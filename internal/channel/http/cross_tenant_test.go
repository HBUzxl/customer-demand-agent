package http_test

import (
	"bytes"
	"net/http"
	"testing"
)

// postAnalyzeWithTenant 带 X-Tenant-ID 头 POST /api/analyze，返回状态码。
func postAnalyzeWithTenant(t *testing.T, tsURL, tenant, body string) int {
	t.Helper()
	req, _ := http.NewRequest("POST", tsURL+"/api/analyze", nopCloser{bytes.NewReader([]byte(body))})
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenant)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// TestCrossTenantSessionRefusal 验证租户 B 用租户 A 的 session_id 调 /api/analyze 被拒（403）。
// （A 先认领 session——analyze 因 LLM 不可用会失败，但 EnsureSession 已建会话；B 续传该 session_id → 拒绝）
func TestCrossTenantSessionRefusal(t *testing.T) {
	ts, _ := setupServer(t)
	body := `{"session_id":"sess-shared","text":"A的需求"}`
	// A 先认领
	postAnalyzeWithTenant(t, ts.URL, "tenantA", body)
	// B 用同一 session_id → 跨租户拒绝 → 403
	if code := postAnalyzeWithTenant(t, ts.URL, "tenantB", body); code != http.StatusForbidden {
		t.Errorf("租户 B 用 A 的 session_id 应被拒（403），got %d", code)
	}
	// A 自己仍可用（同租户不误拒）
	if code := postAnalyzeWithTenant(t, ts.URL, "tenantA", body); code != http.StatusOK {
		t.Errorf("租户 A 用自己的 session_id 应正常（200，LLM 失败也走 SSE），got %d", code)
	}
}
