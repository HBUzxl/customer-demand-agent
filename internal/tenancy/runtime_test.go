package tenancy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/history"
	"customer-demand-agent/internal/memory/longterm"
)

// newTestRegistry 构造测试用租户注册表（空系统基线 + 临时 tenants 目录）。
func newTestRegistry(t *testing.T) (*Registry, string) {
	t.Helper()
	dir := t.TempDir()
	sysWiki := longterm.NewWikiStore(filepath.Join(dir, "system", "wiki"))
	if err := sysWiki.Load(); err != nil {
		t.Fatal(err)
	}
	cfgStore, err := config.Load(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	return NewRegistry(nil, sysWiki, filepath.Join(dir, "tenants"), cfgStore, nil), dir
}

// validHexID 返回一个合法租户 ID（32 位 hex，匹配白名单）。
func validHexID() string { return strings.Repeat("a", 32) }

// TestForTenantRejectsPathTraversal 白名单拒绝目录穿越：非法形态一律报错
// （§6.4 纵深防御）——路径穿越在数据面打开前就闭合。
func TestForTenantRejectsPathTraversal(t *testing.T) {
	reg, _ := newTestRegistry(t)
	ctx := context.Background()
	bad := []string{"../../etc", "..", "foo", "租户", "a/b", "a\\b", "", "aa"}
	for _, id := range bad {
		if _, err := reg.ForTenant(ctx, id); err == nil {
			t.Errorf("ForTenant 非法租户 ID %q 应被拒绝", id)
		}
		if err := reg.Provision(ctx, id); err == nil {
			t.Errorf("Provision 非法租户 ID %q 应被拒绝", id)
		}
		if err := reg.MigrateLegacy(ctx, id, "x.db", "x"); err == nil {
			t.Errorf("MigrateLegacy 非法租户 ID %q 应被拒绝", id)
		}
	}
}

// TestForTenantNotProvisioned 数据面未初始化（未 Provision）→ ErrNotProvisioned。
func TestForTenantNotProvisioned(t *testing.T) {
	reg, _ := newTestRegistry(t)
	_, err := reg.ForTenant(context.Background(), validHexID())
	if !errors.Is(err, ErrNotProvisioned) {
		t.Fatalf("未初始化应 ErrNotProvisioned，got %v", err)
	}
}

// TestForTenantLazyLoadAndCache Provision 后 ForTenant 懒加载缓存：同 ID 返回
// 同一 Runtime（幂等），不同租户各自独立数据面（History 实例不同 = 隔离边界）。
func TestForTenantLazyLoadAndCache(t *testing.T) {
	reg, dir := newTestRegistry(t)
	ctx := context.Background()
	ta := validHexID()
	tb := strings.Repeat("b", 32)
	if err := reg.Provision(ctx, ta); err != nil {
		t.Fatal(err)
	}
	if err := reg.Provision(ctx, tb); err != nil {
		t.Fatal(err)
	}
	rt1, err := reg.ForTenant(ctx, ta)
	if err != nil {
		t.Fatal(err)
	}
	rt2, err := reg.ForTenant(ctx, ta)
	if err != nil {
		t.Fatal(err)
	}
	if rt1 != rt2 {
		t.Fatal("同租户应缓存同一 Runtime")
	}
	rtB, err := reg.ForTenant(ctx, tb)
	if err != nil {
		t.Fatal(err)
	}
	if rt1 == rtB {
		t.Fatal("不同租户应为独立 Runtime")
	}
	if rt1.History.TenantID() != ta || rtB.History.TenantID() != tb {
		t.Fatalf("History 应绑定各自租户: %s vs %s", rt1.History.TenantID(), rtB.History.TenantID())
	}
	// 数据面落盘位置（tenant.json 三方核对）
	if _, err := os.Stat(filepath.Join(dir, "tenants", ta, "tenant.json")); err != nil {
		t.Fatalf("租户目录应含 tenant.json: %v", err)
	}
	// 数据隔离：A 写会话，B 不可见
	_ = rt1.History.EnsureSession("", "shared-id", "A的会话", "")
	if _, err := rtB.History.GetSession(nil, "shared-id", ""); err == nil {
		t.Fatal("A 的会话不应出现在 B 的 history.db（数据面隔离）")
	}
}

// TestProvisionIdempotent 重复 Provision 幂等（active 目录已存在 → 直接成功）。
func TestProvisionIdempotent(t *testing.T) {
	reg, _ := newTestRegistry(t)
	ctx := context.Background()
	id := validHexID()
	if err := reg.Provision(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := reg.Provision(ctx, id); err != nil {
		t.Fatalf("重复 Provision 应幂等: %v", err)
	}
	// 中断残留（.staging）不污染：放一个假 staging，Provision 仍能清理重做
	_ = os.MkdirAll(filepath.Join(reg.TenantsDir(), id+".staging"), 0o700)
	if err := reg.Provision(ctx, id); err != nil {
		t.Fatalf("staging 残留应被清理重做: %v", err)
	}
}

// TestCloseTenantAndReopen CloseTenant 后重新 ForTenant 重建（进程回收语义）。
func TestCloseTenantAndReopen(t *testing.T) {
	reg, _ := newTestRegistry(t)
	ctx := context.Background()
	id := validHexID()
	if err := reg.Provision(ctx, id); err != nil {
		t.Fatal(err)
	}
	rt, err := reg.ForTenant(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	_ = rt.History.EnsureSession("", "s1", "标题", "")
	if err := reg.CloseTenant(id); err != nil {
		t.Fatal(err)
	}
	rt2, err := reg.ForTenant(ctx, id)
	if err != nil {
		t.Fatalf("CloseTenant 后应可重新加载: %v", err)
	}
	// 数据持久（同一 history.db），会话仍在
	if _, err := rt2.History.GetSession(nil, "s1", ""); err != nil {
		t.Fatalf("重开后会话应保留: %v", err)
	}
}

// TestMigrateLegacy 旧单租户数据迁入租户数据面：历史库会话原样迁移；
// wiki 只搬 用户记忆（客户），产品不落租户覆盖层；幂等。
func TestMigrateLegacy(t *testing.T) {
	dir := t.TempDir()
	// 旧 wiki：产品记忆 + 用户记忆/客户
	legacyWiki := filepath.Join(dir, "legacy-wiki")
	_ = os.MkdirAll(filepath.Join(legacyWiki, "产品记忆"), 0o755)
	_ = os.MkdirAll(filepath.Join(legacyWiki, "用户记忆", "客户"), 0o755)
	_ = os.WriteFile(filepath.Join(legacyWiki, "产品记忆", "雷池.md"), []byte("---\ntype: product\ntitle: 雷池\n---\nx"), 0o644)
	_ = os.WriteFile(filepath.Join(legacyWiki, "用户记忆", "客户", "某集团.md"), []byte("---\ntype: customer\ntitle: 某集团\n---\n客户画像"), 0o644)
	// 旧 history.db：真实 SQLite 含会话
	legacyDB := filepath.Join(dir, "history.db")
	h, err := history.Open(legacyDB)
	if err != nil {
		t.Fatal(err)
	}
	_ = h.EnsureSession("", "legacy-sess", "旧会话", "")
	if _, err := h.AppendMessage("legacy-sess", "user", "你好", ""); err != nil {
		t.Fatal(err)
	}
	if err := h.Close(); err != nil {
		t.Fatal(err)
	}

	reg, _ := newTestRegistry(t)
	ctx := context.Background()
	id := validHexID()
	if err := reg.MigrateLegacy(ctx, id, legacyDB, legacyWiki); err != nil {
		t.Fatal(err)
	}
	// 幂等（active 目录已存在）
	if err := reg.MigrateLegacy(ctx, id, legacyDB, legacyWiki); err != nil {
		t.Fatalf("重复迁移应幂等: %v", err)
	}

	rt, err := reg.ForTenant(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	// 会话迁入（消息 1 条）
	det, err := rt.History.GetSession(nil, "legacy-sess", "")
	if err != nil || len(det.Messages) != 1 {
		t.Fatalf("旧会话应迁入租户历史库: %v (%d)", err, len(det.Messages))
	}
	// 客户记忆进租户 wiki
	if _, err := rt.Wiki.GetEntry("customer", "某集团"); err != nil {
		t.Fatalf("客户记忆应迁入租户 wiki: %v", err)
	}
	// 产品不进租户覆盖层（GetEntry product 走系统层，空 → 不存在）
	if _, err := rt.Wiki.GetEntry("product", "雷池"); err == nil {
		t.Fatal("产品记忆不应落租户覆盖层（系统基线另走播种）")
	}
}
