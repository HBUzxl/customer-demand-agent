package http_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

// TestMessageEndpoint 统一入口 /api/message（ADR-014）：SSE 200 + session 事件。
// setupServer 的 LLM 指向不可达端点 → done 不会来，但 session/round 事件 + 200 可验证。
func TestMessageEndpoint(t *testing.T) {
	ts, _ := setupServer(t)

	body := `{"text":"你好"}`
	req, _ := http.NewRequest("POST", ts.URL+"/api/message", nopCloser{bytes.NewReader([]byte(body))})
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/api/message 应 200，got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("应为 SSE，got Content-Type %q", ct)
	}
	// 读部分 body 验证 session 事件先到（不读完——LLM 不可达会阻塞到超时）
	buf := make([]byte, 512)
	n, _ := resp.Body.Read(buf)
	head := string(buf[:n])
	if !strings.Contains(head, `"type":"session"`) {
		t.Errorf("首个事件应为 session，got: %s", head)
	}
}

// TestAnalyzeShimDeprecated 旧端点 /api/analyze 仍可用但带 Deprecation 头。
func TestAnalyzeShimDeprecated(t *testing.T) {
	ts, _ := setupServer(t)
	req, _ := http.NewRequest("POST", ts.URL+"/api/analyze", nopCloser{bytes.NewReader([]byte(`{"text":"hi"}`))})
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/api/analyze shim 应 200，got %d", resp.StatusCode)
	}
	if resp.Header.Get("Deprecation") != "true" {
		t.Error("/api/analyze shim 应带 Deprecation: true 头")
	}
}

// TestChatShimStillWorks 旧端点 /api/chat（{session_id, question} body）shim 正常。
func TestChatShimStillWorks(t *testing.T) {
	ts, _ := setupServer(t)
	// 先建会话（tenantA 认领）
	if code := postAnalyzeWithTenant(t, ts.URL, "tenantA", `{"session_id":"shim-s","text":"hi"}`); code != http.StatusOK {
		t.Fatalf("认领会话失败: %d", code)
	}
	// chat shim：question 字段
	req, _ := http.NewRequest("POST", ts.URL+"/api/chat", nopCloser{bytes.NewReader([]byte(`{"session_id":"shim-s","question":"部署方式？"}`))})
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenantA")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/api/chat shim 应 200，got %d", resp.StatusCode)
	}
	if resp.Header.Get("Deprecation") != "true" {
		t.Error("/api/chat shim 应带 Deprecation: true 头")
	}
}
