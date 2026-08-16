package leads

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// List：结构化返回 + 参数纪律与 leads_search 完全一致（仅 stage+分页、封顶 50）。
func TestListStructured(t *testing.T) {
	var gotQuery string
	s, _ := newSvc(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"items":[{"id":"L1","customer":"某电商","stage":"mql"},{"id":"L2"}],"total":3}`))
	})
	lst, err := s.List(context.Background(), "mql", 0, 100) // page=0/pageSize=100 → 归一化 1/50
	if err != nil {
		t.Fatal(err)
	}
	if want := "page=1&page_size=50&stage=mql"; gotQuery != want {
		t.Errorf("query 组装错：got %s want %s", gotQuery, want)
	}
	if lst.Count != 2 || lst.Page != 1 || len(lst.Items) != 2 {
		t.Errorf("结构化字段不对: %+v", lst)
	}
	if lst.Total == nil || *lst.Total != 3 {
		t.Errorf("total 应为 3: %+v", lst.Total)
	}
	// items 原样透传平台字段
	var it map[string]any
	if err := json.Unmarshal(lst.Items[0], &it); err != nil || it["customer"] != "某电商" {
		t.Errorf("items 应原样透传: %s", lst.Items[0])
	}
	// 序列化形状与工具层回填一致（count/total/page/items 四键）
	b, _ := json.Marshal(lst)
	for _, k := range []string{`"count"`, `"total"`, `"page"`, `"items"`} {
		if !strings.Contains(string(b), k) {
			t.Errorf("序列化缺 %s: %s", k, b)
		}
	}
}

// List：包络识别不出 → 整包作为单元素透传（面板兜底可见，不吞数据）。
func TestListUnknownEnvelopePassthrough(t *testing.T) {
	s, _ := newSvc(t, okJSON(`{"weird":{"nested":true}}`))
	lst, err := s.List(context.Background(), "", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if lst.Count != 1 || len(lst.Items) != 1 || !strings.Contains(string(lst.Items[0]), `"weird"`) {
		t.Errorf("未知包络应整包单元素透传: %+v", lst)
	}
	if lst.Total != nil {
		t.Errorf("未知包络 total 应为 nil: %+v", lst.Total)
	}
}

// Stats：metric 白名单外本地拒绝（零网络请求）。
func TestStatsUnknownMetricLocalReject(t *testing.T) {
	s, mp := newSvc(t, okJSON(`{}`))
	if _, err := s.Stats(context.Background(), "total_value"); err == nil {
		t.Error("未知 metric 应报错")
	}
	if n := mp.hits.Load(); n != 0 {
		t.Errorf("未知 metric 不应有网络请求，got %d", n)
	}
}

// Stats：合法 metric 原样透传平台响应。
func TestStatsPassthrough(t *testing.T) {
	s, _ := newSvc(t, okJSON(`{"total_leads":42,"mql":7}`))
	raw, err := s.Stats(context.Background(), "summary")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"mql":7`) {
		t.Errorf("stats 应原样透传: %s", raw)
	}
}
