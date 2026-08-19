package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/taskbg"
	"customer-demand-agent/internal/tenancy"
)

// analyzeReq is the request body for POST /api/analyze.
type analyzeReq struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
	Customer  string `json:"customer"` // 可选：会话关联客户（注入客户画像，跨会话统一视图）
	Branch    string `json:"branch"`   // 可选：分叉后续写指定分支（checkpoint-tree 编辑重发链路）
}

// handleAnalyze: POST /api/analyze（deprecated shim → 统一 Message 循环，ADR-014）。
// 以 SSE 流式返回整个 Agent 轨迹（思考/内容/工具调用/结果/完成）。
func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Link", `</api/message>; rel="successor-version"`)
	s.handleMessage(w, r)
}

// handleMessage 是统一的消息入口（ADR-014）：任意输入（寒暄/需求/追问）都走
// 同一个自主循环，Agent 自主决定回应方式。
func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	var req analyzeReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	req.Customer = strings.TrimSpace(req.Customer)
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text 不能为空")
		return
	}
	if len([]rune(req.Text)) > 50000 {
		writeError(w, http.StatusRequestEntityTooLarge, "text 不能超过 50000 个字符")
		return
	}
	if len([]rune(req.Customer)) > 120 {
		writeError(w, http.StatusBadRequest, "customer 不能超过 120 个字符")
		return
	}
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = newSessionID()
	}
	s.messageCore(w, r, sessionID, req.Text, req.Customer, req.Branch)
}

// messageCore 创建 Run 并立即返回 202（F0：执行与连接解耦）。
// 真正的 Agent 循环跑在 RunManager 的 goroutine 里；事件进缓冲，
// 客户端通过 GET /api/sessions/{id}/stream 订阅（replay+live）。
// 多租户：scope 解析租户运行时，EnsureSession 带 owner_user_id，
// Run 以 RunKey{TenantID, SessionID} 隔离（§9.3）。
func (s *Server) messageCore(w http.ResponseWriter, r *http.Request, sessionID, text, customer, branch string) {
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "无法解析租户: %v", err)
		return
	}
	// 会话标题：仅新会话落占位标题（首行），已有会话不覆盖——标题由 G4 后台
	// 任务按累计的用户提问归纳（避免每轮被 current 消息首行/最后一问覆盖）。
	exists, err := rt.History.SessionExists(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "检查会话状态失败")
		return
	}
	var title string
	if !exists {
		title = firstLine(text)
	}
	// 客户归属只允许在创建会话时写入。已有会话先以空 customer 做作用域校验，
	// 再拒绝跨客户重绑，避免恶意/误用客户端把一段历史对话挂到另一客户画像。
	// 相同 customer 的幂等重试允许通过，但不会再次更新归属。
	bindCustomer := customer
	if exists {
		bindCustomer = ""
	}
	if err := rt.History.EnsureSessionScoped(&sc, sessionID, title, bindCustomer); err != nil {
		// 会话 id 可能由客户端提供；不可见与不存在统一 404，避免暴露同租户
		// 其他成员的会话是否存在。
		writeError(w, http.StatusNotFound, "会话不存在或无权访问")
		return
	}
	if exists && customer != "" {
		det, derr := rt.History.GetSession(&sc, sessionID, "")
		if derr != nil {
			writeError(w, http.StatusNotFound, "会话不存在或无权访问")
			return
		}
		if strings.TrimSpace(det.Session.Customer) != customer {
			writeError(w, http.StatusConflict, "已有会话不可修改客户归属，请新建会话")
			return
		}
	}

	key := domain.RunKey{TenantID: sc.TenantID, SessionID: sessionID}
	runID := newSessionID()
	fullRunID := "run-" + runID[5:]
	err = s.runs.Start(key, fullRunID, sc.UserID, func(ctx context.Context, emit func(agent.Event)) {
		// user 消息在 Run 确认占用后写入（并发 409 不留孤儿消息）
		if branch != "" {
			if _, aerr := rt.History.AppendMessageBranch(sessionID, branch, "user", text, ""); aerr != nil {
				log.Printf("[warn] user 消息落分支失败 会话 %s 分支 %s: %v", sessionID, branch, aerr)
			}
		} else if _, aerr := rt.History.AppendMessage(sessionID, "user", text, ""); aerr != nil {
			log.Printf("[warn] user 消息落库失败 会话 %s: %v", sessionID, aerr)
		}
		s.executeTurn(ctx, sc, rt, sessionID, branch, text, emit)
	})
	if err != nil {
		writeError(w, http.StatusConflict, "%v", err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"session_id": sessionID,
		"run_id":     fullRunID,
	})
}

