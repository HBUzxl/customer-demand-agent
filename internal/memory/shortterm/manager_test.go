package shortterm

import (
	"testing"

	"customer-demand-agent/internal/domain"
)

func TestCheckpointChain(t *testing.T) {
	m := NewManager()

	if m.HasHistory() {
		t.Fatal("new manager should have no history")
	}

	// op 判断：无历史 → initial
	if op := DetermineOp(m, "文档A"); op != domain.OpInitial {
		t.Fatalf("expected OpInitial, got %v", op)
	}

	// 创建 initial checkpoint
	result := &domain.AnalysisResult{DemandAnalysis: "首次分析"}
	cp1 := m.CreateInitialCheckpoint("文档A原文", result)
	if cp1.Type != domain.CheckpointInitial || cp1.Document != "文档A原文" {
		t.Fatalf("initial checkpoint wrong: %+v", cp1)
	}
	if !m.HasHistory() {
		t.Fatal("should have history after initial")
	}

	// 同文档追问 → followup
	if op := DetermineOp(m, "文档A原文"); op != domain.OpFollowup {
		t.Fatalf("same doc should be OpFollowup, got %v", op)
	}
	cp2 := m.CreateFollowupCheckpoint("雷池怎么报价？", "雷池标准版...")
	if cp2.Type != domain.CheckpointFollowup || cp2.PrevID != cp1.ID {
		t.Fatalf("followup wrong: %+v", cp2)
	}

	// 换文档 → reanalysis
	if op := DetermineOp(m, "完全不同的文档B"); op != domain.OpReanalysis {
		t.Fatalf("different doc should be OpReanalysis, got %v", op)
	}
	cp3 := m.CreateReanalysisCheckpoint("文档B原文", &domain.AnalysisResult{})
	if cp3.Type != domain.CheckpointReanalysis || cp3.PrevID != cp2.ID {
		t.Fatalf("reanalysis wrong: %+v", cp3)
	}

	// 链完整性
	chain := m.CheckpointChain()
	if len(chain) != 3 {
		t.Fatalf("expected chain length 3, got %d", len(chain))
	}
	if m.CurrentCheckpoint().ID != cp3.ID {
		t.Fatal("current should be cp3")
	}
	if m.PreviousCheckpoint().ID != cp2.ID {
		t.Fatal("previous should be cp2")
	}
}

func TestNotes(t *testing.T) {
	m := NewManager()
	m.AppendNote("观察1")
	m.AppendNote("观察2")

	// 创建 checkpoint 时消费 notes
	cp := m.CreateInitialCheckpoint("doc", &domain.AnalysisResult{})
	if len(cp.Notes) != 2 || cp.Notes[0] != "观察1" {
		t.Fatalf("notes not consumed into checkpoint: %+v", cp.Notes)
	}

	// 消费后清空
	if drained := m.DrainNotes(); drained != nil {
		t.Fatalf("notes should be empty after drain, got %v", drained)
	}
}

func TestBuildContext(t *testing.T) {
	m := NewManager()
	m.CreateInitialCheckpoint("文档原文比较长...", &domain.AnalysisResult{DemandAnalysis: "x"})

	ctx := m.BuildContext(domain.OpFollowup)
	if ctx.Op != domain.OpFollowup {
		t.Fatal("op mismatch")
	}
	if ctx.Current == nil {
		t.Fatal("current checkpoint missing")
	}
}
