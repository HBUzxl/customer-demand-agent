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
	"time"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/history"
	"customer-demand-agent/internal/leads"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/assembler"
	"customer-demand-agent/internal/memory/shortterm"
	"customer-demand-agent/internal/memory/tools"
	"customer-demand-agent/internal/model"
)

const (
	// DefaultMaxIterations 防止工具调用死循环（默认；可经 SetMaxIterations 覆盖）。
	DefaultMaxIterations = 15
	// DefaultTurnTimeout 是单次用户消息的总执行上限。模型单次调用仍受各模型
	// timeout 约束；这里防止多轮 fallback/工具组合把后台 Run 永久占住。
	DefaultTurnTimeout = 5 * time.Minute
	// 工具预算是模型输出之外的确定性安全边界：阻止一次响应扇出大量工具，
	// 或用超大 JSON 参数消耗内存/污染轨迹。
	maxToolCallsPerRound = 8
	maxToolCallsPerTurn  = 24
	maxToolArgumentBytes = 32 << 10
	defaultToolTimeout   = 20 * time.Second
	toolMemoryGetMany    = "memory_get_many"
	maxLocatorCalls      = 3 // search/recall/list：定位候选，不允许无限改写关键词
	maxDetailReadCalls   = 5 // get/get_many：主页 + 少量关键子文档后必须收敛
)

// loopRepeatThreshold 同一「工具+参数」签名重复次数达到该值即判定为死循环。
// 检测到后在触底前注入纠正指令，给模型一次收敛机会，而非空转耗光 maxIter。
const loopRepeatThreshold = 3

// loopRecoveryPrompt 死循环纠正指令（RoleUser 注入，协议无关，模型响应最强）。
// 检测到死循环时追加为一条用户消息，模型通常据此收敛为最终答案。
const loopRecoveryPrompt = `【系统】检测到你的工具调用陷入死循环：同一工具/同一参数反复调用，或分析提交连续被系统拒绝。为节省用户等待时间，本轮到此为止——请立即停止调用任何工具，仅用自然语言直接给出最终回答。若已掌握的信息足够，直接给出结论；若信息不足，明确列出还需要用户补充哪些信息即可。不要再调用任何工具。`

// analysisFinalPrompt 在结构化分析通过门禁后进入单独的最终回答阶段。此阶段
// 不再提供工具，避免模型重复提交或继续改写已验收结论；同时禁止引入新事实。
const analysisFinalPrompt = `【系统】结构化分析已通过校验并保存。现在只输出面向用户的简洁最终回答：概括需求判断、产品匹配及能力边界，并列出最关键的待确认项或下一步。不得再调用任何工具，不得新增结构化分析中没有依据的事实或产品能力。`

const emptyResponseRetryPrompt = `【系统】上一轮模型返回了空响应。请继续本轮任务并立即给出有效结果：若结构化分析已通过，只输出最终自然语言总结；否则基于已有证据完成必要的 analysis_submit 或明确说明仍需确认的信息。禁止返回空内容。`

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
	Cached bool   `json:"cached,omitempty"`
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
	Cached   bool                   `json:"cached,omitempty"`   // tool_result 是否命中本轮只读缓存
	Question string                 `json:"question,omitempty"` // ask_user 事件：问题
	Options  []AskOption            `json:"options,omitempty"`  // ask_user 事件：选项
}

