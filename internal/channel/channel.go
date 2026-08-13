// Package channel defines the message-entry abstraction. The agent core never
// sees raw HTTP or DingTalk formats — only the normalized api.InboundMessage /
// api.OutboundMessage. HTTP is the first implementation; DingTalk webhook is
// reserved (ADR-012).
package channel

import (
	"context"

	"customer-demand-agent/internal/api"
)

// Channel is a message transport (HTTP / DingTalk / future).
// 一期只实现 HTTPChannel；DingTalkChannel 是接口预留。
type Channel interface {
	// Name returns the channel identifier ("http" / "dingtalk").
	Name() string
	// Process handles an inbound message end-to-end and returns the outbound reply.
	Process(ctx context.Context, in *api.InboundMessage) (*api.OutboundMessage, error)
}

// Processor is the contract a channel delegates to (typically the agent +
// memory + history orchestration). Keeping it separate lets DingTalk reuse
// the exact same processing pipeline as HTTP.
type Processor interface {
	Process(ctx context.Context, in *api.InboundMessage) (*api.OutboundMessage, error)
}
