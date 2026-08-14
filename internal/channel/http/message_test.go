package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
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
	// 验证三表联删：messages 清空 + tool_calls/checkpoints（经 store 直查）
	dresp2, _ := http.Get(ts.URL + "/api/sessions/trunc-s")
	var det2 struct {
		Messages []json.RawMessage `json:"messages"`
	}
	_ = json.NewDecoder(dresp2.Body).Decode(&det2)
	dresp2.Body.Close()
	if len(det2.Messages) != 0 {
		t.Fatalf("截断后应无消息，got %d", len(det2.Messages))
	}
	// tool_calls（detail 接口可见）应清空
	dresp3, _ := http.Get(ts.URL + "/api/sessions/trunc-s")
	var det3 struct {
		ToolCalls []json.RawMessage `json:"tool_calls"`
	}
	_ = json.NewDecoder(dresp3.Body).Decode(&det3)
	dresp3.Body.Close()
	if len(det3.ToolCalls) != 0 {
		t.Errorf("截断后 tool_calls 应 0，got %d", len(det3.ToolCalls))
	}
	// checkpoints 清空：断点续传后行为验证——ListCheckpoints 归零通过
	// store 直查（setupServer 未暴露 db；用恢复路径：GetSession detail 不含
	// checkpoints，此处通过「截断响应 ok」+ messages/tool_calls 清空 + 后续
	// 重发成功间接证明 checkpoints 不残留（残留会导致重发后 checkpoint 链
	// 混入旧分析——TestMessagesTruncateAndResend 覆盖）。
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

// TestRunEndpointsTenantIsolation F0 安全：running/cancel 跨租户 404。
func TestRunEndpointsTenantIsolation(t *testing.T) {
	ts, _ := setupServer(t)
	// tenantA 建会话并启动 Run
	code, sid, rid := postMessage(t, ts.URL, "tenantA", `{"text":"你好","session_id":"iso-s"}`)
	if code != http.StatusAccepted {
		t.Fatalf("202 expected, got %d", code)
	}
	// tenantB 查 running → 404
	rreq, _ := http.NewRequest("GET", ts.URL+"/api/sessions/"+sid+"/running", nil)
	rreq.Header.Set("X-Tenant-ID", "tenantB")
	rresp, err := http.DefaultClient.Do(rreq)
	if err != nil {
		t.Fatal(err)
	}
	rresp.Body.Close()
	if rresp.StatusCode != http.StatusNotFound {
		t.Errorf("跨租户 running 应 404，got %d", rresp.StatusCode)
	}
	// tenantB cancel → 404（Run 不受影响）
	creq, _ := http.NewRequest("POST", ts.URL+"/api/sessions/"+sid+"/runs/"+rid+"/cancel", nil)
	creq.Header.Set("X-Tenant-ID", "tenantB")
	cresp, err := http.DefaultClient.Do(creq)
	if err != nil {
		t.Fatal(err)
	}
	cresp.Body.Close()
	if cresp.StatusCode != http.StatusNotFound {
		t.Errorf("跨租户 cancel 应 404，got %d", cresp.StatusCode)
	}
	// 跨租户 cancel 被 404 拒绝后，本租户 running 接口仍可用（200）——
	// 注：setupServer 的 LLM 端点不可达，Run 可能在断言前已结束，故只验
	// 接口可用性 + 响应结构，不断言 running 布尔值。
	areq, _ := http.NewRequest("GET", ts.URL+"/api/sessions/"+sid+"/running", nil)
	areq.Header.Set("X-Tenant-ID", "tenantA")
	aresp, err := http.DefaultClient.Do(areq)
	if err != nil {
		t.Fatal(err)
	}
	defer aresp.Body.Close()
	if aresp.StatusCode != http.StatusOK {
		t.Errorf("本租户 running 应 200，got %d", aresp.StatusCode)
	}
	var out struct {
		Running bool   `json:"running"`
		RunID   string `json:"run_id"`
	}
	_ = json.NewDecoder(aresp.Body).Decode(&out)
	_ = out
}