// AskOption 是 ask_user 的一个选项。
type AskOption struct {
	Label       string `json:"label"`                 // 按钮文案
	Value       string `json:"value,omitempty"`       // 发回的值（空=label）
	Description string `json:"description,omitempty"` // 补充说明
	Input       string `json:"input,omitempty"`       // 选此项需用户补充的信息提示（如"请输入客户名称"）；前端先弹输入再回传（label：输入）
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
	modelMgr      *model.Manager
	tools         *tools.Registry
	assembler     *assembler.Assembler
	sessions      *shortterm.SessionManager
	saveCP        func(sessionID, branch string, cp *domain.Checkpoint)        // checkpoint 持久化回调（断点续传；branch=当前分支）
	loadCP        func(sessionID, branch string) ([]*domain.Checkpoint, error) // checkpoint 读取回调（断点续传）
	customerOf    func(sessionID string) string
	bindCustomer  func(sessionID, customer string) error                                                            // 客户绑定 Agent 自主化（2026-08-15）
	maxIters      int                                                                                               // 单轮最大工具循环（console-config 可配置；0=默认 15）
	histSearch    func(scope *domain.TenantScope, sessionID, query string, limit int) ([]history.MessageHit, error) // 历史检索（P1 history_search 工具）
	leads         *leads.Service                                                                                    // 商机平台工具（nil=未启用：不注册，模型无感知）
	productExists func(name string) bool                                                                            // P0-02 证据门禁：产品存在性校验（runtime 用本租户 CompositeStore）
}

// New 创建 Agent。
func New(modelMgr *model.Manager, tr *tools.Registry, asm *assembler.Assembler, sessions *shortterm.SessionManager) *Agent {
	return &Agent{modelMgr: modelMgr, tools: tr, assembler: asm, sessions: sessions}
}

// SetCheckpointSink 设置 checkpoint 持久化回调（断点续传）。branch 区分分支
// （P0-04）：编辑重发的分支 checkpoint 写回同一分支，前缀共享 main。
func (a *Agent) SetCheckpointSink(fn func(sessionID, branch string, cp *domain.Checkpoint)) {
	a.saveCP = fn
}

