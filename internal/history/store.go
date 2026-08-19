// Package history persists the complete trajectory (messages, tool calls,
// checkpoints) of every session to SQLite. This is distinct from short-term
// memory: short-term is working memory for the LLM (in-process checkpoints),
// history is the durable record for humans / audit / replay (ADR-010).
//
// 多租户（ADR-017）：每个租户独立 history.db（tenants/<id>/history.db），
// Store 实例本身即租户边界——同一实例上的 session_id 自然属于该租户。
// tenant_meta 表三方核对（文件内置 tenant_id 与构造传入一致，防路径错配）。
// sessions.owner_user_id/visibility/deleted_at 支撑租户内用户级可见性。
package history

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"customer-demand-agent/internal/domain"
)

// Store 是 SQLite 历史记录存储。tenantID 绑定构造传入的租户（OpenTenant）；
// Open 构造的 legacy Store tenantID 为空（迁移/单租户测试形态）。
type Store struct {
	mu       sync.Mutex
	db       *sql.DB
	tenantID string
}

// hasColumn 探测表某列是否存在（中间态库兼容，老库无 branch_id/tenant_id 时退化）。
func hasColumn(db *sql.DB, table, col string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, dflt, pk sql.NullString
		var dfltVal sql.NullString
		_ = dfltVal
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == col {
			return true, nil
		}
	}
	return false, nil
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
// 注意：重建期间必须临时关闭 foreign_keys（DROP 父表会触发对子行的 FK 校验）。
func addTenantColumnDefault(db *sql.DB) error {
	// PRAGMA foreign_keys 在事务内是 no-op，必须在 BEGIN 之前切换。
	if _, err := db.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	defer func() { _, _ = db.Exec(`PRAGMA foreign_keys=ON`) }()
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

// TenantID 返回 Store 绑定的租户 id（Open 构造的 legacy Store 返回空串）。
func (s *Store) TenantID() string { return s.tenantID }

// RecentMissQueries 收集近期 memory_search 零命中查询（Lint 的 miss 统计输入）。
// 受当前 Store（=租户）作用域约束，不再暴露裸 DB 连接。
func (s *Store) RecentMissQueries(limit int) []string {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var out []string
	rows, err := s.db.Query(`SELECT params_json FROM tool_calls
		WHERE tool_name='memory_search' AND result_json LIKE '%"count":0%'
		ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var params string
		if rows.Scan(&params) != nil {
			continue
		}
		var p struct {
			Query string `json:"query"`
		}
		if json.Unmarshal([]byte(params), &p) == nil && p.Query != "" {
			out = append(out, p.Query)
		}
	}
	return out
}

// verifyForeignKeys 自检 SQLite 连接是否已启用外键约束（多租户隔离依赖）。
func verifyForeignKeys(db *sql.DB) error {
	var fk int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		return fmt.Errorf("检查 foreign_keys 失败: %w", err)
	}
	if fk != 1 {
		return fmt.Errorf("SQLite foreign_keys 未启用（fk=%d）——拒绝启动，防止隔离静默失效", fk)
	}
	return nil
}

// schema 是建表 DDL。
const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    session_id  TEXT PRIMARY KEY,

    title        TEXT,
    customer     TEXT,
    title_pinned INTEGER NOT NULL DEFAULT 0, -- 用户手动重命名=1，防自动标题覆盖（2026-08-18）
    created_at   TIMESTAMP NOT NULL,
    updated_at   TIMESTAMP NOT NULL
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
    branch_id    TEXT NOT NULL DEFAULT 'main', -- P0-04：分叉持久化（main=主线，bX-ts=分支）
    last_seq     INTEGER NOT NULL DEFAULT 0,   -- P0-04：该 checkpoint 对应轮次的消息 seq（分叉前缀判定）
    cp_id        TEXT NOT NULL,
    cp_type      TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    created_at   TIMESTAMP NOT NULL,
    FOREIGN KEY(session_id) REFERENCES sessions(session_id)
);
CREATE INDEX IF NOT EXISTS idx_checkpoints_session ON checkpoints(session_id, branch_id, id);
CREATE INDEX IF NOT EXISTS idx_checkpoints_session ON checkpoints(session_id, id);

CREATE TABLE IF NOT EXISTS tenant_meta (
    tenant_id  TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS customer_refs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    customer   TEXT NOT NULL,
    ref_type   TEXT NOT NULL DEFAULT 'session',
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY(session_id) REFERENCES sessions(session_id)
);
CREATE INDEX IF NOT EXISTS idx_customer_refs_session ON customer_refs(session_id);

CREATE TABLE IF NOT EXISTS audit_logs (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    ts            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor_user_id TEXT,
    action        TEXT NOT NULL,
    obj_type      TEXT,
    obj_id        TEXT,
    result        TEXT,
    ip            TEXT,
    ua            TEXT
);
CREATE INDEX IF NOT EXISTS idx_audit_logs_ts ON audit_logs(ts);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

// Open 打开（或创建）历史数据库并建表（legacy 单租户形态，tenantID 为空）。
// 多租户路径请用 OpenTenant。
func Open(dbPath string) (*Store, error) {
	return open(dbPath, "")
}

// OpenTenant 打开指定租户的独立历史库：构造即绑定 tenantID，并在 tenant_meta
// 表中核对一致性（文件内置租户 id 与传入一致；缺失则写入，防止路径错配/串库）。
func OpenTenant(tenantID, dbPath string) (*Store, error) {
	if tenantID == "" {
		return nil, errors.New("OpenTenant 要求非空 tenantID")
	}
	return open(dbPath, tenantID)
}

func open(dbPath, tenantID string) (*Store, error) {
	// 确保父目录存在（modernc/sqlite 不会自动创建）
	if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建 history 目录: %w", err)
		}
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("打开 history db: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping history db: %w", err)
	}
	// SQLite 外键默认不自动强制执行（多租户隔离纵深依赖此约束）。显式自检：
	// 本连接必须返回 1，否则拒绝启动（防止 DSN 改动导致隔离静默失效）。
	if err := verifyForeignKeys(db); err != nil {
		_ = db.Close()
		return nil, err
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
	// checkpoint-tree：messages.branch_id（编辑重发=开分支，旧消息不删挂新分支）
	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN branch_id TEXT NOT NULL DEFAULT 'main'`); err != nil {
		if !strings.Contains(err.Error(), "duplicate column") {
			_ = db.Close()
			return nil, fmt.Errorf("messages.branch_id 迁移: %w", err)
		}
	}
	if _, err := db.Exec(`ALTER TABLE tool_calls ADD COLUMN message_id INTEGER`); err != nil {
		// column already exists → 忽略；其它错误才致命
		if !strings.Contains(err.Error(), "duplicate column") {
			_ = db.Close()
			return nil, fmt.Errorf("迁移 tool_calls.message_id: %w", err)
		}
	}
	// P0-04：checkpoints 分叉持久化（老库补 branch_id/last_seq；新库建表已含）。
	for col, ddl := range map[string]string{
		"branch_id": `ALTER TABLE checkpoints ADD COLUMN branch_id TEXT NOT NULL DEFAULT 'main'`,
		"last_seq":  `ALTER TABLE checkpoints ADD COLUMN last_seq INTEGER NOT NULL DEFAULT 0`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				_ = db.Close()
				return nil, fmt.Errorf("迁移 checkpoints.%s: %w", col, err)
			}
		}
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_checkpoints_session ON checkpoints(session_id, branch_id, id)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("checkpoints 分支索引: %w", err)
	}
	// 多租户会话级授权列（幂等 ALTER）。
	for col, ddl := range map[string]string{
		"owner_user_id": `ALTER TABLE sessions ADD COLUMN owner_user_id TEXT`,
		"customer_ref":  `ALTER TABLE sessions ADD COLUMN customer_ref TEXT`,
		"visibility":    `ALTER TABLE sessions ADD COLUMN visibility TEXT NOT NULL DEFAULT 'private'`,
		"deleted_at":    `ALTER TABLE sessions ADD COLUMN deleted_at TIMESTAMP`,
		"title_pinned":  `ALTER TABLE sessions ADD COLUMN title_pinned INTEGER NOT NULL DEFAULT 0`, // 手动重命名防覆盖
	} {
		if _, err := db.Exec(ddl); err != nil {
			if !strings.Contains(err.Error(), "duplicate column") {
				_ = db.Close()
				return nil, fmt.Errorf("迁移 sessions.%s: %w", col, err)
			}
		}
	}
	// 租户三方核对：内置 tenant_meta 必须与构造 tenantID 一致（防止租户 A 误连 B 的库）。
	if tenantID != "" {
		var got string
		err := db.QueryRow(`SELECT tenant_id FROM tenant_meta LIMIT 1`).Scan(&got)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			if _, err := db.Exec(`INSERT INTO tenant_meta(tenant_id) VALUES(?)`, tenantID); err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("写入 tenant_meta: %w", err)
			}
		case err != nil:
			_ = db.Close()
			return nil, fmt.Errorf("读取 tenant_meta: %w", err)
		case got != tenantID:
			_ = db.Close()
			return nil, fmt.Errorf("tenant_meta 不匹配：文件=%s 构造=%s（禁止串库）", got, tenantID)
		}
	}
	return &Store{db: db, tenantID: tenantID}, nil
}

