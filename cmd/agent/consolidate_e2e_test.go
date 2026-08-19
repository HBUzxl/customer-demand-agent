package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"customer-demand-agent/internal/auth"
	cdahttp "customer-demand-agent/internal/channel/http"
	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/identity"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/model"
	"customer-demand-agent/internal/tenancy"
)

// TestConsolidateFullChainHTTP P11 真·端到端（可复现）：mock LLM 网关（HTTP
// 边界）+ 真实多租户装配（identity 控制面 + tenancy registry + buildTaskRunner）
// + 真实 httpapi 服务 + 真实租户 wiki 落盘。
// 链路：POST /api/tasks/consolidate（202）→ Runner 执行（真 exec：读观察注记
// → LLM（mock）→ 解析 → UpsertEntry pending_review）→ GET /api/tasks 轮询到
// done → GET /api/memory 条目 pending_review+固化内容并入 → POST
// /api/review/{type}/{title}/approve → GET 条目 verified。
func TestConsolidateFullChainHTTP(t *testing.T) {
	// 1) mock LLM 网关：返回固化 JSON
	calls := 0
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"银行高优关注大模型训练数据出境，券商关心投研LLM生成合规\",\"tags\":[\"金融\",\"大模型\",\"数据出境\",\"合规\"],\"content\":\"## 固化要点\\n\\n- 银行：大模型训练数据能否出境是高频关切\\n- 券商：投研报告用 LLM 生成的合规性存疑\"}"}}]}`)
	}))
	defer mockLLM.Close()

	dir := t.TempDir()
	ctx := context.Background()

	// 2) 平台系统知识（空基线）+ 配置 Store（tenancy.NewRegistry 需要）
	sysWikiDir := filepath.Join(dir, "system", "wiki")
	_ = os.MkdirAll(sysWikiDir, 0o755)
	sysWiki := longterm.NewWikiStore(sysWikiDir)
	if err := sysWiki.Load(); err != nil {
		t.Fatal(err)
	}
	store, err := config.Load(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}

	// 3) 真实 model manager（指向 mock 网关）
	registry := model.NewRegistry(5 * time.Second)
	cfgModel := model.ModelConfig{Name: "default", Endpoint: mockLLM.URL, APIKey: "k", Model: "m"}
	_ = registry.Register(cfgModel)
	rc := model.RouterConfig{Default: "default"}
	mgr := model.NewManager(llm.NewClient(), registry, &rc)

	// 4) 多租户装配：身份控制面 + 租户注册表（provisioner 注入注册流程）
	reg := tenancy.NewRegistry(mgr, sysWiki, filepath.Join(dir, "tenants"), store, nil)
	idStore, err := identity.Open(filepath.Join(dir, "control", "identity.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer idStore.Close()
	idSvc := identity.NewService(idStore)
	idSvc.SetProvisioner(reg.Provision)
	authRes, err := idSvc.Register(ctx, identity.RegisterRequest{
		OrgName: "测试组织", DisplayName: "管理员", Email: "admin@test.local", Password: "password-123456",
	}, "127.0.0.1", "e2e")
	if err != nil {
		t.Fatal(err)
	}
	cookieMgr := auth.NewCookieManager(false)

	// 5) 预置待固化条目（租户 wiki：industry 写覆盖层）
	rt, err := reg.ForTenant(ctx, authRes.Tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	content := "---\ntype: industry\ntitle: 固化E2E\nsummary: 待固化\n---\n\n正文。\n\n### 观察\n\n- [high] 银行问过大模型训练数据出境\n- [medium] 券商关心投研 LLM 生成合规"
	if err := rt.Wiki.UpsertEntry(&longterm.Entry{Type: "industry", Title: "固化E2E", Content: content, Summary: "待固化"}); err != nil {
		t.Fatal(err)
	}

	// 6) 真实 runner + httpapi（身份控制面完整装配）
	runner := buildTaskRunner(reg, mgr)
	authMW := identity.NewMiddleware(idSvc, cookieMgr)
	loginLimiter := auth.NewRateLimiter(10, time.Minute)
	server := cdahttp.New(cdahttp.Config{Store: store, Runtimes: reg, ModelMgr: mgr, Registry: registry, Tasks: runner, LogRing: cdahttp.NewLogRing(100)})
	server.SetAuth(&cdahttp.AuthDeps{
		Identity:            idSvc,
		Cookies:             cookieMgr,
		Middleware:          authMW,
		LoginLimiter:        loginLimiter,
		RegistrationEnabled: func() bool { return true },
	})
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	// 携带登录 Cookie + CSRF 的请求构造器（CSRF 在 register 签发后未轮换，仍有效）
	req := func(method, path string, body io.Reader, csrf bool) *http.Request {
		r, _ := http.NewRequest(method, ts.URL+path, body)
		r.AddCookie(&http.Cookie{Name: cookieMgr.Name(), Value: authRes.TokenRaw})
		if csrf {
			r.Header.Set("X-CSRF-Token", authRes.CSRFRaw)
		}
		return r
	}

	// 7) POST /api/tasks/consolidate → 202
	cr, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/tasks/consolidate", strings.NewReader(`{"type":"industry","title":"固化E2E"}`))
	cr.AddCookie(&http.Cookie{Name: cookieMgr.Name(), Value: authRes.TokenRaw})
	cr.Header.Set("X-CSRF-Token", authRes.CSRFRaw)
	res, err := http.DefaultClient.Do(cr)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("consolidate 应 202，got %d", res.StatusCode)
	}
	// 8) 轮询任务 done
	deadline := time.Now().Add(5 * time.Second)
	status := ""
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		lr := req(http.MethodGet, "/api/tasks", nil, false)
		res, err := http.DefaultClient.Do(lr)
		if err != nil {
			t.Fatal(err)
		}
		var out struct {
			Items []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
				Result string `json:"result"`
			} `json:"items"`
		}
		_ = json.NewDecoder(res.Body).Decode(&out)
		res.Body.Close()
		if len(out.Items) > 0 && out.Items[0].Type == "consolidate" {
			status = out.Items[0].Status
			if status != "running" {
				if status != "done" {
					t.Fatalf("任务应 done，got %s", status)
				}
				break
			}
		}
	}
	if status != "done" {
		t.Fatal("任务未在期限内完成")
	}
	if calls == 0 {
		t.Fatal("应真实调用 LLM（mock 网关）")
	}
	// 9) 条目 pending_review + 固化内容并入
	e, _ := rt.Wiki.GetEntry("industry", "固化E2E")
	if e == nil || string(e.Status) != "pending_review" {
		t.Fatalf("固化后应 pending_review: %+v", e)
	}
	if !strings.Contains(e.Content, "固化要点") || !strings.Contains(e.Content, "银行") {
		t.Fatalf("固化产出应并入观察内容: %q", e.Content)
	}
	// 10) HTTP 审批 → verified
	ar := req(http.MethodPost, "/api/review/industry/"+url.PathEscape("固化E2E")+"/approve", nil, true)
	res2, err := http.DefaultClient.Do(ar)
	if err != nil {
		t.Fatal(err)
	}
	res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("approve 应 200，got %d", res2.StatusCode)
	}
	e2, _ := rt.Wiki.GetEntry("industry", "固化E2E")
	if e2 == nil || string(e2.Status) != "verified" {
		t.Fatalf("审批后应 verified: %+v", e2)
	}
}
