// Package llm implements an OpenAI-compatible chat completion client with
// function-calling support. This is the raw HTTP layer; routing/fallback/retry
// live in package model.
package llm

import (
	"bytes"
	"context"
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
	Protocol    string // openai-chat / openai-response / anthropic
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

// NewClientWithHTTP 用指定的 *http.Client 创建客户端。
// 用于探测端点注入 SSRF 硬化 transport（连接时 IP 校验 + 重定向再校验）。
func NewClientWithHTTP(h *http.Client) *Client {
	if h == nil {
		h = &http.Client{Timeout: 120 * time.Second}
	}
	return &Client{http: h}
}

// ChatRequest 是单次对话请求。
type ChatRequest struct {
	Model       string
	Messages    []domain.Message
	Tools       []domain.Tool // 可选：function calling
	JSONOutput  bool          // 是否要求 json_object 输出
	Temperature *float64      // 覆盖配置默认温度
	MaxTokens   int           // 覆盖配置默认上限
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

	url, headers, body, err := buildRequest(cfg, req, false)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("构造请求: %w", err)
	}
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

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

	return parseChatResponse(raw, cfg.Protocol)
}

// buildBody 已移除：请求构造按协议分派到 protocol.go。
// openaiMessage / openaiToolCall 保留（protocol.go 引用）。

type openaiMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
}

type openaiToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
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

// ChatStream 流式调用。逐 chunk 调 cb 推送思考/回答增量，返回累积的完整响应。
func (c *Client) ChatStream(ctx context.Context, cfg Config, req *ChatRequest, cb DeltaCallbacks) (*ChatResponse, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("LLM endpoint 未配置")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("LLM api_key 未配置（分析接口不可用）")
	}
	req.JSONOutput = false // 流式 + json_object 部分网关不兼容，靠 prompt 约束输出

	url, headers, body, err := buildRequest(cfg, req, true)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("构造请求: %w", err)
	}
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
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

	switch proto(cfg.Protocol) {
	case ProtocolOpenAIResponse:
		return c.parseStreamOpenAIResponse(resp.Body, cb)
	case ProtocolAnthropic:
		return c.parseStreamAnthropic(resp.Body, cb)
	default:
		return c.parseStreamOpenAIChat(resp.Body, cb)
	}
}
