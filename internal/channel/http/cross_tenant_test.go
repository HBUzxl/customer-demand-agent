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

// TestCrossTenantSessionRefusal 验证租户 B 用租户 A 的 session_id 调 /api/message 被拒（403）。
// （A 先认领 session——message 返回 202，EnsureSession 已建会话；B 复用该 session_id → 拒绝）
func TestCrossTenantSessionRefusal(t *testing.T) {
	ts, _ := setupServer(t)
	body := `{"session_id":"sess-shared","text":"A的需求"}`
	// A 先认领
	postAnalyzeWithTenant(t, ts.URL, "tenantA", body)
	// B 用同一 session_id → 跨租户拒绝 → 403
	if code := postAnalyzeWithTenant(t, ts.URL, "tenantB", body); code != http.StatusForbidden {
		t.Errorf("租户 B 用 A 的 session_id 应被拒（403），got %d", code)
	}
	// A 自己仍可用（同租户不误拒；session Run 占用则 409 也算通过）
	code := postAnalyzeWithTenant(t, ts.URL, "tenantA", body)
	if code != http.StatusAccepted && code != http.StatusConflict {
		t.Errorf("租户 A 用自己的 session_id 应 202/409，got %d", code)
	}
}