// SetCheckpointSource 设置 checkpoint 读取回调（断点续传）。branch 非空时
// 读分支记忆 = 共享前缀（main 前缀）+ 分支自身 checkpoint（P0-04）。
func (a *Agent) SetCheckpointSource(fn func(sessionID, branch string) ([]*domain.Checkpoint, error)) {
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
// scope 携带当前登录作用域：P0-03 修复——history_search 必须遵守用户级
// 可见性（非 admin 只检索自己 owner 的会话），不得绕过。
func (a *Agent) SetHistorySearcher(fn func(scope *domain.TenantScope, sessionID, query string, limit int) ([]history.MessageHit, error)) {
	a.histSearch = fn
}

// SetProductResolver 设置产品存在性校验回调（P0-02 证据门禁：analysis_submit
// 推荐的产品必须真实存在于产品库，杜绝编造产品名。runtime 用本租户
// CompositeStore.GetProduct 注入）。未设置（legacy 测试路径）时跳过存在性校验。
func (a *Agent) SetProductResolver(fn func(name string) bool) {
	a.productExists = fn
}

// SetLeads 注入商机平台工具（lead-manager 接入）。nil（未启用）时工具定义
// 根本不注册——模型完全看不到，避免「调用了不存在的工具」（外部数据源启停
// 是常态，干净注册比「恒注册、运行时报未配置」更合适）；启用时同步告知
// assembler 注入数据源身份行（同客户身份行模式）。
func (a *Agent) SetLeads(s *leads.Service) {
	a.leads = s
	a.assembler.SetLeadsEnabled(s != nil)
}

// memKey 短期记忆键：main/空分支直接用 sessionID（兼容既有缓存），分支用
// 复合键隔离——编辑重发的分支有独立的记忆实例，兄弟分支互不串扰（P0-04）。
func memKey(sessionID, branch string) string {
	if branch == "" || branch == "main" {
		return sessionID
	}
	return sessionID + "\x00" + branch
}

// restoreIfNeeded 若内存无 checkpoint 且历史有，则恢复短期记忆（断点续传）。
// 分支场景：loadCP 返回共享前缀+分支自身，恢复到该分支独立的记忆实例。
func (a *Agent) restoreIfNeeded(sessionID, branch string) *shortterm.Manager {
	key := memKey(sessionID, branch)
	stm := a.sessions.Get(key)
	if stm.HasHistory() || a.loadCP == nil {
		return stm
	}
	if cps, err := a.loadCP(sessionID, branch); err == nil && len(cps) > 0 {
		a.sessions.Restore(key, cps)
		stm = a.sessions.Get(key)
	}
	return stm
}

// turnState 收集本轮自主循环的产物。
type turnState struct {
	session         string                // 本轮会话（history_search 默认范围）
	scope           *domain.TenantScope   // 本轮登录作用域（P0-03：history_search 遵守用户级可见性）
	userText        string                // 本轮原始输入（可行性与不确定性一致性门禁）
	pendingQuestion *Event                // ask_user 待答问题（轮次终止时随 done 发出）
	analysis        *AnalysisSubmission   // 最后一次 analysis_submit 的结果（nil = 本轮未提交分析）
	answered        []domain.AnsweredInfo // 本轮记录的"追问已回答"（missing_answer）
	readProducts    map[string]bool       // P0-02 证据门禁：本轮 memory_get 读过的产品（推荐前必须读详情）
}

// Message 是前台对话的统一入口（ADR-014）：处理任意输入（寒暄/需求/追问），
// Agent 自主决策回应方式。返回最终自然语言答案 + 可选的结构化分析 + 轨迹。
//
// scope 提供当前登录作用域（多租户）：assembler 按 scope.UserID 解析本租户
// 使用者画像（替代全局 default_user）；工具读写经本租户 memory registry
// （运行时闭包保证租户边界）。nil scope 表示 legacy/dingtalk 单租户路径。
//
// 记忆系统覆盖所有前台轮次（ADR-015/Q1）：无论是否提交分析都建 checkpoint——
// 提交过建 initial/reanalysis，纯聊天/追问建轻量 followup，对话链完整。
func (a *Agent) Message(ctx context.Context, scope *domain.TenantScope, sessionID, branch, text string, emit func(Event)) (string, *domain.AnalysisResult, *Trace, error) {
	stm := a.restoreIfNeeded(sessionID, branch)
	// DetermineOp 仅作 checkpoint 类型提示与上下文拼装参考，不决定 Agent 行为。
	op := shortterm.DetermineOp(stm, text)

	sessCtx := stm.BuildContext(op)
	// 会话关联的客户（跨会话客户上下文：非空时 assembler 注入该客户画像）。
	if a.customerOf != nil {
		sessCtx.Customer = a.customerOf(sessionID)
	}

	msgList := a.assembler.Assemble(scope, op, sessCtx, text)
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
	st := &turnState{session: sessionID, scope: scope, userText: text, readProducts: map[string]bool{}}

	turnCtx, cancel := context.WithTimeout(ctx, DefaultTurnTimeout)
	defer cancel()
	content, err := a.runStreaming(turnCtx, msgList, trace, emit, st)
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
		a.saveCP(sessionID, branch, cp) // 持久化到当前分支（断点续传；P0-04）
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

// runStreaming 是自主循环的核心（流式）：每轮调 LLM 时只推送安全的阶段状态，
// 不向客户端暴露模型 reasoning_content 原始思维链；最终正文仍流式推送。
// 工具调用/结果作为事件推送，直到无工具调用得到最终答案。
// 不再强制 JSON 输出——输出形态由 Agent 自主决定（ADR-013）。
//
// 死循环防御（2026-08-19 全面加固）：
//   - ask_user 是轮次终止式：调用后立即结束本轮，不再继续调 LLM（原实现在
//     调用后仍会再跑一轮，模型若继续调工具即空耗迭代）。
//   - 重复调用检测：同一「工具+compact 参数」签名重复 ≥3 次，或 analysis_submit
//     连续提交 ≥3 次始终未成功，判定为死循环 → 注入纠正指令给一次收敛机会；
//     纠正后仍循环 → 优雅降级报错（不再裸报「超过最大迭代次数」）。
func (a *Agent) runStreaming(ctx context.Context, msgList []domain.Message, trace *Trace, emit func(Event), st *turnState) (string, error) {
	toolDefs := append(a.tools.DefinitionsFor(st.scope), memoryGetManyDef(), analysisSubmitDef(), missingAnswerDef(), historySearchDef(), askUserDef(), bindCustomerDef())
	if a.leads != nil {
		toolDefs = append(toolDefs, a.leads.Definitions()...) // 启用才注册（nil=模型无感知）
	}

	callCounts := map[string]int{}   // 「工具\x00compact参数」→ 已调用次数
	analysisSubmits := 0             // 本轮 analysis_submit 累计调用（含被拒）
	warned := false                  // 是否已注入死循环纠正指令
	warnReason := ""                 // 首次命中死循环的原因（纠正后仍循环时报错用）
	totalToolCalls := 0              // 本轮累计工具数（确定性预算）
	readCache := map[string]string{} // 相同只读工具+参数在一轮内复用结果
	forceFinal := ""                 // "analysis" / "recovery"：下一次调用不再提供工具
	locatorCalls, detailReadCalls := 0, 0
	locatorBudgetNoted, detailBudgetNoted := false, false
	emptyResponseRetried := false

	// forceFinal 允许在最后一个工具轮后额外做一次“无工具最终回答”，不让收敛
	// 阶段因为恰好触及 maxIter 而丢失已通过校验的分析。
	for iter := 0; iter < a.maxIter() || forceFinal != ""; iter++ {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("Agent 执行已取消或超时: %w", err)
		}
		emitEvent(emit, Event{Type: EventRound, Round: iter + 1})
		emitEvent(emit, Event{Type: EventReasoning, Text: safeReasoningStatus(iter, forceFinal)})
		activeTools := toolDefs
		if forceFinal != "" {
			activeTools = nil
		} else {
			if locatorCalls >= maxLocatorCalls {
				activeTools = withoutTools(activeTools, "memory_search", "memory_recall", "memory_list")
				if !locatorBudgetNoted {
					locatorBudgetNoted = true
					msgList = append(msgList, domain.Message{Role: domain.RoleUser, Content: "【查证 Harness】候选定位预算已用完。不要再改写关键词搜索；请使用已有候选读取必要详情，然后提交分析。未找到的事实标注知识库未证实。"})
				}
			}
			if detailReadCalls >= maxDetailReadCalls {
				activeTools = withoutTools(activeTools, "memory_get", toolMemoryGetMany)
				if !detailBudgetNoted {
					detailBudgetNoted = true
					msgList = append(msgList, domain.Message{Role: domain.RoleUser, Content: "【查证 Harness】详情取证预算已用完。请基于已读取证据立即调用 analysis_submit；无法证明的商业、交付或竞品结论必须标注需确认。"})
				}
			}
		}
		var roundContent strings.Builder
		streamFinalContent := forceFinal != ""
		req := &llm.ChatRequest{
			Messages: msgList,
			Tools:    activeTools,
		}
		cb := llm.DeltaCallbacks{
			// 原始 reasoning_content 可能包含思维链、提示词片段和未经核验草稿，
			// 仅消费以维持流式连接，不持久化、不转发。
			OnReasoning: func(string) {},
			OnContent: func(s string) {
				roundContent.WriteString(s)
				// 只有明确的无工具收敛阶段可以边生成边展示；普通轮先缓冲，
				// 避免“草稿文本 + 随后工具调用”被用户误认作最终结论。
				if streamFinalContent {
					emitEvent(emit, Event{Type: EventContent, Text: s})
				}
			},
		}
		resp, err := a.modelMgr.ChatStream(ctx, model.TaskAnalysis, req, cb)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败（第 %d 轮）: %w", iter+1, err)
		}

		msgList = append(msgList, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			if strings.TrimSpace(resp.Message.Content) == "" {
				if !emptyResponseRetried {
					emptyResponseRetried = true
					msgList = append(msgList, domain.Message{Role: domain.RoleUser, Content: emptyResponseRetryPrompt})
					continue
				}
				return "", fmt.Errorf("模型连续两轮返回空内容（无 tool_calls 也无 content），已停止")
			}
			if !streamFinalContent {
				emitEvent(emit, Event{Type: EventContent, Text: roundContent.String()})
			}
			return resp.Message.Content, nil
		}

		// 无工具收敛阶段仍返回 tool_calls：不执行模型越界请求。
		if forceFinal == "recovery" || warned {
			return "", loopStuckError(warnReason)
		}
		if forceFinal == "analysis" {
			return "", fmt.Errorf("分析已提交，但模型在最终回答阶段仍请求调用工具，已安全停止")
		}
		if err := validateToolBatch(resp.Message.ToolCalls, totalToolCalls); err != nil {
			return "", err
		}
		totalToolCalls += len(resp.Message.ToolCalls)

		for _, tc := range resp.Message.ToolCalls {
			if err := ctx.Err(); err != nil {
				return "", fmt.Errorf("Agent 执行已取消或超时: %w", err)
			}
			emitEvent(emit, Event{Type: EventToolCall, Tool: tc.Function.Name, Params: tc.Function.Arguments})
			sig := tc.Function.Name + "\x00" + compactJSON(tc.Function.Arguments)
			var result string
			cached := false
			budgetBlocked := false
			switch tc.Function.Name {
			case "memory_search", "memory_recall", "memory_list":
				budgetBlocked = locatorCalls >= maxLocatorCalls
			case "memory_get", toolMemoryGetMany:
				budgetBlocked = detailReadCalls >= maxDetailReadCalls
			}
			if budgetBlocked {
				result = `{"error":"查证 Harness 预算已用完；请基于已有证据收敛结论，未证实项标注需确认"}`
				trace.ToolCalls = append(trace.ToolCalls, ToolCallRecord{Tool: tc.Function.Name, Params: tc.Function.Arguments, Result: result})
			} else {
				result, cached = readCache[sig]
				if !cached || !cacheableReadTool(tc.Function.Name) {
					toolCtx, cancel := context.WithTimeout(ctx, defaultToolTimeout)
					result = a.executeTool(toolCtx, tc, trace, st)
					cancel()
					cached = false
					if cacheableReadTool(tc.Function.Name) && !toolResultHasError(result) {
						readCache[sig] = result
					}
				} else {
					trace.ToolCalls = append(trace.ToolCalls, ToolCallRecord{
						Tool: tc.Function.Name, Params: tc.Function.Arguments,
						Result: truncateForTrace(result), Cached: true,
					})
				}
			}
			emitEvent(emit, Event{Type: EventToolResult, Tool: tc.Function.Name, Result: truncateForTrace(result), Cached: cached})
			msgList = append(msgList, domain.Message{
				Role:       domain.RoleTool,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    result,
			})
			callCounts[sig]++
			if !budgetBlocked {
				switch tc.Function.Name {
				case "memory_search", "memory_recall", "memory_list":
					locatorCalls++
				case "memory_get", toolMemoryGetMany:
					detailReadCalls++
				}
			}
			if tc.Function.Name == ToolAnalysisSubmit {
				analysisSubmits++
			}
			// ask_user 是终止式工具：同一个 assistant 响应里即使还夹带了其他
			// tool_calls，也不得在提问后继续执行（尤其是写/删类副作用）。
			if st.pendingQuestion != nil {
				break
			}
		}

		// ask_user 轮次终止式：调用后本轮即结束（等待用户选择），不再空转。
		if st.pendingQuestion != nil {
			return resp.Message.Content, nil
		}

		// 结构化分析一旦通过门禁便冻结，下一轮不再提供任何工具，只允许把
		// 已验收结果整理成自然语言，避免重复提交、继续写记忆或引入新主张。
		if st.analysis != nil {
			forceFinal = "analysis"
			msgList = append(msgList, domain.Message{Role: domain.RoleUser, Content: analysisFinalPrompt})
			continue
		}

		// 死循环检测：命中 → 首次注入纠正指令并清空计数给一次收敛机会；
		// 纠正后仍命中 → 优雅降级失败。
		if reason := detectLoop(callCounts, analysisSubmits, st); reason != "" {
			if !warned {
				warned = true
				warnReason = reason
				callCounts = map[string]int{}
				analysisSubmits = 0
				forceFinal = "recovery"
				msgList = append(msgList, domain.Message{Role: domain.RoleUser, Content: loopRecoveryPrompt})
				continue
			}
			return "", loopStuckError(warnReason)
		}
	}
	return "", loopExhaustedError(callCounts, analysisSubmits, a.maxIter())
}