// Close 关闭数据库。
func (s *Store) Close() error {
	return s.db.Close()
}

// EnsureSession 创建会话记录（ownerUserID 为会话归属人；若已存在则更新
// updated_at/title/customer）。ownerUserID 为空表示无归属（Agent 内部回调——
// 租户内 admin/owner 可见）。title 为空串时不覆盖已有标题。
func (s *Store) EnsureSession(ownerUserID, sessionID, title, customer string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM sessions WHERE session_id=?`, sessionID).Scan(&exists)
	switch {
	case err == nil:
		// 已存在：更新。title 仅在非空且未被用户手动固定（title_pinned=1）时覆盖，
		// 避免 AI 自动标题/首行标题覆盖用户手动重命名（2026-08-18）。
		_, err = s.db.Exec(`UPDATE sessions SET updated_at=?,
			title=CASE WHEN ?!='' AND title_pinned IS NOT 1 THEN ? ELSE title END,
			customer=CASE WHEN ?!='' THEN ? ELSE customer END,
			owner_user_id=CASE WHEN ?='' THEN owner_user_id ELSE ? END
			WHERE session_id=?`,
			now, title, title, customer, customer, ownerUserID, ownerUserID, sessionID)
		return err
	case errors.Is(err, sql.ErrNoRows):
		// 不存在：插入（tenant_id 列由 DDL DEFAULT 自动填——OpenTenant 已核对租户）
		_, err := s.db.Exec(`INSERT INTO sessions(session_id, owner_user_id, title, customer, created_at, updated_at)
			VALUES(?,?,?,?,?,?)`, sessionID, ownerUserID, title, customer, now, now)
		return err
	default:
		return err
	}
}

// EnsureSessionScoped 以登录作用域创建或更新会话。与先查存在性再调用
// EnsureSession 不同，本方法在同一临界区内校验 owner_user_id，防止同租户的
// 普通成员仅凭猜中 session_id 就把他人会话改挂到自己名下。
//
// owner/admin 可维护租户内会话，但不会改变原 owner；普通成员只能维护自己或
// 历史无归属会话。已删除、不可见和不存在性对外统一由调用方映射为安全错误。
func (s *Store) EnsureSessionScoped(scope *domain.TenantScope, sessionID, title, customer string) error {
	if scope == nil || !scope.Valid() {
		return fmt.Errorf("会话不存在或无权访问")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	var owner sql.NullString
	var deletedAt sql.NullTime
	err := s.db.QueryRow(`SELECT owner_user_id, deleted_at FROM sessions WHERE session_id=?`, sessionID).Scan(&owner, &deletedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = s.db.Exec(`INSERT INTO sessions(session_id, owner_user_id, title, customer, created_at, updated_at)
			VALUES(?,?,?,?,?,?)`, sessionID, scope.UserID, title, customer, now, now)
		return err
	case err != nil:
		return err
	case deletedAt.Valid:
		return fmt.Errorf("会话不存在或无权访问")
	case !scope.IsTenantAdmin() && owner.String != "" && owner.String != scope.UserID:
		return fmt.Errorf("会话不存在或无权访问")
	}

	// 已有会话只更新内容属性，绝不重写 owner_user_id。
	_, err = s.db.Exec(`UPDATE sessions SET updated_at=?,
		title=CASE WHEN ?!='' AND title_pinned IS NOT 1 THEN ? ELSE title END,
		customer=CASE WHEN ?!='' THEN ? ELSE customer END
		WHERE session_id=? AND deleted_at IS NULL`,
		now, title, title, customer, customer, sessionID)
	return err
}

// BindCustomerOnce 给尚未归属客户的会话补绑定；已有相同归属视为幂等成功，
// 已绑定到其他客户则拒绝。它供 Agent 的 session_bind_customer 使用，避免
// 模型在后续轮次把整段历史对话跨客户重挂。
func (s *Store) BindCustomerOnce(sessionID, customer string) error {
	customer = strings.TrimSpace(customer)
	if customer == "" {
		return fmt.Errorf("customer 不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`UPDATE sessions SET customer=?, updated_at=?
		WHERE session_id=? AND deleted_at IS NULL AND COALESCE(customer,'')=''`,
		customer, time.Now().UTC(), sessionID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		return nil
	}
	var current string
	if err := s.db.QueryRow(`SELECT COALESCE(customer,'') FROM sessions
		WHERE session_id=? AND deleted_at IS NULL`, sessionID).Scan(&current); err != nil {
		return err
	}
	if strings.TrimSpace(current) == customer {
		return nil
	}
	return fmt.Errorf("会话已归属客户 %q，不允许改绑", current)
}

// SessionExists 判断会话是否存在（轻量存在性检查，不加载消息；调用方负责 scope）。
func (s *Store) SessionExists(sessionID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM sessions WHERE session_id=?`, sessionID).Scan(&one)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	default:
		return false, err
	}
}

