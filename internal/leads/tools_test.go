package leads

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// newSvc 起一个 mock 平台并构造 Service。
func newSvc(t *testing.T, script ...func(w http.ResponseWriter, r *http.Request)) (*Service, *mockPlatform) {
	t.Helper()
	mp := newMockPlatform(t, script...)
	return New(Options{BaseURL: mp.srv.URL, APIKey: "lm_pat_test", RatePerMin: 1_000_000, Burst: 100, Timeout: 2 * time.Second}), mp
}

func exec(t *testing.T, s *Service, args string) string {
	t.Helper()
	out, err := s.Execute(context.Background(), ToolSearch, json.RawMessage(args))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return out
}

// ── leads_search ────────────────────────────────────────────

func TestSearchStageAndPaging(t *testing.T) {
	var gotQuery string
	s, _ := newSvc(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"items":[{"id":"L1","customer":"某电商","stage":"mql"}],"total":42}`))
	})
	out := exec(t, s, `{"stage":"mql"}`)
	if want := "page=1&page_size=10&stage=mql"; gotQuery != want {
		t.Errorf("query 组装错：got %s want %s", gotQuery, want)
	}
	// 结果整形
	var res struct {
		Count int              `json:"count"`
		Total int              `json:"total"`
		Page  int              `json:"page"`
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("结果应合法 JSON: %v\n%s", err, out)
	}
	if res.Count != 1 || res.Total != 42 || res.Page != 1 || len(res.Items) != 1 {
		t.Errorf("整形结果不对: %s", out)
	}
	if res.Items[0]["customer"] != "某电商" {
		t.Errorf("items 应原样透传平台字段: %s", out)
	}
}

func TestSearchPageSizeCap(t *testing.T) {
	var gotQuery string
	s, _ := newSvc(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`[]`))
	})
	exec(t, s, `{"page_size":100}`)
	if !strings.Contains(gotQuery, "page_size=50") {
		t.Errorf("page_size 应封顶 50，got query %s", gotQuery)
	}
	// 显式页码与页大小透传
	exec(t, s, `{"stage":"mql","page":3,"page_size":20}`)
	if !strings.Contains(gotQuery, "page=3") || !strings.Contains(gotQuery, "page_size=20") {
		t.Errorf("分页参数应透传，got %s", gotQuery)
	}
}

// 未记载参数（query/sort/keyword 等）一律忽略，绝不透传给平台。
func TestSearchUnknownParamsIgnored(t *testing.T) {
	var gotQuery string
	s, _ := newSvc(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`[]`))
	})
	exec(t, s, `{"stage":"mql","query":"某电商","sort":"value","keyword":"x","created_after":"2026-01-01"}`)
	for _, banned := range []string{"query=", "sort=", "keyword=", "created_after="} {
		if strings.Contains(gotQuery, banned) {
			t.Errorf("未记载参数被透传（违反「未明确=无」）: %s 含 %s", gotQuery, banned)
		}
	}
	if !strings.Contains(gotQuery, "stage=mql") {
		t.Errorf("stage 应透传: %s", gotQuery)
	}
}

// 响应包络容错解析：常见形状识别 + 识别不出整体透传。
func TestSearchEnvelopeTolerance(t *testing.T) {
	cases := []struct {
		name      string
		platform  string
		wantCount int
		wantTotal *int // nil = 断言 total 为 null（未知）
	}{
		{"items+total", `{"items":[{"id":"1"},{"id":"2"}],"total":7}`, 2, ptr(7)},
		{"data 数组", `{"data":[{"id":"1"}]}`, 1, nil},
		{"list 数组", `{"list":[{"id":"1"},{"id":"2"},{"id":"3"}],"count":99}`, 3, ptr(99)},
		{"顶层裸数组", `[{"id":"1"},{"id":"2"}]`, 2, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newSvc(t, okJSON(tc.platform))
			out := exec(t, s, `{}`)
			var res map[string]json.RawMessage
			if err := json.Unmarshal([]byte(out), &res); err != nil {
				t.Fatalf("输出应合法 JSON: %v\n%s", err, out)
			}
			var items []json.RawMessage
			if err := json.Unmarshal(res["items"], &items); err != nil {
				t.Fatalf("items 应为数组: %v\n%s", err, out)
			}
			if len(items) != tc.wantCount {
				t.Errorf("count 错: %s", out)
			}
			if tc.wantTotal != nil {
				var total int
				_ = json.Unmarshal(res["total"], &total)
				if total != *tc.wantTotal {
					t.Errorf("total 应为 %d: %s", *tc.wantTotal, out)
				}
			}
		})
	}
	// 完全未知形状 → 整体透传（raw 兜底）
	s, _ := newSvc(t, okJSON(`{"weird":{"nested":true}}`))
	out := exec(t, s, `{}`)
	if !strings.Contains(out, `"raw"`) || !strings.Contains(out, `"weird"`) {
		t.Errorf("未知包络应整体透传: %s", out)
	}
}

func ptr(i int) *int { return &i }

// ── leads_get ───────────────────────────────────────────────

func TestGetRequiresID(t *testing.T) {
	s, mp := newSvc(t, okJSON(`{}`))
	if _, err := s.Execute(context.Background(), ToolGet, json.RawMessage(`{}`)); err == nil {
		t.Error("缺 id 应报错")
	}
	if _, err := s.Execute(context.Background(), ToolGet, json.RawMessage(`{"id":"  "}`)); err == nil {
		t.Error("空白 id 应报错")
	}
	if _, err := s.Execute(context.Background(), ToolGet, json.RawMessage(`{"id":"a/b"}`)); err == nil {
		t.Error("含斜杠的 id 应报错")
	}
	if n := mp.hits.Load(); n != 0 {
		t.Errorf("参数校验失败不应有网络请求，got %d", n)
	}
}

func TestGetLead(t *testing.T) {
	var gotPath string
	s, _ := newSvc(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"L9","customer":"某银行","stage":"sql"}`))
	})
	out, err := s.Execute(context.Background(), ToolGet, json.RawMessage(`{"id":"L9"}`))
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/leads/L9" {
		t.Errorf("路径应为 /api/leads/L9，got %s", gotPath)
	}
	if !strings.Contains(out, `"lead"`) || !strings.Contains(out, "某银行") {
		t.Errorf("详情应嵌入 lead 字段: %s", out)
	}
	if strings.Contains(out, "activities") {
		t.Errorf("默认不拉活动: %s", out)
	}
}

