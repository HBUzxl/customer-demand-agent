// Package shortterm implements short-term memory as a Checkpoint chain.
//
// 设计参考小米 MiMo Code：维护结构化状态快照，而非对话文本缓冲。
// 这个 Agent 是一次深度分析 + 少量追问，不是聊天机器人——追问时只需
// 读 current checkpoint，不重放整个对话（见 ADR-003）。
//
// 三种 checkpoint：initial（首次分析）/ followup（追问）/ reanalysis（换文档重分析）。
// notes.md 是 Agent 唯一的临时写入通道，创建 checkpoint 时消费并清空。
package shortterm

import (
	"fmt"
	"sync"
	"time"

	"customer-demand-agent/internal/domain"
)

// Manager 管理单个会话的 checkpoint 链 + notes 便签。
// 一个会话对应一个 Manager 实例（由上层 SessionManager 创建）。
type Manager struct {
	mu           sync.Mutex
	chain        []*domain.Checkpoint // 有序链：chain[len-1] 为 current
	notes        []string            // 临时便签
	rawDocument  string             // 当前原始文档
}

// NewManager 创建一个空的短期记忆管理器。
func NewManager() *Manager {
	return &Manager{}
}

// SetDocument 设置当前会话的原始客户文档。
func (m *Manager) SetDocument(doc string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rawDocument = doc
}

// Document 返回当前原始文档。
func (m *Manager) Document() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rawDocument
}

// CreateInitialCheckpoint 创建首次分析 checkpoint。
func (m *Manager) CreateInitialCheckpoint(document string, analysis *domain.AnalysisResult) *domain.Checkpoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rawDocument = document
	cp := &domain.Checkpoint{
		ID:        newCheckpointID("initial"),
		Type:      domain.CheckpointInitial,
		Document:  document,
		Analysis:  analysis,
		Notes:     m.drainNotesLocked(),
		CreatedAt: time.Now(),
	}
	m.chain = append(m.chain, cp)
	return cp
}

// CreateFollowupCheckpoint 创建追问 checkpoint，链在前一个之后。
func (m *Manager) CreateFollowupCheckpoint(question, answer string) *domain.Checkpoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	prevID := ""
	if len(m.chain) > 0 {
		prevID = m.chain[len(m.chain)-1].ID
	}
	cp := &domain.Checkpoint{
		ID:        newCheckpointID("followup"),
		Type:      domain.CheckpointFollowup,
		PrevID:    prevID,
		Question:  question,
		Answer:    answer,
		Notes:     m.drainNotesLocked(),
		CreatedAt: time.Now(),
	}
	m.chain = append(m.chain, cp)
	return cp
}

// CreateReanalysisCheckpoint 创建重新分析 checkpoint（换文档）。
func (m *Manager) CreateReanalysisCheckpoint(document string, analysis *domain.AnalysisResult) *domain.Checkpoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	prevID := ""
	if len(m.chain) > 0 {
		prevID = m.chain[len(m.chain)-1].ID
	}
	m.rawDocument = document
	cp := &domain.Checkpoint{
		ID:        newCheckpointID("reanalysis"),
		Type:      domain.CheckpointReanalysis,
		PrevID:    prevID,
		Document:  document,
		Analysis:  analysis,
		Notes:     m.drainNotesLocked(),
		CreatedAt: time.Now(),
	}
	m.chain = append(m.chain, cp)
	return cp
}

// CurrentCheckpoint 返回链尾 checkpoint（无则 nil）。
func (m *Manager) CurrentCheckpoint() *domain.Checkpoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.chain) == 0 {
		return nil
	}
	return m.chain[len(m.chain)-1]
}

// PreviousCheckpoint 返回倒数第二个 checkpoint（无则 nil）。
func (m *Manager) PreviousCheckpoint() *domain.Checkpoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.chain) < 2 {
		return nil
	}
	return m.chain[len(m.chain)-2]
}

// CheckpointChain 返回整条链的副本。
func (m *Manager) CheckpointChain() []*domain.Checkpoint {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*domain.Checkpoint, len(m.chain))
	copy(out, m.chain)
	return out
}

// HasHistory 是否已有 checkpoint（用于判断 initial vs followup）。
func (m *Manager) HasHistory() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.chain) > 0
}

// ── Notes 便签（Agent 唯一临时写入通道）────────────────────────

// AppendNote 追加一条临时观察。
func (m *Manager) AppendNote(note string) {
	if note = trim(note); note == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notes = append(m.notes, note)
}

// DrainNotes 取出并清空全部便签。
func (m *Manager) DrainNotes() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.drainNotesLocked()
}

func (m *Manager) drainNotesLocked() []string {
	if len(m.notes) == 0 {
		return nil
	}
	out := m.notes
	m.notes = nil
	return out
}

// ── BuildContext（供 assembler 使用）───────────────────────────

// BuildContext 根据 op 类型构建会话上下文。
func (m *Manager) BuildContext(op domain.CheckpointOp) *domain.SessionContext {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx := &domain.SessionContext{Op: op}
	if len(m.chain) > 0 {
		ctx.Current = m.chain[len(m.chain)-1]
	}
	if len(m.chain) >= 2 {
		ctx.Previous = m.chain[len(m.chain)-2]
	}
	// 文档摘要：优先用最新文档的开头
	if m.rawDocument != "" {
		ctx.DocumentSummary = summarizeDoc(m.rawDocument)
	} else if ctx.Current != nil && ctx.Current.Document != "" {
		ctx.DocumentSummary = summarizeDoc(ctx.Current.Document)
	}
	return ctx
}

// DetermineOp 根据是否有历史 + 是否换文档，决定拼装 op。
func DetermineOp(m *Manager, incoming string) domain.CheckpointOp {
	if m == nil || !m.HasHistory() {
		return domain.OpInitial
	}
	// 有历史但文档变了 → reanalysis；否则 followup
	doc := m.Document()
	if doc != "" && incoming != "" && !sameDoc(doc, incoming) {
		return domain.OpReanalysis
	}
	return domain.OpFollowup
}

// summarizeDoc 取文档前 200 字作为摘要。
func summarizeDoc(doc string) string {
	r := []rune(doc)
	if len(r) > 200 {
		return string(r[:200]) + "…"
	}
	return doc
}

// sameDoc 简易判断两段文本是否同一文档（前 100 字一致）。
func sameDoc(a, b string) bool {
	ra, rb := []rune(a), []rune(b)
	if len(ra) > 100 {
		ra = ra[:100]
	}
	if len(rb) > 100 {
		rb = rb[:100]
	}
	return string(ra) == string(rb)
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\n' || s[len(s)-1] == '\t' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}

func newCheckpointID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// SessionManager 管理多会话的短期记忆（按 sessionID 隔离）。
type SessionManager struct {
	mu       sync.Mutex
	sessions map[string]*Manager
}

// NewSessionManager 创建会话级短期记忆管理器。
func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]*Manager)}
}

// Get 获取或创建某会话的 Manager。
func (sm *SessionManager) Get(sessionID string) *Manager {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	m, ok := sm.sessions[sessionID]
	if !ok {
		m = NewManager()
		sm.sessions[sessionID] = m
	}
	return m
}

// Delete 丢弃某会话的短期记忆。
func (sm *SessionManager) Delete(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, sessionID)
}