// IsSessionTitlePinned 判断会话标题是否被用户手动重命名固定（title_pinned=1）。
// 固定后 AI 自动标题不再覆盖（G4）。
func (s *Store) IsSessionTitlePinned(sessionID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var pinned int
	if err := s.db.QueryRow(`SELECT title_pinned FROM sessions WHERE session_id=?`, sessionID).Scan(&pinned); err != nil {
		return false, err
	}
	return pinned == 1, nil
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

// AppendCheckpoint 追加一个 checkpoint 快照（P0-04：按分支持久化；branch 为空
// 视为 main）。last_seq 记录该 checkpoint 对应轮次的当前分支最大消息 seq——
// 分叉后前缀判定用：cp.last_seq < 分支点 ⇒ 属于共享前缀。
func (s *Store) AppendCheckpoint(sessionID, branch string, cp *domain.Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if branch == "" {
		branch = "main"
	}
	payload, err := encodeJSON(cp)
	if err != nil {
		return err
	}
	var lastSeq int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(seq),0) FROM messages WHERE session_id=? AND branch_id=?`,
		sessionID, branch).Scan(&lastSeq); err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO checkpoints(session_id, branch_id, last_seq, cp_id, cp_type, payload_json, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)`,
		sessionID, branch, lastSeq, cp.ID, cp.Type, payload, time.Now().UTC())
	if err != nil {
		return err
	}
	// P3 链滚动归档：超软上限时删最老的 followup（保留 initial/reanalysis 骨架；
	// 与内存 Manager.compactChainLocked 同规则，防 restore 后链无界增长）。
	// 注意 MAX(0, COUNT-上限)：SQLite 的 LIMIT 负数=无限制，会把 followup 全删。
	_, err = s.db.Exec(`DELETE FROM checkpoints WHERE rowid IN (
		SELECT rowid FROM checkpoints WHERE session_id=? AND branch_id=? AND cp_type='followup'
		ORDER BY id ASC LIMIT (
			SELECT MAX(0, (SELECT COUNT(*) FROM checkpoints WHERE session_id=? AND branch_id=?) - ?)))`,
		sessionID, branch, sessionID, branch, chainSoftLimitSQLite)
	return err
}

