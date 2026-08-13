// Package api defines the message types at the boundary between a Channel
// (HTTP / DingTalk / future) and the Agent core. The agent never sees raw
// HTTP or DingTalk formats — only these normalized types. See ADR-012.
package api

import (
	"customer-demand-agent/internal/domain"
)

// InboundMessage is a normalized incoming message from some channel.
type InboundMessage struct {
	TenantID  string // 多租户标识（一期只区分不鉴权，ADR-011）
	SessionID string // 会话标识（空则由后端创建）
	Content   string // 用户输入（客户沟通文本 / 追问）
	Source    string // 来源渠道："http" / "dingtalk"
}

// OutboundMessage is the normalized reply the Agent produces for a channel.
type OutboundMessage struct {
	SessionID string                 `json:"session_id"`
	Content   string                 `json:"content"`              // 文本回复（追问场景）
	Analysis  *domain.AnalysisResult `json:"analysis,omitempty"`   // 结构化分析结果（分析场景）
	Trace     []ToolCallTrace        `json:"tool_calls,omitempty"` // 工具调用轨迹
}

// ToolCallTrace is one tool invocation in the agent's reasoning trace.
type ToolCallTrace struct {
	Tool   string `json:"tool"`
	Params string `json:"params"`
	Result string `json:"result"`
}

// NewOutbound constructs an OutboundMessage with an analysis result + trace.
func NewOutbound(sessionID string, analysis *domain.AnalysisResult, traces []ToolCallTrace, content string) *OutboundMessage {
	return &OutboundMessage{
		SessionID: sessionID,
		Content:   content,
		Analysis:  analysis,
		Trace:     traces,
	}
}
