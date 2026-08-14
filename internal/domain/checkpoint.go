package domain

import "time"

// CheckpointType 是短期记忆 checkpoint 的三种类型。
type CheckpointType string

const (
	// CheckpointInitial 首次分析后的状态快照
	CheckpointInitial CheckpointType = "initial"
	// CheckpointFollowup 追问后的状态
	CheckpointFollowup CheckpointType = "followup"
	// CheckpointReanalysis 换了新文档重新分析
	CheckpointReanalysis CheckpointType = "reanalysis"
)

// CheckpointOp 是 prompt 拼装时的操作类型（决定拼装哪种上下文）。
type CheckpointOp string

const (
	// OpInitial 首次分析（无历史 checkpoint）
	OpInitial CheckpointOp = "initial"
	// OpFollowup 追问（有 current checkpoint）
	OpFollowup CheckpointOp = "followup"
	// OpReanalysis 重新分析（已有 checkpoint，换文档）
	OpReanalysis CheckpointOp = "reanalysis"
)

// Checkpoint 是短期记忆的结构化状态快照。
//
// 设计参考 MiMo Code：维护结构化状态而非对话文本缓冲。
// 追问时只需读 current checkpoint，不重放整个对话。
type Checkpoint struct {
	ID        string          `json:"id"`
	Type      CheckpointType  `json:"type"`
	PrevID    string          `json:"prev_id,omitempty"`
	Document  string          `json:"document,omitempty"` // 原始客户文本（initial/reanalysis）
	Analysis  *AnalysisResult `json:"analysis,omitempty"` // 结构化分析结果
	Question  string          `json:"question,omitempty"` // 追问内容（followup）
	Answer    string          `json:"answer,omitempty"`   // 回答内容（followup）
	Answered  []AnsweredInfo  `json:"answered,omitempty"` // 本轮记录的"追问已回答"（missing_answer 工具）
	CreatedAt time.Time       `json:"created_at"`
}

// AnsweredInfo 是一条被回答的追问（missing_answer 工具产生，追问闭环）。
type AnsweredInfo struct {
	Item   string `json:"item"`   // 原 missing_info 追问
	Answer string `json:"answer"` // 得到的答案
}

// SessionContext 是为 prompt 拼装器提供的会话上下文（来自短期记忆）。
type SessionContext struct {
	Op              CheckpointOp
	Current         *Checkpoint    // 当前 checkpoint（追问/重分析时有）
	Previous        *Checkpoint    // 前一个 checkpoint（可选，用于上下文）
	LastAnalysis    *Checkpoint    // 链上最近带 Analysis 的 checkpoint（跨任意多轮纯聊天保持）
	DocumentSummary string         // 当前文档摘要
	Customer        string         // 会话关联的客户名（非空时 assembler 注入该客户画像）
	Answered        []AnsweredInfo // 链上已回答的追问（倒序聚合，追问闭环）
}
