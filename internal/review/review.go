// Package review implements the 待审核 (pending-review) workflow for
// AI-written industry memory. AI-written threat/compliance/industry entries
// are tagged pending_review — usable in analysis with down-weighted confidence
// AND visible to humans in a review queue (ADR-005). Approved → verified,
// rejected → deleted.
package review

import (
	"fmt"

	"customer-demand-agent/internal/memory/longterm"
)

// ReviewItem is one entry awaiting human approval.
type ReviewItem struct {
	Type     string   `json:"type"`
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	Tags     []string `json:"tags"`
	Content  string   `json:"content"`
	Status   string   `json:"status"`
	Revision bool     `json:"revision,omitempty"` // P0-07：true=已验证条目的修订（批准=替换原版本）
}

// Service wraps the Wiki store's review operations.
// store 是 longterm.Store 接口（Composite）——待审/审批只落租户覆盖层，
// 系统只读基线（product）不参与审核队列。
type Service struct {
	store longterm.Store
}

// New creates a review service.
func New(store longterm.Store) *Service {
	return &Service{store: store}
}

// Pending returns all entries awaiting review.
func (s *Service) Pending() []ReviewItem {
	entries := s.store.PendingReviews()
	out := make([]ReviewItem, 0, len(entries))
	for _, e := range entries {
		out = append(out, ReviewItem{
			Type: string(e.Type), Title: e.Title, Summary: e.Summary,
			Tags: e.Tags, Content: e.Content, Status: string(e.Status),
			Revision: e.RevisionOf != "",
		})
	}
	return out
}

// Approve marks a pending entry as verified.
func (s *Service) Approve(typeStr, title string) error {
	return s.store.ApproveEntry(typeStr, title)
}

// Reject deletes a pending entry.
func (s *Service) Reject(typeStr, title string) error {
	return s.store.RejectEntry(typeStr, title)
}

// ApproveByID 是 Approve 的别名（前端可能用 id，这里 id = "type/title"）。
func (s *Service) ApproveByID(id string) error {
	t, title, ok := splitID(id)
	if !ok {
		return fmt.Errorf("非法 id: %q", id)
	}
	return s.Approve(t, title)
}

// RejectByID 是 Reject 的别名。
func (s *Service) RejectByID(id string) error {
	t, title, ok := splitID(id)
	if !ok {
		return fmt.Errorf("非法 id: %q", id)
	}
	return s.Reject(t, title)
}

// splitID 解析 "type/title" 形式的 id。
func splitID(id string) (string, string, bool) {
	for i := 0; i < len(id); i++ {
		if id[i] == '/' {
			return id[:i], id[i+1:], true
		}
	}
	return "", "", false
}