func TestGetWithActivities(t *testing.T) {
	var paths []string
	s, _ := newSvc(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/api/leads/L1":
			_, _ = w.Write([]byte(`{"id":"L1"}`))
		case "/api/leads/L1/activities":
			_, _ = w.Write([]byte(`{"items":[{"act":"call"}]}`))
		}
	})
	out, err := s.Execute(context.Background(), ToolGet, json.RawMessage(`{"id":"L1","with_activities":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[0] != "/api/leads/L1" || paths[1] != "/api/leads/L1/activities" {
		t.Errorf("应串行调用详情+活动，got %v", paths)
	}
	if !strings.Contains(out, `"activities"`) || !strings.Contains(out, `"call"`) {
		t.Errorf("活动应嵌入结果: %s", out)
	}
}

// ── leads_stats ─────────────────────────────────────────────

func TestStatsMetricMapping(t *testing.T) {
	mapping := map[string]string{
		"summary":            "/api/leads/stats",
		"conversion":         "/api/stats/conversion",
		"trends":             "/api/stats/conversion/trends",
		"ml_creators":        "/api/stats/ml-creators",
		"product_conversion": "/api/stats/product-conversion",
	}
	for metric, wantPath := range mapping {
		var gotPath string
		s, _ := newSvc(t, func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			_, _ = w.Write([]byte(`{"k":1}`))
		})
		out, err := s.Execute(context.Background(), ToolStats, json.RawMessage(`{"metric":"`+metric+`"}`))
		if err != nil {
			t.Fatalf("%s: %v", metric, err)
		}
		if gotPath != wantPath {
			t.Errorf("metric %s 应映射 %s，got %s", metric, wantPath, gotPath)
		}
		if !strings.Contains(out, `"data"`) {
			t.Errorf("%s: 结果应含 data: %s", metric, out)
		}
	}
}

func TestStatsBadMetric(t *testing.T) {
	s, mp := newSvc(t, okJSON(`{}`))
	if _, err := s.Execute(context.Background(), ToolStats, json.RawMessage(`{"metric":"total_value"}`)); err == nil {
		t.Error("未知 metric 应报错")
	}
	if n := mp.hits.Load(); n != 0 {
		t.Errorf("未知 metric 不应有网络请求，got %d", n)
	}
}

// ── 错误映射（给 LLM 看的中文行动指引） ─────────────────────

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		name     string
		platform func(http.ResponseWriter, *http.Request)
		want     string
	}{
		{"401", status(401), "Token 失效或过期"},
		{"403", status(403), "不要重试或绕过"},
		{"404", status(404), "不存在"},
		{"429持续", statusWith429("0"), "频率超限"},
		{"500持续", status(500), "服务端"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newSvc(t, tc.platform)
			out := exec(t, s, `{}`)
			if !strings.Contains(out, `"error"`) || !strings.Contains(out, tc.want) {
				t.Errorf("错误映射文案不对: %s", out)
			}
		})
	}
	// 超时/连接失败 → VPN 口径
	sv := New(Options{BaseURL: "http://127.0.0.1:1", APIKey: "k", Timeout: 100 * time.Millisecond, RatePerMin: 1_000_000, Burst: 100})
	out := exec2(t, sv)
	if !strings.Contains(out, "连不上商机平台") || !strings.Contains(out, "VPN") {
		t.Errorf("连接失败应提示 VPN: %s", out)
	}
}

func exec2(t *testing.T, s *Service) string {
	t.Helper()
	out, err := s.Execute(context.Background(), ToolSearch, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return out
}

// stats 403 → 统计专用文案。
func TestStats403Message(t *testing.T) {
	s, _ := newSvc(t, status(403))
	out, err := s.Execute(context.Background(), ToolStats, json.RawMessage(`{"metric":"trends"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "无该统计权限") {
		t.Errorf("stats 403 应用统计专用文案: %s", out)
	}
}

// ── Definitions / Handles ───────────────────────────────────

func TestDefinitionsAndHandles(t *testing.T) {
	s, _ := newSvc(t, okJSON(`{}`))
	defs := s.Definitions()
	if len(defs) != 3 {
		t.Fatalf("应 3 个工具，got %d", len(defs))
	}
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Function.Name] = true
		b, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("工具定义不可序列化: %v", err)
		}
		if !json.Valid(b) {
			t.Errorf("工具定义 %s 非法 JSON", d.Function.Name)
		}
	}
	for _, want := range []string{ToolSearch, ToolGet, ToolStats} {
		if !names[want] || !s.Handles(want) {
			t.Errorf("缺少工具 %s", want)
		}
	}
	if s.Handles("memory_search") {
		t.Error("Handles 不应接管非本包工具")
	}
}
