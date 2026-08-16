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
	"customer-demand-agent/internal/leads"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/assembler"
	"customer-demand-agent/internal/memory/shortterm"
	"customer-demand-agent/internal/memory/tools"
	"customer-demand-agent/internal/model"
)

// DefaultMaxIterations 防止工具调用死循环（默认；可经 SetMaxIterations 覆盖）。
const DefaultMaxIterations = 15

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
	Question string                 `json:"question,omitempty"` // ask_user 事件：问题
	Options  []AskOption            `json:"options,omitempty"`  // ask_user 事件：选项
}

// AskOption 是 ask_user 的一个选项。
type AskOption struct {
	Label       string `json:"label"`                 // 按钮文案
	Value       string `json:"value,omitempty"`       // 发回的值（空=label）
	Description string `json:"description,omitempty"` // 补充说明
}

// 事件类型常量。
const (
	EventRound      = "round"
	EventAskUser    = "ask_user" // 向用户提问并给选项（轮次终止式）
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
	modelMgr     *model.Manager
	tools        *tools.Registry
	assembler    *assembler.Assembler
	sessions     *shortterm.SessionManager
	saveCP       func(sessionID string, cp *domain.Checkpoint)        // checkpoint 持久化回调（断点续传）
	loadCP       func(sessionID string) ([]*domain.Checkpoint, error) // checkpoint 读取回调（断点续传）
	customerOf   func(sessionID string) string
	bindCustomer func(sessionID, customer string) error                                 // 客户绑定 Agent 自主化（2026-08-15）
	maxIters     int                                                                    // 单轮最大工具循环（console-config 可配置；0=默认 15）
	histSearch   func(sessionID, query string, limit int) ([]history.MessageHit, error) // 历史检索（P1 history_search 工具）
	leads        *leads.Service                                                         // 商机平台工具（nil=未启用：不注册，模型无感知）
}

// New 创建 Agent。
func New(modelMgr *model.Manager, tr *tools.Registry, asm *assembler.Assembler, sessions *shortterm.SessionManager) *Agent {
	return &Agent{modelMgr: modelMgr, tools: tr, assembler: asm, sessions: sessions}
}

// SetCheckpointSink 设置 checkpoint 持久化回调（断点续传）。
func (a *Agent) SetCheckpointSink(fn func(sessionID string, cp *domain.Checkpoint)) {
	a.saveCP = fn
}

// SetCheckpointSource 设置 checkpoint 读取回调（断点续传）。
func (a *Agent) SetCheckpointSource(fn func(sessionID string) ([]*domain.Checkpoint, error)) {
	a.loadCP = fn
}

// SetMaxIterations 设置单轮最大工具循环数（≤0 恢复默认）。
func (a *Agent) SetMaxIterations(n int) { a.maxIters = n }

func (a *Agent) maxIter() int {
	if a.maxIters > 0 {
		return a.maxIters
	}
	return DefaultMaxIterations
}

// SetCustomerBinder 设置会话→客户绑定写回调（session_bind_customer 工具用）。
func (a *Agent) SetCustomerBinder(fn func(sessionID, customer string) error) {
	a.bindCustomer = fn
}

// SetCustomerResolver 设置会话→客户名解析回调（跨会话客户上下文：非空时
// assembler 注入该客户的画像，同一客户多次沟通不再各说各话）。
func (a *Agent) SetCustomerResolver(fn func(sessionID string) string) {
	a.customerOf = fn
}

// SetHistorySearcher 设置历史检索回调（history_search 工具的后端：
// Agent 可回溯原始对话轨迹，P1——checkpoint 注入摘要不够用时兜底）。
func (a *Agent) SetHistorySearcher(fn func(sessionID, query string, limit int) ([]history.MessageHit, error)) {
	a.histSearch = fn
}

// SetLeads 注入商机平台工具（lead-manager 接入）。nil（未启用）时工具定义
// 根本不注册——模型完全看不到，避免「调用了不存在的工具」（外部数据源启停
// 是常态，干净注册比「恒注册、运行时报未配置」更合适）；启用时同步告知
// assembler 注入数据源身份行（同客户身份行模式）。
func (a *Agent) SetLeads(s *leads.Service) {
	a.leads = s
	a.assembler.SetLeadsEnabled(s != nil)
}

// restoreIfNeeded 若内存无 checkpoint 且历史有，则恢复短期记忆（断点续传）。
func (a *Agent) restoreIfNeeded(sessionID string) *shortterm.Manager {
	stm := a.sessions.Get(sessionID)
	if stm.HasHistory() || a.loadCP == nil {
		return stm
	}
	if cps, err := a.loadCP(sessionID); err == nil && len(cps) > 0 {
		a.sessions.Restore(sessionID, cps)
		stm = a.sessions.Get(sessionID)
	}
	return stm
}

