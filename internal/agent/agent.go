// Package agent implements the autonomous agent loop: LLM + function-calling
// tool dispatch, running until the model produces a final answer (no tool
// calls). This is NOT a fixed workflow — the model decides which tools to
// call and in what order (ADR-009).
//
// 业务能力工具化（ADR-013）：需求分析不是固定输出格式，而是 analysis_submit
// 工具。寒暄/闲聊/追问由 Agent 自主判断直接自然语言回答；真实客户需求才提交
// 结构化分析。统一入口 Message()（ADR-014），不再区分 Analyze/Chat 两种模式。
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/history"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/assembler"
	"customer-demand-agent/internal/memory/shortterm"
	"customer-demand-agent/internal/memory/tools"
	"customer-demand-agent/internal/model"
)

// MaxIterations 防止工具调用死循环。
const MaxIterations = 15

// Trace 记录一次 Agent 运行的调用轨迹（供前端展示与回放）。
type Trace struct {
	SystemPrompt string // 本轮拼装的 system prompt（回放用，ADR-014）
	ToolCalls    []ToolCallRecord
}

// ToolCallRecord 是一次工具调用的记录。
type ToolCallRecord struct {
	Tool   string `json:"tool"`
	Params string `json:"params"`
	Result string `json:"result"`
}

// Event 是流式推送的一条 Agent 轨迹事件。覆盖思考/内容/工具调用/工具结果/完成/错误。
type Event struct {
	Type     string                 `json:"type"`               // round/reasoning/content/tool_call/tool_result/done/error
	Text     string                 `json:"text,omitempty"`     // reasoning/content 增量
	Round    int                    `json:"round,omitempty"`    // round 事件的轮次号
	Tool     string                 `json:"tool,omitempty"`     // tool_call/tool_result 的工具名
	Params   string                 `json:"params,omitempty"`   // tool_call 的参数（JSON）
	Result   string                 `json:"result,omitempty"`   // tool_result 的结果（JSON）
	Analysis *domain.AnalysisResult `json:"analysis,omitempty"` // done 事件：本轮提交过 analysis_submit 才有
	Content  string                 `json:"content,omitempty"`  // done 事件：最终自然语言答案（恒有）
	Error    string                 `json:"error,omitempty"`    // error 事件
	RunID    string                 `json:"run_id,omitempty"`   // F0：产生该事件的 Run（前端多轮过滤用）
}

// 事件类型常量。
const (
	EventRound      = "round"
	EventReasoning  = "reasoning"
	EventContent    = "content"
	EventToolCall   = "tool_call"
	EventToolResult = "tool_result"
	EventDone       = "done"
	EventError      = "error"
)

// emitEvent 安全推送（emit 为 nil 时忽略）。
func emitEvent(emit func(Event), e Event) {
	if emit != nil {
		emit(e)
	}
}

// Agent 是自主分析智能体。
type Agent struct {
	modelMgr   *model.Manager
	tools      *tools.Registry
	assembler  *assembler.Assembler
	sessions   *shortterm.SessionManager
	saveCP     func(tenantID, sessionID string, cp *domain.Checkpoint)                          // checkpoint 持久化回调（断点续传）
	loadCP     func(tenantID, sessionID string) ([]*domain.Checkpoint, error)                   // checkpoint 读取回调（断点续传）
	customerOf func(tenantID, sessionID string) string                                          // 会话关联客户名（跨会话客户上下文）
	histSearch func(tenantID, sessionID, query string, limit int) ([]history.MessageHit, error) // 历史检索（P1 history_search 工具）
}

// New 创建 Agent。
func New(modelMgr *model.Manager, tr *tools.Registry, asm *assembler.Assembler, sessions *shortterm.SessionManager) *Agent {
	return &Agent{modelMgr: modelMgr, tools: tr, assembler: asm, sessions: sessions}
}

// SetCheckpointSink 设置 checkpoint 持久化回调（断点续传）。tenantID 用于跨租户隔离。
func (a *Agent) SetCheckpointSink(fn func(tenantID, sessionID string, cp *domain.Checkpoint)) {
	a.saveCP = fn
}

// SetCheckpointSource 设置 checkpoint 读取回调（断点续传）。tenantID 用于跨租户隔离。
func (a *Agent) SetCheckpointSource(fn func(tenantID, sessionID string) ([]*domain.Checkpoint, error)) {
	a.loadCP = fn
}

