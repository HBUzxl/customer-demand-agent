// Package llm implements an OpenAI-compatible chat completion client with
// function-calling support. This is the raw HTTP layer; routing/fallback/retry
// live in package model.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"customer-demand-agent/internal/domain"
)

// Config 是单个模型端点的配置（与 model.ModelConfig 字段对齐）。
type Config struct {
	Name        string
	Endpoint    string
	APIKey      string
	Model       string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
}

// Client 是 OpenAI 兼容的 chat completion 客户端。
type Client struct {
	http *http.Client
}

// NewClient 创建客户端（复用连接池）。
func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 120 * time.Second}}
}

// ChatRequest 是单次对话请求。
type ChatRequest struct {
	Model       string
	Messages    []domain.Message
	Tools       []domain.Tool         // 可选：function calling
	JSONOutput  bool                  // 是否要求 json_object 输出
	Temperature *float64              // 覆盖配置默认温度
	MaxTokens   int                   // 覆盖配置默认上限
}

// ChatResponse 是单次对话响应（解析后的核心字段）。
type ChatResponse struct {
	Message      domain.Message // assistant 消息（含 content + tool_calls）
	FinishReason string         // stop / tool_calls / length
	Usage        Usage
}

// Usage 是 token 用量。
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Chat 调用 chat completions，返回解析后的 assistant 消息。
func (c *Client) Chat(ctx context.Context, cfg Config, req *ChatRequest) (*ChatResponse, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("LLM endpoint 未配置")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("LLM api_key 未配置（分析接口不可用）")
	}

	body, err := c.buildBody(cfg, req, false)
	if err != nil {
		return nil, err
	}

	url := cfg.Endpoint
	if !endsWith(url, "/chat/completions") {
		url = trimRightSlash(url) + "/chat/completions"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("构造请求: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, &TransportError{Cause: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(raw),
			Retryable:  isRetryableStatus(resp.StatusCode),
		}
	}

	return parseResponse(raw)
}

// buildBody 构造 OpenAI 兼容的请求体。stream=true 时加上流式标记。
func (c *Client) buildBody(cfg Config, req *ChatRequest, stream bool) ([]byte, error) {
	m := map[string]any{
		"model":    firstNonEmpty(req.Model, cfg.Model),
		"messages": req.Messages,
	}
	if stream {
		m["stream"] = true
	}
	temp := cfg.Temperature
	if req.Temperature != nil {
		temp = *req.Temperature
	}
	if temp > 0 {
		m["temperature"] = temp
	}
	maxTok := req.MaxTokens
	if maxTok <= 0 {
		maxTok = cfg.MaxTokens
	}
	if maxTok > 0 {
		m["max_tokens"] = maxTok
	}
	if len(req.Tools) > 0 {
		m["tools"] = req.Tools
		m["tool_choice"] = "auto"
	}
	if req.JSONOutput {
		m["response_format"] = map[string]string{"type": "json_object"}
	}
	return json.Marshal(m)
}

// ── OpenAI 响应结构 ───────────────────────────────────────────

type openaiResponse struct {
	Choices []struct {
		Message      openaiMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

type openaiMessage struct {
	Role      string             `json:"role"`
	Content   string             `json:"content"`
	ToolCalls []openaiToolCall   `json:"tool_calls,omitempty"`
}

type openaiToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func parseResponse(raw []byte) (*ChatResponse, error) {
	var r openaiResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("解析响应 JSON: %w; body: %s", err, truncateStr(string(raw), 300))
	}
	if r.Error != nil {
		return nil, &APIError{
			StatusCode: 0,
			Body:       r.Error.Message,
			Retryable:  false, // 内容/参数类错误默认不重试
		}
	}
	if len(r.Choices) == 0 {
		return nil, fmt.Errorf("响应无 choices; body: %s", truncateStr(string(raw), 300))
	}
	ch := r.Choices[0]
	msg := domain.Message{
		Role:    domain.RoleAssistant,
		Content: ch.Message.Content,
	}
	for _, tc := range ch.Message.ToolCalls {
		msg.ToolCalls = append(msg.ToolCalls, domain.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: domain.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		})
	}
	return &ChatResponse{
		Message:      msg,
		FinishReason: ch.FinishReason,
		Usage: Usage{
			PromptTokens:     r.Usage.PromptTokens,
			CompletionTokens: r.Usage.CompletionTokens,
			TotalTokens:      r.Usage.TotalTokens,
		},
	}, nil
}

