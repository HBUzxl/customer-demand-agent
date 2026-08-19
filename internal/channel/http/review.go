package http

import (
	"net/http"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/memory/longterm"
)

// handleReviewPending: GET /api/review/pending —— 待审列表（含重叠提示：
// 与同类型已验证条目正文 bigram Jaccard>0.6 → overlap_titles，辅助人审
// 判断是否重复/矛盾，wiki-hygiene F3）。
func (s *Server) handleReviewPending(w http.ResponseWriter, r *http.Request) {
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	items := rt.Review.Pending()
	type withOverlap struct {
		Type          string   `json:"type"`
		Title         string   `json:"title"`
		Summary       string   `json:"summary"`
		Tags          []string `json:"tags"`
		Content       string   `json:"content"`
		Status        string   `json:"status"`
		OverlapTitles []string `json:"overlap_titles,omitempty"`
	}
	out := make([]withOverlap, 0, len(items))
	for _, it := range items {
		wo := withOverlap{Type: it.Type, Title: it.Title, Summary: it.Summary, Tags: it.Tags, Content: it.Content, Status: it.Status}
		for _, ex := range rt.Wiki.ListEntry(it.Type, 0, 200) {
			if ex.Title == it.Title || ex.Status == longterm.StatusArchived {
				continue
			}
			if j := contentSimilarity(it.Content, ex.Content); j > 0.6 {
				wo.OverlapTitles = append(wo.OverlapTitles, ex.Title)
			}
		}
		out = append(out, wo)
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(out), "items": out})
}

// contentSimilarity CJK bigram Jaccard（与 taskbg.RunLint overlap 同口径）。
func contentSimilarity(a, b string) float64 {
	bg := func(s string) map[string]struct{} {
		rs := []rune(s)
		out := map[string]struct{}{}
		for i := 0; i+1 < len(rs); i++ {
			if isCJKRune(rs[i]) && isCJKRune(rs[i+1]) {
				out[string(rs[i:i+2])] = struct{}{}
			}
		}
		return out
	}
	A, B := bg(a), bg(b)
	if len(A) == 0 || len(B) == 0 {
		return 0
	}
	inter := 0
	for k := range A {
		if _, ok := B[k]; ok {
			inter++
		}
	}
	union := len(A) + len(B) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func isCJKRune(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF)
}

// handleReviewApprove: POST /api/review/{type}/{title}/approve
func (s *Server) handleReviewApprove(w http.ResponseWriter, r *http.Request) {
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	typ := r.PathValue("type")
	title := r.PathValue("title")
	if err := rt.Review.Approve(typ, title); err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "已批准 " + typ + "/" + title})
}

// handleReviewReject: POST /api/review/{type}/{title}/reject
func (s *Server) handleReviewReject(w http.ResponseWriter, r *http.Request) {
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	typ := r.PathValue("type")
	title := r.PathValue("title")
	if err := rt.Review.Reject(typ, title); err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "已拒绝 " + typ + "/" + title})
}

// parseType converts a string to a MemoryType.
func parseType(s string) (domain.MemoryType, bool) {
	return domain.ParseMemoryType(s)
}