// SetCustomerResolver 设置会话→客户名解析回调（跨会话客户上下文：非空时
// assembler 注入该客户的画像，同一客户多次沟通不再各说各话）。
func (a *Agent) SetCustomerResolver(fn func(tenantID, sessionID string) string) {
	a.customerOf = fn
}

// SetHistorySearcher 设置历史检索回调（history_search 工具的后端：
// Agent 可回溯原始对话轨迹，P1——checkpoint 注入摘要不够用时兜底）。
func (a *Agent) SetHistorySearcher(fn func(tenantID, sessionID, query string, limit int) ([]history.MessageHit, error)) {
	a.histSearch = fn
}

// restoreIfNeeded 若内存无 checkpoint 且历史有，则恢复短期记忆（断点续传）。
// 按 tenant 作用域读取，读不到他租户的 checkpoint。
func (a *Agent) restoreIfNeeded(tenantID, sessionID string) *shortterm.Manager {
	stm := a.sessions.Get(sessionID)
	if stm.HasHistory() || a.loadCP == nil {
		return stm
	}
	if cps, err := a.loadCP(tenantID, sessionID); err == nil && len(cps) > 0 {
		a.sessions.Restore(sessionID, cps)
		stm = a.sessions.Get(sessionID)
	}
	return stm
}

// turnState 收集本轮自主循环的产物。
type turnState struct {
	tenantID string                // 本轮租户（history_search 跨会话检索用）
	session  string                // 本轮会话（history_search 默认范围）
	analysis *AnalysisSubmission   // 最后一次 analysis_submit 的结果（nil = 本轮未提交分析）
	answered []domain.AnsweredInfo // 本轮记录的"追问已回答"（missing_answer）
}

// Message 是前台对话的统一入口（ADR-014）：处理任意输入（寒暄/需求/追问），
// Agent 自主决策回应方式。返回最终自然语言答案 + 可选的结构化分析 + 轨迹。
//
// 记忆系统覆盖所有前台轮次（ADR-015/Q1）：无论是否提交分析都建 checkpoint——
// 提交过建 initial/reanalysis，纯聊天/追问建轻量 followup，对话链完整。
func (a *Agent) Message(ctx context.Context, tenantID, sessionID, text string, emit func(Event)) (string, *domain.AnalysisResult, *Trace, error) {
	stm := a.restoreIfNeeded(tenantID, sessionID)
	// DetermineOp 仅作 checkpoint 类型提示与上下文拼装参考，不决定 Agent 行为。
	op := shortterm.DetermineOp(stm, text)

	sessCtx := stm.BuildContext(op)
	// 会话关联的客户（跨会话客户上下文：非空时 assembler 注入该客户画像）。
	if a.customerOf != nil {
		sessCtx.Customer = a.customerOf(tenantID, sessionID)
	}

	msgList := a.assembler.Assemble(op, sessCtx, text)
	trace := &Trace{}
	// 捕获本轮全部 system 消息（回放可见整个调用状态）：模板 + 动态注入
	// （产品目录在模板内；客户身份行/会话状态是独立 system 消息）。
	var sb strings.Builder
	for _, m := range msgList {
		if m.Role == domain.RoleSystem {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(m.Content)
		}
	}
	trace.SystemPrompt = sb.String()
	st := &turnState{tenantID: tenantID, session: sessionID}

	content, err := a.runStreaming(ctx, msgList, trace, emit, st)
	if err != nil {
		emitEvent(emit, Event{Type: EventError, Error: err.Error()})
		return "", nil, trace, err
	}

	// checkpoint：全轮次创建（ADR-015/Q1）。
	var cp *domain.Checkpoint
	if st.analysis != nil {
		if st.analysis.IsReanalysis || (op == domain.OpReanalysis && stm.HasHistory()) {
			cp = stm.CreateReanalysisCheckpoint(text, &st.analysis.AnalysisResult)
		} else {
			cp = stm.CreateInitialCheckpoint(text, &st.analysis.AnalysisResult)
		}
	} else {
		// 纯聊天/追问：轻量 followup（Question/Answer 照记，Analysis 空）。
		cp = stm.CreateFollowupCheckpoint(text, content)
	}
	if cp != nil && len(st.answered) > 0 {
		cp.Answered = st.answered // 本轮记录的追问答案（随 checkpoint 持久化，追问闭环）
	}
	if a.saveCP != nil && cp != nil {
		a.saveCP(tenantID, sessionID, cp) // 持久化（断点续传，按 tenant 隔离）
	}

	// done：content 恒有；analysis 仅本轮提交过才有。
	var analysis *domain.AnalysisResult
	if st.analysis != nil {
		analysis = &st.analysis.AnalysisResult
	}
	emitEvent(emit, Event{Type: EventDone, Content: content, Analysis: analysis})
	return content, analysis, trace, nil
}

