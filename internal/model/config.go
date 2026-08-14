// Package model implements runtime model scheduling: registration, routing
// by task type, and a fallback chain with exponential-backoff retry.
//
// 模型管理是"被调用的基础设施"。所有模型默认支持 function call，
// 不维护 FC 能力标签（ADR-007）。
package model

import (
	"time"

	"customer-demand-agent/internal/llm"
)

// ModelConfig 是一个已注册模型的配置。
type ModelConfig struct {
	Name        string  `json:"name"`        // 标识，如 "deepseek-v4-pro"
	Endpoint    string  `json:"endpoint"`    // API 网关地址
	APIKey      string  `json:"api_key"`     // 密钥（前端不读回）
	Protocol    string  `json:"protocol"`    // 接口协议：openai-chat / openai-response / anthropic
	Model       string  `json:"model"`       // 实际模型名
	Temperature float64 `json:"temperature"` // 默认 0.3
	MaxTokens   int     `json:"max_tokens"`  // 最大输出 token
}

// 协议常量。
const (
	ProtocolOpenAIChat     = "openai-chat"
	ProtocolOpenAIResponse = "openai-response"
	ProtocolAnthropic      = "anthropic"
)

// Protocol 返回协议（空则默认 openai-chat）。
func (m ModelConfig) Proto() string {
	if m.Protocol == "" {
		return ProtocolOpenAIChat
	}
	return m.Protocol
}

// ToLLMConfig 转为 llm.Config（Timeout 由调用方注入全局值）。
func (m ModelConfig) ToLLMConfig(timeout time.Duration) llm.Config {
	return llm.Config{
		Name:        m.Name,
		Endpoint:    m.Endpoint,
		APIKey:      m.APIKey,
		Protocol:    m.Proto(),
		Model:       m.Model,
		Temperature: m.Temperature,
		MaxTokens:   m.MaxTokens,
		Timeout:     timeout,
	}
}

// TaskType 是模型路由的任务类型。
type TaskType string

const (
	TaskAnalysis   TaskType = "analysis"   // 主分析（含工具调用的自主循环）
	TaskBackground TaskType = "background" // 后台任务（预留：摘要/压缩等独立任务）
)

// RouterConfig 是路由 + 回退配置。
type RouterConfig struct {
	Default  string              `json:"default"`  // 默认模型名
	Routes   map[TaskType]string `json:"routes"`   // 任务类型 → 模型名（可选）
	Fallback FallbackPolicy      `json:"fallback"` // 回退策略
}

// FallbackPolicy 是回退 + 重试策略。
type FallbackPolicy struct {
	MaxRetries int      `json:"max_retries"`     // 每个模型重试次数，默认 2
	BackoffMs  int      `json:"backoff_base_ms"` // 退避基数（毫秒），默认 200
	Chain      []string `json:"chain"`           // 回退链：模型名顺序
}
