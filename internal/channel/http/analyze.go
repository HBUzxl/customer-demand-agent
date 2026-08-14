package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"customer-demand-agent/internal/agent"
)

// analyzeReq is the request body for POST /api/analyze.
type analyzeReq struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
	Customer  string `json:"customer"` // 可选：会话关联客户（注入客户画像，跨会话统一视图）
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
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text 不能为空")
		return
	}
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = newSessionID()
	}
	s.messageCore(w, r, sessionID, req.Text, req.Customer)
}

// messageCore 创建 Run 并立即返回 202（F0：执行与连接解耦）。
// 真正的 Agent 循环跑在 RunManager 的 goroutine 里；事件进缓冲，
// 客户端通过 GET /api/sessions/{id}/stream 订阅（replay+live）。
func (s *Server) messageCore(w http.ResponseWriter, r *http.Request, sessionID, text, customer string) {
	tenant := tenantFrom(r)
	if err := s.history.EnsureSession(tenant, sessionID, firstLine(text), customer); err != nil {
		writeError(w, http.StatusForbidden, "无权访问该会话: %v", err)
		return
	}

	runID := newSessionID()
	fullRunID := "run-" + runID[5:]
	err := s.runs.Start(sessionID, fullRunID, func(ctx context.Context, emit func(agent.Event)) {
		// user 消息在 Run 确认占用后写入（并发 409 不留孤儿消息）
		if _, aerr := s.history.AppendMessage(sessionID, "user", text, ""); aerr != nil {
			log.Printf("[warn] user 消息落库失败 会话 %s: %v", sessionID, aerr)
		}
		s.executeTurn(ctx, tenant, sessionID, text, emit)
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

// executeTurn 是 Run 的执行体：完整 Agent 循环 + 落库（与连接彻底解耦）。
// ADR-015：前台全记忆不变——Run 仍是完整循环，变的只是执行宿主。
func (s *Server) executeTurn(ctx context.Context, tenant, sessionID, text string, emit func(agent.Event)) {
	emit(agent.Event{Type: "session", Content: sessionID})

	content, _, trace, err := s.agent.Message(ctx, tenant, sessionID, text, emit)
	if err != nil {
		if ctx.Err() != nil {
			// 显式取消（用户点了停止）：已流出的部分内容落库 + 停止标记——
			// 刷新/切回后中断结果不丢（中断态持久化）。
			if strings.TrimSpace(content) != "" {
				_, _ = s.history.AppendMessage(sessionID, "assistant", content+"\n\n（已停止）", "")
			} else {
				_, _ = s.history.AppendMessage(sessionID, "assistant", "（已停止）", "")
			}
			log.Printf("[info] Run 被取消，会话 %s 中止（部分内容已落库）: %v", sessionID, err)
			return
		}
		_, _ = s.history.AppendMessage(sessionID, "assistant", "[失败] "+err.Error(), "")
		return
	}

	// 回放轨迹持久化：system prompt（整个调用状态可见）+ assistant 自然语言 +
	// 工具调用（message_id 归属 assistant 消息，回放按时间线表格还原）。
	if trace != nil && trace.SystemPrompt != "" {
		_, _ = s.history.AppendMessage(sessionID, "system", trace.SystemPrompt, "")
	}
	assistantID, _ := s.history.AppendMessage(sessionID, "assistant", content, "")
	if trace != nil {
		for _, tc := range trace.ToolCalls {
			_, _ = s.history.AppendToolCall(sessionID, assistantID, tc.Tool, tc.Params, tc.Result)
		}
	}
}

// handleSessionStream 订阅会话事件流（SSE）：先 replay since 之后的事件，
// 再续传 live；Run 结束且缓冲追平后关流。
func (s *Server) handleSessionStream(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	tenant := tenantFrom(r)
	if _, err := s.history.GetSession(tenant, sessionID); err != nil {
		writeError(w, http.StatusNotFound, "会话不存在: %v", err)
		return
	}
	since := 0
	if v := r.URL.Query().Get("since"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			since = n
		}
	}
	replay, live, done, unsub := s.runs.Subscribe(sessionID, since)
	defer unsub()
	curRun := s.runs.RunID(sessionID) // 当前 Run（replay 里历史 Run 的事件无标注——由游标语义隔离）

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
			writeSSESeq(w, flusher, be.Seq, curRun, be.Event)
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
				writeSSESeq(w, flusher, be.Seq, curRun, be.Event)
				lastSeq = be.Seq
			}
		case <-done:
			// Run 结束：从缓冲补发 lastSeq 之后的所有事件（live 通道
			// 滞留 + 溢出丢弃一次性兜底），再关流
			tail := s.runs.ReplayAfter(sessionID, lastSeq)
			for _, be := range tail {
				if be.Seq > lastSeq {
					writeSSESeq(w, flusher, be.Seq, curRun, be.Event)
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
// 租户校验：只返回本租户会话的运行态（跨租户 404）。
func (s *Server) handleSessionRunning(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	tenant := tenantFrom(r)
	if _, err := s.history.GetSession(tenant, sessionID); err != nil {
		writeError(w, http.StatusNotFound, "会话不存在: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"running": s.runs.Running(sessionID),
		"run_id":  s.runs.RunID(sessionID),
	})
}

// handleRunCancel 显式取消会话的活跃 Run（唯一停止途径）。
// 租户校验：只能取消本租户会话的 Run（跨租户 404）。
func (s *Server) handleRunCancel(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	runID := r.PathValue("run_id")
	tenant := tenantFrom(r)
	if _, err := s.history.GetSession(tenant, sessionID); err != nil {
		writeError(w, http.StatusNotFound, "会话不存在: %v", err)
		return
	}
	if err := s.runs.Cancel(sessionID, runID); err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelling"})
}

// handleMessagesTruncate 截断会话到指定 seq 之后（编辑重发用）：
// 删除 seq 之后的消息 + 关联 tool_calls + 全部 checkpoints（F1）。
func (s *Server) handleMessagesTruncate(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	tenant := tenantFrom(r)
	// 先租户归属校验（防跨租户探测运行态），再检查运行状态
	if _, err := s.history.GetSession(tenant, sessionID); err != nil {
		writeError(w, http.StatusNotFound, "会话不存在: %v", err)
		return
	}
	if s.runs.Running(sessionID) {
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
	deleted, err := s.history.TruncateAfter(tenant, sessionID, after)
	if err != nil {
		writeError(w, http.StatusBadRequest, "截断失败: %v", err)
		return
	}
	s.runs.DropSession(sessionID)
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted_messages": deleted})
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
	s.messageCore(w, r, req.SessionID, req.Question, "")
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
