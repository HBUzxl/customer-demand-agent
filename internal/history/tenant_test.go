package history

import (
	"strings"
	"testing"

	"customer-demand-agent/internal/domain"
)

// TestOpenTenantThreeWayVerify 三方核对：OpenTenant 写入 tenant_meta；
// 同租户重开成功；不同租户 ID 打开同一库 → 拒绝（防路径错配/串库）。
func TestOpenTenantThreeWayVerify(t *testing.T) {
	dir := t.TempDir()
	p := dir + "/t.db"
	ta := strings.Repeat("a", 32)
	tb := strings.Repeat("b", 32)
	s, err := OpenTenant(ta, p)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	// 同租户重开成功
	s2, err := OpenTenant(ta, p)
	if err != nil {
		t.Fatalf("同租户重开应成功: %v", err)
	}
	s2.Close()
	// 异租户打开同一库 → 拒绝
	if _, err := OpenTenant(tb, p); err == nil {
		t.Fatal("租户 ID 与 tenant_meta 不一致应拒绝（串库）")
	}
}

// TestForeignKeysEnabled 自检：OpenTenant 连接必须启用外键约束
// （open 内部已强制，若关闭则拒绝启动——此处锁定契约）。
func TestForeignKeysEnabled(t *testing.T) {
	s := openTestStore(t)
	var fk int
	if err := s.db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys 应为 1，got %d", fk)
	}
	// 外键生效：往不存在的会话插消息应失败
	if _, err := s.AppendMessage("no-such-session", "user", "x", ""); err == nil {
		t.Fatal("外键应拒绝孤儿消息（会话不存在）")
	}
}

