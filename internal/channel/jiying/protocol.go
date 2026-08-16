package jiying

import "encoding/json"

// 协议常量：Envelope 版本、消息 kind 与可靠事件名。
const (
	protocolVersion = 1

	kindRequest  = "request"
	kindResponse = "response"
	kindEvent    = "event"

	eventMessageCreated        = "message.created"
	eventTopicClosed           = "topic.closed"
	eventChoiceResponseCreated = "choice.response_created"
)

// envelope 是 request/response/event 共用的外层结构，字段按 kind 选择性填充。
type envelope struct {
	V       int             `json:"v"`
	Kind    string          `json:"kind"`
	ID      string          `json:"id,omitempty"`
	ReplyTo string          `json:"reply_to,omitempty"`
	Method  string          `json:"method,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	OK      *bool           `json:"ok,omitempty"`
	Error   *apiError       `json:"error,omitempty"`
	Cursor  int64           `json:"cursor,omitempty"`
	Event   string          `json:"event,omitempty"`
}

// apiError 是失败响应的 error 字段。
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// rpcError 是应用视角的 RPC 调用失败。
type rpcError struct {
	Method  string
	Code    string
	Message string
}

func (e *rpcError) Error() string {
	if e.Code != "" {
		return e.Method + " 失败（" + e.Code + "）: " + e.Message
	}
	return e.Method + " 失败: " + e.Message
}

// ── 事件载荷 ─────────────────────────────────────────────

// messageCreatedPayload 是 message.created 的载荷。
type messageCreatedPayload struct {
	Conversation conversation `json:"conversation"`
	Sender       user         `json:"sender"`
	Message      message      `json:"message"`
}

// choiceResponsePayload 是 choice.response_created 的载荷。
type choiceResponsePayload struct {
	Conversation  conversation   `json:"conversation"`
	ChoiceMessage message        `json:"choice_message"`
	Sender        user           `json:"sender"`
	Response      choiceResponse `json:"response"`
}

// topicClosedPayload 是 topic.closed 的载荷。
type topicClosedPayload struct {
	Archived             bool   `json:"archived"`
	ConversationID       string `json:"conversation_id"`
	ParentConversationID string `json:"parent_conversation_id"`
	SourceMessageID      string `json:"source_message_id"`
}

type conversation struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Type           string         `json:"type"` // app / group / topic
	CreatedByAppID string         `json:"created_by_app_id,omitempty"`
	Parent         *conversation  `json:"parent,omitempty"`
	SourceMessage  *sourceMessage `json:"source_message,omitempty"`
}

type sourceMessage struct {
	ID  string `json:"id"`
	Seq int64  `json:"seq"`
}

type user struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // user / app
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
}

type message struct {
	ID               string      `json:"id"`
	Seq              int64       `json:"seq"`
	ReplyToMessageID string      `json:"reply_to_message_id,omitempty"`
	Body             messageBody `json:"body"`
	Summary          string      `json:"summary"`
	CreatedAt        string      `json:"created_at"`
}

// messageBody 覆盖平台消息类型；本渠道读 text/markdown，发 markdown/choice。
type messageBody struct {
	Type        string         `json:"type"` // text / markdown / choice / link / card / chart / image / file
	Content     string         `json:"content,omitempty"`
	ContentType string         `json:"content_type,omitempty"` // choice 的问题正文渲染方式
	Selection   string         `json:"selection,omitempty"`    // choice: single / multiple
	Options     []choiceOption `json:"options,omitempty"`
	Caption     string         `json:"caption,omitempty"`
	CaptionType string         `json:"caption_type,omitempty"`
}

type choiceOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type choiceResponse struct {
	ID        string   `json:"id"`
	OptionIDs []string `json:"option_ids"`
	CreatedAt string   `json:"created_at"`
}

// text 提取消息体中的可读文本（text/markdown 取 content，其余取 summary）。
func (p messageCreatedPayload) text() string {
	if p.Message.Body.Type == "text" || p.Message.Body.Type == "markdown" {
		if p.Message.Body.Content != "" {
			return p.Message.Body.Content
		}
	}
	if p.Message.Summary != "" {
		return p.Message.Summary
	}
	return p.Message.Body.Content
}

// ── RPC 载荷 ─────────────────────────────────────────────

// messageSendPayload 是 message.send 的载荷。
type messageSendPayload struct {
	Target           messageTarget `json:"target"`
	ReplyToMessageID string        `json:"reply_to_message_id,omitempty"`
	Message          messageBody   `json:"message"`
}

type messageTarget struct {
	Type           string `json:"type"`
	UserID         string `json:"user_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
}

// ackPayload 是 events.ack 的载荷。
type ackPayload struct {
	Cursor int64 `json:"cursor"`
}

// topicCreatePayload 是 conversation.topic.create 的载荷。
type topicCreatePayload struct {
	ConversationID  string `json:"conversation_id"`
	SourceMessageID string `json:"source_message_id"`
}
