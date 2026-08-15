// Package history persists the complete trajectory (messages, tool calls,
// checkpoints) of every session to SQLite. This is distinct from short-term
// memory: short-term is working memory for the LLM (in-process checkpoints),
// history is the durable record for humans / audit / replay (ADR-010).
//
// 单租户（de-tenancy，ADR-011 废止）：tenant_id 列恒保留（见 schema DDL
// NOT NULL DEFAULT）；代码不读不写——INSERT 不提及该列、SELECT 不查。
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

// hasTenantColumnAny 探测 sessions.tenant_id 列是否存在。
func hasTenantColumnAny(db *sql.DB) bool {
	rows, err := db.Query(`PRAGMA table_info(sessions)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull int
		var dflt sql.NullString
		var pk int
		if rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk) == nil && name == "tenant_id" {
			return true
		}
	}
	return false
}

// hasTenantColumnWithoutDefault 探测 sessions.tenant_id 列存在且无默认值
// （de-tenancy 前的老库形态——需要补 DEFAULT 迁移）。
func hasTenantColumnWithoutDefault(db *sql.DB) bool {
	rows, err := db.Query(`PRAGMA table_info(sessions)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull int
		var dflt sql.NullString
		var pk int
		if rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk) == nil &&
			name == "tenant_id" && !dflt.Valid {
			return true
		}
	}
	return false
}

// addTenantColumnDefault 表重建给 tenant_id 补 DEFAULT 'default'
// （列保留、历史租户值原样 copy——只是让无列 INSERT 合法）。幂等。
func addTenantColumnDefault(db *sql.DB) error {
	if _, err := db.Exec(`BEGIN`); err != nil {
		return err
	}
	rollback := func() { _, _ = db.Exec(`ROLLBACK`) }
	stmts := []string{
		`CREATE TABLE sessions_new (
    session_id TEXT PRIMARY KEY,
    tenant_id  TEXT NOT NULL DEFAULT 'default',
    title      TEXT,
    customer   TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)`,
		`INSERT INTO sessions_new(session_id, tenant_id, title, customer, created_at, updated_at)
			SELECT session_id, tenant_id, title, customer, created_at, updated_at FROM sessions`,
		`DROP TABLE sessions`,
		`ALTER TABLE sessions_new RENAME TO sessions`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			rollback()
			return err
		}
	}
	if _, err := db.Exec(`COMMIT`); err != nil {
		rollback()
		return err
	}
	return nil
}

// DB 暴露底层连接（后台任务直查用——Lint 的检索 miss 统计）。
func (s *Store) DB() *sql.DB { return s.db }

// schema 是建表 DDL。
const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    session_id  TEXT PRIMARY KEY,

    title       TEXT,
    customer    TEXT,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL
);


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
	// de-tenancy（目标契约：tenant_id 列保留、代码不读写）：
	// ①列缺失（曾经历删列迁移的中间态库）：ALTER 补列（带 DEFAULT，幂等）；
	// ②列在但 NOT NULL 无默认（de-tenancy 前老库）：表重建补 DEFAULT
	//   （列保留、历史值不动），之后 INSERT 完全不提及该列。均幂等。
	if !hasTenantColumnAny(db) {
		if _, err := db.Exec(`ALTER TABLE sessions ADD COLUMN tenant_id TEXT NOT NULL DEFAULT 'default'`); err != nil {
			// 并发迁移竞态（duplicate column）可容忍；其余错误（磁盘/权限）硬失败
			if !strings.Contains(err.Error(), "duplicate column") {
				_ = db.Close()
				return nil, fmt.Errorf("tenant_id 列补全迁移: %w", err)
			}
		}
	} else if hasTenantColumnWithoutDefault(db) {
		if err := addTenantColumnDefault(db); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("老库 tenant_id 列补默认值迁移: %w", err)
		}
	}
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

// EnsureSession 创建会话记录；若已存在则更新 updated_at/title/customer。
func (s *Store) EnsureSession(sessionID, title, customer string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM sessions WHERE session_id=?`, sessionID).Scan(&exists)
	switch {
	case err == nil:
		// 已存在：更新
		_, err = s.db.Exec(`UPDATE sessions SET updated_at=?,
			title=CASE WHEN ?!='' THEN ? ELSE title END,
			customer=CASE WHEN ?!='' THEN ? ELSE customer END WHERE session_id=?`,
			now, title, title, customer, customer, sessionID)
		return err
	case errors.Is(err, sql.ErrNoRows):
		// 不存在：插入（INSERT 不涉及 tenant_id 列——DDL DEFAULT 自动填，de-tenancy）
		_, err := s.db.Exec(`INSERT INTO sessions(session_id, title, customer, created_at, updated_at)
			VALUES(?,?,?,?,?)`, sessionID, title, customer, now, now)
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

// AppendCheckpoint 追加一个 checkpoint 快照。
func (s *Store) AppendCheckpoint(sessionID string, cp *domain.Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := encodeJSON(cp)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO checkpoints(session_id, cp_id, cp_type, payload_json, created_at)
		VALUES(?, ?, ?, ?, ?)`,
		sessionID, cp.ID, cp.Type, payload, time.Now().UTC())
	if err != nil {
		return err
	}
	// P3 链滚动归档：超软上限时删最老的 followup（保留 initial/reanalysis 骨架；
	// 与内存 Manager.compactChainLocked 同规则，防 restore 后链无界增长）。
	// 注意 MAX(0, COUNT-上限)：SQLite 的 LIMIT 负数=无限制，会把 followup 全删。
	_, err = s.db.Exec(`DELETE FROM checkpoints WHERE rowid IN (
		SELECT rowid FROM checkpoints WHERE session_id=? AND cp_type='followup'
		ORDER BY id ASC LIMIT (
			SELECT MAX(0, (SELECT COUNT(*) FROM checkpoints WHERE session_id=?) - ?)))`,
		sessionID, sessionID, chainSoftLimitSQLite)
	return err
}

