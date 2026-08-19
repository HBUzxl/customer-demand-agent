package history

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"customer-demand-agent/internal/domain"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// TestCheckpointRoundTrip de-tenancy 后：checkpoint 按 session 存取（无租户维度）。
func TestCheckpointRoundTrip(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("", "sess-x", "标题", "")
	cp := &domain.Checkpoint{ID: "cp-1", Type: domain.CheckpointInitial}
	if err := s.AppendCheckpoint("sess-x", "", cp); err != nil {
		t.Fatal(err)
	}
	cps, err := s.ListCheckpoints("sess-x", "")
	if err != nil || len(cps) != 1 {
		t.Fatalf("按 session 取回 checkpoint 失败: %v (%d)", err, len(cps))
	}
}

// TestEnsureSessionSameTenantUpdate 验证同租户重复 EnsureSession 正常更新（不误拒）。
func TestEnsureSessionSameTenantUpdate(t *testing.T) {
	s := openTestStore(t)
	if err := s.EnsureSession("", "s1", "标题1", ""); err != nil {
		t.Fatalf("首次: %v", err)
	}
	if err := s.EnsureSession("", "s1", "标题2", "客户X"); err != nil {
		t.Fatalf("同租户更新: %v", err)
	}
	det, err := s.GetSession(nil, "s1", "")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if det.Session.Title != "标题2" || det.Session.Customer != "客户X" {
		t.Errorf("同租户更新未生效: %+v", det.Session)
	}
}

// TestSessionExistsTitlePinned G4 辅助方法：SessionExists 判存在、IsSessionTitlePinned
// 判手动重命名固定（固定后 AI 标题不再覆盖）。
func TestSessionExistsTitlePinned(t *testing.T) {
	s := openTestStore(t)
	// 不存在
	if exists, _ := s.SessionExists("s-nope"); exists {
		t.Fatal("不存在会话应返回 false")
	}
	if _, err := s.IsSessionTitlePinned("s-nope"); err == nil {
		t.Fatal("不存在会话 IsSessionTitlePinned 应报错")
	}
	// 新建：未固定
	if err := s.EnsureSession("", "s-pin", "初始", ""); err != nil {
		t.Fatal(err)
	}
	if exists, _ := s.SessionExists("s-pin"); !exists {
		t.Fatal("已建会话应返回 true")
	}
	if pinned, _ := s.IsSessionTitlePinned("s-pin"); pinned {
		t.Fatal("新建会话不应固定")
	}
	// EnsureSession 自动更新不置固定（标题仍可被 AI 覆盖）
	if err := s.EnsureSession("", "s-pin", "自动标题", ""); err != nil {
		t.Fatal(err)
	}
	if pinned, _ := s.IsSessionTitlePinned("s-pin"); pinned {
		t.Fatal("自动标题更新不应置 title_pinned")
	}
	// 手动重命名 → 固定
	scope := &domain.TenantScope{TenantID: "t1", UserID: "u1", Roles: []string{domain.RoleOwner}}
	if err := s.RenameSession(scope, "s-pin", "用户改名"); err != nil {
		t.Fatal(err)
	}
	if pinned, _ := s.IsSessionTitlePinned("s-pin"); !pinned {
		t.Fatal("手动重命名后应置 title_pinned")
	}
}