// appendMsg 分支感知的消息落库（P0-04 全链路同分支）：分支续写走
// AppendMessageBranch（消息挂当前分支），主链/main 走 AppendMessage。
// 工具调用经返回的 message_id 归属对应消息，天然落在同一分支。
func appendMsg(rt *tenancy.Runtime, sessionID, branch, role, content, toolCallID string) (int64, error) {
	if branch != "" && branch != "main" {
		return rt.History.AppendMessageBranch(sessionID, branch, role, content, toolCallID)
	}
	return rt.History.AppendMessage(sessionID, role, content, toolCallID)
}

// executeTurn 是 Run 的执行体：完整 Agent 循环 + 落库（与连接彻底解耦）。
// ADR-015：前台全记忆不变——Run 仍是完整循环，变的只是执行宿主（租户运行时）。
// P0-04：branch 续写时 user/system/assistant/error 消息全部写同一分支。
func (s *Server) executeTurn(ctx context.Context, sc domain.TenantScope, rt *tenancy.Runtime, sessionID, branch, text string, emit func(agent.Event)) {
	emit(agent.Event{Type: "session", Content: sessionID})

	// 取消时 Agent.Message 返回空 content——在 emit 层累积流出的内容增量，
	// 取消落库用（中断态持久化：刷新/切回后部分结果不丢）。
	var streamed strings.Builder
	emitWrap := func(e agent.Event) {
		if e.Type == agent.EventContent {
			streamed.WriteString(e.Text)
		}
		emit(e)
	}

	content, _, trace, err := rt.Agent.Message(ctx, &sc, sessionID, branch, text, emitWrap)
	if err != nil {
		if ctx.Err() != nil {
			// 显式取消（用户点了停止）：已流出内容 + 停止标记落库
			partial := streamed.String()
			if strings.TrimSpace(partial) != "" {
				_, _ = appendMsg(rt, sessionID, branch, "assistant", partial+"\n\n（已停止）", "")
			} else {
				_, _ = appendMsg(rt, sessionID, branch, "assistant", "（已停止）", "")
			}
			log.Printf("[info] Run 被取消，会话 %s 中止（部分内容已落库）: %v", sessionID, err)
			return
		}
		_, _ = appendMsg(rt, sessionID, branch, "assistant", "[失败] "+err.Error(), "")
		return
	}

	// 回放轨迹持久化：system prompt（整个调用状态可见）+ assistant 自然语言 +
	// 工具调用（message_id 归属 assistant 消息，回放按时间线表格还原）。
	if trace != nil && trace.SystemPrompt != "" {
		_, _ = appendMsg(rt, sessionID, branch, "system", trace.SystemPrompt, "")
	}
	assistantID, _ := appendMsg(rt, sessionID, branch, "assistant", content, "")
	if trace != nil {
		for _, tc := range trace.ToolCalls {
			_, _ = rt.History.AppendToolCall(sessionID, assistantID, tc.Tool, tc.Params, tc.Result)
		}
	}
	// G4 会话标题：每轮完成且标题未被用户手动固定时，后台按累计的用户提问
	// 归纳为简短标题（替代首行截断 / 最后一问的丑标题）。
	if s.tasks != nil {
		s.scheduleSessionTitle(rt, &sc, sessionID, branch)
	}
}