// chainSoftLimitSQLite 与 shortterm.chainSoftLimit 保持一致（软上限）。
const chainSoftLimitSQLite = 50

// ListCheckpoints 读回某会话某分支的 checkpoint 链（按创建顺序，断点续传用）。
// 分支记忆 = 共享前缀（main 上 last_seq < 分支点的 checkpoint）+ 该分支自己的
// checkpoint（P0-04）；main 直接取全链。branch 为空视为 main。
func (s *Store) ListCheckpoints(sessionID, branch string) ([]*domain.Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	hasBranch, _ := hasColumn(s.db, "checkpoints", "branch_id")
	if !hasBranch {
		rows, err := s.db.Query(`SELECT c.payload_json FROM checkpoints c
			JOIN sessions s ON s.session_id = c.session_id
			WHERE c.session_id=? ORDER BY c.id ASC`, sessionID)
		if err != nil {
			return nil, err
		}
		return scanCheckpoints(rows)
	}
	if branch == "" || branch == "main" {
		// main：只看主线 checkpoint（兄弟分支的 checkpoint 不回灌主视图）。
		rows, err := s.db.Query(`SELECT c.payload_json FROM checkpoints c
			JOIN sessions s ON s.session_id = c.session_id
			WHERE c.session_id=? AND c.branch_id='main' ORDER BY c.id ASC`, sessionID)
		if err != nil {
			return nil, err
		}
		return scanCheckpoints(rows)
	}
	// 分支：前缀（main 且 last_seq < 分支点）+ 分支自身 checkpoint，按 id 序。
	after := branchPoint(branch)
	q := `SELECT c.payload_json FROM checkpoints c
		JOIN sessions s ON s.session_id = c.session_id
		WHERE c.session_id=? AND (
			(c.branch_id='main' AND c.last_seq < ?) OR c.branch_id=?
		) ORDER BY c.id ASC`
	rows, err := s.db.Query(q, sessionID, after, branch)
	if err != nil {
		return nil, err
	}
	return scanCheckpoints(rows)
}