// ── 错误类型 ──────────────────────────────────────────────────

// TransportError 是网络层错误（可重试）。
type TransportError struct{ Cause error }

func (e *TransportError) Error() string { return "网络错误: " + e.Cause.Error() }

// APIError 是 API 返回的非 2xx。
type APIError struct {
	StatusCode int
	Body       string
	Retryable  bool
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API 错误 %d: %s", e.StatusCode, truncateStr(e.Body, 200))
}

// IsRetryable 判断错误是否值得重试。
func IsRetryable(err error) bool {
	if te, ok := err.(*TransportError); ok {
		_ = te
		return true
	}
	if ae, ok := err.(*APIError); ok {
		return ae.Retryable
	}
	return false
}

func isRetryableStatus(code int) bool {
	return code == 429 || code >= 500
}

// ── 辅助 ──────────────────────────────────────────────────────

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func trimRightSlash(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '/') {
		s = s[:len(s)-1]
	}
	return s
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ── 流式调用 ──────────────────────────────────────────────────

// DeltaCallbacks 在流式过程中接收增量文本。
type DeltaCallbacks struct {
	OnReasoning func(text string) // 思考增量（reasoning_content）
	OnContent   func(text string) // 回答增量（content）
}

// accToolCall 累积一个工具调用的碎片（按 index 归并）。
type accToolCall struct {
	id, typ, name string
	args          strings.Builder
}

// streamChunk 是一条 SSE data 的 JSON 结构。
type streamChunk struct {
	Choices []struct {
		Delta struct {
			ReasoningContent string `json:"reasoning_content"`
			Content          string `json:"content"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// ChatStream 流式调用 chat completions。逐 chunk 调 cb 推送思考/回答增量，
// 返回累积后的完整响应（含 tool_calls）。
func (c *Client) ChatStream(ctx context.Context, cfg Config, req *ChatRequest, cb DeltaCallbacks) (*ChatResponse, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("LLM endpoint 未配置")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("LLM api_key 未配置（分析接口不可用）")
	}
	req.JSONOutput = false // 流式 + json_object 部分网关不兼容，靠 prompt 约束输出
	body, err := c.buildBody(cfg, req, true)
	if err != nil {
		return nil, err
	}
	url := cfg.Endpoint
	if !endsWith(url, "/chat/completions") {
		url = trimRightSlash(url) + "/chat/completions"
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("构造请求: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, &TransportError{Cause: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(raw), Retryable: isRetryableStatus(resp.StatusCode)}
	}

	scanner := bufio.NewScanner(resp.Body)
	var contentB strings.Builder
	tcAcc := map[int]*accToolCall{}
	var order []int // 保持 tool_calls 顺序
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
		if json.Unmarshal([]byte(payload), &ch) != nil {
			continue
		}
		if len(ch.Choices) == 0 {
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

	// 组装最终响应
	msg := domain.Message{Role: domain.RoleAssistant, Content: contentB.String()}
	if len(order) > 0 {
		for _, idx := range order {
			acc := tcAcc[idx]
			msg.ToolCalls = append(msg.ToolCalls, domain.ToolCall{
				ID:   acc.id,
				Type: acc.typ,
				Function: domain.ToolCallFunction{Name: acc.name, Arguments: acc.args.String()},
			})
		}
	}
	if finishReason == "" {
		if len(msg.ToolCalls) > 0 {
			finishReason = "tool_calls"
		} else {
			finishReason = "stop"
		}
	}
	return &ChatResponse{Message: msg, FinishReason: finishReason}, nil
}
