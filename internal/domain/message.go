package domain

// Role 是 LLM 对话消息的角色。
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message 是与 LLM 交互的对话消息（OpenAI 兼容格式）。
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // assistant 发起的工具调用
	ToolCallID string     `json:"tool_call_id,omitempty"` // role=tool 时关联的调用 ID
	Name       string     `json:"name,omitempty"`         // 工具名（role=tool）
}

// ToolCall 是 LLM 发起的一次 function calling 调用。
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // 固定 "function"
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction 是工具调用的函数信息。
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串形式的参数
}

// Tool 是暴露给 LLM 的 function calling 工具定义。
type Tool struct {
	Type     string       `json:"type"` // 固定 "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction 是工具的函数声明（名称/描述/参数 schema）。
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"` // JSON Schema
}
