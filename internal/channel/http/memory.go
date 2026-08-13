package http

import (
	"net/http"
	"strconv"

	"customer-demand-agent/internal/memory/longterm"
)

// handleMemorySearch: GET /api/memory/search?q=&type=&limit=
func (s *Server) handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	typ := r.URL.Query().Get("type")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	hits := s.storeWiki.SearchEntry(q, typ, limit)
	writeJSON(w, http.StatusOK, map[string]any{"count": len(hits), "items": hits})
}

// handleMemoryList: GET /api/memory/list?type=&offset=&limit=
func (s *Server) handleMemoryList(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("type")
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries := s.storeWiki.ListEntry(typ, offset, limit)
	writeJSON(w, http.StatusOK, map[string]any{"count": len(entries), "items": entries})
}

// handleMemoryGet: GET /api/memory/{type}/{title}
func (s *Server) handleMemoryGet(w http.ResponseWriter, r *http.Request) {
	typ := r.PathValue("type")
	title := r.PathValue("title")
	entry, err := s.storeWiki.GetEntry(typ, title)
	if err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

// memoryUpsertReq is the body for POST /api/memory (human-facing, bypasses AI permission).
type memoryUpsertReq struct {
	Type     string   `json:"type"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Tags     []string `json:"tags"`
	Aliases  []string `json:"aliases"`
	Summary  string   `json:"summary"`
	Status   string   `json:"status"`
}

// handleMemoryUpsert: POST /api/memory
// 这是人工管理入口（产品经理/管理员用），不经过 AI 权限校验，可直接写 product/user。
func (s *Server) handleMemoryUpsert(w http.ResponseWriter, r *http.Request) {
	var req memoryUpsertReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	mt, ok := parseType(req.Type)
	if !ok {
		writeError(w, http.StatusBadRequest, "未知 type: %q", req.Type)
		return
	}
	status := longterm.StatusVerified
	if req.Status != "" {
		status = longterm.EntryStatus(req.Status)
	}
	// 若已存在，以旧条目为基底覆盖可编辑字段，保留其结构化 frontmatter（如产品 capabilities）
	entry := &longterm.Entry{
		Type:    mt,
		Title:   req.Title,
		Content: req.Content,
		Tags:    req.Tags,
		Aliases: req.Aliases,
		Summary: req.Summary,
		Status:  status,
	}
	if existing, err := s.storeWiki.GetEntry(req.Type, req.Title); err == nil {
		entry = existing // existing.typed 已填充（结构化字段保留）
		entry.Content = req.Content
		if req.Tags != nil {
			entry.Tags = req.Tags
		}
		if req.Aliases != nil {
			entry.Aliases = req.Aliases
		}
		entry.Summary = req.Summary
		entry.Status = status
	}
	if err := s.storeWiki.UpsertEntry(entry); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "type": req.Type, "title": req.Title})
}

// handleMemoryDelete: DELETE /api/memory/{type}/{title}?archive=true
func (s *Server) handleMemoryDelete(w http.ResponseWriter, r *http.Request) {
	typ := r.PathValue("type")
	title := r.PathValue("title")
	archive := r.URL.Query().Get("archive") != "false"
	if err := s.storeWiki.DeleteEntry(typ, title, archive); err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	action := "已归档"
	if !archive {
		action = "已删除"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": action})
}
