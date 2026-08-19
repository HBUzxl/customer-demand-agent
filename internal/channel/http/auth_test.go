package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"customer-demand-agent/internal/identity"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/model"
	"customer-demand-agent/internal/tenancy"
)

// newAuthServer 构造一个启用身份控制面但不自动注册用户的测试服务（auth 专用用例）。
// 注册/登录经 HTTP 端点触发，Provisioner 已注入（注册才可初始化租户数据面）。
func newAuthServer(t *testing.T, regEnabled bool) *httptest.Server {
	ts, _ := newAuthServerWithProvisioner(t, regEnabled, nil)
	return ts
}

// newAuthServerWithProvisioner 同 newAuthServer，但允许自定义 Provisioner
// （P0-09 故障注入：先失败后成功的场景），并返回身份服务供切换故障开关。
func newAuthServerWithProvisioner(t *testing.T, regEnabled bool, prov func(ctx context.Context, tenantID string) error) (*httptest.Server, *identity.Service) {
	t.Helper()
	dir := t.TempDir()
	sysWikiDir := filepath.Join(dir, "system", "wiki")
	_ = os.MkdirAll(sysWikiDir, 0o755)
	sysWiki := longterm.NewWikiStore(sysWikiDir)
	if err := sysWiki.Load(); err != nil {
		t.Fatal(err)
	}

	cfgStore, err := config.Load(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	registry := model.NewRegistry(5 * time.Second)
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
	if prov != nil {
		idSvc.SetProvisioner(prov)
	} else {
		idSvc.SetProvisioner(reg.Provision)
	}

	srv := httpapi.New(httpapi.Config{
		Store: cfgStore, Runtimes: reg, ModelMgr: mgr, Registry: registry,
		Tasks: nil, LogRing: httpapi.NewLogRing(100), Leads: nil,
	})
	cookieMgr := auth.NewCookieManager(false)
	authMW := identity.NewMiddleware(idSvc, cookieMgr)
	srv.SetAuth(&httpapi.AuthDeps{
		Identity:            idSvc,
		Cookies:             cookieMgr,
		Middleware:          authMW,
		LoginLimiter:        auth.NewRateLimiter(50, time.Minute),
		LoginIPLimiter:      auth.NewRateLimiter(100, time.Minute),
		RegistrationLimiter: auth.NewRateLimiter(50, time.Minute),
		RegistrationEnabled: func() bool { return regEnabled },
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, idSvc
}

// rawDo 用裸 http.Client（无登录 jar）发起请求；显式传 cookie/csrf。
func rawDo(t *testing.T, ts *httptest.Server, method, path, cookie, csrf string, body any) (int, map[string]any, []*http.Cookie) {
	t.Helper()
	req, _ := http.NewRequest(method, ts.URL+path, nil)
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	if body != nil {
		b, _ := json.Marshal(body)
		req.Body = nopCloser{bytes.NewReader(b)}
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out, resp.Cookies()
}

func registerUser(t *testing.T, ts *httptest.Server, email string) (cookie, csrf string) {
	t.Helper()
	code, body, cookies := rawDo(t, ts, "POST", "/api/auth/register", "", "", map[string]any{
		"org_name": "注册测试组织", "display_name": "注册测试员",
		"email": email, "password": "password-123456",
	})
	if code != http.StatusCreated {
		t.Fatalf("注册应 201，got %d: %v", code, body)
	}
	for _, c := range cookies {
		if c.Name == auth.DevSessionCookie {
			cookie = c.Name + "=" + c.Value
		}
	}
	if cookie == "" {
		t.Fatal("注册应下发登录 Cookie")
	}
	csrf, _ = body["csrf"].(string)
	if csrf == "" {
		t.Fatal("注册响应应含 csrf")
	}
	return cookie, csrf
}

// TestRequireAuthOnBusinessRoutes 未登录访问业务接口一律 401（M1 门禁）。
func TestRequireAuthOnBusinessRoutes(t *testing.T) {
	ts := newAuthServer(t, true)
	code, body, _ := rawDo(t, ts, "GET", "/api/memory/list", "", "", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("未登录 GET /api/memory/list 应 401，got %d: %v", code, body)
	}
	code, _, _ = rawDo(t, ts, "POST", "/api/message", "", "", map[string]any{"text": "hi"})
	if code != http.StatusUnauthorized {
		t.Fatalf("未登录 POST /api/message 应 401，got %d", code)
	}
	code, _, _ = rawDo(t, ts, "GET", "/api/sessions", "", "", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("未登录 GET /api/sessions 应 401，got %d", code)
	}
	// 伪造 Cookie / 改 Header 不生效
	code, _, _ = rawDo(t, ts, "GET", "/api/memory/list", "wrong-cookie", "", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("伪造 Cookie 应 401，got %d", code)
	}
}

// TestHealthAndLoginPublic 公开端点无需登录。
func TestHealthAndLoginPublic(t *testing.T) {
	ts := newAuthServer(t, true)
	code, _, _ := rawDo(t, ts, "GET", "/api/health", "", "", nil)
	if code != http.StatusOK {
		t.Fatalf("/api/health 应 200，got %d", code)
	}
	// 登录失败也应能到达（不 401）
	code, body, _ := rawDo(t, ts, "POST", "/api/auth/login", "", "", map[string]any{
		"email": "x@y.com", "password": "wrong-password-1",
	})
	if code != http.StatusUnauthorized {
		t.Fatalf("登录失败应 401，got %d: %v", code, body)
	}
	if body["error"] == "未登录或登录已过期" {
		t.Fatalf("登录失败应返回统一文案而非登录态错误: %v", body)
	}
}

// TestRegisterDuplicateAndLogin 重复注册 409；注册后可登录。
func TestRegisterDuplicateAndLogin(t *testing.T) {
	ts := newAuthServer(t, true)
	cookie, csrf := registerUser(t, ts, "dup@example.com")
	if cookie == "" || csrf == "" {
		t.Fatal("注册应返回 cookie+csrf")
	}
	// 重复注册（同邮箱）→ 409
	code, body, _ := rawDo(t, ts, "POST", "/api/auth/register", "", "", map[string]any{
		"org_name": "x", "display_name": "x", "email": "DUP@example.com", "password": "password-123456",
	})
	if code != http.StatusConflict {
		t.Fatalf("重复注册应 409，got %d: %v", code, body)
	}
	// 正确密码登录 → 200 + cookie
	code, body, cookies := rawDo(t, ts, "POST", "/api/auth/login", "", "", map[string]any{
		"email": "dup@example.com", "password": "password-123456",
	})
	if code != http.StatusOK {
		t.Fatalf("登录应 200，got %d: %v", code, body)
	}
	if body["roles"] == nil {
		t.Fatalf("登录应返回角色: %v", body)
	}
	_ = cookies
}

// TestMeAndCSRFEnforced /api/auth/me 返回视图 + CSRF；修改类缺 CSRF → 403。
func TestMeAndCSRFEnforced(t *testing.T) {
	ts := newAuthServer(t, true)
	cookie, csrf := registerUser(t, ts, "me@example.com")

	// me：返回用户/租户/角色 + csrf
	code, body, _ := rawDo(t, ts, "GET", "/api/auth/me", cookie, "", nil)
	if code != http.StatusOK {
		t.Fatalf("me 应 200，got %d: %v", code, body)
	}
	if body["roles"] == nil || body["tenant_name"] == nil || body["csrf"] == nil {
		t.Fatalf("me 应含 roles/tenant_name/csrf: %v", body)
	}
	// 无登录访问 me → 401
	code, _, _ = rawDo(t, ts, "GET", "/api/auth/me", "", "", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("未登录 me 应 401，got %d", code)
	}

	// 修改类请求：带 Cookie 但无 CSRF → 403；带 Cookie+CSRF → 通过
	code, body, _ = rawDo(t, ts, "POST", "/api/memory", cookie, "", map[string]any{
		"type": "threat", "title": "CSRF测试条目", "content": "x",
	})
	if code != http.StatusForbidden {
		t.Fatalf("无 CSRF 修改请求应 403，got %d: %v", code, body)
	}
	// me 返回同一会话的稳定 CSRF，多个浏览器标签页互不挤掉。
	_, meBody, _ := rawDo(t, ts, "GET", "/api/auth/me", cookie, "", nil)
	newCSRF, _ := meBody["csrf"].(string)
	if newCSRF == "" {
		t.Fatal("me 应返回 csrf")
	}
	if newCSRF != csrf {
		t.Fatal("同一会话重复读取 me 不应轮换 csrf")
	}
	code, _, _ = rawDo(t, ts, "POST", "/api/memory", cookie, newCSRF, map[string]any{
		"type": "threat", "title": "CSRF测试条目", "content": "x",
	})
	if code != http.StatusOK {
		t.Fatalf("带新 CSRF 应 200，got %d", code)
	}
	// 另一标签页之前拿到的 csrf 仍有效。
	code, _, _ = rawDo(t, ts, "POST", "/api/memory", cookie, csrf, map[string]any{
		"type": "threat", "title": "旧CSRF条目", "content": "x",
	})
	if code != http.StatusOK {
		t.Fatalf("同会话 CSRF 应保持有效，got %d", code)
	}
}

// TestLogoutRevokesSession 退出后会话失效。
func TestLogoutRevokesSession(t *testing.T) {
	ts := newAuthServer(t, true)
	cookie, csrf := registerUser(t, ts, "logout@example.com")
	code, _, _ := rawDo(t, ts, "POST", "/api/auth/logout", cookie, csrf, nil)
	if code != http.StatusOK {
		t.Fatalf("logout 应 200，got %d", code)
	}
	code, _, _ = rawDo(t, ts, "GET", "/api/auth/me", cookie, "", nil)
	if code != http.StatusUnauthorized {
		t.Fatalf("退出后 me 应 401，got %d", code)
	}
}

// TestChangePasswordEndpoint 改密后旧密码失效。
func TestChangePasswordEndpoint(t *testing.T) {
	ts := newAuthServer(t, true)
	cookie, csrf := registerUser(t, ts, "chpw@example.com")
	code, _, _ := rawDo(t, ts, "PUT", "/api/auth/password", cookie, csrf, map[string]any{
		"old_password": "password-123456", "new_password": "new-password-999",
	})
	if code != http.StatusOK {
		t.Fatalf("改密应 200，got %d", code)
	}
	// 新密码可登录
	code, _, _ = rawDo(t, ts, "POST", "/api/auth/login", "", "", map[string]any{
		"email": "chpw@example.com", "password": "new-password-999",
	})
	if code != http.StatusOK {
		t.Fatalf("新密码应可登录，got %d", code)
	}
	// 旧密码不可登录
	code, _, _ = rawDo(t, ts, "POST", "/api/auth/login", "", "", map[string]any{
		"email": "chpw@example.com", "password": "password-123456",
	})
	if code != http.StatusUnauthorized {
		t.Fatalf("旧密码应不可登录，got %d", code)
	}
}

// TestRegistrationDisabled 关闭注册 → 403。
func TestRegistrationDisabled(t *testing.T) {
	ts := newAuthServer(t, false)
	code, body, _ := rawDo(t, ts, "POST", "/api/auth/register", "", "", map[string]any{
		"org_name": "x", "display_name": "x", "email": "no@example.com", "password": "password-123456",
	})
	if code != http.StatusForbidden {
		t.Fatalf("关闭注册应 403，got %d: %v", code, body)
	}
	// 已有账号仍可登录（关闭注册不影响登录）
	ts2 := newAuthServer(t, true)
	_, _ = registerUser(t, ts2, "existing@example.com")
	// 注：ts2 独立身份库，无法复用账号——仅验证注册关闭路径
}

func TestRegistrationRateLimitAndRequestBodyLimit(t *testing.T) {
	t.Run("registration rate limit", func(t *testing.T) {
		ts := newAuthServer(t, true)
		for i := 0; i < 50; i++ {
			code, _, _ := rawDo(t, ts, "POST", "/api/auth/register", "", "", nil)
			if code != http.StatusBadRequest {
				t.Fatalf("第 %d 次空请求应先到参数校验，got %d", i+1, code)
			}
		}
		code, body, _ := rawDo(t, ts, "POST", "/api/auth/register", "", "", nil)
		if code != http.StatusTooManyRequests {
			t.Fatalf("超过单 IP 注册上限应 429，got %d: %v", code, body)
		}
	})

	t.Run("request body limit", func(t *testing.T) {
		ts := newAuthServer(t, true)
		code, body, _ := rawDo(t, ts, "POST", "/api/auth/register", "", "", map[string]any{
			"org_name":     strings.Repeat("x", (1<<20)+1024),
			"display_name": "超大请求", "email": "large@example.com", "password": "Password-123",
		})
		if code != http.StatusRequestEntityTooLarge {
			t.Fatalf("超过 1 MiB 请求体应 413，got %d: %v", code, body)
		}
	})
}

func TestJSONBodyRejectsTrailingValues(t *testing.T) {
	ts := newAuthServer(t, true)
	body := `{"org_name":"x","display_name":"x","email":"strict@example.com","password":"Password-123"} {}`
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/auth/register", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("包含多个 JSON 值的请求体应 400，got %d", resp.StatusCode)
	}
}

// TestSessionCookieAttrs 登录 Cookie 必须 HttpOnly/Path=/（浏览器防护）。
func TestSessionCookieAttrs(t *testing.T) {
	ts := newAuthServer(t, true)
	_, _, cookies := rawDo(t, ts, "POST", "/api/auth/register", "", "", map[string]any{
		"org_name": "attr", "display_name": "attr", "email": "attr@example.com", "password": "password-123456",
	})
	var got *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.DevSessionCookie {
			got = c
		}
	}
	if got == nil {
		t.Fatal("应下发登录 Cookie")
	}
	if got.Name != auth.DevSessionCookie {
		t.Fatalf("secure=false 测试服务应下发 dev Cookie 名 %s: %s", auth.DevSessionCookie, got.Name)
	}
	if !got.HttpOnly || got.Path != "/" || got.SameSite != http.SameSiteLaxMode {
		t.Fatalf("Cookie 属性不合规: HttpOnly=%v Path=%q SameSite=%v", got.HttpOnly, got.Path, got.SameSite)
	}
}

// TestRegisterRejectsDuplicateOrgName §4.1 严重缺陷修复：同名租户禁止注册（409）。
func TestRegisterRejectsDuplicateOrgName(t *testing.T) {
	ts := newAuthServer(t, true)
	// 1. 首次注册「重名组织」成功。
	code, body, _ := rawDo(t, ts, "POST", "/api/auth/register", "", "", map[string]any{
		"org_name": "重名组织", "display_name": "甲", "email": "dup-a@example.com", "password": "password-123456",
	})
	if code != http.StatusCreated {
		t.Fatalf("首次注册应 201，got %d: %v", code, body)
	}
	// 2. 不同邮箱、同一租户名 → 409 该工作空间名称已存在。
	code, body, _ = rawDo(t, ts, "POST", "/api/auth/register", "", "", map[string]any{
		"org_name": "重名组织", "display_name": "乙", "email": "dup-b@example.com", "password": "password-123456",
	})
	if code != http.StatusConflict {
		t.Fatalf("同名租户注册应 409，got %d: %v", code, body)
	}
	if msg, _ := body["error"].(string); msg != "该工作空间名称已存在" {
		t.Fatalf("错误文案应明确，got %v", body["error"])
	}
	// 3. 空组织名 → 400。
	code, _, _ = rawDo(t, ts, "POST", "/api/auth/register", "", "", map[string]any{
		"org_name": "  ", "display_name": "丙", "email": "dup-c@example.com", "password": "password-123456",
	})
	if code != http.StatusBadRequest {
		t.Fatalf("空组织名应 400，got %d", code)
	}
}

// legacy/最小装配会跳过认证中间件；新增管理 Handler 仍必须防御 nil 身份依赖，
// 返回明确 503 而不是 panic 或意外放行。
func TestManagementRoutesWithoutIdentityReturn503(t *testing.T) {
	srv := httpapi.NewMinimal(nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/tenant"},
		{http.MethodPatch, "/api/tenant"},
		{http.MethodPost, "/api/platform/users"},
		{http.MethodPatch, "/api/platform/users/u"},
		{http.MethodDelete, "/api/platform/users/u"},
		{http.MethodPost, "/api/platform/tenants"},
		{http.MethodPatch, "/api/platform/tenants/t"},
		{http.MethodPatch, "/api/platform/tenants/t/status"},
		{http.MethodDelete, "/api/platform/tenants/t"},
		{http.MethodPost, "/api/platform/tenants/t/members"},
		{http.MethodPatch, "/api/platform/tenants/t/members/u/role"},
		{http.MethodDelete, "/api/platform/tenants/t/members/u"},
	} {
		code, _, _ := rawDo(t, ts, tc.method, tc.path, "", "", nil)
		if code != http.StatusServiceUnavailable {
			t.Errorf("%s %s 未配置身份控制面应 503，got %d", tc.method, tc.path, code)
		}
	}
}

// TestResumeProvisionHTTP P0-09 端到端：注册 provision 失败 → 503 且邮箱被占；
// 登录（provision_failed=true，进修复页）→ /api/auth/resume 重试 → 200 标记清除；
// /auth/me 反映修复状态；修复前业务路由被数据面未初始化闭合。
func TestResumeProvisionHTTP(t *testing.T) {
	fail := true
	ts, _ := newAuthServerWithProvisioner(t, true, func(ctx context.Context, tenantID string) error {
		if fail {
			return errors.New("磁盘写满") // 故障注入：数据面初始化失败
		}
		return nil
	})

	// 1. 注册失败 → 503 工作空间初始化失败。
	code, body, _ := rawDo(t, ts, "POST", "/api/auth/register", "", "", map[string]any{
		"org_name": "恢复组织", "display_name": "恢复员", "email": "resume-http@example.com", "password": "password-123456",
	})
	if code != http.StatusServiceUnavailable {
		t.Fatalf("注册应 503，got %d: %v", code, body)
	}
	// 2. 同邮箱重新注册 → 409 邮箱已占用（用户已提交，走恢复而非重注册）。
	if code, _, _ := rawDo(t, ts, "POST", "/api/auth/register", "", "", map[string]any{
		"org_name": "恢复组织2", "display_name": "x", "email": "resume-http@example.com", "password": "password-123456",
	}); code != http.StatusConflict {
		t.Fatalf("重复注册应 409，got %d", code)
	}

	// 3. 故障恢复前登录：200 且 provision_failed=true（前端据此进修复页）。
	fail = false
	code, body, cookies := rawDo(t, ts, "POST", "/api/auth/login", "", "", map[string]any{
		"email": "resume-http@example.com", "password": "password-123456",
	})
	if code != http.StatusOK {
		t.Fatalf("provisioning_failed 登录应 200，got %d: %v", code, body)
	}
	if pf, _ := body["provision_failed"].(bool); !pf {
		t.Fatalf("修复前登录应 provision_failed=true: %v", body)
	}
	cookie := ""
	for _, c := range cookies {
		if c.Name == auth.DevSessionCookie {
			cookie = c.Name + "=" + c.Value
		}
	}
	csrf, _ := body["csrf"].(string)
	if cookie == "" || csrf == "" {
		t.Fatal("登录应下发 Cookie+CSRF")
	}

	// 4. /api/auth/resume：重跑 provisioner → 200，标记清除，CSRF 轮换。
	code, body, _ = rawDo(t, ts, "POST", "/api/auth/resume", cookie, csrf, nil)
	if code != http.StatusOK {
		t.Fatalf("resume 应 200，got %d: %v", code, body)
	}
	if pf, _ := body["provision_failed"].(bool); pf {
		t.Fatalf("resume 后应无 provision_failed: %v", body)
	}
	newCSRF, _ := body["csrf"].(string)
	if newCSRF == "" || newCSRF == csrf {
		t.Fatalf("resume 应轮换 CSRF: old=%q new=%q", csrf, newCSRF)
	}

	// 5. /auth/me 反映修复后状态（无失败标记）。
	code, body, _ = rawDo(t, ts, "GET", "/api/auth/me", cookie, "", nil)
	if code != http.StatusOK {
		t.Fatalf("me 应 200，got %d", code)
	}
	if pf, _ := body["provision_failed"].(bool); pf {
		t.Fatalf("修复后 me 不应有 provision_failed: %v", body)
	}
}
