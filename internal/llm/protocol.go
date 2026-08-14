package llm

import (
	"encoding/json"
	"fmt"

	"customer-demand-agent/internal/domain"
)

// 协议常量（与 model 包对齐）。
const (
	ProtocolOpenAIChat     = "openai-chat"
	ProtocolOpenAIResponse = "openai-response"
	ProtocolAnthropic      = "anthropic"
)

// proto 返回有效协议（空/未知回退 openai-chat）。
func proto(p string) string {
	switch p {
	case ProtocolOpenAIResponse, ProtocolAnthropic:
		return p
	default:
		return ProtocolOpenAIChat
	}
}

// buildRequest 按协议构造请求：返回 (url, headers, body)。
func buildRequest(cfg Config, req *ChatRequest, stream bool) (string, map[string]string, []byte, error) {
	p := proto(cfg.Protocol)
	switch p {
	case ProtocolOpenAIResponse:
		return buildOpenAIResponse(cfg, req, stream)
	case ProtocolAnthropic:
		return buildAnthropic(cfg, req, stream)
	default:
		return buildOpenAIChat(cfg, req, stream)
	}
}

// parseChatResponse 按协议解析非流式响应。
func parseChatResponse(raw []byte, protocol string) (*ChatResponse, error) {
	switch proto(protocol) {
	case ProtocolOpenAIResponse:
		return parseOpenAIResponse(raw)
	case ProtocolAnthropic:
		return parseAnthropic(raw)
	default:
		return parseOpenAIChat(raw)
	}
}

// ── openai-chat ─────────────────────────────────────────────

func buildOpenAIChat(cfg Config, req *ChatRequest, stream bool) (string, map[string]string, []byte, error) {
	url := joinURL(cfg.Endpoint, "/chat/completions")
	m := map[string]any{
		"model":    firstNonEmpty(req.Model, cfg.Model),
		"messages": req.Messages,
	}
	if stream {
		m["stream"] = true
	}
	if t := tempOf(cfg, req); t > 0 {
		m["temperature"] = t
	}
	if mt := maxTokOf(cfg, req); mt > 0 {
		m["max_tokens"] = mt
	}
	if len(req.Tools) > 0 {
		m["tools"] = req.Tools
		m["tool_choice"] = "auto"
	}
	if req.JSONOutput {
		m["response_format"] = map[string]string{"type": "json_object"}
	}
	body, err := json.Marshal(m)
	return url, map[string]string{"Authorization": "Bearer " + cfg.APIKey}, body, err
}