// TestTenantScopeSharedReadOwnerWrite 同租户会话读取共享；普通成员只能修改自己
// 创建的会话，owner/admin 可维护全部。跨租户隔离由独立 Store 保证。
func TestTenantScopeSharedReadOwnerWrite(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("user-a", "s-a", "A的会话", "")
	_ = s.EnsureSession("user-b", "s-b", "B的会话", "")
	_ = s.EnsureSession("", "s-nil", "无归属会话", "") // Agent 回调写入（无 owner）

	// 非 admin（user-a）也可读取本租户全部会话。
	scopeA := &domain.TenantScope{TenantID: "t1", UserID: "user-a", Roles: []string{domain.RoleAnalyst}}
	list, err := s.ListSessions(scopeA, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, it := range list {
		ids[it.SessionID] = true
	}
	if !ids["s-a"] || !ids["s-b"] || !ids["s-nil"] {
		t.Fatalf("user-a 应见本租户全部会话: %+v", ids)
	}
	if _, err := s.GetSession(scopeA, "s-b", ""); err != nil {
		t.Fatalf("同租户成员应可读取他人会话: %v", err)
	}
	if err := s.RenameSession(scopeA, "s-b", "越权修改"); err == nil {
		t.Fatal("共享读取不应允许普通成员修改他人会话")
	}
	// admin（owner）：看全租户
	scopeAdmin := &domain.TenantScope{TenantID: "t1", UserID: "user-b", Roles: []string{domain.RoleOwner}}
	list2, err := s.ListSessions(scopeAdmin, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list2) != 3 {
		t.Fatalf("admin 应见全部 3 会话，got %d", len(list2))
	}
	// admin 可读 B 的会话
	if _, err := s.GetSession(scopeAdmin, "s-b", ""); err != nil {
		t.Fatalf("admin 读他人会话应成功: %v", err)
	}
	// nil scope（租户内部/Agent 回调）：全量可见
	if _, err := s.GetSession(nil, "s-b", ""); err != nil {
		t.Fatalf("nil scope 应全量可见: %v", err)
	}
}

// TestSearchMessagesSharedWithinTenant history_search 与会话列表保持一致：同租户
// 成员可检索共享历史，跨租户仍由独立数据库隔离。
func TestSearchMessagesSharedWithinTenant(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("user-a", "s-a", "A的会话", "")
	_ = s.EnsureSession("user-b", "s-b", "B的会话", "")
	_ = s.EnsureSession("", "s-nil", "无归属会话", "")
	_, _ = s.AppendMessage("s-a", "user", "A 提到预算 50 万", "")
	_, _ = s.AppendMessage("s-b", "user", "B 提到预算 200 万", "")
	_, _ = s.AppendMessage("s-nil", "user", "无归属会话提到预算 10 万", "")

	// 非 admin：也命中同租户全部共享原文。
	scopeA := &domain.TenantScope{TenantID: "t1", UserID: "user-a", Roles: []string{domain.RoleAnalyst}}
	hits, err := s.SearchMessages(scopeA, "", "预算", 10)
	if err != nil {
		t.Fatal(err)
	}
	foundB := false
	for _, h := range hits {
		if h.SessionID == "s-b" {
			foundB = true
		}
	}
	if !foundB {
		t.Fatalf("同租户成员应搜到共享会话原文，got %+v", hits)
	}
	var foundA, foundNil bool
	for _, h := range hits {
		if h.SessionID == "s-a" {
			foundA = true
		}
		if h.SessionID == "s-nil" {
			foundNil = true
		}
	}
	if !foundA || !foundNil || len(hits) != 3 {
		t.Fatalf("应命中本租户全部 3 条原文: %+v", hits)
	}
	// admin：全量可见
	scopeAdmin := &domain.TenantScope{TenantID: "t1", UserID: "user-b", Roles: []string{domain.RoleOwner}}
	if hits, _ = s.SearchMessages(scopeAdmin, "", "预算", 10); len(hits) != 3 {
		t.Fatalf("admin 应全量可见（3 条），got %d", len(hits))
	}
	// nil scope（legacy/Agent 回调）：全量可见
	if hits, _ = s.SearchMessages(nil, "", "预算", 10); len(hits) != 3 {
		t.Fatalf("nil scope 应全量可见（3 条），got %d", len(hits))
	}
}

// TestRenameSession 手动重命名：置 title_pinned 防自动标题覆盖；越权/空名拒绝。
func TestRenameSession(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("user-a", "s-a", "原标题", "")
	scopeA := &domain.TenantScope{TenantID: "t1", UserID: "user-a", Roles: []string{domain.RoleAnalyst}}
	scopeB := &domain.TenantScope{TenantID: "t1", UserID: "user-b", Roles: []string{domain.RoleAnalyst}}

	// 越权重命名他人会话 → 拒绝
	if err := s.RenameSession(scopeB, "s-a", "抢占"); err == nil {
		t.Fatal("越权重命名他人会话应拒绝")
	}
	// 空标题 → 拒绝
	if err := s.RenameSession(scopeA, "s-a", "   "); err == nil {
		t.Fatal("空标题应拒绝")
	}
	// 正常重命名
	if err := s.RenameSession(scopeA, "s-a", "用户改名"); err != nil {
		t.Fatalf("重命名失败: %v", err)
	}
	det, err := s.GetSession(scopeA, "s-a", "")
	if err != nil {
		t.Fatal(err)
	}
	if det.Session.Title != "用户改名" {
		t.Fatalf("标题应为「用户改名」，got %q", det.Session.Title)
	}
	// 重命名后 EnsureSession 自动标题不再覆盖（title_pinned=1）
	if err := s.EnsureSession("user-a", "s-a", "新的首行标题", ""); err != nil {
		t.Fatal(err)
	}
	det2, _ := s.GetSession(scopeA, "s-a", "")
	if det2.Session.Title != "用户改名" {
		t.Fatalf("title_pinned 后自动标题不应覆盖，got %q", det2.Session.Title)
	}
	// 未重命名的会话仍可被自动标题更新
	if err := s.EnsureSession("user-a", "s-nopin", "初始", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureSession("user-a", "s-nopin", "自动标题", ""); err != nil {
		t.Fatal(err)
	}
	det3, _ := s.GetSession(scopeA, "s-nopin", "")
	if det3.Session.Title != "自动标题" {
		t.Fatalf("未固定会话应可自动更新标题，got %q", det3.Session.Title)
	}
}

// TestEnsureSessionScopedRejectsTakeover 验证同租户普通成员不能通过复用
// session_id 抢占他人会话；owner/admin 可维护但不会改变原归属。
func TestEnsureSessionScopedRejectsTakeover(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("user-a", "shared-session-id", "A 的会话", "客户A")
	scopeA := &domain.TenantScope{TenantID: "t1", UserID: "user-a", Roles: []string{domain.RoleAnalyst}}
	scopeB := &domain.TenantScope{TenantID: "t1", UserID: "user-b", Roles: []string{domain.RoleAnalyst}}
	scopeAdmin := &domain.TenantScope{TenantID: "t1", UserID: "admin", Roles: []string{domain.RoleAdmin}}

	if err := s.EnsureSessionScoped(scopeB, "shared-session-id", "B 抢占", "客户B"); err == nil {
		t.Fatal("普通成员不应能接管他人会话")
	}
	if _, err := s.GetSession(scopeB, "shared-session-id", ""); err != nil {
		t.Fatal("抢占失败不影响同租户共享读取")
	}
	det, err := s.GetSession(scopeA, "shared-session-id", "")
	if err != nil {
		t.Fatal(err)
	}
	if det.Session.Title != "A 的会话" || det.Session.Customer != "客户A" {
		t.Fatalf("抢占失败后原会话不应被修改: %+v", det.Session)
	}

	if err := s.EnsureSessionScoped(scopeAdmin, "shared-session-id", "管理员维护", "客户A"); err != nil {
		t.Fatalf("admin 应可维护租户内会话: %v", err)
	}
	// admin 维护后也不能把 owner 改成自己；A/B 继续共享读取。
	if _, err := s.GetSession(scopeA, "shared-session-id", ""); err != nil {
		t.Fatalf("admin 维护不应改变原 owner: %v", err)
	}
	if _, err := s.GetSession(scopeB, "shared-session-id", ""); err != nil {
		t.Fatal("admin 维护后 B 应继续读取共享会话")
	}

	if err := s.EnsureSessionScoped(scopeB, "new-b", "B 新会话", ""); err != nil {
		t.Fatalf("B 应可创建自己的新会话: %v", err)
	}
	if _, err := s.GetSession(scopeB, "new-b", ""); err != nil {
		t.Fatalf("B 应可读取自己创建的会话: %v", err)
	}
}

// TestRecentMissQueriesScoped 收集近期 memory_search 零命中（Lint miss 输入），
// 不再暴露裸 DB 连接。
func TestRecentMissQueriesScoped(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("", "s-miss", "标题", "")
	aid, err := s.AppendMessage("s-miss", "assistant", "答", "")
	if err != nil {
		t.Fatal(err)
	}
	// 零命中
	if _, err := s.AppendToolCall("s-miss", aid, "memory_search", `{"query":"不存在的产品X"}`, `{"count":0}`); err != nil {
		t.Fatal(err)
	}
	// 有命中（不应进 miss 收集）
	if _, err := s.AppendToolCall("s-miss", aid, "memory_search", `{"query":"雷池"}`, `{"count":1}`); err != nil {
		t.Fatal(err)
	}
	miss := s.RecentMissQueries(10)
	if len(miss) != 1 || !strings.Contains(miss[0], "不存在的产品X") {
		t.Fatalf("应只收集零命中查询: %+v", miss)
	}
}
