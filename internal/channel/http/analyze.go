package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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

// messageCore 是所有消息入口共享的核心逻辑（tenant 从请求头解析）。
func (s *Server) messageCore(w http.ResponseWriter, r *http.Request, sessionID, text, customer string) {
	tenant := tenantFrom(r)
	if err := s.history.EnsureSession(tenant, sessionID, firstLine(text), customer); err != nil {
		writeError(w, http.StatusForbidden, "无权访问该会话: %v", err)
		return
	}
	_, _ = s.history.AppendMessage(sessionID, "user", text, "")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "当前环境不支持流式响应")
		return
	}
	setSSEHeaders(w)
	// 先发一个 session_id 事件，客户端立刻拿到
	writeSSE(w, flusher, agent.Event{Type: "session", Content: sessionID})

	content, _, trace, err := s.agent.Message(r.Context(), tenant, sessionID, text, func(e agent.Event) {
		writeSSE(w, flusher, e)
	})
	if err != nil {
		if r.Context().Err() != nil {
			// 客户端断开（刷新页面/关闭连接/超时掐断）：不是系统失败，
			// 不往历史写 [失败]——用户主动离开，留痕只会污染会话。
			log.Printf("[warn] 客户端断开，会话 %s 流式中止: %v", sessionID, err)
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

// writeSSE writes one SSE event and flushes.
func writeSSE(w http.ResponseWriter, flusher http.Flusher, e agent.Event) {
	data, _ := json.Marshal(e)
	fmt.Fprintf(w, "data: %s\n\n", data)
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