func parseOpenAIChat(raw []byte) (*ChatResponse, error) {
	var r struct {
		Choices []struct {
			Message      openaiMessage `json:"message"`
			FinishReason string        `json:"finish_reason"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("解析 openai-chat 响应: %w", err)
	}
	if r.Error != nil {
		return nil, &APIError{StatusCode: 0, Body: r.Error.Message}
	}
	if len(r.Choices) == 0 {
		return nil, fmt.Errorf("openai-chat 响应无 choices")
	}
	ch := r.Choices[0]
	msg := domain.Message{Role: domain.RoleAssistant, Content: ch.Message.Content}
	for _, tc := range ch.Message.ToolCalls {
		msg.ToolCalls = append(msg.ToolCalls, domain.ToolCall{
			ID: tc.ID, Type: tc.Type,
			Function: domain.ToolCallFunction{Name: tc.Function.Name, Arguments: tc.Function.Arguments},
		})
	}
	return &ChatResponse{Message: msg, FinishReason: ch.FinishReason}, nil
}

// ── openai-response ─────────────────────────────────────────

func buildOpenAIResponse(cfg Config, req *ChatRequest, stream bool) (string, map[string]string, []byte, error) {
	url := joinURL(cfg.Endpoint, "/responses")
	// input 是扁平的消息数组（复用 messages）
	input := make([]map[string]any, 0, len(req.Messages))
	for _, msg := range req.Messages {
		item := map[string]any{"role": string(msg.Role), "content": msg.Content}
		if len(msg.ToolCalls) > 0 {
			item["tool_calls"] = msg.ToolCalls
		}
		if msg.ToolCallID != "" {
			item = map[string]any{
				"type":    "function_call_output",
				"call_id": msg.ToolCallID,
				"output":  msg.Content,
			}
		}
		input = append(input, item)
	}
	m := map[string]any{"model": firstNonEmpty(req.Model, cfg.Model), "input": input}
	if stream {
		m["stream"] = true
	}
	if t := tempOf(cfg, req); t > 0 {
		m["temperature"] = t
	}
	if mt := maxTokOf(cfg, req); mt > 0 {
		m["max_output_tokens"] = mt
	}
	if len(req.Tools) > 0 {
		m["tools"] = req.Tools
	}
	body, err := json.Marshal(m)
	return url, map[string]string{"Authorization": "Bearer " + cfg.APIKey}, body, err
}

func parseOpenAIResponse(raw []byte) (*ChatResponse, error) {
	var r struct {
		Output []struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
			CallID    string `json:"call_id"`
		} `json:"output"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("解析 openai-response 响应: %w", err)
	}
	if r.Error != nil {
		return nil, &APIError{StatusCode: 0, Body: r.Error.Message}
	}
	msg := domain.Message{Role: domain.RoleAssistant}
	for _, out := range r.Output {
		switch out.Type {
		case "message":
			for _, c := range out.Content {
				if c.Type == "output_text" || c.Type == "text" {
					msg.Content += c.Text
				}
			}
		case "function_call":
			msg.ToolCalls = append(msg.ToolCalls, domain.ToolCall{
				ID: out.CallID, Type: "function",
				Function: domain.ToolCallFunction{Name: out.Name, Arguments: out.Arguments},
			})
		}
	}
	return &ChatResponse{Message: msg, FinishReason: "stop"}, nil
}

// ── anthropic ───────────────────────────────────────────────

func buildAnthropic(cfg Config, req *ChatRequest, stream bool) (string, map[string]string, []byte, error) {
	url := joinURL(cfg.Endpoint, "/v1/messages")
	// 转换 tools 为 anthropic 格式
	tools := make([]map[string]any, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, map[string]any{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": t.Function.Parameters,
		})
	}
	maxTok := maxTokOf(cfg, req)
	if maxTok <= 0 {
		maxTok = 4096 // anthropic API 强制要求 max_tokens>0，否则每个请求 400
	}
	m := map[string]any{
		"model":      firstNonEmpty(req.Model, cfg.Model),
		"max_tokens": maxTok,
		"messages":   req.Messages,
	}
	if stream {
		m["stream"] = true
	}
	if t := tempOf(cfg, req); t > 0 {
		m["temperature"] = t
	}
	if len(tools) > 0 {
		m["tools"] = tools
	}
	body, err := json.Marshal(m)
	headers := map[string]string{
		"x-api-key":         cfg.APIKey,
		"anthropic-version": "2023-06-01",
		"content-type":      "application/json",
	}
	return url, headers, body, err
}

func parseAnthropic(raw []byte) (*ChatResponse, error) {
	var r struct {
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			ID    string `json:"id"`
			Name  string `json:"name"`
			Input any    `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Error      *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("解析 anthropic 响应: %w", err)
	}
	if r.Error != nil {
		return nil, &APIError{StatusCode: 0, Body: r.Error.Message}
	}
	msg := domain.Message{Role: domain.RoleAssistant}
	for _, c := range r.Content {
		switch c.Type {
		case "text":
			msg.Content += c.Text
		case "tool_use":
			args, _ := json.Marshal(c.Input)
			msg.ToolCalls = append(msg.ToolCalls, domain.ToolCall{
				ID: c.ID, Type: "function",
				Function: domain.ToolCallFunction{Name: c.Name, Arguments: string(args)},
			})
		}
	}
	finish := r.StopReason
	if finish == "" {
		if len(msg.ToolCalls) > 0 {
			finish = "tool_calls"
		} else {
			finish = "stop"
		}
	}
	return &ChatResponse{Message: msg, FinishReason: finish}, nil
}

// ── 辅助 ────────────────────────────────────────────────────

func joinURL(endpoint, suffix string) string {
	if endsWith(endpoint, suffix) {
		return endpoint
	}
	return trimRightSlash(endpoint) + suffix
}

func tempOf(cfg Config, req *ChatRequest) float64 {
	if req.Temperature != nil {
		return *req.Temperature
	}
	return cfg.Temperature
}

func maxTokOf(cfg Config, req *ChatRequest) int {
	if req.MaxTokens > 0 {
		return req.MaxTokens
	}
	return cfg.MaxTokens
}