func withoutTools(defs []domain.Tool, names ...string) []domain.Tool {
	deny := make(map[string]struct{}, len(names))
	for _, name := range names {
		deny[name] = struct{}{}
	}
	out := make([]domain.Tool, 0, len(defs))
	for _, def := range defs {
		if _, blocked := deny[def.Function.Name]; !blocked {
			out = append(out, def)
		}
	}
	return out
}

// safeReasoningStatus 只公开阶段状态，不公开模型原始思维链。
func safeReasoningStatus(iter int, forceFinal string) string {
	switch forceFinal {
	case "analysis":
		return "正在整理已校验的分析结论…"
	case "recovery":
		return "正在收敛执行结果…"
	case "":
		if iter == 0 {
			return "正在理解需求并规划查证步骤…"
		}
	}
	return "正在结合工具结果核对结论…"
}

// validateToolBatch 在执行任何副作用前验证整批调用，避免部分执行后才发现
// 超预算或协议损坏。参数 JSON 的语义错误仍回填给模型自纠正。
func validateToolBatch(calls []domain.ToolCall, total int) error {
	if len(calls) > maxToolCallsPerRound {
		return fmt.Errorf("模型单轮请求了 %d 个工具，超过上限 %d，已拒绝整批执行", len(calls), maxToolCallsPerRound)
	}
	if total+len(calls) > maxToolCallsPerTurn {
		return fmt.Errorf("本轮累计工具调用将达到 %d，超过上限 %d，已停止", total+len(calls), maxToolCallsPerTurn)
	}
	ids := make(map[string]struct{}, len(calls))
	for _, tc := range calls {
		if strings.TrimSpace(tc.ID) == "" || strings.TrimSpace(tc.Function.Name) == "" {
			return fmt.Errorf("模型返回了缺少 id 或 name 的无效工具调用")
		}
		if _, exists := ids[tc.ID]; exists {
			return fmt.Errorf("模型返回了重复 tool_call id: %s", tc.ID)
		}
		ids[tc.ID] = struct{}{}
		if len(tc.Function.Arguments) > maxToolArgumentBytes {
			return fmt.Errorf("工具 %s 参数大小 %d 字节，超过上限 %d", tc.Function.Name, len(tc.Function.Arguments), maxToolArgumentBytes)
		}
	}
	return nil
}