// TestTenantColumnMigration de-tenancy 契约：列保留不读写——老库的
// NOT NULL 无默认 tenant_id 列在 Open 时补 DEFAULT 'default'（表重建，
// 列与历史值原样保留），此后无列 INSERT 合法。幂等。
func TestTenantColumnMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "old.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// 老形态 schema（de-tenancy 前）
	if _, err := db.Exec(`CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    tenant_id  TEXT NOT NULL,
    title      TEXT,
    customer   TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sessions(session_id, tenant_id, title, customer, created_at, updated_at)
		VALUES('legacy-1', 'corp-a', '老会话', '', '2026-01-01', '2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Open 触发迁移
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open 应完成迁移: %v", err)
	}
	defer s.Close()
	// 1) 列保留且带默认值
	rows, err := s.db.Query(`PRAGMA table_info(sessions)`)
	if err != nil {
		t.Fatal(err)
	}
	colFound := false
	hasDefault := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull int
		var dflt sql.NullString
		var pk int
		_ = rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk)
		if name == "tenant_id" {
			colFound = true
			hasDefault = dflt.Valid && strings.Trim(dflt.String, `"'`) == "default"
		}
	}
	rows.Close()
	if !colFound {
		t.Fatal("tenant_id 列应保留（契约：列保留）")
	}
	if !hasDefault {
		t.Fatal("tenant_id 列应有 DEFAULT 'default'")
	}
	// 历史租户值原样保留
	var legacyVal string
	_ = s.db.QueryRow(`SELECT tenant_id FROM sessions WHERE session_id='legacy-1'`).Scan(&legacyVal)
	if legacyVal != "corp-a" {
		t.Fatalf("历史租户值应保留，got %q", legacyVal)
	}
	// 2) 数据保留
	if err := s.EnsureSession("", "legacy-1", "老会话", ""); err != nil {
		t.Fatal(err)
	}
	sessions, err := s.ListSessions(nil, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range sessions {
		if it.SessionID == "legacy-1" {
			found = true
		}
	}
	if !found {
		t.Fatal("迁移应保留老会话数据")
	}
	// 3) 新会话正常写入（无列 INSERT）
	if err := s.EnsureSession("", "new-1", "新会话", ""); err != nil {
		t.Fatal(err)
	}
}

