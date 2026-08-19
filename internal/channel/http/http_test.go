package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
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
	"customer-demand-agent/internal/tenancy"
)

// 包级认证态：setupServer 注册测试用户后，后续所有请求经 testClient（带登录
// Cookie）发出，修改类请求带 testCSRF。测试包内测试串行执行，无需重置清理。
var (
	testClient *http.Client
	testCSRF   string
)

// setupServer 构建一个完整可用的 HTTP server（多租户装配：system wiki +
// identity 控制面 + tenancy registry + mock 模型），并注册一个测试用户
// （此后业务请求默认已登录）。返回租户复合记忆（rt.Wiki）。
func setupServer(t *testing.T) (*httptest.Server, longterm.Store) {
	t.Helper()
	// 用临时 config.json 加载（测试真实持久化路径）
	configFile := filepath.Join(t.TempDir(), "config.json")
	_ = os.WriteFile(configFile, []byte(`{"server":{"addr":":0"},"default_tenant":"default","models":[{"name":"default","endpoint":"http://localhost:1","api_key":"k","model":"m","temperature":0.3,"max_tokens":1024,"timeout_sec":5}],"router":{"default":"default","routes":{},"fallback":{"max_retries":1,"backoff_base_ms":10,"chain":null}}}`), 0o600)
	return setupServerWithConfig(t, configFile)
}

