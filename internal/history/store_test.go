package history

import (
	"path/filepath"
	"testing"

	"customer-demand-agent/internal/domain"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// TestCrossTenantCheckpointIsolation 验证跨租户隔离（核心数据泄漏修复）：
// 租户 B 用租户 A 的 session_id：① EnsureSession 被拒（防占用/续传）；② 读不到 A 的 checkpoint。
func TestCrossTenantCheckpointIsolation(t *testing.T) {
	s := openTestStore(t)

	// A 建会话 + 写 checkpoint
	if err := s.EnsureSession("tenantA", "sess-shared", "A的需求", ""); err != nil {
		t.Fatalf("A EnsureSession: %v", err)
	}
	cp := &domain.Checkpoint{
		ID: "cp1", Type: domain.CheckpointInitial, Document: "A的机密文档",
		Analysis: &domain.AnalysisResult{DemandAnalysis: "A的结论"},
	}
	if err := s.AppendCheckpoint("tenantA", "sess-shared", cp); err != nil {
		t.Fatalf("A AppendCheckpoint: %v", err)
	}

	// B 用 A 的 session_id EnsureSession → 应被拒
	if err := s.EnsureSession("tenantB", "sess-shared", "B想劫持", ""); err == nil {
		t.Error("租户 B 用 A 的 session_id EnsureSession 应失败（跨租户拒绝），got nil")
	}

	// B 直接读 A 的 checkpoint → 应空（JOIN sessions 校验）
	cps, err := s.ListCheckpoints("tenantB", "sess-shared")
	if err != nil {
		t.Fatalf("B ListCheckpoints: %v", err)
	}
	if len(cps) != 0 {
		t.Errorf("B 不应读到 A 的 checkpoint，got %d 条", len(cps))
	}

	// 回归：A 自己能读到
	cpsA, err := s.ListCheckpoints("tenantA", "sess-shared")
	if err != nil {
		t.Fatalf("A ListCheckpoints: %v", err)
	}
	if len(cpsA) != 1 || cpsA[0].ID != "cp1" {
		t.Errorf("A 应读到自己的 1 条 checkpoint，got %+v", cpsA)
	}
}

// TestEnsureSessionSameTenantUpdate 验证同租户重复 EnsureSession 正常更新（不误拒）。
func TestEnsureSessionSameTenantUpdate(t *testing.T) {
	s := openTestStore(t)
	if err := s.EnsureSession("t1", "s1", "标题1", ""); err != nil {
		t.Fatalf("首次: %v", err)
	}
	if err := s.EnsureSession("t1", "s1", "标题2", "客户X"); err != nil {
		t.Fatalf("同租户更新: %v", err)
	}
	det, err := s.GetSession("t1", "s1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if det.Session.Title != "标题2" || det.Session.Customer != "客户X" {
		t.Errorf("同租户更新未生效: %+v", det.Session)
	}
}
