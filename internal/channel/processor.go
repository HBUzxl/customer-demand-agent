package channel

import (
	"context"
	"strings"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/api"
	"customer-demand-agent/internal/history"
)

// AgentProcessor 把 agent 适配为 channel.Processor：异步渠道（即应/钉钉）
// 与 HTTP 渠道共用同一处理管线——跑 agent.Message 并落库。history 为 nil 时
// 跳过落库（checkpoint 仍由 agent 的 sink 回调持久化）。
type AgentProcessor struct {
	ag      *agent.Agent
	history *history.Store
	onEvent func(agent.Event) // 可选：流式事件转发；nil 则丢弃
}

// NewAgentProcessor 构造处理器。hist 可传 nil（仅不落消息历史）。
func NewAgentProcessor(ag *agent.Agent, hist *history.Store) *AgentProcessor {
	return &AgentProcessor{ag: ag, history: hist}
}

// SetOnEvent 注册流式事件回调（如即应把 ask_user 转成 choice 消息）；
// 须在 Process 前调用。
func (p *AgentProcessor) SetOnEvent(fn func(agent.Event)) {
	p.onEvent = fn
}

// Process 执行一轮 agent 自主循环并归一化为 OutboundMessage。
func (p *AgentProcessor) Process(ctx context.Context, in *api.InboundMessage) (*api.OutboundMessage, error) {
	sessionID := in.SessionID
	if p.history != nil && sessionID != "" {
		_ = p.history.EnsureSession(sessionID, firstLine(in.Content), "")
		_, _ = p.history.AppendMessage(sessionID, "user", in.Content, "")
	}

	content, analysis, trace, err := p.ag.Message(ctx, sessionID, in.Content, p.onEvent)
	if err != nil {
		return nil, err
	}

	if p.history != nil && sessionID != "" {
		assistantID, _ := p.history.AppendMessage(sessionID, "assistant", content, "")
		if trace != nil {
			for _, tc := range trace.ToolCalls {
				_, _ = p.history.AppendToolCall(sessionID, assistantID, tc.Tool, tc.Params, tc.Result)
			}
		}
	}

	traces := make([]api.ToolCallTrace, 0)
	if trace != nil {
		for _, tc := range trace.ToolCalls {
			traces = append(traces, api.ToolCallTrace{Tool: tc.Tool, Params: tc.Params, Result: tc.Result})
		}
	}
	return api.NewOutbound(sessionID, analysis, traces, content), nil
}

// firstLine 取首行作会话标题兜底。
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	if len([]rune(s)) > 40 {
		return string([]rune(s)[:40]) + "…"
	}
	return s
}