// scheduleSessionTitle 后台生成/更新会话标题：把本会话累计的用户提问原文交给
// LLM 归纳为一个简短标题（G4）。仅当标题未被用户手动重命名固定（title_pinned）
// 时提交；多轮对话随提问累积，标题持续收敛到真实主题（而非首行/最后一问）。
func (s *Server) scheduleSessionTitle(rt *tenancy.Runtime, sc *domain.TenantScope, sessionID, branch string) {
	det, err := rt.History.GetSessionInternal(sessionID, branch)
	if err != nil {
		return
	}
	if pinned, err := rt.History.IsSessionTitlePinned(sessionID); err != nil || pinned {
		return // 用户手动重命名过：不再自动覆盖
	}
	var asks []string
	for _, m := range det.Messages {
		if m.Role == "user" && strings.TrimSpace(m.Content) != "" {
			asks = append(asks, m.Content)
		}
	}
	if len(asks) == 0 {
		return
	}
	input := strings.Join(asks, "\n---\n")
	if n := len([]rune(input)); n > 1000 {
		input = string([]rune(input)[:1000]) + "…"
	}
	s.tasks.Submit(sc, fmt.Sprintf("title-%d", time.Now().UnixNano()), taskbg.TaskTitle,
		taskbg.TaskResource{Type: "session", TypeID: sessionID, Title: input})
}

// handleSessionStream 订阅会话事件流（SSE）：先 replay since 之后的事件，
// 再续传 live；Run 结束且缓冲追平后关流。
func (s *Server) handleSessionStream(w http.ResponseWriter, r *http.Request) {
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	sessionID := r.PathValue("id")
	if _, err := rt.History.GetSession(&sc, sessionID, ""); err != nil {
		writeError(w, http.StatusNotFound, "会话不存在: %v", err)
		return
	}
	since := 0
	if v := r.URL.Query().Get("since"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			since = n
		}
	}
	key := domain.RunKey{TenantID: sc.TenantID, SessionID: sessionID}
	replay, live, done, unsub := s.runs.Subscribe(key, since)
	defer unsub()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "当前环境不支持流式响应")
		return
	}
	setSSEHeaders(w)
	// 支持 Last-Event-ID 自动续传（浏览器 EventSource 原生）；
	// 自定义 fetch 客户端用 since 参数，效果相同。
	if v := r.Header.Get("Last-Event-ID"); v != "" && since == 0 {
		if n, err := strconv.Atoi(v); err == nil {
			since = n
		}
	}
	// replay 阶段（带 seq 游标；历史 Run 事件 data schema 不变）
	for _, be := range replay {
		if be.Seq > since {
			writeSSESeq(w, flusher, be.Seq, be.RunID, be.Event)
		}
	}
	// 无活跃 Run：replay 完即结束
	if done == nil {
		return
	}
	// live 阶段；Run 结束（done 关闭）后从缓冲按最后 seq 补尾——
	// 慢订阅者 live 通道溢出丢弃的事件由此兜底，保证不丢不重。
	lastSeq := since
	if len(replay) > 0 && replay[len(replay)-1].Seq > lastSeq {
		lastSeq = replay[len(replay)-1].Seq
	}
	for {
		select {
		case be := <-live:
			if be.Seq > lastSeq {
				writeSSESeq(w, flusher, be.Seq, be.RunID, be.Event)
				lastSeq = be.Seq
			}
		case <-done:
			// Run 结束：从缓冲补发 lastSeq 之后的所有事件（live 通道
			// 滞留 + 溢出丢弃一次性兜底），再关流
			tail := s.runs.ReplayAfter(key, lastSeq)
			for _, be := range tail {
				if be.Seq > lastSeq {
					writeSSESeq(w, flusher, be.Seq, be.RunID, be.Event)
					lastSeq = be.Seq
				}
			}
			return
		case <-r.Context().Done():
			return // 客户端断开：只是取消订阅，Run 继续
		}
	}
}