// TestTenantColumnRestore 中间态库（曾经历删列迁移、列已不存在）：
// Open 时 ALTER 补回带 DEFAULT 的列——契约「列恒在」对任意历史形态成立。
func TestTenantColumnRestore(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "mid.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    title      TEXT,
    customer   TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sessions(session_id, title) VALUES('m1', '中间态')`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rows, _ := s.db.Query(`PRAGMA table_info(sessions)`)
	colOK := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull int
		var dflt sql.NullString
		var pk int
		_ = rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk)
		if name == "tenant_id" && dflt.Valid {
			colOK = true
		}
	}
	rows.Close()
	if !colOK {
		t.Fatal("列应被补回且带默认值")
	}
	if err := s.EnsureSession("", "m2", "新行", ""); err != nil {
		t.Fatalf("无列 INSERT 应合法: %v", err)
	}
	var v string
	_ = s.db.QueryRow(`SELECT tenant_id FROM sessions WHERE session_id='m2'`).Scan(&v)
	if v != "default" {
		t.Fatalf("新行应为默认值，got %q", v)
	}
}

// TestSearchSessions 会话内容搜索：标题命中优先 + 内容命中带 snippet。
func TestSearchSessions(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("", "s-title", "医院挂号系统刷号", "")
	_ = s.EnsureSession("", "s-content", "随便聊聊", "")
	if _, err := s.AppendMessage("s-content", "user", "客户官网被挂马要过等保三级", ""); err != nil {
		t.Fatal(err)
	}
	hits, err := s.SearchSessions(nil, "挂马", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].SessionID != "s-content" {
		t.Fatalf("内容词应命中 s-content: %+v", hits)
	}
	if !strings.Contains(hits[0].Snippet, "挂马") {
		t.Fatalf("snippet 应含关键词: %q", hits[0].Snippet)
	}
	// 标题命中 + 同会话内容也命中 → 合并且排前
	hits2, _ := s.SearchSessions(nil, "刷号", 10)
	if len(hits2) != 1 || hits2[0].SessionID != "s-title" || !hits2[0].HitTitle {
		t.Fatalf("标题词应命中 s-title 且优先: %+v", hits2)
	}
}

// TestBranchAfterCheckpointTree checkpoint-tree git 式分叉（2026-08-15 修订）：
// main 完整保留（零丢失）+共享前缀复制到新分支+分支独立续写互不干扰。
func TestBranchAfterCheckpointTree(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("", "br-s1", "分支测试", "")
	for _, txt := range []string{"第一轮", "第二轮", "第三轮"} {
		if _, err := s.AppendMessage("br-s1", "user", txt, ""); err != nil {
			t.Fatal(err)
		}
		if _, err := s.AppendMessage("br-s1", "assistant", "答"+txt, ""); err != nil {
			t.Fatal(err)
		}
	}
	branch, copied, err := s.BranchAfter("br-s1", 3)
	if err != nil || copied != 2 || branch == "" {
		t.Fatalf("分叉失败: %v %d %s", err, copied, branch)
	}
	// main 完整保留 6 条（原路径不动——git 语义）
	det, err := s.GetSession(nil, "br-s1", "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(det.Messages) != 6 {
		t.Fatalf("main 应完整保留 6 条，got %d", len(det.Messages))
	}
	// 分支 = 前缀副本 2 条 + 独立续写 1 条
	if _, err := s.AppendMessageBranch("br-s1", branch, "user", "第二轮改写", ""); err != nil {
		t.Fatal(err)
	}
	detB, err := s.GetSession(nil, "br-s1", branch)
	if err != nil {
		t.Fatal(err)
	}
	if len(detB.Messages) != 3 {
		t.Fatalf("分支应 2 副本+1 续写=3 条，got %d", len(detB.Messages))
	}
	if detB.Messages[len(detB.Messages)-1].Content != "第二轮改写" {
		t.Fatalf("分支末条应为续写内容")
	}
	// 分支续写不影响 main
	det2, _ := s.GetSession(nil, "br-s1", "main")
	if len(det2.Messages) != 6 {
		t.Fatalf("分支续写不应影响 main: %d", len(det2.Messages))
	}
	// branches 含 main + 新分支
	detAll, _ := s.GetSession(nil, "br-s1", "")
	hasMain, hasB := false, false
	for _, b := range detAll.Branches {
		if b == "main" {
			hasMain = true
		}
		if b == branch {
			hasB = true
		}
	}
	if !hasMain || !hasB {
		t.Fatalf("branches 应含 main+新分支: %v", detAll.Branches)
	}
}

// TestBranchCheckpointFullChain P0-04 端到端：编辑重发后「消息+工具调用+checkpoint」
// 全部写同一分支；按分支读回完整路径；checkpoint 链=共享前缀+分支自身；
// 兄弟分支互不串扰（main 保留尾部 cp，分支看不到；分支 cp 也不回灌 main）。
func TestBranchCheckpointFullChain(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("", "fc", "全链路", "")

	// 主链两轮：u1/a1（cp1 initial）+ u2/a2（cp2 followup）
	_, _ = s.AppendMessage("fc", "user", "u1", "")
	aid1, _ := s.AppendMessage("fc", "assistant", "a1", "")
	_, _ = s.AppendToolCall("fc", aid1, "memory_search", `{"q":"1"}`, `{"n":1}`)
	cp1 := &domain.Checkpoint{ID: "cp1", Type: domain.CheckpointInitial, Document: "客户原始需求"}
	_ = s.AppendCheckpoint("fc", "", cp1)
	_, _ = s.AppendMessage("fc", "user", "u2", "")
	aid2, _ := s.AppendMessage("fc", "assistant", "a2", "")
	_, _ = s.AppendToolCall("fc", aid2, "memory_search", `{"q":"2"}`, `{"n":2}`)
	cp2 := &domain.Checkpoint{ID: "cp2", Type: domain.CheckpointFollowup, Question: "u2", Answer: "a2"}
	_ = s.AppendCheckpoint("fc", "", cp2)

	// 编辑 seq=3 的 u2 → 分叉（git 式：main 原路径不动）
	branch, copied, err := s.BranchAfter("fc", 3)
	if err != nil || copied != 2 || branch == "" {
		t.Fatalf("分叉失败: %v %d %s", err, copied, branch)
	}

	// 分支续写完整一轮：user + assistant + tool_call + checkpoint 全部挂分支
	_, _ = s.AppendMessageBranch("fc", branch, "user", "u2改", "")
	baid, _ := s.AppendMessageBranch("fc", branch, "assistant", "a2改", "")
	_, _ = s.AppendToolCall("fc", baid, "analysis_submit", `{"p":"x"}`, `{"ok":true}`)
	cp3 := &domain.Checkpoint{ID: "cp3", Type: domain.CheckpointFollowup, Question: "u2改", Answer: "a2改"}
	if err := s.AppendCheckpoint("fc", branch, cp3); err != nil {
		t.Fatal(err)
	}

	// 1) DB 层断言：分支消息/tool_calls/checkpoint 归属同一 branch_id
	var bn int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE session_id='fc' AND branch_id=?`, branch).Scan(&bn)
	if bn != 4 { // 2 前缀副本 + 2 续写
		t.Fatalf("分支应有 4 条消息（2副本+2续写），got %d", bn)
	}
	var btc int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM tool_calls WHERE session_id='fc' AND message_id IN
		(SELECT id FROM messages WHERE session_id='fc' AND branch_id=?)`, branch).Scan(&btc)
	if btc != 1 {
		t.Fatalf("分支应有 1 条工具调用，got %d", btc)
	}
	var bcp int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM checkpoints WHERE session_id='fc' AND branch_id=?`, branch).Scan(&bcp)
	if bcp != 1 {
		t.Fatalf("分支应有 1 个 checkpoint，got %d", bcp)
	}

	// 2) 分支记忆 = 共享前缀（main 且 last_seq<分支点）+ 分支自身
	cps, err := s.ListCheckpoints("fc", branch)
	if err != nil {
		t.Fatal(err)
	}
	if len(cps) != 2 || cps[0].ID != "cp1" || cps[1].ID != "cp3" {
		t.Fatalf("分支 checkpoint 应为 [cp1, cp3]，got %v", ids(cps))
	}

	// 3) main 完整保留：4 条消息 + 2 个 checkpoint（cp2 尾部不回灌分支，也不丢）
	detM, err := s.GetSession(nil, "fc", "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(detM.Messages) != 4 {
		t.Fatalf("main 应完整保留 4 条，got %d", len(detM.Messages))
	}
	mcps, _ := s.ListCheckpoints("fc", "main")
	if len(mcps) != 2 || mcps[0].ID != "cp1" || mcps[1].ID != "cp2" {
		t.Fatalf("main checkpoint 应为 [cp1, cp2]，got %v", ids(mcps))
	}

	// 4) 分支视图完整路径：副本 2 + 续写 2，tool_call 归属分支 assistant
	detB, err := s.GetSession(nil, "fc", branch)
	if err != nil {
		t.Fatal(err)
	}
	if len(detB.Messages) != 4 {
		t.Fatalf("分支视图应有 4 条消息，got %d", len(detB.Messages))
	}
	if detB.Messages[2].Content != "u2改" || detB.Messages[3].Content != "a2改" {
		t.Fatalf("分支末尾应为续写内容: %q %q", detB.Messages[2].Content, detB.Messages[3].Content)
	}
	foundBranchTool := false
	for _, tc := range detB.ToolCalls {
		if tc.ToolName == "analysis_submit" {
			foundBranchTool = true
		}
	}
	if !foundBranchTool {
		t.Fatalf("分支视图应含续写的工具调用: %+v", detB.ToolCalls)
	}
	if len(detB.Checkpoints) != 2 || detB.Checkpoints[0].ID != "cp1" || detB.Checkpoints[1].ID != "cp3" {
		t.Fatalf("分支视图 checkpoint 摘要应为 [cp1, cp3]: %+v", detB.Checkpoints)
	}

	// 5) 分支独立续写不污染 main（兄弟隔离）
	_, _ = s.AppendMessageBranch("fc", branch, "user", "再次追问", "")
	detM2, _ := s.GetSession(nil, "fc", "main")
	if len(detM2.Messages) != 4 {
		t.Fatalf("分支续写不应影响 main: %d", len(detM2.Messages))
	}
}

