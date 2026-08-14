package shortterm

import (
	"fmt"
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

// TestChainCompaction P3：链超软上限时压缩最老 followup，保留分析骨架
// （initial/reanalysis 全留 + LastAnalysis 依赖不破）。
func TestChainCompaction(t *testing.T) {
	m := NewManager()
	m.CreateInitialCheckpoint("doc", &domain.AnalysisResult{DemandAnalysis: "首发"})
	for i := 0; i < 80; i++ {
		m.CreateFollowupCheckpoint(fmt.Sprintf("问%d", i), "答")
	}
	chain := m.CheckpointChain()
	if len(chain) > chainSoftLimit {
		t.Fatalf("链应 ≤%d，got %d", chainSoftLimit, len(chain))
	}
	// 骨架保留：initial 在链上
	sawInitial := false
	for _, cp := range chain {
		if cp.Type == domain.CheckpointInitial {
			sawInitial = true
		}
	}
	if !sawInitial {
		t.Fatal("initial 骨架被压缩掉了（LastAnalysis 依赖破坏）")
	}
	// LastAnalysis 仍可回溯
	ctx := m.BuildContext(domain.OpFollowup)
	if ctx.LastAnalysis == nil || ctx.LastAnalysis.Analysis == nil {
		t.Fatal("LastAnalysis 应跨压缩保持")
	}
}
