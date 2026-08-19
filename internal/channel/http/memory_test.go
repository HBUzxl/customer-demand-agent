package http_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/memory/longterm"
)

// TestMemoryHistoryEndpoint P6 时效查询契约：覆盖归档后 /history API 返回
// 版本链（count/invalid_at/superseded_by/status=archived）。
func TestMemoryHistoryEndpoint(t *testing.T) {
	ts, wiki := setupServer(t)
	defer ts.Close()
	// v1 → 覆盖 v2（实质变更触发 superseded 归档）
	if err := wiki.UpsertEntry(&longterm.Entry{Type: domain.MemoryIndustry, Title: "历史链条目", Content: "第一版老结论内容", Summary: "v1", Status: longterm.StatusVerified}); err != nil {
		t.Fatal(err)
	}
	if err := wiki.UpsertEntry(&longterm.Entry{Type: domain.MemoryIndustry, Title: "历史链条目", Content: "第二版新结论内容差异足够大", Summary: "v2", Status: longterm.StatusVerified}); err != nil {
		t.Fatal(err)
	}
	res, err := testClient.Get(ts.URL + "/api/memory/industry/" + url.PathEscape("历史链条目") + "/history")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out struct {
		Count   int              `json:"count"`
		History []longterm.Entry `json:"history"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Count == 0 || len(out.History) == 0 {
		t.Fatalf("应返回历史版本链: %+v", out)
	}
	h := out.History[0]
	if h.InvalidAt == "" || h.SupersededBy != "历史链条目" || h.Status != longterm.StatusArchived {
		t.Fatalf("历史版本应带时效字段+archived: %+v", h)
	}
}

func TestCustomerCreateForConversation(t *testing.T) {
	ts, _ := setupServer(t)
	body := map[string]any{
		"name": "示例银行", "industry": "金融", "scale": "大型",
		"existing_security": []string{"安恒 WAF"}, "notes": "监管通报后关注规则更新时效",
	}
	code, out := do(t, ts, "POST", "/api/customers", body)
	if code != http.StatusCreated || out["name"] != "示例银行" {
		t.Fatalf("客户登记应 201，got code=%d body=%v", code, out)
	}
	code, got := do(t, ts, "GET", "/api/memory/customer/示例银行", nil)
	if code != http.StatusOK {
		t.Fatalf("登记后租户内应可读取客户画像，got %d", code)
	}
	if got["status"] != "verified" || !strings.Contains(fmt.Sprint(got["content"]), "监管通报") {
		t.Fatalf("客户画像应 verified 且保留备注，got %v", got)
	}
	code, _ = do(t, ts, "POST", "/api/customers", body)
	if code != http.StatusConflict {
		t.Fatalf("同名客户登记不得覆盖已有画像，got %d", code)
	}
}
