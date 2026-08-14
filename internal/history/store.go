// Package history persists the complete trajectory (messages, tool calls,
// checkpoints) of every session to SQLite. This is distinct from short-term
// memory: short-term is working memory for the LLM (in-process checkpoints),
// history is the durable record for humans / audit / replay (ADR-010).
//
// 多租户：tenant_id 隔离 sessions 与 messages，产品/行业/客户记忆为
// 组织级共享知识（不在此隔离，见 ADR-011）。
package history

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"customer-demand-agent/internal/domain"
)

// Store 是 SQLite 历史记录存储。
type Store struct {
	mu sync.Mutex
	db *sql.DB
}

// schema 是建表 DDL。
const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    session_id  TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL,
    title       TEXT,
    customer    TEXT,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_tenant ON sessions(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS messages (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id   TEXT NOT NULL,
    role         TEXT NOT NULL,
    content      TEXT,
    tool_call_id TEXT,
    seq          INTEGER NOT NULL,
    created_at   TIMESTAMP NOT NULL,
    FOREIGN KEY(session_id) REFERENCES sessions(session_id)
);
CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, seq);

CREATE TABLE IF NOT EXISTS tool_calls (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id   TEXT NOT NULL,
    message_id   INTEGER,           -- 归属的 assistant 消息 id（回放归属边界）
    tool_name    TEXT NOT NULL,
    params_json  TEXT,
    result_json  TEXT,
    seq          INTEGER NOT NULL,
    created_at   TIMESTAMP NOT NULL,
    FOREIGN KEY(session_id) REFERENCES sessions(session_id)
);
CREATE INDEX IF NOT EXISTS idx_toolcalls_session ON tool_calls(session_id, seq);

CREATE TABLE IF NOT EXISTS checkpoints (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id   TEXT NOT NULL,
    cp_id        TEXT NOT NULL,
    cp_type      TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    created_at   TIMESTAMP NOT NULL,
    FOREIGN KEY(session_id) REFERENCES sessions(session_id)
);
CREATE INDEX IF NOT EXISTS idx_checkpoints_session ON checkpoints(session_id, id);
`

// Open 打开（或创建）历史数据库并建表。
func Open(dbPath string) (*Store, error) {
	// 确保父目录存在（modernc/sqlite 不会自动创建）
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建 history 目录: %w", err)
		}
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("打开 history db: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping history db: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("建表: %w", err)
	}
	// 轻量迁移：老库的 tool_calls 没有 message_id 列（回放归属边界，2026-08-14）。
	if _, err := db.Exec(`ALTER TABLE tool_calls ADD COLUMN message_id INTEGER`); err != nil {
		// column already exists → 忽略；其它错误才致命
		if !strings.Contains(err.Error(), "duplicate column") {
			_ = db.Close()
			return nil, fmt.Errorf("迁移 tool_calls.message_id: %w", err)
		}
	}
	return &Store{db: db}, nil
}

// Close 关闭数据库。
func (s *Store) Close() error {
	return s.db.Close()
}

// EnsureSession 创建会话记录；若已存在则校验 tenant 归属后更新 updated_at/title。
// 跨租户：若 session 已存在但属于别的 tenant，拒绝（防跨租户占用/续传/读 checkpoint）。
func (s *Store) EnsureSession(tenantID, sessionID, title, customer string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	var owner string
	err := s.db.QueryRow(`SELECT tenant_id FROM sessions WHERE session_id=?`, sessionID).Scan(&owner)
	switch {
	case err == nil:
		// 已存在：校验归属
		if owner != tenantID {
			return fmt.Errorf("会话 %s 不属于租户 %s", sessionID, tenantID)
		}
		_, err = s.db.Exec(`UPDATE sessions SET updated_at=?,
			title=CASE WHEN ?!='' THEN ? ELSE title END,
			customer=CASE WHEN ?!='' THEN ? ELSE customer END WHERE session_id=?`,
			now, title, title, customer, customer, sessionID)
		return err
	case errors.Is(err, sql.ErrNoRows):
		// 不存在：插入
		_, err = s.db.Exec(`INSERT INTO sessions(session_id, tenant_id, title, customer, created_at, updated_at)
			VALUES(?,?,?,?,?,?)`, sessionID, tenantID, title, customer, now, now)
		return err
	default:
		return err
	}
}

// AppendMessage 追加一条消息，返回自增 id。
func (s *Store) AppendMessage(sessionID, role, content, toolCallID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seq, err := s.nextSeq("messages", sessionID)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO messages(session_id, role, content, tool_call_id, seq, created_at)
		VALUES(?, ?, ?, ?, ?, ?)`,
		sessionID, role, content, toolCallID, seq, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AppendToolCall 追加一次工具调用记录。messageID 是归属的 assistant 消息 id