// chainSoftLimitSQLite 与 shortterm.chainSoftLimit 保持一致（软上限）。
const chainSoftLimitSQLite = 50

// ListCheckpoints 读回某会话的 checkpoint 链（按创建顺序，用于断点续传）。
// 按会话读回 checkpoint 链（de-tenancy 后无租户维度）。
func (s *Store) ListCheckpoints(sessionID string) ([]*domain.Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT c.payload_json FROM checkpoints c
		JOIN sessions s ON s.session_id = c.session_id
		WHERE c.session_id=? ORDER BY c.id ASC`, sessionID)
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
	Title     string    `json:"title"`
	Customer  string    `json:"customer"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListSessions 列出会话（分页，按更新时间倒序）。
func (s *Store) ListSessions(limit, offset int) ([]SessionListItem, error) {
	if limit <= 0 {
		limit = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT session_id, title, customer, created_at, updated_at
		FROM sessions ORDER BY updated_at DESC LIMIT ? OFFSET ?`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionListItem
	for rows.Next() {
		var it SessionListItem
		if err := rows.Scan(&it.SessionID, &it.Title, &it.Customer, &it.CreatedAt, &it.UpdatedAt); err != nil {
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
	Session     SessionListItem  `json:"session"`
	Messages    []MessageRecord  `json:"messages"`
	ToolCalls   []ToolCallRecord `json:"tool_calls"`
	Checkpoints []CheckpointRec  `json:"checkpoints"` // P4：回放页 checkpoint 行
}

// CheckpointRec 是回放用的 checkpoint 摘要（不含完整 payload）。
type CheckpointRec struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	HasAnalysis bool   `json:"has_analysis"`
	Question    string `json:"question,omitempty"`
	Answer      string `json:"answer,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// MessageHit 是历史检索的一条命中。
type MessageHit struct {
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// SearchMessages 按关键词检索历史消息。
// sessionID 为空时跨该租户全部会话；大小写不敏感的子串匹配（P1 起步——
// 历史量级小，LIKE 足够；后续可升级 bigram 权重检索对齐 longterm）。
func (s *Store) SearchMessages(sessionID, query string, limit int) ([]MessageHit, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT m.session_id, m.role, m.content, m.created_at
		FROM messages m
		WHERE (? = '' OR m.session_id = ?)
			AND m.content LIKE '%' || ? || '%'
		ORDER BY m.created_at DESC LIMIT ?`,
		sessionID, sessionID, query, limit)
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

// SessionSearchHit 是会话搜索结果（标题或内容命中）。
type SessionSearchHit struct {
	SessionID string `json:"session_id"`
	Title     string `json:"title"`
	Customer  string `json:"customer"`
	Snippet   string `json:"snippet"`   // 内容命中片段（标题命中为空）
	Role      string `json:"role"`      // 命中消息角色（user/assistant）
	HitTitle  bool   `json:"hit_title"` // true=标题命中（排序优先）
	UpdatedAt string `json:"updated_at"`
}

// SearchSessions 会话搜索：标题 LIKE 命中（优先）+ 消息内容 LIKE 命中
// （带 snippet，截取关键词前后 30 字符）。按 标题命中>内容命中、更新时间倒序。
func (s *Store) SearchSessions(query string, limit int) ([]SessionSearchHit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	like := "%" + query + "%"
	// 内容命中（每会话取最近一条命中消息）
	rows, err := s.db.Query(`SELECT m.session_id, m.role, m.content, m.created_at,
			COALESCE(ss.title,''), COALESCE(ss.customer,''), COALESCE(ss.updated_at, m.created_at)
		FROM messages m
		JOIN sessions ss ON ss.session_id = m.session_id
		WHERE m.content LIKE ?
		ORDER BY m.created_at DESC`, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type rawHit struct {
		h   SessionSearchHit
		src string
	}
	var contents []rawHit
	seen := map[string]bool{}
	for rows.Next() {
		var h SessionSearchHit
		var content, createdAt string
		if err := rows.Scan(&h.SessionID, &h.Role, &content, &createdAt, &h.Title, &h.Customer, &h.UpdatedAt); err != nil {
			continue
		}
		if seen[h.SessionID] {
			continue // 每会话一条（最近）
		}
		seen[h.SessionID] = true
		h.Snippet = makeSnippet(content, query, 60)
		h.HitTitle = false
		h.UpdatedAt = createdAt
		contents = append(contents, rawHit{h, content})
	}
	// 标题命中
	trows, err := s.db.Query(`SELECT session_id, title, customer, updated_at
		FROM sessions WHERE title LIKE ? ORDER BY updated_at DESC LIMIT ?`, like, limit)
	if err != nil {
		return nil, err
	}
	defer trows.Close()
	titleHits := map[string]bool{}
	var titleList []SessionSearchHit
	for trows.Next() {
		var h SessionSearchHit
		var updated string
		if err := trows.Scan(&h.SessionID, &h.Title, &h.Customer, &updated); err != nil {
			continue
		}
		titleHits[h.SessionID] = true
		h.HitTitle = true
		h.UpdatedAt = updated
		titleList = append(titleList, h)
	}
	// 合并：标题命中在前（其会话若也有内容命中，补 snippet），内容命中排后
	byID := map[string]SessionSearchHit{}
	for _, c := range contents {
		byID[c.h.SessionID] = c.h
	}
	out := titleList
	for i := range out {
		if c, ok := byID[out[i].SessionID]; ok {
			out[i].Snippet = c.Snippet
			out[i].Role = c.Role
		}
	}
	for _, c := range contents {
		if !titleHits[c.h.SessionID] {
			out = append(out, c.h)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// makeSnippet 截取关键词前后文（content 命中片段）。
func makeSnippet(content, query string, width int) string {
	i := strings.Index(content, query)
	if i < 0 {
		if len([]rune(content)) <= width {
			return content
		}
		return string([]rune(content)[:width]) + "…"
	}
	start := i - width/2
	if start < 0 {
		start = 0
	}
	end := start + width + len(query)
	if end > len(content) {
		end = len(content)
		start = end - width - len(query)
		if start < 0 {
			start = 0
		}
	}
	snip := content[start:end]
	if start > 0 {
		snip = "…" + snip
	}
	if end < len(content) {
		snip += "…"
	}
	return strings.ReplaceAll(snip, "\n", " ")
}

// GetSession 返回会话详情。
func (s *Store) GetSession(sessionID string) (*SessionDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var det SessionDetail
	var sess SessionListItem
	err := s.db.QueryRow(`SELECT session_id, title, customer, created_at, updated_at
		FROM sessions WHERE session_id=?`, sessionID).
		Scan(&sess.SessionID, &sess.Title, &sess.Customer, &sess.CreatedAt, &sess.UpdatedAt)
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

	// P4：checkpoint 摘要（回放页时间线行）。已持 s.mu——不调 ListCheckpoints
	// （它也要 Lock），直接轻量查询 payload 自行解。
	crows, cerr := s.db.Query(`SELECT cp_id, cp_type, payload_json, created_at FROM checkpoints
		WHERE session_id=? ORDER BY id ASC`, sessionID)
	if cerr == nil {
		defer crows.Close()
		for crows.Next() {
			var cpID, cpType, payload string
			var createdAt time.Time
			if err := crows.Scan(&cpID, &cpType, &payload, &createdAt); err != nil {
				continue
			}
			var cp domain.Checkpoint
			if json.Unmarshal([]byte(payload), &cp) != nil {
				continue
			}
			det.Checkpoints = append(det.Checkpoints, CheckpointRec{
				ID: cpID, Type: cpType, HasAnalysis: cp.Analysis != nil,
				Question: truncateStr(cp.Question, 60), Answer: truncateStr(cp.Answer, 60),
				CreatedAt: createdAt.UTC().Format(time.RFC3339Nano), // ISO——前端本地化（与消息一致；直传 HH:MM:SS 会把 UTC 当本地显示）
			})
		}
	}
	return &det, nil
}

// truncateStr 截断字符串（checkpoint 摘要用）。
func truncateStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// DeleteSession 删除某会话及其全部轨迹。
func (s *Store) DeleteSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	// 会话存在性校验
	var cnt int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM sessions WHERE session_id=?`, sessionID).Scan(&cnt); err != nil {
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
// 返回删除的消息条数。
func (s *Store) TruncateAfter(sessionID string, after int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 三表联删走单事务（中途失败整体回滚，不产生部分删除）
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }() // 已提交时为 no-op
	if _, err := tx.Exec(`DELETE FROM tool_calls WHERE session_id=? AND message_id IN
		(SELECT id FROM messages WHERE session_id=? AND seq >= ?)`, sessionID, sessionID, after); err != nil {
		return 0, err
	}
	res, err := tx.Exec(`DELETE FROM messages WHERE session_id=? AND seq >= ?`, sessionID, after)
	if err != nil {
		return 0, err
	}
	deleted, _ := res.RowsAffected()
	// checkpoints 是链式的，截断对话后短期记忆整体作废（保守但一致）
	if _, err := tx.Exec(`DELETE FROM checkpoints WHERE session_id=?`, sessionID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int(deleted), nil
}