func cacheableReadTool(name string) bool {
	switch name {
	case "memory_search", "memory_get", toolMemoryGetMany, "memory_recall", "memory_list", ToolHistorySearch:
		return true
	default:
		return false
	}
}

func toolResultHasError(result string) bool {
	var v map[string]any
	return json.Unmarshal([]byte(result), &v) == nil && v["error"] != nil
}

// detectLoop 判定本轮工具调用是否陷入死循环。返回拒绝原因（空=正常）。
//  1. 同一「工具+参数」签名重复 ≥ loopRepeatThreshold 次（同一工具同一参数反复调用）；
//  2. analysis_submit 累计提交 ≥ loopRepeatThreshold 次且始终未成功
//     （st.analysis==nil，通常是 P0-02 证据门禁拒绝后模型反复重提）。
func detectLoop(callCounts map[string]int, analysisSubmits int, st *turnState) string {
	for sig, n := range callCounts {
		if n >= loopRepeatThreshold {
			tool := sig
			if i := strings.IndexByte(sig, '\x00'); i >= 0 {
				tool = sig[:i]
			}
			return fmt.Sprintf("重复调用 %s ×%d", tool, n)
		}
	}
	if analysisSubmits >= loopRepeatThreshold && st.analysis == nil {
		return fmt.Sprintf("analysis_submit 连续提交 %d 次被拒（证据门禁未通过）", analysisSubmits)
	}
	return ""
}

