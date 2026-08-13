// Package dingtalk is a RESERVED channel implementation for DingTalk webhook
// integration. It is intentionally a stub: the Channel abstraction (package
// channel) and api.InboundMessage/OutboundMessage already define the seam, so
// wiring DingTalk later means implementing Process() here without touching the
// agent core (ADR-012).
//
// 未来对接钉钉时实现：
//   - Receive：接收钉钉机器人 webhook 推送，解析为 api.InboundMessage
//   - Reply：主动推送回复到群聊/单聊（钉钉需异步推送，与 HTTP 同步返回不同）
package dingtalk

import (
	"context"
	"fmt"

	"customer-demand-agent/internal/api"
	"customer-demand-agent/internal/channel"
)

// Channel is the (stub) DingTalk channel. It implements channel.Channel but
// returns not-implemented until the webhook integration is built.
type Channel struct {
	processor channel.Processor
}

// New creates a DingTalk channel bound to the shared processor pipeline.
func New(processor channel.Processor) *Channel {
	return &Channel{processor: processor}
}

// Name returns the channel identifier.
func (c *Channel) Name() string { return "dingtalk" }

// Process is not yet implemented. Future: parse DingTalk webhook payload →
// InboundMessage → processor.Process → push reply via DingTalk API.
func (c *Channel) Process(ctx context.Context, in *api.InboundMessage) (*api.OutboundMessage, error) {
	return nil, fmt.Errorf("钉钉渠道尚未实现（ADR-012 预留口子）")
}

// RegisterWebhook would mount the DingTalk webhook endpoint on a mux.
// Stub: future implementation parses signed webhook payloads.
func (c *Channel) RegisterWebhook() error {
	return fmt.Errorf("钉钉 webhook 注册尚未实现")
}