// setupServerWithConfig 用指定 config.json 构造完整服务（验证回写文件用）。
func setupServerWithConfig(t *testing.T, configFile string) (*httptest.Server, longterm.Store) {
	t.Helper()
	dir := t.TempDir()

	// 平台系统知识：system wiki 播种雷池产品（复合记忆 product → system 只读层）。
	sysWikiDir := filepath.Join(dir, "system", "wiki")
	_ = os.MkdirAll(filepath.Join(sysWikiDir, "产品记忆"), 0o755)
	_ = os.WriteFile(filepath.Join(sysWikiDir, "产品记忆", "雷池.md"),
		[]byte("---\ntype: product\ntitle: 雷池\ntags: [\"WAF\"]\n---\nx"), 0o644)
	sysWiki := longterm.NewWikiStore(sysWikiDir)
	if err := sysWiki.Load(); err != nil {
		t.Fatal(err)
	}

	cfgStore, err := config.Load(configFile)
	if err != nil {
		t.Fatal(err)
	}
	registry := model.NewRegistry(10 * time.Second)
	for _, m := range cfgStore.Get().Models {
		_ = registry.Register(m)
	}
	rc := cfgStore.Get().Router
	mgr := model.NewManager(llm.NewClient(), registry, &rc)

	// 多租户装配：身份控制面 + 租户运行时注册表（provisioner 注入注册流程）。
	reg := tenancy.NewRegistry(mgr, sysWiki, filepath.Join(dir, "tenants"), cfgStore, nil)
	idStore, err := identity.Open(filepath.Join(dir, "control", "identity.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { idStore.Close() })
	idSvc := identity.NewService(idStore)
	idSvc.SetProvisioner(reg.Provision)

	srv := httpapi.New(httpapi.Config{
		Store: cfgStore, Runtimes: reg, ModelMgr: mgr, Registry: registry,
		Tasks: nil, LogRing: httpapi.NewLogRing(100), Leads: nil,
	})
	cookieMgr := auth.NewCookieManager(false) // 本地 http 测试：Secure=false
	authMW := identity.NewMiddleware(idSvc, cookieMgr)
	srv.SetAuth(&httpapi.AuthDeps{
		Identity:            idSvc,
		Cookies:             cookieMgr,
		Middleware:          authMW,
		LoginLimiter:        auth.NewRateLimiter(20, time.Minute),
		RegistrationEnabled: func() bool { return true },
	})

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	// 直接经身份服务注册测试用户（确定性捕获 tenantID + 登录 Token + CSRF），
	// 再手动把会话 Cookie 注入 jar（等价于 HTTP 注册的 Set-Cookie）。
	ctx := context.Background()
	res, err := idSvc.Register(ctx, identity.RegisterRequest{
		OrgName: "测试组织", DisplayName: "测试员",
		Email: "test@example.com", Password: "password-123456",
	}, "127.0.0.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	// 测试用户授予平台管理员：config/probe/console 平台路由测试需要（§4.2）。
	if err := idSvc.GrantPlatformAdmin(res.User.ID, true); err != nil {
		t.Fatal(err)
	}
	rt, err := reg.ForTenant(ctx, res.Tenant.ID)
	if err != nil {
		t.Fatal(err)
	}

	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse(ts.URL)
	jar.SetCookies(u, []*http.Cookie{{Name: cookieMgr.Name(), Value: res.TokenRaw, Path: "/"}})
	testClient = &http.Client{Jar: jar, Transport: csrfInjectingTransport{base: http.DefaultTransport}}
	testCSRF = res.CSRFRaw

	return ts, rt.Wiki
}

// csrfInjectingTransport 自动给修改类请求补 X-CSRF-Token（测试客户端统一带 CSRF，
// 避免每个手写请求都要手动加头）。
type csrfInjectingTransport struct{ base http.RoundTripper }

func (t csrfInjectingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead && req.Method != http.MethodOptions && testCSRF != "" {
		req.Header.Set("X-CSRF-Token", testCSRF)
	}
	return t.base.RoundTrip(req)
}

func do(t *testing.T, ts *httptest.Server, method, path string, body any) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(method, ts.URL+path, nil)
	if body != nil {
		b, _ := json.Marshal(body)
		req.Body = nopCloser{bytes.NewReader(b)}
		req.Header.Set("Content-Type", "application/json")
	}
	client := testClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

// slowModelServer 返回一个延迟 delay 后返回合法 OpenAI chat completion 的 mock 模型，
// 使 Agent Run 保持忙态（用于并发 409 等时序敏感断言）。
func slowModelServer(t *testing.T, delay time.Duration) *httptest.Server {
	t.Helper()
	ml := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"好的"}}],"usage":{"total_tokens":10}}`))
	}))
	t.Cleanup(ml.Close)
	return ml
}

// setupServerSlowModel 用延迟模型构造完整服务（模型 endpoint 指向本地慢 mock）。
func setupServerSlowModel(t *testing.T, delay time.Duration) *httptest.Server {
	t.Helper()
	ml := slowModelServer(t, delay)
	configFile := filepath.Join(t.TempDir(), "config.json")
	cfg := fmt.Sprintf(`{"server":{"addr":":0"},"models":[{"name":"default","endpoint":%q,"api_key":"k","model":"m","temperature":0.3,"max_tokens":1024,"timeout_sec":30}],"router":{"default":"default","routes":{},"fallback":{"max_retries":0,"backoff_base_ms":10,"chain":null}}}`, ml.URL)
	_ = os.WriteFile(configFile, []byte(cfg), 0o600)
	ts, _ := setupServerWithConfig(t, configFile)
	return ts
}

func TestMemoryUpsertReviewFlow(t *testing.T) {
	ts, store := setupServer(t)

	// 1. AI 路径写入 threat（经 wiki store 打 pending——人工通道已不能传
	// status，P10；此测试锁定审核流本身）
	if err := store.UpsertEntry(&longterm.Entry{
		Type: domain.MemoryThreat, Title: "测试威胁", Content: "内容",
		Status: longterm.StatusPendingReview,
	}); err != nil {
		t.Fatal(err)
	}

	// 2. 出现在审核队列
	code, body := do(t, ts, "GET", "/api/review/pending", nil)
	if code != 200 || int(body["count"].(float64)) != 1 {
		t.Fatalf("expected 1 pending, got %v", body)
	}

	// 3. 批准
	code, _ = do(t, ts, "POST", "/api/review/threat/测试威胁/approve", nil)
	if code != 200 {
		t.Fatalf("approve failed: %d", code)
	}

	// 4. 队列清空
	_, body = do(t, ts, "GET", "/api/review/pending", nil)
	if int(body["count"].(float64)) != 0 {
		t.Fatalf("pending should be empty after approve, got %v", body)
	}

	// 5. memory get 确认状态
	_, body = do(t, ts, "GET", "/api/memory/threat/测试威胁", nil)
	if body["status"] != "verified" {
		t.Fatalf("status should be verified, got %v", body["status"])
	}
}

func TestMemorySearchAndList(t *testing.T) {
	ts, _ := setupServer(t)

	// list products → 雷池（复合记忆 product → system 只读层）
	_, body := do(t, ts, "GET", "/api/memory/list?type=product", nil)
	if int(body["count"].(float64)) != 1 {
		t.Fatalf("expected 1 product, got %v", body)
	}

	// search 雷池
	_, body = do(t, ts, "GET", "/api/memory/search?q=雷池&type=product", nil)
	if int(body["count"].(float64)) != 1 {
		t.Fatalf("search 雷池 expected 1, got %v", body)
	}
}

func TestConfigGetPut(t *testing.T) {
	ts, _ := setupServer(t)

	code, body := do(t, ts, "GET", "/api/config", nil)
	if code != 200 {
		t.Fatalf("config get failed: %d", code)
	}
	models := body["models"].([]any)
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	// api_key 应被脱敏（原始 'k' 不应泄露）
	m := models[0].(map[string]any)
	if m["api_key"] == "k" {
		t.Errorf("api_key should be masked, got raw value")
	}
}

// TestMessageCustomerFieldPersisted 客户身份端到端（ADR-016 L2）：
// /api/message 带 customer → EnsureSession 落库 → 会话详情读回。
// LLM 不可达会失败，但 customer 落库发生在 LLM 调用前——验证不受影响。
func TestMessageCustomerFieldPersisted(t *testing.T) {
	ts, _ := setupServer(t)
	// 带 customer 建会话（F0：202 + Run）
	code, body := do(t, ts, "POST", "/api/message", map[string]any{
		"text": "你好", "session_id": "sess-cust", "customer": "某跨境电商",
	})
	if code != 202 {
		t.Fatalf("消息应 202，got %d: %v", code, body)
	}
	code2, body2 := do(t, ts, "GET", "/api/sessions/sess-cust", nil)
	if code2 != 200 {
		t.Fatalf("读取会话应 200，got %d", code2)
	}
	cust, _ := body2["session"].(map[string]any)["customer"].(string)
	if cust != "某跨境电商" {
		t.Fatalf("customer 应已落库为 某跨境电商，got %q", cust)
	}
}

// TestMessageDoesNotOverwriteSessionTitle G4 标题修复：已有会话再发消息，
// 会话标题不被当前消息首行覆盖（否则标题会退化成「最后一问」而非用户提问归纳）。
// 标题的 LLM 归纳由后台 TaskTitle 任务完成（本测试 Tasks=nil，只验证不覆盖）。
func TestMessageDoesNotOverwriteSessionTitle(t *testing.T) {
	ts, _ := setupServer(t)
	title := func() string {
		code, body := do(t, ts, "GET", "/api/sessions/sess-title", nil)
		if code != 200 {
			t.Fatalf("读取会话应 200，got %d", code)
		}
		s, _ := body["session"].(map[string]any)["title"].(string)
		return s
	}
	waitIdle := func() {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			code, body := do(t, ts, "GET", "/api/sessions/sess-title/running", nil)
			if code == 200 {
				if running, _ := body["running"].(bool); !running {
					return
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("Run 未在超时内结束")
	}
	// 第一条消息建会话：标题=首行占位
	if code, _ := do(t, ts, "POST", "/api/message", map[string]any{"text": "帮我分析雷池WAF采购需求", "session_id": "sess-title"}); code != 202 {
		t.Fatalf("首条消息应 202，got %d", code)
	}
	waitIdle()
	if got := title(); got != "帮我分析雷池WAF采购需求" {
		t.Fatalf("新会话标题应为首行占位，got %q", got)
	}
	// 第二条消息（新提问）→ 标题不应被其首行覆盖
	if code, _ := do(t, ts, "POST", "/api/message", map[string]any{"text": "预算大概100万", "session_id": "sess-title"}); code != 202 {
		t.Fatalf("第二条消息应 202，got %d", code)
	}
	waitIdle()
	if got := title(); got != "帮我分析雷池WAF采购需求" {
		t.Fatalf("已有会话标题不应被当前消息首行覆盖，got %q", got)
	}
}

// TestMemoryUpsertIgnoresStatusParam P10：人工通道不接受 status 传参——
// 传 pending_review 也必须落 verified（pending 是 AI 写入专属语义）。
func TestMemoryUpsertIgnoresStatusParam(t *testing.T) {
	ts, store := setupServer(t)
	code, _ := do(t, ts, "POST", "/api/memory", map[string]any{
		"type": "threat", "title": "人工条目", "content": "x", "status": "pending_review",
	})
	if code != 200 {
		t.Fatalf("upsert 应 200，got %d", code)
	}
	e, err := store.GetEntry("threat", "人工条目")
	if err != nil {
		t.Fatal(err)
	}
	if e.Status != longterm.StatusVerified {
		t.Fatalf("人工写入必须 verified（忽略 status 传参），got %s", e.Status)
	}
}
