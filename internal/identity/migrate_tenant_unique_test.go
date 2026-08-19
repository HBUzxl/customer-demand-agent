package identity

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// TestMigrateTenantsUniqueDedupes §4.1 严重缺陷修复：
// 存量重复 name/slug 被幂等解析（保留最早创建者，后建追加前 8 位 ID），
// 唯一索引建立后禁止新重复（注册并发竞态兜底）。
func TestMigrateTenantsUniqueDedupes(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "identity.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE tenants (
		id         TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		slug       TEXT NOT NULL,
		status     TEXT NOT NULL DEFAULT 'provisioning',
		created_by TEXT,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	now := "2026-08-18 00:00:00 +0000 UTC"
	// 两个「内测」租户（id 升序确定 keep 者）+ 一个唯一租户。
	seed := []struct{ id, name, slug, ts string }{
		{"11111111111111111111111111111111", "内测", "tenant", now},
		{"22222222222222222222222222222222", "内测", "tenant", now},
		{"33333333333333333333333333333333", "Acme", "acme", now},
	}
	for _, r := range seed {
		if _, err := db.Exec(`INSERT INTO tenants (id,name,slug,status,created_at,updated_at)
			VALUES (?,?,?,?,?,?)`, r.id, r.name, r.slug, "active", r.ts, r.ts); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	// Open 触发迁移（schema + migrateUsersPlatformAdmin + migrateTenantsUnique）。
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// 解析后：name / slug 全部唯一；后建「内测」→「内测 · 22222222」。
	names, slugs := map[string]bool{}, map[string]bool{}
	rows, err := store.db.Query(`SELECT name, slug FROM tenants`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var n, s string
		if err := rows.Scan(&n, &s); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		if names[n] {
			t.Fatalf("租户名仍重复: %s", n)
		}
		if slugs[s] {
			t.Fatalf("slug 仍重复: %s", s)
		}
		names[n], slugs[s] = true, true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !names["内测"] || !names["内测 · 22222222"] || !names["Acme"] {
		t.Fatalf("改名结果不符: %v", names)
	}
	if !slugs["tenant"] || !slugs["tenant-22222222"] || !slugs["acme"] {
		t.Fatalf("slug 改名结果不符: %v", slugs)
	}

	// 唯一索引兜底：插入重复名 / 重复 slug 均被拒绝。
	if _, err := store.db.Exec(`INSERT INTO tenants (id,name,slug,status,created_at,updated_at)
		VALUES (?,?,?,?,?,?)`, "44444444444444444444444444444444", "Acme", "acme-x", "active", now, now); err == nil {
		t.Fatal("插入重复租户名应被唯一索引拒绝")
	}
	if _, err := store.db.Exec(`INSERT INTO tenants (id,name,slug,status,created_at,updated_at)
		VALUES (?,?,?,?,?,?)`, "55555555555555555555555555555555", "Acme-x", "acme", "active", now, now); err == nil {
		t.Fatal("插入重复 slug 应被唯一索引拒绝")
	}
	// 正常插入仍可。
	if _, err := store.db.Exec(`INSERT INTO tenants (id,name,slug,status,created_at,updated_at)
		VALUES (?,?,?,?,?,?)`, "66666666666666666666666666666666", "新组织", "new", "active", now, now); err != nil {
		t.Fatalf("正常插入应成功: %v", err)
	}
}

// TestMigrateTenantsUniqueIdempotent 已建唯一索引 → 幂等返回，不改数据。
func TestMigrateTenantsUniqueIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "identity.db")
	if _, err := Open(dbPath); err != nil {
		t.Fatal(err)
	}
	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ok, err := indexExists(store.db, "idx_tenants_name")
	if err != nil || !ok {
		t.Fatalf("唯一索引应存在: ok=%v err=%v", ok, err)
	}
	// 幂等：再跑一次迁移不报错。
	if err := migrateTenantsUnique(store.db); err != nil {
		t.Fatalf("二次迁移应幂等成功: %v", err)
	}
}
