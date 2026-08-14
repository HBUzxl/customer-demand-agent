package shortterm

import (
	"testing"

	"customer-demand-agent/internal/domain"
)

// TestRestoreChain 验证断点续传：从持久化的 checkpoint 链重建短期记忆。
func TestRestoreChain(t *testing.T) {
	cp1 := &domain.Checkpoint{ID: "cp1", Type: domain.CheckpointInitial, Document: "客户A文档", Analysis: &domain.AnalysisResult{DemandAnalysis: "首次结论"}}
	cp2 := &domain.Checkpoint{ID: "cp2", Type: domain.CheckpointFollowup, PrevID: "cp1", Question: "追问1", Answer: "回答1"}

	sm := NewSessionManager()
	sm.Restore("sess-x", []*domain.Checkpoint{cp1, cp2})

	m := sm.Get("sess-x")
	if !m.HasHistory() {
		t.Fatal("恢复后应有历史")
	}
	if m.CurrentCheckpoint().ID != "cp2" {
		t.Fatalf("current 应为 cp2，got %s", m.CurrentCheckpoint().ID)
	}
	if m.PreviousCheckpoint().ID != "cp1" {
		t.Fatalf("previous 应为 cp1")
	}
	// 恢复后 DetermineOp 应识别为 followup（同文档）
	if op := DetermineOp(m, "客户A文档"); op != domain.OpFollowup {
		t.Fatalf("恢复后同文档追问应为 followup，got %v", op)
	}
	// 前一个 checkpoint 的分析结论应保留（followup 通过 PrevID 引用它）
	ctx := m.BuildContext(domain.OpFollowup)
	if ctx.Current == nil || ctx.Current.Question != "追问1" {
		t.Fatalf("恢复的 followup checkpoint 丢失: %+v", ctx.Current)
	}
	if ctx.Previous == nil || ctx.Previous.Analysis == nil || ctx.Previous.Analysis.DemandAnalysis != "首次结论" {
		t.Fatalf("恢复的 initial checkpoint 分析结论丢失: %+v", ctx.Previous)
	}
}
