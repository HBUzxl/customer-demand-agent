package http

import (
	"net/http"

	"customer-demand-agent/internal/domain"
)

// handleReviewPending: GET /api/review/pending
func (s *Server) handleReviewPending(w http.ResponseWriter, r *http.Request) {
	items := s.review.Pending()
	writeJSON(w, http.StatusOK, map[string]any{"count": len(items), "items": items})
}

// handleReviewApprove: POST /api/review/{type}/{title}/approve
func (s *Server) handleReviewApprove(w http.ResponseWriter, r *http.Request) {
	typ := r.PathValue("type")
	title := r.PathValue("title")
	if err := s.review.Approve(typ, title); err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "已批准 " + typ + "/" + title})
}

// handleReviewReject: POST /api/review/{type}/{title}/reject
func (s *Server) handleReviewReject(w http.ResponseWriter, r *http.Request) {
	typ := r.PathValue("type")
	title := r.PathValue("title")
	if err := s.review.Reject(typ, title); err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "已拒绝 " + typ + "/" + title})
}

// parseType converts a string to a MemoryType.
func parseType(s string) (domain.MemoryType, bool) {
	return domain.ParseMemoryType(s)
}