// handleSessionRunning 会话是否有活跃 Run（前端运行指示/切回恢复用）。
// 返回会话运行态（不存在/已结束 → false）。
func (s *Server) handleSessionRunning(w http.ResponseWriter, r *http.Request) {
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	sessionID := r.PathValue("id")
	if _, err := rt.History.GetSession(&sc, sessionID, ""); err != nil {
		writeError(w, http.StatusNotFound, "会话不存在: %v", err)
		return
	}
	key := domain.RunKey{TenantID: sc.TenantID, SessionID: sessionID}
	writeJSON(w, http.StatusOK, map[string]any{
		"running": s.runs.Running(key),
		"run_id":  s.runs.RunID(key),
	})
}

// handleRunCancel 显式取消会话的活跃 Run（唯一停止途径）。
// 取消会话 Run（不存在或已结束 → 404）。
func (s *Server) handleRunCancel(w http.ResponseWriter, r *http.Request) {
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	sessionID := r.PathValue("id")
	runID := r.PathValue("run_id")
	if ok, err := rt.History.CanWriteSession(&sc, sessionID); err != nil || !ok {
		writeError(w, http.StatusNotFound, "会话不存在: %v", err)
		return
	}
	if err := s.runs.Cancel(domain.RunKey{TenantID: sc.TenantID, SessionID: sessionID}, runID); err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelling"})
}

// handleMessagesTruncate 截断会话到指定 seq 之后（编辑重发用）：
// 删除 seq 之后的消息 + 关联 tool_calls + 全部 checkpoints（F1）。
func (s *Server) handleMessagesTruncate(w http.ResponseWriter, r *http.Request) {
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	sessionID := r.PathValue("id")
	// 共享读取不代表可修改：编辑重发仍限会话创建者或租户 owner/admin。
	if ok, err := rt.History.CanWriteSession(&sc, sessionID); err != nil || !ok {
		writeError(w, http.StatusNotFound, "会话不存在: %v", err)
		return
	}
	key := domain.RunKey{TenantID: sc.TenantID, SessionID: sessionID}
	if s.runs.Running(key) {
		writeError(w, http.StatusConflict, "会话正在运行，先停止再编辑")
		return
	}
	after := 0
	if v := r.URL.Query().Get("seq"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			after = n
		}
	}
	if after <= 0 {
		writeError(w, http.StatusBadRequest, "seq 必须为正整数（该消息及其之后将被删除）")
		return
	}
	// checkpoint-tree：编辑重发=软分叉（旧消息挂新分支保留），返回分支 ID
	branchID, moved, err := rt.History.BranchAfter(sessionID, after)
	if err != nil {
		writeError(w, http.StatusBadRequest, "分叉失败: %v", err)
		return
	}
	s.runs.DropSession(key)
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "branch_id": branchID, "branched_messages": moved})
}

// chatReq is the request body for POST /api/chat (deprecated shim, body 映射用).
type chatReq struct {
	SessionID string `json:"session_id"`
	Question  string `json:"question"`
}

// handleChat 以 SSE 流式处理追问（deprecated shim：question → text 走统一入口）。
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Link", `</api/message>; rel="successor-version"`)
	var req chatReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	if req.SessionID == "" || req.Question == "" {
		writeError(w, http.StatusBadRequest, "session_id 和 question 不能为空")
		return
	}
	s.messageCore(w, r, req.SessionID, req.Question, "", "")
}

// setSSEHeaders sets SSE response headers.
func setSSEHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no") // nginx 不缓冲
}

// writeSSESeq 写带 SSE id（事件游标）与 x-run（产生事件的 Run）的事件。
// data payload 的 agent.Event schema 保持不变——run_id 走 SSE 字段行
// （x-run:），浏览器 EventSource 忽略未知字段，自定义客户端可解析。
func writeSSESeq(w http.ResponseWriter, flusher http.Flusher, seq int, runID string, e agent.Event) {
	data, _ := json.Marshal(e)
	if runID != "" {
		fmt.Fprintf(w, "id: %d\nx-run: %s\ndata: %s\n\n", seq, runID, data)
	} else {
		fmt.Fprintf(w, "id: %d\ndata: %s\n\n", seq, data)
	}
	flusher.Flush()
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	if len([]rune(s)) > 40 {
		return string([]rune(s)[:40]) + "…"
	}
	return s
}