// turnState 收集本轮自主循环的产物。
type turnState struct {
	session         string                // 本轮会话（history_search 默认范围）
	pendingQuestion *Event                // ask_user 待答问题（轮次终止时随 done 发出）
	analysis        *AnalysisSubmission   // 最后一次 analysis_submit 的结果（nil = 本轮未提交分析）
	answered        []domain.AnsweredInfo // 本轮记录的"追问已回答"（missing_answer）
}

// Message 是前台对话的统一入口（ADR-014）：处理任意输入（寒暄/需求/追问），
// Agent 自主决策回应方式。返回最终自然语言答案 + 可选的结构化分析 + 轨迹。
//
// 记忆系统覆盖所有前台轮次（ADR-015/Q1）：无论是否提交分析都建 checkpoint——
// 提交过建 initial/reanalysis，纯聊天/追问建轻量 followup，对话链完整。
func (a *Agent) Message(ctx context.Context, sessionID, text string, emit func(Event)) (string, *domain.AnalysisResult, *Trace, error) {
	stm := a.restoreIfNeeded(sessionID)
	// DetermineOp 仅作 checkpoint 类型提示与上下文拼装参考，不决定 Agent 行为。
	op := shortterm.DetermineOp(stm, text)

	sessCtx := stm.BuildContext(op)
	// 会话关联的客户（跨会话客户上下文：非空时 assembler 注入该客户画像）。
	if a.customerOf != nil {
		sessCtx.Customer = a.customerOf(sessionID)
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
	st := &turnState{session: sessionID}

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
		a.saveCP(sessionID, cp) // 持久化（断点续传）
	}

	// done：content 恒有；analysis 仅本轮提交过才有。
	var analysis *domain.AnalysisResult
	if st.analysis != nil {
		analysis = &st.analysis.AnalysisResult
	}
	// F3 ask_user：待答问题在 done 前发出（前端渲染按钮组，done 收尾）
	if st.pendingQuestion != nil {
		emitEvent(emit, *st.pendingQuestion)
	}
	emitEvent(emit, Event{Type: EventDone, Content: content, Analysis: analysis})
	return content, analysis, trace, nil
}

// runStreaming 是自主循环的核心（流式）：每轮调 LLM 时推送思考/内容增量，
// 工具调用/结果作为事件推送，直到无工具调用得到最终答案。
// 不再强制 JSON 输出——输出形态由 Agent 自主决定（ADR-013）。
func (a *Agent) runStreaming(ctx context.Context, msgList []domain.Message, trace *Trace, emit func(Event), st *turnState) (string, error) {
	toolDefs := append(a.tools.Definitions(), analysisSubmitDef(), missingAnswerDef(), historySearchDef(), askUserDef(), bindCustomerDef())
	if a.leads != nil {
		toolDefs = append(toolDefs, a.leads.Definitions()...) // 启用才注册（nil=模型无感知）
	}

	for iter := 0; iter < a.maxIter(); iter++ {
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
			result := a.executeTool(ctx, tc, trace, st)
			emitEvent(emit, Event{Type: EventToolResult, Tool: tc.Function.Name, Result: truncateForTrace(result)})
			msgList = append(msgList, domain.Message{
				Role:       domain.RoleTool,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    result,
			})
		}
	}
	return "", fmt.Errorf("超过最大迭代次数 %d，可能存在工具调用死循环", a.maxIter())
}

// executeTool 执行单次工具调用，记录轨迹。
// analysis_submit / missing_answer 是 agent 层业务工具：拦截解析后挂到轮次
// 状态，不进 memory registry。leads_* 分发到商机平台服务（ctx 传播：Run
// cancel 即断外部 HTTP 调用）。
func (a *Agent) executeTool(ctx context.Context, tc domain.ToolCall, trace *Trace, st *turnState) string {
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
	case ToolAskUser:
		result = a.execAskUser(tc.Function.Arguments, st)
	case ToolBindCustomer:
		result = a.execBindCustomer(tc.Function.Arguments, st.session)
	default:
		args := json.RawMessage(tc.Function.Arguments)
		var r string
		var err error
		if a.leads != nil && a.leads.Handles(tc.Function.Name) {
			r, err = a.leads.Execute(ctx, tc.Function.Name, args)
		} else {
			r, err = a.tools.Execute(tc.Function.Name, args)
		}
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

// TemplateRaw 返回外置 prompt 模板原文（console C3 转发）。
func (a *Agent) TemplateRaw() string { return a.assembler.TemplateRaw() }
