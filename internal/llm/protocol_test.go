package llm

import (
	"encoding/json"
	"strings"
	"testing"

	"customer-demand-agent/internal/domain"
)

// ── 非流式解析器 ───────────────────────────────────────────────

// TestParseOpenAIChat 覆盖 openai-chat 非流式解析：正常内容 + tool_calls + body 级 error + 空 choices。
func TestParseOpenAIChat(t *testing.T) {
	// happy path：content + tool_calls
	raw := `{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":"想一下","tool_calls":[{"id":"c1","type":"function","function":{"name":"memory_search","arguments":"{\"q\":\"雷池\"}"}}]}}]}`
	resp, err := parseOpenAIChat([]byte(raw))
	if err != nil {
		t.Fatalf("happy: %v", err)
	}
	if resp.Message.Content != "想一下" || resp.FinishReason != "tool_calls" {
		t.Errorf("content/finish: %+v", resp)
	}
	if len(resp.Message.ToolCalls) != 1 || resp.Message.ToolCalls[0].Function.Name != "memory_search" {
		t.Errorf("tool_calls: %+v", resp.Message.ToolCalls)
	}
	if resp.Message.ToolCalls[0].Function.Arguments == "" {
		t.Error("tool_call arguments 丢失")
	}

	// body 级 error
	if _, err := parseOpenAIChat([]byte(`{"error":{"message":"rate limited"}}`)); err == nil {
		t.Error("body error 应返回 error")
	}

	// 空 choices
	if _, err := parseOpenAIChat([]byte(`{"choices":[]}`)); err == nil {
		t.Error("空 choices 应返回 error")
	}

	// 畸形 JSON
	if _, err := parseOpenAIChat([]byte(`not json`)); err == nil {
		t.Error("畸形 JSON 应返回 error")
	}
}

// TestParseOpenAIResponse 覆盖 openai-response 非流式解析：message(output_text) + function_call。
func TestParseOpenAIResponse(t *testing.T) {
	raw := `{"output":[
		{"type":"message","role":"assistant","content":[{"type":"output_text","text":"结论A"}]},
		{"type":"function_call","name":"memory_search","arguments":"{\"q\":\"X\"}","call_id":"fc_1"}
	]}`
	resp, err := parseOpenAIResponse([]byte(raw))
	if err != nil {
		t.Fatalf("happy: %v", err)
	}
	if !strings.Contains(resp.Message.Content, "结论A") {
		t.Errorf("content 丢失: %q", resp.Message.Content)
	}
	if len(resp.Message.ToolCalls) != 1 || resp.Message.ToolCalls[0].ID != "fc_1" ||
		resp.Message.ToolCalls[0].Function.Name != "memory_search" {
		t.Errorf("function_call 解析错: %+v", resp.Message.ToolCalls)
	}

	// body error
	if _, err := parseOpenAIResponse([]byte(`{"error":{"message":"bad"}}`)); err == nil {
		t.Error("body error 应返回 error")
	}
}

// TestParseAnthropic 覆盖 anthropic 非流式解析：text + tool_use + stop_reason 推断。
func TestParseAnthropic(t *testing.T) {
	raw := `{"content":[
		{"type":"text","text":"分析中"},
		{"type":"tool_use","id":"tu_1","name":"memory_search","input":{"q":"雷池"}}
	],"stop_reason":"tool_use"}`
	resp, err := parseAnthropic([]byte(raw))
	if err != nil {
		t.Fatalf("happy: %v", err)
	}
	if !strings.Contains(resp.Message.Content, "分析中") {
		t.Errorf("text 丢失: %q", resp.Message.Content)
	}
	if len(resp.Message.ToolCalls) != 1 || resp.Message.ToolCalls[0].Function.Name != "memory_search" {
		t.Errorf("tool_use 解析错: %+v", resp.Message.ToolCalls)
	}
	// tool_use 的 input 应序列化回 arguments
	if resp.Message.ToolCalls[0].Function.Arguments == "" {
		t.Error("tool_use arguments 丢失")
	}
	if resp.FinishReason != "tool_use" {
		t.Errorf("stop_reason: %q", resp.FinishReason)
	}

	// 无 stop_reason 但有 tool_calls → 推断 tool_calls
	raw2 := `{"content":[{"type":"tool_use","id":"tu_2","name":"x","input":{}}]}`
	resp2, _ := parseAnthropic([]byte(raw2))
	if resp2.FinishReason != "tool_calls" {
		t.Errorf("应推断 tool_calls，got %q", resp2.FinishReason)
	}
}

// ── 流式解析器 ─────────────────────────────────────────────────

// sseReader 把多行 data: payload 拼成一个 reader（模拟 SSE）。
func sseReader(lines ...string) *strings.Reader {
	return strings.NewReader(strings.Join(lines, "\n") + "\n")
}

