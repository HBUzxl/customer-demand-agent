package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	httpapi "customer-demand-agent/internal/channel/http"
	"customer-demand-agent/internal/leads"
)

// newLeadsTestServer 构造带商机面板代理的最小 HTTP 服务。
// platform 为 nil 时模拟「未接入」态（SetLeads 不注入）。
func newLeadsTestServer(t *testing.T, platform http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httpapi.NewMinimal(nil, nil)
	if platform != nil {
		mp := httptest.NewServer(platform)
		t.Cleanup(mp.Close)
		srv.SetLeads(leads.New(leads.Options{
			BaseURL: mp.URL, APIKey: "lm_pat_test",
			Timeout: 2 * time.Second, RatePerMin: 1_000_000, Burst: 100,
		}))
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// 未接入 → 200 {"enabled":false}（前端渲染引导态而非报错）。
func TestLeadsDashboardDisabled(t *testing.T) {
	ts := newLeadsTestServer(t, nil)
	code, body := do(t, ts, "GET", "/api/leads/dashboard", nil)
	if code != 200 {
		t.Fatalf("未接入应 200，got %d", code)
	}
	if body["enabled"] != false {
		t.Errorf("未接入应 enabled=false: %v", body)
	}
	code, body = do(t, ts, "GET", "/api/leads/stats", nil)
	if code != 200 || body["enabled"] != false {
		t.Errorf("stats 未接入应 200+enabled=false: %d %v", code, body)
	}
}

// 列表透传：默认口径 stage=mql + 分页归一化 + Authorization 头 + items 原样。
func TestLeadsDashboardList(t *testing.T) {
	var gotPath, gotQuery, gotAuth string
	ts := newLeadsTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery, gotAuth = r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"items":[{"id":"L1","customer":"某电商","stage":"mql","owner":"张三"}],"total":3}`))
	})
	code, body := do(t, ts, "GET", "/api/leads/dashboard", nil)
	if code != 200 {
		t.Fatalf("应 200，got %d", code)
	}
	if gotPath != "/api/leads" {
		t.Errorf("应打到平台 /api/leads，got %s", gotPath)
	}
	if gotQuery != "page=1&page_size=20&stage=mql" {
		t.Errorf("默认参数应为 stage=mql&page=1&page_size=20，got %s", gotQuery)
	}
	if gotAuth != "Bearer lm_pat_test" {
		t.Errorf("应带系统 Token 头，got %q", gotAuth)
	}
	if body["enabled"] != true || body["count"] != float64(1) || body["page_size"] != float64(20) {
		t.Errorf("面板元数据不对: %v", body)
	}
	if body["total"] != float64(3) {
		t.Errorf("total 应透传 3: %v", body["total"])
	}
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items 应透传 1 条: %v", body["items"])
	}
	if it := items[0].(map[string]any); it["customer"] != "某电商" {
		t.Errorf("items 字段应原样透传: %v", it)
	}
	// 自定义分页透传
	code, body = do(t, ts, "GET", "/api/leads/dashboard?page=2&page_size=5", nil)
	if code != 200 || body["page"] != float64(2) || body["page_size"] != float64(5) {
		t.Errorf("自定义分页应透传: %d %v", code, body)
	}
}

// 平台连不上 → 502 + VPN 口径（面板用户可行动的提示）。
func TestLeadsDashboardTransportError(t *testing.T) {
	srv := httpapi.NewMinimal(nil, nil)
	srv.SetLeads(leads.New(leads.Options{
		BaseURL: "http://127.0.0.1:1", APIKey: "k",
		Timeout: 200 * time.Millisecond, RatePerMin: 1_000_000, Burst: 100,
	}))
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	code, body := do(t, ts, "GET", "/api/leads/dashboard", nil)
	if code != 502 {
		t.Fatalf("连接失败应 502，got %d", code)
	}
	msg, _ := body["error"].(string)
	if !strings.Contains(msg, "连不上商机平台") || !strings.Contains(msg, "VPN") {
		t.Errorf("应有 VPN 口径提示，got %q", msg)
	}
}

// 平台 401 → 502 + Token 失效口径。
func TestLeadsDashboard401(t *testing.T) {
	ts := newLeadsTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	})
	code, body := do(t, ts, "GET", "/api/leads/dashboard", nil)
	if code != 502 {
		t.Fatalf("401 应映射 502，got %d", code)
	}
	if msg, _ := body["error"].(string); msg == "" {
		t.Error("401 应带错误文案")
	}
}

// stats best-effort：正常透传 / 403 降级为 body.error（面板不整体失败）/ 非法 metric 400。
func TestLeadsStats(t *testing.T) {
	var status atomic.Int32
	status.Store(200)
	ts := newLeadsTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if status.Load() == 403 {
			w.WriteHeader(403)
			return
		}
		_, _ = w.Write([]byte(`{"total":9,"mql":2}`))
	})
	// 正常
	code, body := do(t, ts, "GET", "/api/leads/stats?metric=summary", nil)
	if code != 200 || body["enabled"] != true {
		t.Fatalf("stats 应 200+enabled: %d %v", code, body)
	}
	if d, ok := body["data"].(map[string]any); !ok || d["mql"] != float64(2) {
		t.Errorf("stats data 应透传: %v", body["data"])
	}
	// 403 → 200 + error（best-effort：列表照常，统计区显示原因）
	status.Store(403)
	code, body = do(t, ts, "GET", "/api/leads/stats?metric=summary", nil)
	if code != 200 {
		t.Fatalf("stats 403 应 best-effort 200，got %d", code)
	}
	if msg, _ := body["error"].(string); msg == "" {
		t.Error("stats 403 应带 error 说明")
	}
	// 非法 metric → 400（本地拒绝，零平台请求）
	code, _ = do(t, ts, "GET", "/api/leads/stats?metric=anything", nil)
	if code != 400 {
		t.Errorf("非法 metric 应 400，got %d", code)
	}
}