// runStreaming 是自主循环的核心（流式）：每轮调 LLM 时推送思考/内容增量，
// 工具调用/结果作为事件推送，直到无工具调用得到最终答案。
// 不再强制 JSON 输出——输出形态由 Agent 自主决定（ADR-013）。
func (a *Agent) runStreaming(ctx context.Context, msgList []domain.Message, trace *Trace, emit func(Event), st *turnState) (string, error) {
	toolDefs := append(a.tools.Definitions(), analysisSubmitDef(), missingAnswerDef(), historySearchDef())

	for iter := 0; iter < MaxIterations; iter++ {
		emitEvent(emit, Event{Type: EventRound, Round: iter + 1})
		req := &llm.ChatRequest{
			Messages: msgList,
			Tools:    toolDefs,
		}
		cb := llm.DeltaCallbacks{
			OnReasoning: func(s string) { emitEvent(emit, Event{Type: EventReasoning, Text: s}) },
			OnContent:   func(s string) { emitEvent(emit, Event{Type: EventContent, Text: s}) },
		}
		resp, err := a.modelMgr.ChatStream(ctx, model.TaskAnalysis, req, cb)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败（第 %d 轮）: %w", iter+1, err)
		}

		msgList = append(msgList, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			if strings.TrimSpace(resp.Message.Content) == "" {
				return "", fmt.Errorf("第 %d 轮模型返回空内容（无 tool_calls 也无 content）", iter+1)
			}
			return resp.Message.Content, nil
		}

		for _, tc := range resp.Message.ToolCalls {
			emitEvent(emit, Event{Type: EventToolCall, Tool: tc.Function.Name, Params: tc.Function.Arguments})
			result := a.executeTool(tc, trace, st)
			emitEvent(emit, Event{Type: EventToolResult, Tool: tc.Function.Name, Result: truncateForTrace(result)})
			msgList = append(msgList, domain.Message{
				Role:       domain.RoleTool,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    result,
			})
		}
	}
	return "", fmt.Errorf("超过最大迭代次数 %d，可能存在工具调用死循环", MaxIterations)
}

// executeTool 执行单次工具调用，记录轨迹。
// analysis_submit / missing_answer 是 agent 层业务工具：拦截解析后挂到轮次
// 状态，不进 memory registry。
func (a *Agent) executeTool(tc domain.ToolCall, trace *Trace, st *turnState) string {
	var result string
	switch tc.Function.Name {
	case ToolAnalysisSubmit:
		sub, err := parseSubmission(json.RawMessage(tc.Function.Arguments))
		if err != nil {
			result = fmt.Sprintf(`{"error":"%s"}`, jsonEscape(err.Error()))
		} else {
			st.analysis = sub // 后一次提交覆盖前一次
			result = `{"received":true}`
		}
	case ToolMissingAnswer:
		ans, err := parseMissingAnswer(json.RawMessage(tc.Function.Arguments))
		if err != nil {
			result = fmt.Sprintf(`{"error":"%s"}`, jsonEscape(err.Error()))
		} else {
			st.answered = append(st.answered, ans...)
			result = `{"received":true}`
		}
	case ToolHistorySearch:
		result = a.execHistorySearch(tc.Function.Arguments, st)
	default:
		args := json.RawMessage(tc.Function.Arguments)
		r, err := a.tools.Execute(tc.Function.Name, args)
		if err != nil {
			result = fmt.Sprintf(`{"error":"%s"}`, jsonEscape(err.Error()))
		} else {
			result = r
		}
	}
	trace.ToolCalls = append(trace.ToolCalls, ToolCallRecord{
		Tool:   tc.Function.Name,
		Params: tc.Function.Arguments,
		Result: truncateForTrace(result),
	})
	return result
}

func truncateForTrace(s string) string {
	if len(s) > 500 {
		return s[:500] + "…"
	}
	return s
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return strings.Trim(string(b), "\"")
}
