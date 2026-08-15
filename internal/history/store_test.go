package history

import (
	"database/sql"
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

// TestCheckpointRoundTrip de-tenancy 后：checkpoint 按 session 存取（无租户维度）。
func TestCheckpointRoundTrip(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("sess-x", "标题", "")
	cp := &domain.Checkpoint{ID: "cp-1", Type: domain.CheckpointInitial}
	if err := s.AppendCheckpoint("sess-x", cp); err != nil {
		t.Fatal(err)
	}
	cps, err := s.ListCheckpoints("sess-x")
	if err != nil || len(cps) != 1 {
		t.Fatalf("按 session 取回 checkpoint 失败: %v (%d)", err, len(cps))
	}
}

// TestEnsureSessionSameTenantUpdate 验证同租户重复 EnsureSession 正常更新（不误拒）。
func TestEnsureSessionSameTenantUpdate(t *testing.T) {
	s := openTestStore(t)
	if err := s.EnsureSession("s1", "标题1", ""); err != nil {
		t.Fatalf("首次: %v", err)
	}
	if err := s.EnsureSession("s1", "标题2", "客户X"); err != nil {
		t.Fatalf("同租户更新: %v", err)
	}
	det, err := s.GetSession("s1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if det.Session.Title != "标题2" || det.Session.Customer != "客户X" {
		t.Errorf("同租户更新未生效: %+v", det.Session)
	}
}

// TestTenantColumnMigration de-tenancy 契约：老库带 NOT NULL tenant_id 列时
// Open 一次性重建去列（数据原样保留，幂等）。
func TestTenantColumnMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "old.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// 老形态 schema（de-tenancy 前）
	if _, err := db.Exec(`CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    tenant_id  TEXT NOT NULL,
    title      TEXT,
    customer   TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sessions(session_id, tenant_id, title, customer, created_at, updated_at)
		VALUES('legacy-1', 'corp-a', '老会话', '', '2026-01-01', '2026-01-01')`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Open 触发迁移
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open 应完成迁移: %v", err)
	}
	defer s.Close()
	// 1) 列已消失
	rows, err := s.db.Query(`PRAGMA table_info(sessions)`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull int
		var dflt any
		var pk int
		_ = rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk)
		if name == "tenant_id" {
			rows.Close()
			t.Fatal("迁移后 tenant_id 列应消失")
		}
	}
	rows.Close()
	// 2) 数据保留
	if err := s.EnsureSession("legacy-1", "老会话", ""); err != nil {
		t.Fatal(err)
	}
	sessions, err := s.ListSessions(10, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, it := range sessions {
		if it.SessionID == "legacy-1" {
			found = true
		}
	}
	if !found {
		t.Fatal("迁移应保留老会话数据")
	}
	// 3) 新会话正常写入（无列 INSERT）
	if err := s.EnsureSession("new-1", "新会话", ""); err != nil {
		t.Fatal(err)
	}
}