// （回放归属边界；0 表示未关联——向后兼容）。
func (s *Store) AppendToolCall(sessionID string, messageID int64, toolName, params, result string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seq, err := s.nextSeq("tool_calls", sessionID)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO tool_calls(session_id, message_id, tool_name, params_json, result_json, seq, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)`,
		sessionID, messageID, toolName, params, result, seq, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// AppendCheckpoint 追加一个 checkpoint 快照（校验 tenant 归属）。
func (s *Store) AppendCheckpoint(tenantID, sessionID string, cp *domain.Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var cnt int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE session_id=? AND tenant_id=?`, sessionID, tenantID).Scan(&cnt); err != nil {
		return err
	}
	if cnt == 0 {
		return fmt.Errorf("会话 %s 不属于租户 %s，拒绝写入 checkpoint", sessionID, tenantID)
	}
	payload, err := encodeJSON(cp)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO checkpoints(session_id, cp_id, cp_type, payload_json, created_at)
		VALUES(?, ?, ?, ?, ?)`,
		sessionID, cp.ID, cp.Type, payload, time.Now().UTC())
	return err
}

// ListCheckpoints 读回某会话的 checkpoint 链（按创建顺序，用于断点续传）。
// 跨租户：JOIN sessions 校验归属，非本租户会话返回空（读不到对方 checkpoint）。
func (s *Store) ListCheckpoints(tenantID, sessionID string) ([]*domain.Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT c.payload_json FROM checkpoints c
		JOIN sessions s ON s.session_id = c.session_id
		WHERE c.session_id=? AND s.tenant_id=? ORDER BY c.id ASC`, sessionID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Checkpoint
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var cp domain.Checkpoint
		if err := json.Unmarshal([]byte(payload), &cp); err != nil {
			continue // 跳过损坏的 checkpoint
		}
		out = append(out, &cp)
	}
	return out, rows.Err()
}