// TestParseStreamOpenAIChat 覆盖流式累积：content 分片 + tool_call 按 index 归并 + [DONE]。
func TestParseStreamOpenAIChat(t *testing.T) {
	chunks := []string{
		`data: {"choices":[{"delta":{"content":"hel"}}]}`,
		`data: {"choices":[{"delta":{"content":"lo"}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"memory_search"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"q\":"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"雷池\"}"}}]}}]}`,
		`data: {"choices":[{"finish_reason":"tool_calls"}]}`,
		`data: [DONE]`,
	}
	c := &Client{}
	var sawContent string
	resp, err := c.parseStreamOpenAIChat(sseReader(chunks...), DeltaCallbacks{
		OnContent: func(s string) { sawContent += s },
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if sawContent != "hello" {
		t.Errorf("content 增量: %q", sawContent)
	}
	if resp.Message.Content != "hello" {
		t.Errorf("累积 content: %q", resp.Message.Content)
	}
	if len(resp.Message.ToolCalls) != 1 || resp.Message.ToolCalls[0].Function.Arguments != `{"q":"雷池"}` {
		t.Errorf("tool_call 累积错: %+v", resp.Message.ToolCalls)
	}
	if resp.FinishReason != "tool_calls" {
		t.Errorf("finish: %q", resp.FinishReason)
	}
}

// TestParseStreamOpenAIResponseToolArgsKey 验证 R1 修复：
// output_item.added 用 item_id 注册、function_call_arguments.delta 用同一 item_id 查找 → 参数不丢。
func TestParseStreamOpenAIResponseToolArgsKey(t *testing.T) {
	// 构造 call_id("call_xyz") ≠ item_id("item_abc")，旧代码会丢参数
	chunks := []string{
		`data: {"type":"response.output_item.added","item_id":"item_abc","item":{"type":"function_call","name":"memory_search","call_id":"call_xyz"}}`,
		`data: {"type":"response.function_call_arguments.delta","item_id":"item_abc","delta":"{\"q\":"}`,
		`data: {"type":"response.function_call_arguments.delta","item_id":"item_abc","delta":"\"雷池\"}"}`,
		`data: {"type":"response.output_text.delta","delta":"结论"}`,
		`data: {"type":"response.completed"}`,
	}
	c := &Client{}
	resp, err := c.parseStreamOpenAIResponse(sseReader(chunks...), DeltaCallbacks{})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if len(resp.Message.ToolCalls) != 1 {
		t.Fatalf("应累积 1 个 tool_call，got %d（参数可能丢了）", len(resp.Message.ToolCalls))
	}
	tc := resp.Message.ToolCalls[0]
	if tc.Function.Name != "memory_search" {
		t.Errorf("name: %s", tc.Function.Name)
	}
	if tc.ID != "call_xyz" {
		t.Errorf("应保留 call_id 作 tool call ID，got %q", tc.ID)
	}
	if tc.Function.Arguments != `{"q":"雷池"}` {
		t.Errorf("arguments 应被完整累积（item_id 一致 key），got %q", tc.Function.Arguments)
	}
	if resp.Message.Content != "结论" {
		t.Errorf("content: %q", resp.Message.Content)
	}
}

// TestParseStreamAnthropic 覆盖 anthropic 流式：text_delta + tool_use 累积。
func TestParseStreamAnthropic(t *testing.T) {
	chunks := []string{
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"tu_1","name":"memory_search"}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"q\":"}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"雷池\"}"}}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"结论"}}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use"}}`,
	}
	c := &Client{}
	resp, err := c.parseStreamAnthropic(sseReader(chunks...), DeltaCallbacks{})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if resp.Message.Content != "结论" {
		t.Errorf("text content: %q", resp.Message.Content)
	}
	if len(resp.Message.ToolCalls) != 1 || resp.Message.ToolCalls[0].Function.Arguments != `{"q":"雷池"}` {
		t.Errorf("tool_use 累积错: %+v", resp.Message.ToolCalls)
	}
}

// ── buildAnthropic max_tokens 下限（R2）─────────────────────────

// TestBuildAnthropicMaxTokensFloor 验证 anthropic 在 max_tokens 未配（0）时取下限 4096，
// 且已配值时不被覆盖。
func TestBuildAnthropicMaxTokensFloor(t *testing.T) {
	// 未配 max_tokens → 应回退到 4096
	_, _, body, err := buildAnthropic(Config{Endpoint: "https://x", Protocol: ProtocolAnthropic, APIKey: "k"},
		&ChatRequest{Messages: []domain.Message{{Role: domain.RoleUser, Content: "hi"}}}, false)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if mt, _ := m["max_tokens"].(float64); mt != 4096 {
		t.Errorf("未配 max_tokens 应为 4096，got %v", m["max_tokens"])
	}

	// 已配 2048 → 保持
	_, _, body2, _ := buildAnthropic(Config{Endpoint: "https://x", Protocol: ProtocolAnthropic, APIKey: "k", MaxTokens: 2048},
		&ChatRequest{Messages: []domain.Message{{Role: domain.RoleUser, Content: "hi"}}}, false)
	json.Unmarshal(body2, &m)
	if mt, _ := m["max_tokens"].(float64); mt != 2048 {
		t.Errorf("已配 2048 应保持，got %v", m["max_tokens"])
	}
}
