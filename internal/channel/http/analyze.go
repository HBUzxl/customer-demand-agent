package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"customer-demand-agent/internal/agent"
)

// analyzeReq is the request body for POST /api/analyze.
type analyzeReq struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
}

// handleAnalyze 以 SSE 流式返回整个 Agent 轨迹（思考/内容/工具调用/结果/完成）。
// 客户端收到 data: {type:"done", analysis:{...}} 表示分析完成。
func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	var req analyzeReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text 不能为空")
		return
	}
	tenant := tenantFrom(r)
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = newSessionID()
	}

	_ = s.history.EnsureSession(tenant, sessionID, firstLine(req.Text), "")
	_, _ = s.history.AppendMessage(sessionID, "user", req.Text, "")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "当前环境不支持流式响应")
		return
	}
	setSSEHeaders(w)
	// 先发一个 session_id 事件，客户端立刻拿到
	writeSSE(w, flusher, agent.Event{Type: "session", Content: sessionID})

	result, trace, err := s.agent.AnalyzeStream(r.Context(), sessionID, req.Text, func(e agent.Event) {
		writeSSE(w, flusher, e)
	})

	if err != nil {
		_, _ = s.history.AppendMessage(sessionID, "assistant", "[分析失败] "+err.Error(), "")
		return
	}

	// 持久化 assistant 输出 + 工具调用轨迹
	if result != nil {
		resultJSON, _ := json.Marshal(result)
		_, _ = s.history.AppendMessage(sessionID, "assistant", string(resultJSON), "")
	}
	if trace != nil {
		for _, tc := range trace.ToolCalls {
			_, _ = s.history.AppendToolCall(sessionID, tc.Tool, tc.Params, tc.Result)
		}
	}
}

// chatReq is the request body for POST /api/chat (follow-up question).
type chatReq struct {
	SessionID string `json:"session_id"`
	Question  string `json:"question"`
}

// handleChat 以 SSE 流式处理追问。
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	if req.SessionID == "" || req.Question == "" {
		writeError(w, http.StatusBadRequest, "session_id 和 question 不能为空")
		return
	}
	tenant := tenantFrom(r)
	_ = s.history.EnsureSession(tenant, req.SessionID, "", "")
	_, _ = s.history.AppendMessage(req.SessionID, "user", req.Question, "")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "当前环境不支持流式响应")
		return
	}
	setSSEHeaders(w)
	writeSSE(w, flusher, agent.Event{Type: "session", Content: req.SessionID})

	answer, trace, err := s.agent.ChatStream(r.Context(), req.SessionID, req.Question, func(e agent.Event) {
		writeSSE(w, flusher, e)
	})
	_ = tenant
	if err != nil {
		_, _ = s.history.AppendMessage(req.SessionID, "assistant", "[追问失败] "+err.Error(), "")
		return
	}
	_, _ = s.history.AppendMessage(req.SessionID, "assistant", answer, "")
	if trace != nil {
		for _, tc := range trace.ToolCalls {
			_, _ = s.history.AppendToolCall(req.SessionID, tc.Tool, tc.Params, tc.Result)
		}
	}
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
