// Package http implements the HTTP channel: a REST API over the agent,
// memory, review, history, and config subsystems. Single-tenant (de-tenancy:
// X-Tenant-ID header accepted but ignored, ADR-011 已废止). See ADR-012.
package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/history"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/model"
	"customer-demand-agent/internal/review"
	"customer-demand-agent/internal/taskbg"
)

// Server wires all subsystems behind an HTTP REST API.
type Server struct {
	store     *config.Store
	agent     *agent.Agent
	review    *review.Service
	storeWiki *longterm.WikiStore
	history   *history.Store
	modelMgr  *model.Manager
	registry  *model.Registry
	runs      *RunManager    // F0：服务端 Run 任务（执行与连接解耦）
	tasks     *taskbg.Runner // P11：后台任务（观测台消费）
}

// New creates the HTTP server with all dependencies injected.
func New(store *config.Store, ag *agent.Agent, rv *review.Service, wiki *longterm.WikiStore,
	hist *history.Store, mm *model.Manager, reg *model.Registry, tasks *taskbg.Runner) *Server {
	return &Server{
		store: store, agent: ag, review: rv, storeWiki: wiki,
		history: hist, modelMgr: mm, registry: reg,
		runs: NewRunManager(), tasks: tasks,
	}
}

// Handler returns the configured ServeMux with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// 分析对话（统一入口 + deprecated shims，ADR-014；F0：202 + Run 任务）
	mux.HandleFunc("POST /api/message", s.handleMessage)
	mux.HandleFunc("POST /api/analyze", s.handleAnalyze)
	mux.HandleFunc("POST /api/chat", s.handleChat)
	mux.HandleFunc("GET /api/sessions/{id}/stream", s.handleSessionStream)
	mux.HandleFunc("GET /api/sessions/{id}/running", s.handleSessionRunning)
	mux.HandleFunc("POST /api/sessions/{id}/runs/{run_id}/cancel", s.handleRunCancel)
	mux.HandleFunc("DELETE /api/sessions/{id}/messages/after", s.handleMessagesTruncate)

	// 记忆管理
	mux.HandleFunc("GET /api/memory/search", s.handleMemorySearch)
	mux.HandleFunc("GET /api/memory/list", s.handleMemoryList)
	mux.HandleFunc("GET /api/memory/{type}/{title}", s.handleMemoryGet)
	mux.HandleFunc("POST /api/memory", s.handleMemoryUpsert)
	mux.HandleFunc("DELETE /api/memory/{type}/{title}", s.handleMemoryDelete)

	// 审核
	mux.HandleFunc("GET /api/review/pending", s.handleReviewPending)
	mux.HandleFunc("POST /api/review/{type}/{title}/approve", s.handleReviewApprove)
	mux.HandleFunc("POST /api/review/{type}/{title}/reject", s.handleReviewReject)

	// 配置
	mux.HandleFunc("GET /api/config", s.handleConfigGet)
	mux.HandleFunc("PUT /api/config", s.handleConfigPut)
	mux.HandleFunc("POST /api/config/test", s.handleConfigTest)
	mux.HandleFunc("POST /api/models", s.handleListModels)

	// 会话历史
	mux.HandleFunc("GET /api/tasks", s.handleTasksList)
	mux.HandleFunc("POST /api/tasks/consolidate", s.handleTaskConsolidate)

	mux.HandleFunc("GET /api/sessions", s.handleSessionList)
	mux.HandleFunc("GET /api/sessions/{id}", s.handleSessionGet)
	mux.HandleFunc("DELETE /api/sessions/{id}", s.handleSessionDelete)

	// 健康检查
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return logging(mux)
}

// ── 中间件 ────────────────────────────────────────────────────

// logging wraps a handler with request logging.
func logging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// ── 响应辅助 ──────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, format string, args ...any) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf(format, args...)})
}

// newSessionID 生成一个会话 id。
func newSessionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "sess_" + hex.EncodeToString(b)
}

func decodeBody(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}