// scanCheckpoints 遍历结果集解出 checkpoint 链。
func scanCheckpoints(rows *sql.Rows) ([]*domain.Checkpoint, error) {
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

// branchPoint 从分支 ID 解析分支点（b{after}-{ts} → after；main/未知形态 → 0）。
func branchPoint(branch string) int {
	// 分支 ID 形如 "b5-1755..."：b 后数字即分支点（截断的 message seq）。
	if strings.HasPrefix(branch, "b") {
		if dash := strings.IndexByte(branch, '-'); dash > 1 {
			if n, err := strconv.Atoi(branch[1:dash]); err == nil {
				return n
			}
		}
	}
	return 0
}

// SessionListItem 是会话列表的一项。
type SessionListItem struct {
	SessionID string    `json:"session_id"`
	Title     string    `json:"title"`
	Customer  string    `json:"customer"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListSessions 列出租户共享会话（分页，按更新时间倒序）。Store 已绑定 TenantID，
// 因此同租户成员共享读取不会扩大到其他租户；写操作仍校验 owner/admin。
func (s *Store) ListSessions(scope *domain.TenantScope, limit, offset int) ([]SessionListItem, error) {
	if limit <= 0 {
		limit = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cond, args := s.visibleCond(scope)
	q := `SELECT session_id, title, customer, created_at, updated_at
		FROM sessions WHERE deleted_at IS NULL` + cond + ` ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := s.db.Query(q, args...)
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

// visibleCond 返回租户内共享读取条件。跨租户边界由每租户独立 history.db 和
// tenant_meta 三方核对保证；owner_user_id 只用于修改授权，不再限制同租户读取。
func (s *Store) visibleCond(scope *domain.TenantScope) (string, []any) {
	return "", nil
}

// sessionVisibleTo 判定 scope 是否可访问该会话（不存在/已删除 → false）。
func (s *Store) sessionVisibleTo(scope *domain.TenantScope, sessionID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var cnt int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE session_id=? AND deleted_at IS NULL`, sessionID).Scan(&cnt)
	return err == nil && cnt > 0, err
}

// sessionWritableTo 将共享读取与修改授权分离：owner/admin 可维护租户内全部会话，
// 普通成员只能修改自己创建或无归属的会话。不存在/已删除统一 false。
func (s *Store) sessionWritableTo(scope *domain.TenantScope, sessionID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var owner sql.NullString
	var deletedAt sql.NullTime
	err := s.db.QueryRow(`SELECT owner_user_id, deleted_at FROM sessions WHERE session_id=?`, sessionID).Scan(&owner, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if deletedAt.Valid {
		return false, nil
	}
	if scope == nil || scope.IsTenantAdmin() {
		return true, nil
	}
	return owner.String == "" || owner.String == scope.UserID, nil
}

// CanWriteSession 供 HTTP 层在取消 Run、编辑重发等 Store 外副作用前做同一授权。
func (s *Store) CanWriteSession(scope *domain.TenantScope, sessionID string) (bool, error) {
	return s.sessionWritableTo(scope, sessionID)
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
	Branches    []string         `json:"branches"`    // checkpoint-tree：全部分支（main + b*）
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

// SearchMessages 按关键词检索历史消息（租户库内；scope 控制用户级可见性）。
// sessionID 为空时跨该租户全部可见会话；大小写不敏感的子串匹配（P1 起步——
// 历史量级小，LIKE 足够；后续可升级 bigram 权重检索对齐 longterm）。
func (s *Store) SearchMessages(scope *domain.TenantScope, sessionID, query string, limit int) ([]MessageHit, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cond, args := s.visibleCond(scope)
	q := `SELECT m.session_id, m.role, m.content, m.created_at
		FROM messages m
		JOIN sessions s ON s.session_id = m.session_id
		WHERE s.deleted_at IS NULL
			AND (? = '' OR m.session_id = ?)
			AND m.content LIKE '%' || ? || '%'` + cond + `
		ORDER BY m.created_at DESC LIMIT ?`
	args = append([]any{sessionID, sessionID, query}, args...)
	args = append(args, limit)
	rows, err := s.db.Query(q, args...)
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
// scope 控制用户级可见性（非 admin 只搜自己会话）。
func (s *Store) SearchSessions(scope *domain.TenantScope, query string, limit int) ([]SessionSearchHit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	like := "%" + query + "%"
	cond, args := s.visibleCond(scope)
	// 内容命中（每会话取最近一条命中消息）
	rows, err := s.db.Query(`SELECT m.session_id, m.role, m.content, m.created_at,
			COALESCE(ss.title,''), COALESCE(ss.customer,''), COALESCE(ss.updated_at, m.created_at)
		FROM messages m
		JOIN sessions ss ON ss.session_id = m.session_id
		WHERE ss.deleted_at IS NULL AND m.content LIKE ?`+cond+`
		ORDER BY m.created_at DESC`, append([]any{like}, args...)...)
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
		FROM sessions WHERE deleted_at IS NULL AND title LIKE ?`+cond+`
		ORDER BY updated_at DESC LIMIT ?`, append(append([]any{like}, args...), limit)...)
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

// GetSession 返回会话详情（按分支过滤时只取该分支消息/工具/检查点）。
// scope 控制用户级可见性（越权/不存在/已删除统一返回「不存在或无权访问」）。
// branch 为空=全分支汇总（admin/初始迁移用），否则过滤到该分支。
func (s *Store) GetSession(scope *domain.TenantScope, sessionID, branch string) (*SessionDetail, error) {
	ok, err := s.sessionVisibleTo(scope, sessionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("会话不存在或无权访问")
	}
	return s.GetSessionInternal(sessionID, branch)
}

// GetSessionInternal 租户内部读取会话详情（Agent 回调用——运行已授权，不按
// 用户过滤；仅排除已软删除会话）。调用方须已处于目标租户的 Store 上下文。
func (s *Store) GetSessionInternal(sessionID, branch string) (*SessionDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var det SessionDetail
	var sess SessionListItem
	err := s.db.QueryRow(`SELECT session_id, COALESCE(title,''), COALESCE(customer,''), created_at, updated_at
		FROM sessions WHERE session_id=? AND deleted_at IS NULL`, sessionID).
		Scan(&sess.SessionID, &sess.Title, &sess.Customer, &sess.CreatedAt, &sess.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("会话不存在或无权访问: %w", err)
	}
	det.Session = sess

	// branch_id 列存在则过滤；列缺失（中间态库）时退化到全分支
	hasBranch, _ := hasColumn(s.db, "messages", "branch_id")
	hasCpBranch, _ := hasColumn(s.db, "checkpoints", "branch_id")
	var msgQuery, tcQuery, cpQuery string
	if hasBranch && branch != "" {
		msgQuery = `SELECT id, role, content, tool_call_id, seq, created_at FROM messages
			WHERE session_id=? AND branch_id=? ORDER BY seq`
		tcQuery = `SELECT id, COALESCE(message_id, 0), tool_name, params_json, result_json, seq, created_at FROM tool_calls
			WHERE session_id=? AND message_id IN (SELECT id FROM messages WHERE session_id=? AND branch_id=?) ORDER BY seq`
	}
	// P0-04：分支 checkpoint 摘要 = 共享前缀（main 且 last_seq<分支点）+ 分支自身；
	// main 只看主线（兄弟分支 checkpoint 不回灌）。
	if hasCpBranch && branch != "" {
		cpQuery = `SELECT cp_id, cp_type, payload_json, created_at FROM checkpoints
			WHERE session_id=? AND ((branch_id='main' AND last_seq < ?) OR branch_id=?) ORDER BY id ASC`
	} else if hasCpBranch {
		cpQuery = `SELECT cp_id, cp_type, payload_json, created_at FROM checkpoints
			WHERE session_id=? AND branch_id='main' ORDER BY id ASC`
	} else {
		cpQuery = `SELECT cp_id, cp_type, payload_json, created_at FROM checkpoints
			WHERE session_id=? ORDER BY id ASC`
	}
	if !hasBranch || branch == "" {
		msgQuery = `SELECT id, role, content, tool_call_id, seq, created_at FROM messages
			WHERE session_id=? ORDER BY seq`
		tcQuery = `SELECT id, COALESCE(message_id, 0), tool_name, params_json, result_json, seq, created_at FROM tool_calls
			WHERE session_id=? ORDER BY seq`
	}

	var rows *sql.Rows
	if hasBranch && branch != "" {
		rows, err = s.db.Query(msgQuery, sessionID, branch)
	} else {
		rows, err = s.db.Query(msgQuery, sessionID)
	}
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

	var rows2 *sql.Rows
	if hasBranch && branch != "" {
		rows2, err = s.db.Query(tcQuery, sessionID, sessionID, branch)
	} else {
		rows2, err = s.db.Query(tcQuery, sessionID)
	}
	if err != nil {
		return nil, err
	}
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

	// checkpoint-tree：分支列表（messages.branch_id 去重；main 恒在——会话存在即有主线）
	det.Branches = []string{"main"}
	brows, berr := s.db.Query(`SELECT DISTINCT branch_id FROM messages WHERE session_id=? AND branch_id != 'main'`, sessionID)
	if berr == nil {
		defer brows.Close()
		for brows.Next() {
			var b string
			if brows.Scan(&b) == nil && b != "" {
				det.Branches = append(det.Branches, b)
			}
		}
	}

	// P4：checkpoint 摘要（回放页时间线行）。已持 s.mu——不调 ListCheckpoints
	// （它也要 Lock），直接轻量查询 payload 自行解。
	var crows *sql.Rows
	var cerr error
	if hasCpBranch && branch != "" {
		crows, cerr = s.db.Query(cpQuery, sessionID, branchPoint(branch), branch)
	} else {
		crows, cerr = s.db.Query(cpQuery, sessionID)
	}
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

// DeleteSession 软删除某会话（置 deleted_at；轨迹保留供审计/恢复）。
// scope 控制可见性——越权/不存在统一 404 语义（「会话不存在或无权访问」）。
func (s *Store) DeleteSession(scope *domain.TenantScope, sessionID string) error {
	ok, err := s.sessionWritableTo(scope, sessionID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("会话不存在或无权访问")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`UPDATE sessions SET deleted_at=? WHERE session_id=?`, time.Now().UTC(), sessionID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("会话不存在或无权访问")
	}
	return nil
}

// RenameSession 用户手动重命名会话：更新 title 并置 title_pinned=1
// （后续自动标题/首行标题不再覆盖）。越权/不存在统一 404 语义。
func (s *Store) RenameSession(scope *domain.TenantScope, sessionID, title string) error {
	title = strings.TrimSpace(title)
	if title == "" || len([]rune(title)) > 100 {
		return fmt.Errorf("会话名称需为 1~100 字符")
	}
	ok, err := s.sessionWritableTo(scope, sessionID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("会话不存在或无权访问")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(`UPDATE sessions SET title=?, title_pinned=1, updated_at=? WHERE session_id=?`,
		title, time.Now().UTC(), sessionID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("会话不存在或无权访问")
	}
	return nil
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

// TruncateAfter 编辑重发（checkpoint-tree 语义，2026-08-15）：seq >= after
// 的消息与 checkpoint 不再物理删除——软分叉到新分支（b{after}-{ts}）。
// 主线 main 保留 seq < after；旧对话在新分支上可回看。返回分支 ID。
func (s *Store) TruncateAfter(sessionID string, after int) (string, int, error) {
	return s.BranchAfter(sessionID, after)
}

// BranchAfter 编辑重发的分叉语义（checkpoint-tree，git 式，2026-08-15 修订）：
// **保留原路径**——main 上 A B C D E 全部不动；从截断点**复制**共享前缀
// （A B C 副本挂 branch_id=b{seq}-{ts}），后续新消息写该分支。兄弟分支
// 并存互不干扰，各视图显示自己完整路径（main=原全路径；bN=前缀副本+
// 新消息）。返回 (分支 ID, 复制的消息数)。
func (s *Store) BranchAfter(sessionID string, after int) (string, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	branchID := fmt.Sprintf("b%d-%d", after, time.Now().Unix())
	tx, err := s.db.Begin()
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = tx.Rollback() }()
	// 复制共享前缀（seq < after）到新分支——先读后插（子查询占位符在
	// modernc/sqlite 的绑定顺序问题，读出逐条插最稳）；原消息不动。
	prefix, err := tx.Query(`SELECT role, content, tool_call_id, seq, created_at FROM messages
		WHERE session_id=? AND seq < ? ORDER BY seq`, sessionID, after)
	if err != nil {
		return "", 0, fmt.Errorf("读共享前缀: %w", err)
	}
	type pmsg struct {
		role, content, tcID string
		createdAt           time.Time
	}
	var msgs []pmsg
	for prefix.Next() {
		var m pmsg
		var seq int
		if err := prefix.Scan(&m.role, &m.content, &m.tcID, &seq, &m.createdAt); err != nil {
			prefix.Close()
			return "", 0, err
		}
		msgs = append(msgs, m)
	}
	prefix.Close()
	// 新分支 seq 从全局最大值继续（分支消息与 main 消息在同一 seq 空间）
	var maxSeq int
	if err := tx.QueryRow(`SELECT COALESCE(MAX(seq),0) FROM messages WHERE session_id=?`, sessionID).Scan(&maxSeq); err != nil {
		return "", 0, err
	}
	for _, m := range msgs {
		maxSeq++
		if _, err := tx.Exec(`INSERT INTO messages (session_id, role, content, tool_call_id, seq, created_at, branch_id)
			VALUES(?,?,?,?,?,?,?)`, sessionID, m.role, m.content, m.tcID, maxSeq, m.createdAt, branchID); err != nil {
			return "", 0, fmt.Errorf("复制前缀消息: %w", err)
		}
	}
	affected := len(msgs)
	// 原尾部（seq >= after）保留在 main，不再挪走——git 语义：旧路径完整保留
	if err := tx.Commit(); err != nil {
		return "", 0, err
	}
	return branchID, int(affected), nil
}

// AppendMessageBranch 追加消息到指定分支（分叉后的续写走这里）。
func (s *Store) AppendMessageBranch(sessionID, branch string, role, content, toolCallID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var seq int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(seq),0) FROM messages WHERE session_id=? AND branch_id=?`,
		sessionID, branch).Scan(&seq); err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO messages (session_id, role, content, tool_call_id, seq, created_at, branch_id)
		VALUES(?,?,?,?,?,?,?)`, sessionID, role, content, toolCallID, seq+1, time.Now().UTC(), branch)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}
