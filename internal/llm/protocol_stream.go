package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"customer-demand-agent/internal/domain"
)

// buildChatResponse 把累积后的 tool call map 组装成 ChatResponse（泛型：支持 int/string key）。
func buildChatResponse[K comparable](content string, tcAcc map[K]*accToolCall, order []K, finishReason string) *ChatResponse {
	msg := domain.Message{Role: domain.RoleAssistant, Content: content}
	for _, idx := range order {
		acc := tcAcc[idx]
		msg.ToolCalls = append(msg.ToolCalls, domain.ToolCall{
			ID:       acc.id,
			Type:     acc.typ,
			Function: domain.ToolCallFunction{Name: acc.name, Arguments: acc.args.String()},
		})
	}
	if finishReason == "" {
		if len(msg.ToolCalls) > 0 {
			finishReason = "tool_calls"
		} else {
			finishReason = "stop"
		}
	}
	return &ChatResponse{Message: msg, FinishReason: finishReason}
}

// parseStreamOpenAIChat 解析 openai-chat 的 SSE 流。
func (c *Client) parseStreamOpenAIChat(r io.Reader, cb DeltaCallbacks) (*ChatResponse, error) {
	scanner := bufio.NewScanner(r)
	var contentB strings.Builder
	tcAcc := map[int]*accToolCall{}
	var order []int
	var finishReason string
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			if payload == "[DONE]" {
				break
			}
			continue
		}
		var ch streamChunk
		if json.Unmarshal([]byte(payload), &ch) != nil || len(ch.Choices) == 0 {
			continue
		}
		d := ch.Choices[0].Delta
		if d.ReasoningContent != "" && cb.OnReasoning != nil {
			cb.OnReasoning(d.ReasoningContent)
		}
		if d.Content != "" {
			contentB.WriteString(d.Content)
			if cb.OnContent != nil {
				cb.OnContent(d.Content)
			}
		}
		for _, tc := range d.ToolCalls {
			acc, ok := tcAcc[tc.Index]
			if !ok {
				acc = &accToolCall{}
				tcAcc[tc.Index] = acc
				order = append(order, tc.Index)
			}
			if tc.ID != "" {
				acc.id = tc.ID
			}
			if tc.Type != "" {
				acc.typ = tc.Type
			}
			if tc.Function.Name != "" {
				acc.name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				acc.args.WriteString(tc.Function.Arguments)
			}
		}
		if ch.Choices[0].FinishReason != "" {
			finishReason = ch.Choices[0].FinishReason
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取流: %w", err)
	}
	return buildChatResponse(contentB.String(), tcAcc, order, finishReason), nil
}

// parseStreamOpenAIResponse 解析 openai-response 的 SSE 流。
func (c *Client) parseStreamOpenAIResponse(r io.Reader, cb DeltaCallbacks) (*ChatResponse, error) {
	scanner := bufio.NewScanner(r)
	var contentB strings.Builder
	// tool call 用 call_id 做 key
	tcAcc := map[string]*accToolCall{}
	var order []string
	var finishReason string
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}
		var ev struct {
			Type   string `json:"type"`
			Delta  string `json:"delta"`
			ItemID string `json:"item_id"`
			Item   struct {
				ID     string `json:"id"`
				Type   string `json:"type"`
				Name   string `json:"name"`
				CallID string `json:"call_id"`
			} `json:"item"`
		}
		if json.Unmarshal([]byte(payload), &ev) != nil {
			continue
		}
		switch ev.Type {
		case "response.output_text.delta":
			if ev.Delta != "" {
				contentB.WriteString(ev.Delta) // 始终累积（不应依赖 callback 是否非空）
				if cb.OnContent != nil {
					cb.OnContent(ev.Delta)
				}
			}
		case "response.reasoning_summary_text.delta":
			// 推理摘要只进入 reasoning 通道，不能混入最终答复。Agent 会消费但不
			// 转发该通道，避免把模型内部推理或未经核验的草稿展示给用户。
			if ev.Delta != "" && cb.OnReasoning != nil {
				cb.OnReasoning(ev.Delta)
			}
		case "response.output_item.added":
			if ev.Item.Type == "function_call" {
				// key 用 item_id（与 response.function_call_arguments.delta 的 item_id 一致），
				// 而非 call_id——两者通常不同，旧代码按 call_id 注册却按 item_id 查找，会丢工具参数。
				key := ev.ItemID
				if key == "" {
					key = ev.Item.ID
				}
				acc := &accToolCall{id: ev.Item.CallID, typ: "function", name: ev.Item.Name}
				if acc.id == "" {
					acc.id = key // 兜底：某些网关用 item id 作 call id
				}
				tcAcc[key] = acc
				order = append(order, key)
			}
		case "response.function_call_arguments.delta":
			if acc, ok := tcAcc[ev.ItemID]; ok && ev.Delta != "" {
				acc.args.WriteString(ev.Delta)
			}
		case "response.completed":
			finishReason = "stop"
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取流: %w", err)
	}
	return buildChatResponse(contentB.String(), tcAcc, order, finishReason), nil
}

// parseStreamAnthropic 解析 anthropic 的 SSE 流。
func (c *Client) parseStreamAnthropic(r io.Reader, cb DeltaCallbacks) (*ChatResponse, error) {
	scanner := bufio.NewScanner(r)
	var contentB strings.Builder
	// tool use 用 index 做 key
	tcAcc := map[int]*accToolCall{}
	var order []int
	var finishReason string
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		var ev struct {
			Type  string `json:"type"`
			Index int    `json:"index"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
				StopReason  string `json:"stop_reason"`
			} `json:"delta"`
			ContentBlock struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"content_block"`
		}
		if json.Unmarshal([]byte(payload), &ev) != nil {
			continue
		}
		switch ev.Type {
		case "content_block_start":
			if ev.ContentBlock.Type == "tool_use" {
				acc := &accToolCall{id: ev.ContentBlock.ID, typ: "function", name: ev.ContentBlock.Name}
				tcAcc[ev.Index] = acc
				order = append(order, ev.Index)
			}
		case "content_block_delta":
			if ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
				contentB.WriteString(ev.Delta.Text)
				if cb.OnContent != nil {
					cb.OnContent(ev.Delta.Text)
				}
			} else if ev.Delta.Type == "input_json_delta" && ev.Delta.PartialJSON != "" {
				if acc, ok := tcAcc[ev.Index]; ok {
					acc.args.WriteString(ev.Delta.PartialJSON)
				}
			} else if ev.Delta.Type == "thinking_delta" && ev.Delta.Text != "" && cb.OnReasoning != nil {
				cb.OnReasoning(ev.Delta.Text)
			}
		case "message_delta":
			if ev.Delta.StopReason != "" {
				finishReason = ev.Delta.StopReason
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取流: %w", err)
	}
	return buildChatResponse(contentB.String(), tcAcc, order, finishReason), nil
}
