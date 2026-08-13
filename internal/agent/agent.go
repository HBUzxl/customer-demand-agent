// Package agent implements the autonomous agent loop: LLM + function-calling
// tool dispatch, running until the model produces a final answer (no tool
// calls). This is NOT a fixed workflow — the model decides which tools to
// call and in what order (ADR-009).
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/assembler"
	"customer-demand-agent/internal/memory/shortterm"
	"customer-demand-agent/internal/memory/tools"
	"customer-demand-agent/internal/model"
)

// MaxIterations 防止工具调用死循环。
const MaxIterations = 15

// Trace 记录一次 Agent 运行的工具调用轨迹（供前端展示推理过程）。
type Trace struct {
	ToolCalls []ToolCallRecord
}

// ToolCallRecord 是一次工具调用的记录。
type ToolCallRecord struct {
	Tool   string `json:"tool"`
	Params string `json:"params"`
	Result string `json:"result"`
}

// Event 是流式推送的一条 Agent 轨迹事件。覆盖思考/内容/工具调用/工具结果/完成/错误。
type Event struct {
	Type     string                  `json:"type"`                // round/reasoning/content/tool_call/tool_result/done/error
	Text     string                  `json:"text,omitempty"`      // reasoning/content 增量
	Round    int                     `json:"round,omitempty"`     // round 事件的轮次号
	Tool     string                  `json:"tool,omitempty"`      // tool_call/tool_result 的工具名
	Params   string                  `json:"params,omitempty"`    // tool_call 的参数（JSON）
	Result   string                  `json:"result,omitempty"`    // tool_result 的结果（JSON）
	Analysis *domain.AnalysisResult `json:"analysis,omitempty"`  // done 事件（分析场景）
	Content  string                  `json:"content,omitempty"`   // done 事件（追问场景）
	Error    string                  `json:"error,omitempty"`     // error 事件
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
	modelMgr *model.Manager
	tools    *tools.Registry
	assembler *assembler.Assembler
	sessions *shortterm.SessionManager
}

// New 创建 Agent。
func New(modelMgr *model.Manager, tr *tools.Registry, asm *assembler.Assembler, sessions *shortterm.SessionManager) *Agent {
	return &Agent{modelMgr: modelMgr, tools: tr, assembler: asm, sessions: sessions}
}

// Analyze 对客户文本做需求分析，返回结构化结果 + 轨迹（非流式便捷入口）。
func (a *Agent) Analyze(ctx context.Context, sessionID, input string) (*domain.AnalysisResult, *Trace, error) {
	return a.AnalyzeStream(ctx, sessionID, input, nil)
}

// AnalyzeStream 流式分析：emit 实时推送整个 Agent 轨迹（思考/内容/工具调用/结果），最后返回结果。
func (a *Agent) AnalyzeStream(ctx context.Context, sessionID, input string, emit func(Event)) (*domain.AnalysisResult, *Trace, error) {
	stm := a.sessions.Get(sessionID)
	op := shortterm.DetermineOp(stm, input)

	msgList := a.assembler.Assemble(op, stm.BuildContext(op), input)
	trace := &Trace{}

	finalContent, err := a.runStreaming(ctx, msgList, trace, emit, true)
	if err != nil {
		emitEvent(emit, Event{Type: EventError, Error: err.Error()})
		return nil, trace, err
	}

	result, perr := parseAnalysis(finalContent)
	if perr != nil {
		result = &domain.AnalysisResult{DemandAnalysis: finalContent}
	}

	switch op {
	case domain.OpInitial:
		stm.CreateInitialCheckpoint(input, result)
	case domain.OpReanalysis:
		stm.CreateReanalysisCheckpoint(input, result)
	default:
		stm.CreateInitialCheckpoint(input, result)
	}
	emitEvent(emit, Event{Type: EventDone, Analysis: result})
	return result, trace, nil
}

// Chat 处理追问（非流式便捷入口）。
func (a *Agent) Chat(ctx context.Context, sessionID, question string) (string, *Trace, error) {
	return a.ChatStream(ctx, sessionID, question, nil)
}

// ChatStream 流式追问。
func (a *Agent) ChatStream(ctx context.Context, sessionID, question string, emit func(Event)) (string, *Trace, error) {
	stm := a.sessions.Get(sessionID)
	op := domain.OpFollowup

	msgList := a.assembler.Assemble(op, stm.BuildContext(op), "追问："+question)
	trace := &Trace{}

	answer, err := a.runStreaming(ctx, msgList, trace, emit, false)
	if err != nil {
		emitEvent(emit, Event{Type: EventError, Error: err.Error()})
		return "", trace, err
	}
	stm.CreateFollowupCheckpoint(question, answer)
	emitEvent(emit, Event{Type: EventDone, Content: answer})
	return answer, trace, nil
}

// runStreaming 是自主循环的核心（流式）：每轮调 LLM 时推送思考/内容增量，
// 工具调用/结果作为事件推送，直到无工具调用得到最终答案。
func (a *Agent) runStreaming(ctx context.Context, msgList []domain.Message, trace *Trace, emit func(Event), jsonOutput bool) (string, error) {
	toolDefs := a.tools.Definitions()

	for iter := 0; iter < MaxIterations; iter++ {
		emitEvent(emit, Event{Type: EventRound, Round: iter + 1})
		req := &llm.ChatRequest{
			Messages:   msgList,
			Tools:      toolDefs,
			JSONOutput: jsonOutput,
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
			return resp.Message.Content, nil
		}

		for _, tc := range resp.Message.ToolCalls {
			emitEvent(emit, Event{Type: EventToolCall, Tool: tc.Function.Name, Params: tc.Function.Arguments})
			result := a.executeTool(ctx, tc, trace)
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
func (a *Agent) executeTool(ctx context.Context, tc domain.ToolCall, trace *Trace) string {
	args := json.RawMessage(tc.Function.Arguments)
	result, err := a.tools.Execute(tc.Function.Name, args)
	if err != nil {
		result = fmt.Sprintf(`{"error":"%s"}`, jsonEscape(err.Error()))
	}
	trace.ToolCalls = append(trace.ToolCalls, ToolCallRecord{
		Tool:   tc.Function.Name,
		Params: tc.Function.Arguments,
		Result: truncateForTrace(result),
	})
	return result
}

// ── 结果解析 ──────────────────────────────────────────────────

// parseAnalysis 把 LLM 文本输出解析为 AnalysisResult。
// 先尝试直接 JSON unmarshal；失败则抽取首个 {...} 块再试；仍失败返回错误。
func parseAnalysis(content string) (*domain.AnalysisResult, error) {
	content = strings.TrimSpace(content)
	content = stripCodeFence(content)

	var r domain.AnalysisResult
	if err := json.Unmarshal([]byte(content), &r); err == nil {
		return &r, nil
	}
	// 抽取首个 JSON 对象
	if s := extractJSON(content); s != "" {
		if err := json.Unmarshal([]byte(s), &r); err == nil {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("无法解析为 AnalysisResult JSON")
}

// stripCodeFence 去掉 ```json ... ``` 包裹。
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// 去首行
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
		s = strings.TrimSpace(s)
	}
	return s
}

// extractJSON 从文本中抽出第一个平衡的 {...}。
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	inStr := false
	escape := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escape {
			escape = false
			continue
		}
		if c == '\\' {
			escape = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
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
