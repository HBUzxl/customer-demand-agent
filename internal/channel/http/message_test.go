package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// F0 语义：POST /api/message → 202 + {session_id, run_id}（执行与连接解耦）。

// postMessage 便捷封装：发消息并解析 202 响应。
func postMessage(t *testing.T, tsURL, tenant, body string) (int, string, string) {
	t.Helper()
	req, _ := http.NewRequest("POST", tsURL+"/api/message", nopCloser{bytes.NewReader([]byte(body))})
	req.Header.Set("Content-Type", "application/json")
	if tenant != "" {
		req.Header.Set("X-Tenant-ID", tenant)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	var out struct {
		SessionID string `json:"session_id"`
		RunID     string `json:"run_id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out.SessionID, out.RunID
}

// TestMessageEndpoint202 统一入口 F0 语义：202 + run_id。
func TestMessageEndpoint202(t *testing.T) {
	ts, _ := setupServer(t)
	code, sid, rid := postMessage(t, ts.URL, "", `{"text":"你好"}`)
	if code != http.StatusAccepted {
		t.Fatalf("/api/message 应 202，got %d", code)
	}
	if sid == "" || rid == "" {
		t.Fatalf("202 响应应含 session_id/run_id，got sid=%q rid=%q", sid, rid)
	}
}

// TestMessageBusy409 同会话并发：第二个消息 409。
func TestMessageBusy409(t *testing.T) {
	ts, _ := setupServer(t)
	code1, _, _ := postMessage(t, ts.URL, "", `{"text":"你好","session_id":"busy-s"}`)
	if code1 != http.StatusAccepted {
		t.Fatalf("第一个消息应 202，got %d", code1)
	}
	// Run 仍活跃（LLM 不可达会跑到超时）→ 409
	code2, _, _ := postMessage(t, ts.URL, "", `{"text":"再来","session_id":"busy-s"}`)
	if code2 != http.StatusConflict {
		t.Fatalf("并发应 409，got %d", code2)
	}
}

// TestRunSurvivesDisconnect F0 核心：创建 Run 后立刻断开（响应已读完），
// 运行继续——用订阅流 replay 证明事件在后台持续产生。
func TestRunSurvivesDisconnect(t *testing.T) {
	ts, _ := setupServer(t)
	code, sid, rid := postMessage(t, ts.URL, "", `{"text":"你好","session_id":"disc-s"}`)
	if code != http.StatusAccepted {
		t.Fatalf("202 expected, got %d", code)
	}
	// 不订阅任何流（=客户端断开）——等一小段时间让 Run 跑
	time.Sleep(300 * time.Millisecond)
	// 现在订阅（since=0 replay）：应能看到 Run 已产生的事件
	req, _ := http.NewRequest("GET", ts.URL+"/api/sessions/"+sid+"/stream?since=0", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stream 应 200，got %d", resp.StatusCode)
	}
	// 读一段（Run 活跃时流不结束；读到 session 事件即证明后台在跑）
	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	head := string(buf[:n])
	if !strings.Contains(head, `"type":"session"`) {
		t.Fatalf("replay 应含 session 事件，got: %s", head)
	}
	_ = rid
}

// TestRunCancelExplicit 显式取消：cancel API 后 Run 停止（缓冲不再增长，运行态解除）。
func TestRunCancelExplicit(t *testing.T) {
	ts, _ := setupServer(t)
	code, sid, rid := postMessage(t, ts.URL, "", `{"text":"你好","session_id":"cancel-s"}`)
	if code != http.StatusAccepted {
		t.Fatalf("202 expected, got %d", code)
	}
	// cancel（LLM 不可达 → Run 在等超时，cancel 立即生效）
	creq, _ := http.NewRequest("POST", ts.URL+"/api/sessions/"+sid+"/runs/"+rid+"/cancel", nil)
	cresp, err := http.DefaultClient.Do(creq)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	cresp.Body.Close()
	if cresp.StatusCode != http.StatusOK {
		t.Fatalf("cancel 应 200，got %d", cresp.StatusCode)
	}
	// 等待 Run goroutine 退出 → 再发消息应可 202（409 解除）
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		c, _, _ := postMessage(t, ts.URL, "", `{"text":"重发","session_id":"cancel-s"}`)
		if c == http.StatusAccepted {
			return // 成功：取消解除了忙碌
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("取消后 5s 内未解除忙碌态")
}

// TestMessagesTruncate F1 截断：seq 之后的消息 + tool_calls + checkpoints 全清。
func TestMessagesTruncate(t *testing.T) {
	ts, store := setupServer(t)
	_ = store
	// 用 HTTP 建消息（等 Run 超时落 [失败] 也行——user 消息一定在）
	code, _, _ := postMessage(t, ts.URL, "", `{"text":"第一条","session_id":"trunc-s"}`)
	if code != http.StatusAccepted {
		t.Fatalf("202 expected, got %d", code)
	}
	// 等 [失败] 落库（LLM 不可达，5s timeout 内会落）或直接操作底层 store
	time.Sleep(6 * time.Second) // 与模型 timeout_sec=5 对齐
	// 查当前消息数
	dresp, _ := http.Get(ts.URL + "/api/sessions/trunc-s")
	var det struct {
		Messages []struct {
			ID  int `json:"id"`
			Seq int `json:"seq"`
		} `json:"messages"`
	}
	_ = json.NewDecoder(dresp.Body).Decode(&det)
	dresp.Body.Close()
	if len(det.Messages) < 2 {
		t.Fatalf("应至少 2 条消息（user+失败），got %d", len(det.Messages))
	}
	// 截断 user 消息（seq=1）及其之后
	treq, _ := http.NewRequest("DELETE", ts.URL+"/api/sessions/trunc-s/messages/after?seq=1", nil)
	tresp, err := http.DefaultClient.Do(treq)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
	var tout struct {
		DeletedMessages int `json:"deleted_messages"`
	}
	_ = json.NewDecoder(tresp.Body).Decode(&tout)
	tresp.Body.Close()
	if tresp.StatusCode != http.StatusOK {
		t.Fatalf("truncate 应 200，got %d", tresp.StatusCode)
	}
	// 验证清空
	dresp2, _ := http.Get(ts.URL + "/api/sessions/trunc-s")
	var det2 struct {
		Messages []json.RawMessage `json:"messages"`
	}
	_ = json.NewDecoder(dresp2.Body).Decode(&det2)
	dresp2.Body.Close()
	if len(det2.Messages) != 0 {
		t.Fatalf("截断后应无消息，got %d", len(det2.Messages))
	}
}

// TestAnalyzeShimDeprecated 旧端点 shim 仍可用（202 语义）且带 Deprecation 头。
func TestAnalyzeShimDeprecated(t *testing.T) {
	ts, _ := setupServer(t)
	req, _ := http.NewRequest("POST", ts.URL+"/api/analyze", nopCloser{bytes.NewReader([]byte(`{"text":"hi"}`))})
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("/api/analyze shim 应 202，got %d", resp.StatusCode)
	}
	if resp.Header.Get("Deprecation") != "true" {
		t.Error("/api/analyze shim 应带 Deprecation: true 头")
	}
}

// TestChatShimStillWorks 旧端点 /api/chat（{session_id, question} body）shim 正常。
func TestChatShimStillWorks(t *testing.T) {
	ts, _ := setupServer(t)
	// 先建会话（tenantA 认领）——shim 也是 202
	code, _, _ := postMessage(t, ts.URL, "tenantA", `{"session_id":"shim-s","text":"hi"}`)
	if code != http.StatusAccepted {
		t.Fatalf("认领会话失败: %d", code)
	}
	req, _ := http.NewRequest("POST", ts.URL+"/api/chat", nopCloser{bytes.NewReader([]byte(`{"session_id":"shim-s","question":"部署方式？"}`))})
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenantA")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusAccepted {
		// 会话可能仍被第一个 Run 占用（LLM 不可达）→ 409 也算 shim 工作
		t.Errorf("/api/chat shim 应 202/409，got %d", resp.StatusCode)
	}
	if resp.Header.Get("Deprecation") != "true" {
		t.Error("/api/chat shim 应带 Deprecation: true 头")
	}
}