// compactJSON 把工具参数规范化为稳定签名（去除空白/键序差异），无法解析时退化为原始串。
func compactJSON(s string) string {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return strings.TrimSpace(s)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return strings.TrimSpace(s)
	}
	return string(b)
}

// canonicalProductFromResult 从 memory_get 结果首行解析规范产品名（# 标题（product｜状态）），
// 用于把「模型用别名读取的产品」以规范名入账 readProducts。解析失败返回空串。
func canonicalProductFromResult(r string) string {
	line := r
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
	if i := strings.LastIndex(line, "（product｜"); i >= 0 {
		line = strings.TrimSpace(line[:i])
	}
	if line == "" || line[0] == '{' || line[0] == '"' || strings.HasPrefix(line, "不存在") {
		return ""
	}
	return line
}

// loopStuckError 纠正指令已给、模型仍循环时返回的优雅降级错误（可操作）。
func loopStuckError(reason string) error {
	return fmt.Errorf("Agent 工具调用仍陷入死循环（%s），已停止。请换一种说法重试，或把客户原文拆小后重发。", reason)
}

// loopExhaustedError 迭代触底时返回的错误：附上重复调用摘要，让用户知道卡在哪。
func loopExhaustedError(callCounts map[string]int, analysisSubmits int, max int) error {
	return fmt.Errorf("超过最大迭代次数 %d，可能存在工具调用死循环（%s）。请换一种说法重试。", max, loopSummary(callCounts, analysisSubmits))
}

