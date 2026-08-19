package http

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// LogRing 是进程内日志环形缓冲（观测台 tail 用；重启即失）。
type LogRing struct {
	mu    sync.Mutex
	lines []string
	head  int
	cap   int
	subs  map[chan string]struct{}
}

// NewLogRing 创建环形缓冲。
func NewLogRing(capacity int) *LogRing {
	return &LogRing{
		lines: make([]string, capacity),
		cap:   capacity,
		subs:  make(map[chan string]struct{}),
	}
}

// Write 实现 io.Writer（接 log.SetOutput 用）——写一行+广播订阅者。
func (r *LogRing) Write(p []byte) (int, error) {
	line := time.Now().Format("15:04:05 ") + string(p)
	r.mu.Lock()
	r.lines[r.head%r.cap] = line
	r.head++
	subs := make([]chan string, 0, len(r.subs))
	for ch := range r.subs {
		subs = append(subs, ch)
	}
	r.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- line:
		default:
		}
	}
	return len(p), nil
}

// Tail 返回最近 n 行。
func (r *LogRing) Tail(n int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	start := r.head - n
	if start < 0 {
		start = 0
	}
	for i := start; i < r.head; i++ {
		if l := r.lines[i%r.cap]; l != "" {
			out = append(out, l)
		}
	}
	return out
}

// Subscribe 订阅新行；返回取消函数。
func (r *LogRing) Subscribe() (<-chan string, func()) {
	ch := make(chan string, 64)
	r.mu.Lock()
	r.subs[ch] = struct{}{}
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		delete(r.subs, ch)
		r.mu.Unlock()
	}
}

// handleConsoleLogs: GET /api/platform/logs —— 原始日志 tail（SSE：先 replay 最近 100 行再续传）。
// 平台管理路由（platform_admin 专用）：日志含系统级敏感信息，不对租户开放（§7.5）。
func (s *Server) handleConsoleLogs(w http.ResponseWriter, r *http.Request) {
	if s.logRing == nil {
		writeError(w, http.StatusServiceUnavailable, "日志流未启用")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "当前环境不支持流式响应")
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	// replay
	for _, line := range s.logRing.Tail(100) {
		fmt.Fprintf(w, "data: %s\n\n", line)
	}
	flusher.Flush()
	// live
	ch, unsub := s.logRing.Subscribe()
	defer unsub()
	for {
		select {
		case line := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