func ids(cps []*domain.Checkpoint) []string {
	out := make([]string, len(cps))
	for i, c := range cps {
		out[i] = c.ID
	}
	return out
}

// TestCheckpointChainMigrationZeroLoss 老库线性链零丢失迁移：
// 无 branch_id 列的一期形态库 Open 迁移后消息全部保留在 main。
func TestCheckpointChainMigrationZeroLoss(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "old-linear.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    title TEXT,
    customer TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT,
    tool_call_id TEXT DEFAULT '',
    seq INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sessions (session_id, title, customer, created_at, updated_at) VALUES('legacy-1', '老会话', '', '2026-01-01', '2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 4; i++ {
		if _, err := db.Exec(`INSERT INTO messages (session_id, role, content, seq) VALUES('legacy-1', 'user', ?, ?)`,
			fmt.Sprintf("老消息%d", i), i); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("老库 Open 迁移: %v", err)
	}
	defer s.Close()
	det, err := s.GetSession(nil, "legacy-1", "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(det.Messages) != 4 {
		t.Fatalf("老库 4 条消息应零丢失迁移到 main，got %d", len(det.Messages))
	}
	if det.Messages[0].Content != "老消息1" {
		t.Fatalf("首条内容应保留: %+v", det.Messages[0])
	}
}