// loopSummary 构造死循环摘要（最高频重复调用/分析提交被拒）。
func loopSummary(callCounts map[string]int, analysisSubmits int) string {
	tool, n := "", 0
	for sig, c := range callCounts {
		if c > n {
			n = c
			tool = sig
			if i := strings.IndexByte(sig, '\x00'); i >= 0 {
				tool = sig[:i]
			}
		}
	}
	if n >= 2 {
		return fmt.Sprintf("重复调用 %s ×%d", tool, n)
	}
	if analysisSubmits > 0 {
		return fmt.Sprintf("analysis_submit 提交 %d 次未成功", analysisSubmits)
	}
	return "工具调用未收敛"
}

// executeTool 执行单次工具调用，记录轨迹。
// analysis_submit / missing_answer 是 agent 层业务工具：拦截解析后挂到轮次
// 状态，不进 memory registry。leads_* 分发到商机平台服务（ctx 传播：Run
// cancel 即断外部 HTTP 调用）。
func (a *Agent) executeTool(ctx context.Context, tc domain.ToolCall, trace *Trace, st *turnState) string {
	var result string
	switch tc.Function.Name {
	case ToolAnalysisSubmit:
		if st.analysis != nil {
			result = `{"error":"本轮分析已经提交成功，不允许重复覆盖","received":false}`
			break
		}
		sub, err := parseSubmission(json.RawMessage(tc.Function.Arguments))
		if err != nil {
			result = fmt.Sprintf(`{"error":"%s"}`, jsonEscape(err.Error()))
		} else if reason := a.validateSubmission(sub, st); reason != "" {
			// P0-02 证据门禁：拒绝并回填模型修正（不挂 analysis，模型应补证据后重提）
			result = fmt.Sprintf(`{"error":"%s","received":false,"hint":"修正后重新调用 analysis_submit"}`, jsonEscape(reason))
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
	case toolMemoryGetMany:
		result = a.execMemoryGetMany(tc.Function.Arguments, st)
	case "memory_get":
		// P0-02 证据门禁：记录本轮 memory_get 读过的产品（推荐产品前必须读详情核验能力）。
		// 同时记录结果首行的规范产品名——模型可能用别名/简称发起读取，analysis_submit
		// 里却用全称；规范名入账后证据门禁才匹配得上（2026-08-19 防门禁拼写偏差死循环）。
		var g struct {
			Type  string `json:"type"`
			Title string `json:"title"`
		}
		_ = json.Unmarshal(json.RawMessage(tc.Function.Arguments), &g)
		var r string
		var err error
		if a.leads != nil && a.leads.Handles(tc.Function.Name) {
			r, err = a.leads.Execute(ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
		} else {
			r, err = a.tools.ExecuteFor(st.scope, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
		}
		if err != nil {
			result = fmt.Sprintf(`{"error":"%s"}`, jsonEscape(err.Error()))
		} else {
			result = r
			if g.Type == "product" {
				if g.Title != "" {
					st.readProducts[g.Title] = true
				}
				if canon := canonicalProductFromResult(r); canon != "" {
					st.readProducts[canon] = true
				}
			}
		}
	default:
		args := json.RawMessage(tc.Function.Arguments)
		var r string
		var err error
		if a.leads != nil && a.leads.Handles(tc.Function.Name) {
			r, err = a.leads.Execute(ctx, tc.Function.Name, args)
		} else {
			r, err = a.tools.ExecuteFor(st.scope, tc.Function.Name, args)
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

func memoryGetManyDef() domain.Tool {
	return domain.Tool{
		Type: "function",
		Function: domain.ToolFunction{
			Name: toolMemoryGetMany,
			Description: "批量读取 1-8 条记忆完整内容。多产品方案必须优先用本工具一次取证，替代逐个 memory_get，" +
				"每个产品读取成功后同样满足 analysis_submit 的证据门禁。title 应来自产品目录或 memory_search 结果。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"items": map[string]any{
						"type": "array", "minItems": 1, "maxItems": 8,
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"type":        map[string]any{"type": "string", "enum": []string{"product", "threat", "compliance", "industry", "customer", "user"}},
								"title":       map[string]any{"type": "string"},
								"section":     map[string]any{"type": "string"},
								"body_offset": map[string]any{"type": "integer"},
							},
							"required": []string{"type", "title"},
						},
					},
				},
				"required": []string{"items"},
			},
		},
	}
}

func (a *Agent) execMemoryGetMany(raw string, st *turnState) string {
	type item struct {
		Type       string `json:"type"`
		Title      string `json:"title"`
		Section    string `json:"section,omitempty"`
		BodyOffset int    `json:"body_offset,omitempty"`
	}
	var args struct {
		Items []item `json:"items"`
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return fmt.Sprintf(`{"error":"参数解析: %s"}`, jsonEscape(err.Error()))
	}
	if len(args.Items) == 0 || len(args.Items) > 8 {
		return `{"error":"items 数量必须在 1..8 之间"}`
	}
	type batchResult struct {
		Type    string `json:"type"`
		Title   string `json:"title"`
		Content string `json:"content,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	out := make([]batchResult, 0, len(args.Items))
	for _, in := range args.Items {
		in.Type = strings.TrimSpace(in.Type)
		in.Title = strings.TrimSpace(in.Title)
		if in.Type == "" || in.Title == "" {
			out = append(out, batchResult{Type: in.Type, Title: in.Title, Error: "type/title 不能为空"})
			continue
		}
		one, _ := json.Marshal(in)
		content, err := a.tools.ExecuteFor(st.scope, "memory_get", one)
		if err != nil {
			out = append(out, batchResult{Type: in.Type, Title: in.Title, Error: err.Error()})
			continue
		}
		out = append(out, batchResult{Type: in.Type, Title: in.Title, Content: content})
		if in.Type == "product" {
			st.readProducts[in.Title] = true
			if canon := canonicalProductFromResult(content); canon != "" {
				st.readProducts[canon] = true
			}
		}
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func truncateForTrace(s string) string {
	r := []rune(s)
	if len(r) > 500 {
		return string(r[:500]) + "…"
	}
	return s
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return strings.Trim(string(b), "\"")
}

// TemplateRaw 返回外置 prompt 模板原文（console C3 转发）。
func (a *Agent) TemplateRaw() string { return a.assembler.TemplateRaw() }