// SessionListItem 是会话列表的一项。
type SessionListItem struct {
	SessionID string    `json:"session_id"`
	TenantID  string    `json:"tenant_id"`
	Title     string    `json:"title"`
	Customer  string    `json:"customer"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListSessions 列出某 tenant 的会话（分页，按更新时间倒序）。
func (s *Store) ListSessions(tenantID string, limit, offset int) ([]SessionListItem, error) {
	if limit <= 0 {
		limit = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT session_id, tenant_id, title, customer, created_at, updated_at
		FROM sessions WHERE tenant_id=? ORDER BY updated_at DESC LIMIT ? OFFSET ?`,
		tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionListItem
	for rows.Next() {
		var it SessionListItem
		if err := rows.Scan(&it.SessionID, &it.TenantID, &it.Title, &it.Customer, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// MessageRecord 是一条历史消息。
type MessageRecord struct {
	ID         int64     `json:"id"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	ToolCallID string    `json:"tool_call_id,omitempty"`
	Seq        int       `json:"seq"`
	CreatedAt  time.Time `json:"created_at"`
}

// ToolCallRecord 是一条工具调用历史。
type ToolCallRecord struct {
	ID        int64     `json:"id"`
	MessageID int64     `json:"message_id"` // 归属的 assistant 消息 id（回放归属边界）
	ToolName  string    `json:"tool_name"`
	Params    string    `json:"params"`
	Result    string    `json:"result"`
	Seq       int       `json:"seq"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionDetail 是会话详情（含完整轨迹，供回放）。
type SessionDetail struct {
	Session   SessionListItem  `json:"session"`
	Messages  []MessageRecord  `json:"messages"`
	ToolCalls []ToolCallRecord `json:"tool_calls"`
}

// MessageHit 是历史检索的一条命中。
type MessageHit struct {
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// SearchMessages 按关键词检索历史消息（tenant 隔离）。
// sessionID 为空时跨该租户全部会话；大小写不敏感的子串匹配（P1 起步——
// 历史量级小，LIKE 足够；后续可升级 bigram 权重检索对齐 longterm）。
func (s *Store) SearchMessages(tenantID, sessionID, query string, limit int) ([]MessageHit, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT m.session_id, m.role, m.content, m.created_at
		FROM messages m JOIN sessions se ON se.session_id = m.session_id
		WHERE se.tenant_id = ? AND (? = '' OR m.session_id = ?)
			AND m.content LIKE '%' || ? || '%'
		ORDER BY m.created_at DESC LIMIT ?`,
		tenantID, sessionID, sessionID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MessageHit
	for rows.Next() {
		var h MessageHit
		if err := rows.Scan(&h.SessionID, &h.Role, &h.Content, &h.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, nil
}

// GetSession 返回会话详情（校验 tenant 归属）。
func (s *Store) GetSession(tenantID, sessionID string) (*SessionDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var det SessionDetail
	var sess SessionListItem
	err := s.db.QueryRow(`SELECT session_id, tenant_id, title, customer, created_at, updated_at
		FROM sessions WHERE session_id=? AND tenant_id=?`, sessionID, tenantID).
		Scan(&sess.SessionID, &sess.TenantID, &sess.Title, &sess.Customer, &sess.CreatedAt, &sess.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("会话不存在或无权访问: %w", err)
	}
	det.Session = sess

	rows, err := s.db.Query(`SELECT id, role, content, tool_call_id, seq, created_at FROM messages
		WHERE session_id=? ORDER BY seq`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var m MessageRecord
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.ToolCallID, &m.Seq, &m.CreatedAt); err != nil {
			return nil, err
		}
		det.Messages = append(det.Messages, m)
	}

	rows2, err := s.db.Query(`SELECT id, COALESCE(message_id, 0), tool_name, params_json, result_json, seq, created_at FROM tool_calls
		WHERE session_id=? ORDER BY seq`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var t ToolCallRecord
		if err := rows2.Scan(&t.ID, &t.MessageID, &t.ToolName, &t.Params, &t.Result, &t.Seq, &t.CreatedAt); err != nil {
			return nil, err
		}
		det.ToolCalls = append(det.ToolCalls, t)
	}
	return &det, nil
}

// DeleteSession 删除某会话及其全部轨迹（校验 tenant）。
func (s *Store) DeleteSession(tenantID, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	// 先校验 tenant 归属
	var cnt int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM sessions WHERE session_id=? AND tenant_id=?`, sessionID, tenantID).Scan(&cnt); err != nil {
		_ = tx.Rollback()
		return err
	}
	if cnt == 0 {
		_ = tx.Rollback()
		return fmt.Errorf("会话不存在或无权访问")
	}
	for _, q := range []string{
		`DELETE FROM messages WHERE session_id=?`,
		`DELETE FROM tool_calls WHERE session_id=?`,
		`DELETE FROM checkpoints WHERE session_id=?`,
		`DELETE FROM sessions WHERE session_id=?`,
	} {
		if _, err := tx.Exec(q, sessionID); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// nextSeq 取某表某会话的下一个序号。
func (s *Store) nextSeq(table, sessionID string) (int, error) {
	var seq sql.NullInt64
	q := fmt.Sprintf(`SELECT COALESCE(MAX(seq),0)+1 FROM %s WHERE session_id=?`, table)
	if err := s.db.QueryRow(q, sessionID).Scan(&seq); err != nil {
		return 0, err
	}
	return int(seq.Int64), nil
}

func encodeJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// TruncateAfter 删除会话中 seq >= after 的消息及其关联 tool_calls，
// 并清空该会话全部 checkpoints（F1 编辑重发：截断后重发）。
// 返回删除的消息条数。校验 tenant 归属。
func (s *Store) TruncateAfter(tenantID, sessionID string, after int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// tenant 归属校验（与 ListCheckpoints 同款 JOIN 检查）
	var owner string
	err := s.db.QueryRow(`SELECT tenant_id FROM sessions WHERE session_id=?`, sessionID).Scan(&owner)
	if err != nil || owner != tenantID {
		return 0, fmt.Errorf("无权操作该会话")
	}
	// 该 seq 起的消息对应的 tool_calls（message_id 关联）
	_, err = s.db.Exec(`DELETE FROM tool_calls WHERE session_id=? AND message_id IN
		(SELECT id FROM messages WHERE session_id=? AND seq >= ?)`, sessionID, sessionID, after)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`DELETE FROM messages WHERE session_id=? AND seq >= ?`, sessionID, after)
	if err != nil {
		return 0, err
	}
	deleted, _ := res.RowsAffected()
	// checkpoints 是链式的，截断对话后短期记忆整体作废（保守但一致）
	if _, err := s.db.Exec(`DELETE FROM checkpoints WHERE session_id=?`, sessionID); err != nil {
		return 0, err
	}
	return int(deleted), nil
}
