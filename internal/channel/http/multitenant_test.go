package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"customer-demand-agent/internal/auth"
	httpapi "customer-demand-agent/internal/channel/http"
	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/identity"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/model"
	"customer-demand-agent/internal/taskbg"
	"customer-demand-agent/internal/tenancy"
)

// tenantAuth 绑定一个租户的登录态（Cookie + CSRF）+ 其租户运行时数据面。
type tenantAuth struct {
	Cookie   string // "cda_session=<token>"（HTTP 请求 Cookie 头）
	CSRF     string
	UserID   string
	TenantID string
	Wiki     longterm.Store // 租户复合记忆（直接写 pending 等人工通道不可表达的语义）
	RT       *tenancy.Runtime
}

// setupTwoTenantServer 装配双租户测试服务：同一身份控制面下注册 A/B 两个租户，
// 各自数据面独立（tenants/<uuid>/ 各自 history.db + wiki）。taskbg 用空 exec
// （任务入队即 done，只验证租户过滤）。返回服务 + 身份服务 + A/B 登录态。
func setupTwoTenantServer(t *testing.T) (*httptest.Server, *identity.Service, *tenantAuth, *tenantAuth) {
	t.Helper()
	dir := t.TempDir()
	sysWikiDir := filepath.Join(dir, "system", "wiki")
	_ = os.MkdirAll(filepath.Join(sysWikiDir, "产品记忆"), 0o755)
	_ = os.WriteFile(filepath.Join(sysWikiDir, "产品记忆", "雷池.md"),
		[]byte("---\ntype: product\ntitle: 雷池\ntags: [\"WAF\"]\n---\nx"), 0o644)
	sysWiki := longterm.NewWikiStore(sysWikiDir)
	if err := sysWiki.Load(); err != nil {
		t.Fatal(err)
	}

	cfgStore, err := config.Load(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	registry := model.NewRegistry(10 * time.Second)
	for _, m := range cfgStore.Get().Models {
		_ = registry.Register(m)
	}
	rc := cfgStore.Get().Router
	mgr := model.NewManager(llm.NewClient(), registry, &rc)

	reg := tenancy.NewRegistry(mgr, sysWiki, filepath.Join(dir, "tenants"), cfgStore, nil)
	idStore, err := identity.Open(filepath.Join(dir, "control", "identity.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { idStore.Close() })
	idSvc := identity.NewService(idStore)
	idSvc.SetProvisioner(reg.Provision)

	tasks := taskbg.NewRunner(func(ctx context.Context, task *taskbg.Task) error { return nil })
	srv := httpapi.New(httpapi.Config{
		Store: cfgStore, Runtimes: reg, ModelMgr: mgr, Registry: registry,
		Tasks: tasks, LogRing: httpapi.NewLogRing(100), Leads: nil,
	})
	cookieMgr := auth.NewCookieManager(false)
	authMW := identity.NewMiddleware(idSvc, cookieMgr)
	srv.SetAuth(&httpapi.AuthDeps{
		Identity:            idSvc,
		Cookies:             cookieMgr,
		Middleware:          authMW,
		LoginLimiter:        auth.NewRateLimiter(50, time.Minute),
		RegistrationEnabled: func() bool { return true },
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	ctx := context.Background()
	register := func(org, name, email string) *tenantAuth {
		res, err := idSvc.Register(ctx, identity.RegisterRequest{
			OrgName: org, DisplayName: name, Email: email, Password: "password-123456",
		}, "127.0.0.1", "test")
		if err != nil {
			t.Fatal(err)
		}
		rt, err := reg.ForTenant(ctx, res.Tenant.ID)
		if err != nil {
			t.Fatal(err)
		}
		return &tenantAuth{
			Cookie: cookieMgr.Name() + "=" + res.TokenRaw, CSRF: res.CSRFRaw,
			UserID: res.User.ID, TenantID: res.Tenant.ID, Wiki: rt.Wiki, RT: rt,
		}
	}
	return ts, idSvc, register("A公司", "A用户", "a@example.com"), register("B公司", "B用户", "b@example.com")
}

// titlesOf 提取响应 items 的 title 字段列表。
func titlesOf(body map[string]any) []string {
	items, _ := body["items"].([]any)
	out := make([]string, 0, len(items))
	for _, it := range items {
		m, _ := it.(map[string]any)
		if t, ok := m["title"].(string); ok {
			out = append(out, t)
		}
	}
	return out
}

// sessionIDs 提取会话列表响应的 session_id 字段列表。
func sessionIDs(body map[string]any) []string {
	items, _ := body["items"].([]any)
	out := make([]string, 0, len(items))
	for _, it := range items {
		m, _ := it.(map[string]any)
		if id, ok := m["session_id"].(string); ok {
			out = append(out, id)
		}
	}
	return out
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// TestCrossTenantSessionIsolation A 用同一 session_id 建的会话，B 用任意接口
// 访问一律 404（读/运行态/取消/流式/删除/截断）——数据面隔离在接触
// RunManager 前先由 GetSession(scope) 闭合（设计文档 §9.3）。
func TestCrossTenantSessionIsolation(t *testing.T) {
	ts, _, a, b := setupTwoTenantServer(t)

	// A 建会话（owner=a 的用户；LLM 不可达只影响后台 Run，会话已同步创建）
	if code, body, _ := rawDo(t, ts, "POST", "/api/message", a.Cookie, a.CSRF, map[string]any{
		"text": "你好 A", "session_id": "sess-shared",
	}); code != http.StatusAccepted {
		t.Fatalf("A 发消息应 202，got %d: %v", code, body)
	}

	// A 可见自己会话（200）
	if code, body, _ := rawDo(t, ts, "GET", "/api/sessions/sess-shared", a.Cookie, "", nil); code != http.StatusOK {
		t.Fatalf("A 读自己会话应 200，got %d: %v", code, body)
	}
	// B 用同一 session_id → 404（B 数据面不存在）
	if code, body, _ := rawDo(t, ts, "GET", "/api/sessions/sess-shared", b.Cookie, "", nil); code != http.StatusNotFound {
		t.Fatalf("B 读 A 会话应 404，got %d: %v", code, body)
	}
	// B 查运行态 / 取消 Run / 订阅流 / 删除 / 截断 → 全部 404
	if code, _, _ := rawDo(t, ts, "GET", "/api/sessions/sess-shared/running", b.Cookie, "", nil); code != http.StatusNotFound {
		t.Fatalf("B 查 A 会话运行态应 404，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "POST", "/api/sessions/sess-shared/runs/run-x/cancel", b.Cookie, b.CSRF, nil); code != http.StatusNotFound {
		t.Fatalf("B 取消 A 会话 Run 应 404，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "GET", "/api/sessions/sess-shared/stream", b.Cookie, "", nil); code != http.StatusNotFound {
		t.Fatalf("B 订阅 A 会话流应 404（SSE 不串流），got %d", code)
	}
	// 重命名：B 改 A 会话 → 404；A 改自己 → 200（含 title_pinned 防覆盖）
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/sessions/sess-shared", b.Cookie, b.CSRF, map[string]any{"title": "B改名"}); code != http.StatusNotFound {
		t.Fatalf("B 重命名 A 会话应 404，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/sessions/sess-shared", a.Cookie, a.CSRF, map[string]any{"title": "A改名"}); code != http.StatusOK {
		t.Fatalf("A 重命名自己会话应 200，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "DELETE", "/api/sessions/sess-shared", b.Cookie, b.CSRF, nil); code != http.StatusNotFound {
		t.Fatalf("B 删 A 会话应 404，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "DELETE", "/api/sessions/sess-shared/messages/after?after=1", b.Cookie, b.CSRF, nil); code != http.StatusNotFound {
		t.Fatalf("B 截断 A 会话应 404，got %d", code)
	}
}

// TestSameTenantAnalystCannotTakeOverSession 覆盖“同租户、不同成员”这一层：
// 独立数据面只能防跨租户，owner_user_id 授权还必须防 session_id 抢占。
func TestSameTenantAnalystCannotTakeOverSession(t *testing.T) {
	ts, idSvc, a, _ := setupTwoTenantServer(t)
	ownerScope := &domain.TenantScope{TenantID: a.TenantID, UserID: a.UserID, Roles: []string{domain.RoleOwner}}
	const memberPassword = "Password-123"
	if err := idSvc.MembersAdd(ownerScope, "member-a@example.com", "A租户成员", memberPassword, domain.RoleAnalyst); err != nil {
		t.Fatal(err)
	}
	member, err := idSvc.Login(context.Background(), "member-a@example.com", memberPassword, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	memberCookie := auth.NewCookieManager(false).Name() + "=" + member.TokenRaw

	if err := a.RT.History.EnsureSession(a.UserID, "same-tenant-owned", "Owner 的会话", "客户A"); err != nil {
		t.Fatal(err)
	}
	// 同租户数据共享：analyst 可读取/回放 owner 的会话，但不能接管、改写或删除。
	if code, body, _ := rawDo(t, ts, "GET", "/api/sessions/same-tenant-owned", memberCookie, "", nil); code != http.StatusOK {
		t.Fatalf("同租户 analyst 应可读取共享会话，got %d: %v", code, body)
	}
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/sessions/same-tenant-owned", memberCookie, member.CSRFRaw,
		map[string]any{"title": "越权改名"}); code != http.StatusNotFound {
		t.Fatalf("同租户 analyst 修改他人会话应 404，got %d", code)
	}
	code, body, _ := rawDo(t, ts, "POST", "/api/message", memberCookie, member.CSRFRaw, map[string]any{
		"text": "试图接管", "session_id": "same-tenant-owned", "customer": "客户B",
	})
	if code != http.StatusNotFound {
		t.Fatalf("同租户 analyst 接管他人会话应 404，got %d: %v", code, body)
	}
	ownerView, err := a.RT.History.GetSession(ownerScope, "same-tenant-owned", "")
	if err != nil {
		t.Fatal(err)
	}
	if ownerView.Session.Title != "Owner 的会话" || ownerView.Session.Customer != "客户A" {
		t.Fatalf("被拒后原会话不应被修改: %+v", ownerView.Session)
	}
}

// TestCrossTenantMemoryIsolation 同名客户互不覆盖、互不可见；A 独有记忆 B
// 检索 0 命中 / 直读 404；产品系统层只读不被租户改写。
func TestCrossTenantMemoryIsolation(t *testing.T) {
	ts, _, a, b := setupTwoTenantServer(t)

	// 同名客户「某集团」：A/B 各自建立，内容各自独立（互不覆盖）
	if code, body, _ := rawDo(t, ts, "POST", "/api/memory", a.Cookie, a.CSRF, map[string]any{
		"type": "customer", "title": "某集团", "content": "A 的客户画像",
	}); code != http.StatusOK {
		t.Fatalf("A 建客户应 200，got %d: %v", code, body)
	}
	if code, body, _ := rawDo(t, ts, "POST", "/api/memory", b.Cookie, b.CSRF, map[string]any{
		"type": "customer", "title": "某集团", "content": "B 的客户画像",
	}); code != http.StatusOK {
		t.Fatalf("B 建同名客户应 200，got %d: %v", code, body)
	}
	_, ba, _ := rawDo(t, ts, "GET", "/api/memory/customer/某集团", a.Cookie, "", nil)
	if !strings.Contains(ba["content"].(string), "A 的客户画像") {
		t.Fatalf("A 应读到自己的客户画像: %v", ba)
	}
	_, bb, _ := rawDo(t, ts, "GET", "/api/memory/customer/某集团", b.Cookie, "", nil)
	if !strings.Contains(bb["content"].(string), "B 的客户画像") {
		t.Fatalf("B 应读到自己的客户画像: %v", bb)
	}

	// A 独有客户：B 直读 404、检索 0 命中；A 检索某集团 1 命中
	if code, _, _ := rawDo(t, ts, "POST", "/api/memory", a.Cookie, a.CSRF, map[string]any{
		"type": "customer", "title": "A独有客户", "content": "仅 A",
	}); code != http.StatusOK {
		t.Fatalf("A 建独有客户应 200，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "GET", "/api/memory/customer/A独有客户", b.Cookie, "", nil); code != http.StatusNotFound {
		t.Fatalf("B 读 A 独有客户应 404，got %d", code)
	}
	// 检索隔离：B 搜 A 的客户标题，绝不命中 A 的条目（fuzzy 检索可能命中 B 自己
	// 重叠字，但结果只能是 B 自己的数据）
	_, sb, _ := rawDo(t, ts, "GET", "/api/memory/search?q=A独有客户&type=customer", b.Cookie, "", nil)
	if containsStr(titlesOf(sb), "A独有客户") {
		t.Fatalf("B 检索不应命中 A 的客户: %v", sb)
	}
	_, sa, _ := rawDo(t, ts, "GET", "/api/memory/search?q=某集团&type=customer", a.Cookie, "", nil)
	if !containsStr(titlesOf(sa), "某集团") {
		t.Fatalf("A 检索某集团应命中自己的条目: %v", sa)
	}

	// 产品系统层只读：租户试图覆盖产品 → 403（明确的权限边界）
	if code, body, _ := rawDo(t, ts, "POST", "/api/memory", a.Cookie, a.CSRF, map[string]any{
		"type": "product", "title": "雷池", "content": "租户篡改",
	}); code != http.StatusForbidden {
		t.Fatalf("租户写产品应 403，got %d: %v", code, body)
	}
}

// TestCrossTenantReviewIsolation 审核队列按租户隔离：A/B 各写待审只见自己；
// B 批准 A 的待审 404；同名威胁各自审核互不干扰。
func TestCrossTenantReviewIsolation(t *testing.T) {
	ts, _, a, b := setupTwoTenantServer(t)

	// AI 专属语义：直接经租户 wiki store 打 pending（人工通道强制 verified）
	if err := a.Wiki.UpsertEntry(&longterm.Entry{Type: domain.MemoryThreat, Title: "A威胁", Content: "A内容", Status: longterm.StatusPendingReview}); err != nil {
		t.Fatal(err)
	}
	if err := b.Wiki.UpsertEntry(&longterm.Entry{Type: domain.MemoryThreat, Title: "B威胁", Content: "B内容", Status: longterm.StatusPendingReview}); err != nil {
		t.Fatal(err)
	}
	_, pa, _ := rawDo(t, ts, "GET", "/api/review/pending", a.Cookie, "", nil)
	if int(pa["count"].(float64)) != 1 || !containsStr(titlesOf(pa), "A威胁") || containsStr(titlesOf(pa), "B威胁") {
		t.Fatalf("A 待审应只含 A威胁: %v", pa)
	}
	_, pb, _ := rawDo(t, ts, "GET", "/api/review/pending", b.Cookie, "", nil)
	if int(pb["count"].(float64)) != 1 || !containsStr(titlesOf(pb), "B威胁") || containsStr(titlesOf(pb), "A威胁") {
		t.Fatalf("B 待审应只含 B威胁: %v", pb)
	}
	// B 批准 A 的待审 → 404（不串审）
	if code, _, _ := rawDo(t, ts, "POST", "/api/review/threat/A威胁/approve", b.Cookie, b.CSRF, nil); code != http.StatusNotFound {
		t.Fatalf("B 批准 A 的待审应 404，got %d", code)
	}

	// 同名威胁：A/B 各自建同名待审，A 批准不影响 B
	if err := a.Wiki.UpsertEntry(&longterm.Entry{Type: domain.MemoryThreat, Title: "同名威胁", Content: "A版本", Status: longterm.StatusPendingReview}); err != nil {
		t.Fatal(err)
	}
	if err := b.Wiki.UpsertEntry(&longterm.Entry{Type: domain.MemoryThreat, Title: "同名威胁", Content: "B版本", Status: longterm.StatusPendingReview}); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := rawDo(t, ts, "POST", "/api/review/threat/同名威胁/approve", a.Cookie, a.CSRF, nil); code != http.StatusOK {
		t.Fatalf("A 批准同名威胁应 200，got %d", code)
	}
	_, ga, _ := rawDo(t, ts, "GET", "/api/memory/threat/同名威胁", a.Cookie, "", nil)
	if ga["status"] != "verified" {
		t.Fatalf("A 批准后应为 verified: %v", ga)
	}
	_, gb, _ := rawDo(t, ts, "GET", "/api/memory/threat/同名威胁", b.Cookie, "", nil)
	if gb["status"] != "pending_review" {
		t.Fatalf("B 的同名威胁应保持 pending_review（审核不串）: %v", gb)
	}
}

// TestCrossTenantTaskIsolation 后台任务按租户过滤：A 提交的任务 B 列表不可见。
func TestCrossTenantTaskIsolation(t *testing.T) {
	ts, _, a, b := setupTwoTenantServer(t)

	if code, _, _ := rawDo(t, ts, "POST", "/api/tasks/lint", a.Cookie, a.CSRF, nil); code != http.StatusAccepted {
		t.Fatalf("A 提交 lint 应 202，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "POST", "/api/tasks/consolidate", a.Cookie, a.CSRF, map[string]any{
		"type": "threat", "title": "x",
	}); code != http.StatusAccepted {
		t.Fatalf("A 提交 consolidate 应 202，got %d", code)
	}
	_, la, _ := rawDo(t, ts, "GET", "/api/tasks", a.Cookie, "", nil)
	itemsA, _ := la["items"].([]any)
	if len(itemsA) != 2 {
		t.Fatalf("A 应见 2 条任务，got %d: %v", len(itemsA), la)
	}
	for _, it := range itemsA {
		if m := it.(map[string]any); m["tenant_id"] != a.TenantID {
			t.Fatalf("任务应属 A 租户: %v", m)
		}
	}
	_, lb, _ := rawDo(t, ts, "GET", "/api/tasks", b.Cookie, "", nil)
	itemsB, _ := lb["items"].([]any)
	if len(itemsB) != 0 {
		t.Fatalf("B 不应见 A 的任务，got %d: %v", len(itemsB), lb)
	}
}

// TestCrossTenantSessionListIsolation 会话列表按租户隔离：各自只见自己建的会话。
func TestCrossTenantSessionListIsolation(t *testing.T) {
	ts, _, a, b := setupTwoTenantServer(t)

	if code, _, _ := rawDo(t, ts, "POST", "/api/message", a.Cookie, a.CSRF, map[string]any{
		"text": "A 首条", "session_id": "sess-a1",
	}); code != http.StatusAccepted {
		t.Fatalf("A 建会话应 202，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "POST", "/api/message", b.Cookie, b.CSRF, map[string]any{
		"text": "B 首条", "session_id": "sess-b1",
	}); code != http.StatusAccepted {
		t.Fatalf("B 建会话应 202，got %d", code)
	}
	_, la, _ := rawDo(t, ts, "GET", "/api/sessions?limit=50", a.Cookie, "", nil)
	sidsA := sessionIDs(la)
	if !containsStr(sidsA, "sess-a1") || containsStr(sidsA, "sess-b1") {
		t.Fatalf("A 会话列表应只见 sess-a1: %v", sidsA)
	}
	_, lb, _ := rawDo(t, ts, "GET", "/api/sessions?limit=50", b.Cookie, "", nil)
	sidsB := sessionIDs(lb)
	if !containsStr(sidsB, "sess-b1") || containsStr(sidsB, "sess-a1") {
		t.Fatalf("B 会话列表应只见 sess-b1: %v", sidsB)
	}
}

// addMember 在指定租户下创建给定角色的成员用户并登录（P0-06 RBAC 矩阵用）。
func addMember(t *testing.T, idSvc *identity.Service, tenantID, email, role string) *tenantAuth {
	t.Helper()
	ctx := context.Background()
	userID := auth.NewRandomHex(16)
	hash, err := auth.HashPassword("password-123456")
	if err != nil {
		t.Fatal(err)
	}
	st := idSvc.Store()
	if err := st.CreateUser(&identity.User{
		ID: userID, EmailNorm: identity.NormalizeEmail(email), DisplayName: email,
		PasswordHash: hash, Status: identity.UserStatusActive,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateMembership(&identity.Membership{
		TenantID: tenantID, UserID: userID, Role: role,
		Status: identity.MembershipStatusActive, CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	res, err := idSvc.Login(ctx, email, "password-123456", "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	return &tenantAuth{
		Cookie: auth.DevSessionCookie + "=" + res.TokenRaw, CSRF: res.CSRFRaw,
		UserID: userID, TenantID: tenantID,
	}
}

// TestTenantRoleRBACMatrix P0-06 验收：租户内 RBAC 落到业务修改路由——
// analyst 只读分析（写/审核/任务全 403），reviewer 可审核但不可管理记忆/触发任务，
// admin/owner 全权；未登录一律 401。每条修改路由过 401/403/200 矩阵。
func TestTenantRoleRBACMatrix(t *testing.T) {
	ts, idSvc, a, _ := setupTwoTenantServer(t)
	analyst := addMember(t, idSvc, a.TenantID, "analyst@example.com", domain.RoleAnalyst)
	reviewer := addMember(t, idSvc, a.TenantID, "reviewer@example.com", domain.RoleReviewer)
	admin := addMember(t, idSvc, a.TenantID, "admin@example.com", domain.RoleAdmin)

	// 预置待审威胁：reviewer 批准一个、驳回一个（各自独立，不互相消费）。
	seedPending := func(title string) {
		if err := a.Wiki.UpsertEntry(&longterm.Entry{Type: domain.MemoryThreat, Title: title, Content: "内容", Status: longterm.StatusPendingReview}); err != nil {
			t.Fatal(err)
		}
	}
	seedPending("RBAC批准")
	seedPending("RBAC驳回")
	// 预置一条已审客户（admin 可删除的对象）。
	if err := a.Wiki.UpsertEntry(&longterm.Entry{Type: domain.MemoryCustomer, Title: "RBAC客户", Content: "画像", Status: longterm.StatusVerified}); err != nil {
		t.Fatal(err)
	}

	// ── 未登录 → 401 ──
	for _, tc := range []struct{ method, path string }{
		{"POST", "/api/memory"},
		{"DELETE", "/api/memory/customer/RBAC客户"},
		{"POST", "/api/review/threat/RBAC批准/approve"},
		{"POST", "/api/review/threat/RBAC驳回/reject"},
		{"POST", "/api/tasks/consolidate"},
		{"POST", "/api/tasks/lint"},
	} {
		if code, _, _ := rawDo(t, ts, tc.method, tc.path, "", "", nil); code != http.StatusUnauthorized {
			t.Fatalf("未登录 %s %s 应 401，got %d", tc.method, tc.path, code)
		}
	}

	// ── analyst：只读可，写/审核/任务全 403 ──
	if code, _, _ := rawDo(t, ts, "GET", "/api/memory/list", analyst.Cookie, "", nil); code != http.StatusOK {
		t.Fatalf("analyst 读记忆应 200，got %d", code)
	}
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{"POST", "/api/memory", map[string]any{"type": "customer", "title": "X", "content": "Y"}},
		{"DELETE", "/api/memory/customer/RBAC客户", nil},
		{"POST", "/api/review/threat/RBAC批准/approve", nil},
		{"POST", "/api/review/threat/RBAC驳回/reject", nil},
		{"POST", "/api/tasks/consolidate", map[string]any{"type": "threat", "title": "x"}},
		{"POST", "/api/tasks/lint", nil},
	} {
		if code, _, _ := rawDo(t, ts, tc.method, tc.path, analyst.Cookie, analyst.CSRF, tc.body); code != http.StatusForbidden {
			t.Fatalf("analyst %s %s 应 403，got %d", tc.method, tc.path, code)
		}
	}

	// ── reviewer：审核可（approve/reject 200），管理/任务 403 ──
	if code, _, _ := rawDo(t, ts, "POST", "/api/review/threat/RBAC批准/approve", reviewer.Cookie, reviewer.CSRF, nil); code != http.StatusOK {
		t.Fatalf("reviewer 批准应 200，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "POST", "/api/review/threat/RBAC驳回/reject", reviewer.Cookie, reviewer.CSRF, nil); code != http.StatusOK {
		t.Fatalf("reviewer 驳回应 200，got %d", code)
	}
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{"POST", "/api/memory", map[string]any{"type": "customer", "title": "X", "content": "Y"}},
		{"DELETE", "/api/memory/customer/RBAC客户", nil},
		{"POST", "/api/tasks/consolidate", map[string]any{"type": "threat", "title": "x"}},
		{"POST", "/api/tasks/lint", nil},
	} {
		if code, _, _ := rawDo(t, ts, tc.method, tc.path, reviewer.Cookie, reviewer.CSRF, tc.body); code != http.StatusForbidden {
			t.Fatalf("reviewer %s %s 应 403，got %d", tc.method, tc.path, code)
		}
	}

	// ── admin/owner：全权（写/删/审核/任务全部成功）──
	for _, tc := range []struct {
		role string
		u    *tenantAuth
	}{
		{"admin", admin}, {"owner", a},
	} {
		seedPending("RBAC审核-" + tc.role) // 各自预置待审条目供 approve 消费
		if code, _, _ := rawDo(t, ts, "POST", "/api/memory", tc.u.Cookie, tc.u.CSRF, map[string]any{
			"type": "customer", "title": "RBAC新增", "content": "画像",
		}); code != http.StatusOK {
			t.Fatalf("%s 写记忆应 200，got %d", tc.role, code)
		}
		if code, _, _ := rawDo(t, ts, "DELETE", "/api/memory/customer/RBAC客户", tc.u.Cookie, tc.u.CSRF, nil); code != http.StatusOK {
			t.Fatalf("%s 删记忆应 200，got %d", tc.role, code)
		}
		if code, _, _ := rawDo(t, ts, "POST", "/api/review/threat/RBAC审核-"+tc.role+"/approve", tc.u.Cookie, tc.u.CSRF, nil); code != http.StatusOK {
			t.Fatalf("%s 审核应 200，got %d", tc.role, code)
		}
		if code, _, _ := rawDo(t, ts, "POST", "/api/tasks/consolidate", tc.u.Cookie, tc.u.CSRF, map[string]any{
			"type": "threat", "title": "x",
		}); code != http.StatusAccepted {
			t.Fatalf("%s 触发 consolidate 应 202，got %d", tc.role, code)
		}
		if code, _, _ := rawDo(t, ts, "POST", "/api/tasks/lint", tc.u.Cookie, tc.u.CSRF, nil); code != http.StatusAccepted {
			t.Fatalf("%s 触发 lint 应 202，got %d", tc.role, code)
		}
	}
}

// TestMemoryReadOnlyAndIdentityBoundUserProfiles 验证记忆库权限与使用者模型：
// 普通成员可读取平台知识但不能修改；每个租户成员自动获得 user_id 绑定画像，
// 本人可维护自己的画像但不能修改他人画像；owner 仍可维护租户知识覆盖层。
func TestMemoryReadOnlyAndIdentityBoundUserProfiles(t *testing.T) {
	ts, idSvc, owner, _ := setupTwoTenantServer(t)
	ctx := context.Background()

	if code, body, _ := rawDo(t, ts, "POST", "/api/members", owner.Cookie, owner.CSRF, map[string]any{
		"email": "profile-member@example.com", "display_name": "画像成员",
		"password": "password-123456", "role": domain.RoleAnalyst,
	}); code != http.StatusOK {
		t.Fatalf("owner 添加画像成员应 200，got %d: %v", code, body)
	}
	memberAuth, err := idSvc.Login(ctx, "profile-member@example.com", "password-123456", "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	memberCookie := "cda_session=" + memberAuth.TokenRaw

	// 普通成员读取使用者列表时，身份控制面的当前租户成员会自动同步为画像。
	code, profiles, _ := rawDo(t, ts, "GET", "/api/memory/list?type=user&limit=100", memberCookie, "", nil)
	if code != http.StatusOK {
		t.Fatalf("普通成员读取使用者列表应 200，got %d: %v", code, profiles)
	}
	titlesByID := map[string]string{}
	for _, raw := range profiles["items"].([]any) {
		item := raw.(map[string]any)
		id, _ := item["user_id"].(string)
		title, _ := item["title"].(string)
		if id == "" {
			t.Fatalf("列表不得展示无 user_id 的旧演示画像: %v", item)
		}
		titlesByID[id] = title
	}
	memberTitle := titlesByID[memberAuth.User.ID]
	ownerTitle := titlesByID[owner.UserID]
	if memberTitle == "" || ownerTitle == "" {
		t.Fatalf("应同时生成 owner 与成员画像，got %v", titlesByID)
	}

	// 产品/威胁/合规/行业对普通成员只读：GET 可用，通用写入拒绝。
	if code, _, _ := rawDo(t, ts, "GET", "/api/memory/list?type=product", memberCookie, "", nil); code != http.StatusOK {
		t.Fatalf("普通成员读产品应 200，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "POST", "/api/memory", memberCookie, memberAuth.CSRFRaw, map[string]any{
		"type": "threat", "title": "普通成员伪造威胁", "content": "不应写入",
	}); code != http.StatusForbidden {
		t.Fatalf("普通成员写威胁应 403，got %d", code)
	}

	// 使用者本人可维护自己的内容，但不能改 owner 的画像。
	if code, body, _ := rawDo(t, ts, "POST", "/api/memory", memberCookie, memberAuth.CSRFRaw, map[string]any{
		"type": "user", "title": memberTitle, "content": "偏好简洁结论和明确下一步",
	}); code != http.StatusOK {
		t.Fatalf("本人编辑使用者画像应 200，got %d: %v", code, body)
	}
	if code, body, _ := rawDo(t, ts, "POST", "/api/memory", memberCookie, memberAuth.CSRFRaw, map[string]any{
		"type": "user", "title": ownerTitle, "content": "越权修改",
	}); code != http.StatusForbidden {
		t.Fatalf("成员编辑他人画像应 403，got %d: %v", code, body)
	}

	// owner 可维护租户行业覆盖，但产品仍属于平台系统只读基线。
	if code, body, _ := rawDo(t, ts, "POST", "/api/memory", owner.Cookie, owner.CSRF, map[string]any{
		"type": "industry", "title": "租户行业实践", "content": "本租户验证经验",
	}); code != http.StatusOK {
		t.Fatalf("owner 写租户行业知识应 200，got %d: %v", code, body)
	}
	if code, _, _ := rawDo(t, ts, "POST", "/api/memory", owner.Cookie, owner.CSRF, map[string]any{
		"type": "product", "title": "雷池", "content": "租户篡改",
	}); code != http.StatusForbidden {
		t.Fatalf("owner 写平台产品仍应 403，got %d", code)
	}
}

// TestUserMemberManagement 用户/成员管理（§4.2）：
//   - 平台管理 /api/platform/** 仅 platform_admin；普通用户 403。
//   - 租户成员 /api/members 按 scope 隔离；owner 可增/改/删，analyst 403；
//     跨租户成员不可见（B 看不到 A 添加的成员）。
func TestUserMemberManagement(t *testing.T) {
	ts, idSvc, a, b := setupTwoTenantServer(t)
	ctx := context.Background()
	if err := idSvc.GrantPlatformAdmin(a.UserID, true); err != nil {
		t.Fatal(err)
	}

	// ── 平台管理（仅 platform_admin）──────────────────────────
	// a（platform_admin）可列用户/租户
	code, body, _ := rawDo(t, ts, "GET", "/api/platform/users", a.Cookie, "", nil)
	if code != http.StatusOK {
		t.Fatalf("platform_admin 列用户应 200，got %d: %v", code, body)
	}
	if count, _ := body["count"].(float64); count < 2 {
		t.Fatalf("应列出至少 2 个用户，got %v", body["count"])
	}
	if code, _, _ := rawDo(t, ts, "GET", "/api/platform/tenants", a.Cookie, "", nil); code != http.StatusOK {
		t.Fatalf("platform_admin 列租户应 200，got %d", code)
	}
	// a 可看 B 租户成员（平台视图跨租户只读元数据）
	if code, _, _ := rawDo(t, ts, "GET", "/api/platform/tenants/"+b.TenantID+"/members", a.Cookie, "", nil); code != http.StatusOK {
		t.Fatalf("platform_admin 看 B 租户成员应 200，got %d", code)
	}
	// b（普通用户）访问平台管理 → 403（未登录另测，见 TestTenantRoleRBACMatrix）
	for _, p := range []string{"/api/platform/users", "/api/platform/tenants", "/api/platform/tenants/" + b.TenantID + "/members"} {
		if code, _, _ := rawDo(t, ts, "GET", p, b.Cookie, "", nil); code != http.StatusForbidden {
			t.Fatalf("普通用户 GET %s 应 403，got %d", p, code)
		}
	}

	// ── 租户成员管理（A 的 owner）────────────────────────────
	// A owner 添加新成员（新邮箱 → 建号 + 加成员）
	if code, body, _ := rawDo(t, ts, "POST", "/api/members", a.Cookie, a.CSRF, map[string]any{
		"email": "m1@example.com", "display_name": "成员一", "password": "password-123456", "role": "analyst",
	}); code != http.StatusOK {
		t.Fatalf("owner 添加成员应 200，got %d: %v", code, body)
	}
	// A 列成员 → 2（a + m1）；B 列成员 → 1（只有 b）
	_, ba, _ := rawDo(t, ts, "GET", "/api/members", a.Cookie, "", nil)
	if count, _ := ba["count"].(float64); count != 2 {
		t.Fatalf("A 成员数应为 2，got %v", ba["count"])
	}
	_, bb, _ := rawDo(t, ts, "GET", "/api/members", b.Cookie, "", nil)
	if count, _ := bb["count"].(float64); count != 1 {
		t.Fatalf("B 成员数应为 1，got %v", bb["count"])
	}
	if containsStr(bodyStrings(bb), "m1@example.com") {
		t.Fatalf("B 不应看到 A 添加的成员: %v", bb)
	}

	// B 不能用 A 的成员 user_id 改角色（该 user_id 在 B 租户不存在 → 404）
	// 先拿到 m1 的 user_id
	var m1ID string
	if items, ok := ba["items"].([]any); ok {
		for _, it := range items {
			m := it.(map[string]any)
			if m["email"] == "m1@example.com" {
				m1ID, _ = m["user_id"].(string)
			}
		}
	}
	if m1ID == "" {
		t.Fatal("未找到 m1 的 user_id")
	}
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/members/"+m1ID+"/role", b.Cookie, b.CSRF, map[string]any{"role": "reviewer"}); code != http.StatusNotFound {
		t.Fatalf("B 改 A 成员角色应 404，got %d", code)
	}

	// m1（analyst）无权添加/改角色/移除成员
	m1res, err := idSvc.Login(ctx, "m1@example.com", "password-123456", "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	m1Cookie := "cda_session=" + m1res.TokenRaw
	for _, tc := range []struct {
		method, path string
		body         any
	}{
		{"POST", "/api/members", map[string]any{"email": "x@example.com", "display_name": "X", "password": "password-123456", "role": "analyst"}},
		{"PATCH", "/api/members/" + a.UserID + "/role", map[string]any{"role": "analyst"}},
		{"DELETE", "/api/members/" + a.UserID, nil},
	} {
		if code, _, _ := rawDo(t, ts, tc.method, tc.path, m1Cookie, m1res.CSRFRaw, tc.body); code != http.StatusForbidden {
			t.Fatalf("analyst %s %s 应 403，got %d", tc.method, tc.path, code)
		}
	}

	// A owner 改 m1 角色 → 200；移除 m1 → 200
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/members/"+m1ID+"/role", a.Cookie, a.CSRF, map[string]any{"role": "reviewer"}); code != http.StatusOK {
		t.Fatalf("owner 改成员角色应 200，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "DELETE", "/api/members/"+m1ID, a.Cookie, a.CSRF, nil); code != http.StatusOK {
		t.Fatalf("owner 移除成员应 200，got %d", code)
	}

	// ── 平台禁用账号 → 会话立即失效 ──────────────────────────
	// 先为 B 补一位 owner；唯一 owner 保护会拒绝直接禁用，防止产生无人可管理租户。
	bScope := &domain.TenantScope{TenantID: b.TenantID, UserID: b.UserID, Roles: []string{domain.RoleOwner}}
	if err := idSvc.MembersAdd(bScope, "b-co-owner@example.com", "B 联合所有者",
		"password-123456", domain.RoleOwner); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/platform/users/"+b.UserID+"/status", a.Cookie, a.CSRF, map[string]any{"status": "disabled"}); code != http.StatusOK {
		t.Fatalf("platform_admin 禁用账号应 200，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "GET", "/api/sessions", b.Cookie, "", nil); code != http.StatusUnauthorized {
		t.Fatalf("禁用后 B 会话应失效 401，got %d", code)
	}
}

// TestWorkspaceSwitchAndSharedTenantData 已有账号加入第二租户后，必须通过服务端
// membership 校验切换工作空间；切换后读取目标租户共享记忆，原租户数据不可见。
func TestWorkspaceSwitchAndSharedTenantData(t *testing.T) {
	ts, _, a, b := setupTwoTenantServer(t)

	// A owner 把已有的 B 用户加入 A，复用身份，不传姓名/密码。
	if code, body, _ := rawDo(t, ts, "POST", "/api/members", a.Cookie, a.CSRF, map[string]any{
		"email": "b@example.com", "role": domain.RoleAnalyst,
	}); code != http.StatusOK {
		t.Fatalf("已有用户加入第二租户应 200，got %d: %v", code, body)
	}
	if code, body, _ := rawDo(t, ts, "POST", "/api/memory", a.Cookie, a.CSRF, map[string]any{
		"type": "customer", "title": "A共享客户", "content": "A 租户共享画像",
	}); code != http.StatusOK {
		t.Fatalf("A 创建共享画像应 200，got %d: %v", code, body)
	}
	if code, body, _ := rawDo(t, ts, "POST", "/api/memory", b.Cookie, b.CSRF, map[string]any{
		"type": "customer", "title": "B独有客户", "content": "B 数据",
	}); code != http.StatusOK {
		t.Fatalf("B 创建画像应 200，got %d: %v", code, body)
	}

	// B 当前会话切换到 A；响应轮换 CSRF，并列出两个可用工作空间。
	code, body, _ := rawDo(t, ts, "POST", "/api/auth/switch-tenant", b.Cookie, b.CSRF,
		map[string]any{"tenant_id": a.TenantID})
	if code != http.StatusOK {
		t.Fatalf("切换到 A 应 200，got %d: %v", code, body)
	}
	if body["tenant_id"] != a.TenantID {
		t.Fatalf("响应应切到 A: %v", body)
	}
	if ws, _ := body["workspaces"].([]any); len(ws) != 2 {
		t.Fatalf("应返回两个工作空间: %v", body["workspaces"])
	}
	newCSRF, _ := body["csrf"].(string)
	if newCSRF == "" || newCSRF == b.CSRF {
		t.Fatal("切换租户必须轮换 CSRF")
	}

	if code, body, _ := rawDo(t, ts, "GET", "/api/memory/customer/A共享客户", b.Cookie, "", nil); code != http.StatusOK {
		t.Fatalf("切换后应读取 A 共享画像，got %d: %v", code, body)
	}
	if code, _, _ := rawDo(t, ts, "GET", "/api/memory/customer/B独有客户", b.Cookie, "", nil); code != http.StatusNotFound {
		t.Fatalf("切换到 A 后不应读取 B 数据，got %d", code)
	}
	// 旧 CSRF 已失效；使用新 CSRF 也不能切到无 membership 的租户。
	if code, _, _ := rawDo(t, ts, "POST", "/api/auth/switch-tenant", b.Cookie, b.CSRF,
		map[string]any{"tenant_id": b.TenantID}); code != http.StatusForbidden {
		t.Fatalf("旧 CSRF 应 403，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "POST", "/api/auth/switch-tenant", b.Cookie, newCSRF,
		map[string]any{"tenant_id": strings.Repeat("f", 32)}); code != http.StatusForbidden {
		t.Fatalf("未授权切换应 403，got %d", code)
	}
}

// bodyStrings 提取响应 items 数组全部字符串字段值（判断跨租户内容可见性）。
func bodyStrings(body map[string]any) []string {
	items, _ := body["items"].([]any)
	var out []string
	for _, it := range items {
		m, _ := it.(map[string]any)
		for _, v := range m {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}