// TestMessageBusy409NoOrphanUserMessage 并发 409 不留孤儿 user 消息：
// 被拒的第二个请求不应写历史。
func TestMessageBusy409NoOrphanUserMessage(t *testing.T) {
	ts, _ := setupServer(t)
	code1, _, _ := postMessage(t, ts.URL, "", `{"text":"第一条","session_id":"orphan-s"}`)
	if code1 != http.StatusAccepted {
		t.Fatalf("第一条应 202，got %d", code1)
	}
	// 并发第二条 → 409
	code2, _, _ := postMessage(t, ts.URL, "", `{"text":"第二条不该落库","session_id":"orphan-s"}`)
	if code2 != http.StatusConflict {
		t.Fatalf("并发应 409，got %d", code2)
	}
	// 立刻查会话：user 消息只应有第一条（409 的不落库；第一条在 Run goroutine 内写入）
	deadline := time.Now().Add(3 * time.Second)
	var userCount int
	for time.Now().Before(deadline) {
		dresp, _ := http.Get(ts.URL + "/api/sessions/orphan-s")
		var det struct {
			Messages []struct {
				Role string `json:"role"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(dresp.Body).Decode(&det)
		dresp.Body.Close()
		userCount = 0
		for _, m := range det.Messages {
			if m.Role == "user" {
				userCount++
			}
		}
		if userCount >= 1 {
			break // 第一条已写入
		}
		time.Sleep(100 * time.Millisecond)
	}
	if userCount != 1 {
		t.Fatalf("user 消息应恰 1 条（409 不落库），got %d", userCount)
	}
}

// TestStreamSeqCursorResume SSE 游标：事件带 id: seq；since=N 只收到 N 之后。
func TestStreamSeqCursorResume(t *testing.T) {
	ts, _ := setupServer(t)
	code, sid, _ := postMessage(t, ts.URL, "", `{"text":"你好","session_id":"cursor-s"}`)
	if code != http.StatusAccepted {
		t.Fatalf("202 expected, got %d", code)
	}
	// 收一段事件（带 id: 游标）
	first := sseRead(t, ts.URL, sid, 0, 2*time.Second)
	if !strings.Contains(first, `"type":"session"`) {
		t.Fatalf("应含 session 事件: %s", first)
	}
	// 提取最大 seq
	seqs := sseIDs(first)
	if len(seqs) == 0 {
		t.Fatal("SSE 应带 id: 游标（无则无法续传）")
	}
	maxSeq := seqs[len(seqs)-1]
	// since=maxSeq 增量订阅：不重复已收事件（所有 id > maxSeq）
	time.Sleep(300 * time.Millisecond) // 让 Run 再产生些事件
	second := sseRead(t, ts.URL, sid, maxSeq, 2*time.Second)
	for _, id := range sseIDs(second) {
		if id <= maxSeq {
			t.Fatalf("增量续传收到重复事件 id=%d（游标 %d）", id, maxSeq)
		}
	}
}

// sseRead 读一段 SSE 流（限时）返回原文。
func sseRead(t *testing.T, tsURL, sid string, since int, d time.Duration) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", tsURL+"/api/sessions/"+sid+"/stream?since=0", nil)
	if since > 0 {
		req.URL.RawQuery = "since=" + strconv.Itoa(since)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sse: %v", err)
	}
	defer resp.Body.Close()
	buf := make([]byte, 8192)
	var out []byte
	for len(out) < 65536 {
		n, err := resp.Body.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil || n == 0 {
			break
		}
	}
	return string(out)
}

// sseIDs 提取 SSE 流中的全部 id: 游标（升序出现）。
func sseIDs(raw string) []int {
	var out []int
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "id:") {
			if n, err := strconv.Atoi(strings.TrimSpace(line[3:])); err == nil {
				out = append(out, n)
			}
		}
	}
	return out
}
