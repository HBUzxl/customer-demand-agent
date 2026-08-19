package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/history"
	"customer-demand-agent/internal/identity"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/tenancy"
)

// TestLegacyMigrationToTenant 存量单租户数据自动迁入 Legacy 租户（阶段 5 引导）：
// 旧 history.db（真实 SQLite 含会话）+ 旧 wiki（产品记忆 + 用户记忆）→
// 会话迁入租户历史库、客户/使用者记忆进租户 wiki、产品记忆进系统基线；
// 系统层不携带客户记忆；迁移幂等（重复调用返回同一租户）。
func TestLegacyMigrationToTenant(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// 1) 旧单租户数据：history.db（真实 SQLite）+ wiki（产品记忆 + 用户记忆）
	if err := os.MkdirAll("data/wiki/产品记忆/安全平台", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("data/wiki/用户记忆/客户", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("data/wiki/产品记忆/安全平台/雷池.md", []byte("---\ntype: product\ntitle: 雷池\n---\n\n产品正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("data/wiki/用户记忆/客户/某集团.md", []byte("---\ntype: customer\ntitle: 某集团\n---\n\n客户画像"), 0o644); err != nil {
		t.Fatal(err)
	}
	hist, err := history.Open("data/history.db")
	if err != nil {
		t.Fatal(err)
	}
	_ = hist.EnsureSession("", "sess_legacy1", "旧会话", "")
	if _, err := hist.AppendMessage("sess_legacy1", "user", "你好", ""); err != nil {
		t.Fatal(err)
	}
	if err := hist.Close(); err != nil {
		t.Fatal(err)
	}

	// 2) 平台装配：系统 wiki 播种（无 ./wiki 种子，仅叠加旧库系统层）+ 身份控制面
	store, err := config.Load(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	idStore, err := identity.Open(filepath.Join(dir, "control", "identity.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer idStore.Close()
	seedSystemWiki(filepath.Join(dir, "system", "wiki"), "data/wiki")
	sysWiki := longterm.NewWikiStore(filepath.Join(dir, "system", "wiki"))
	if err := sysWiki.Load(); err != nil {
		t.Fatal(err)
	}
	reg := tenancy.NewRegistry(nil, sysWiki, filepath.Join(dir, "tenants"), store, nil)

	// 3) 迁移 + 幂等（重复调用返回同一租户）
	ten, err := ensureLegacyTenant(ctx, idStore, reg, "data/history.db", "data/wiki")
	if err != nil {
		t.Fatal(err)
	}
	if ten.Slug != "legacy" || ten.Status != identity.TenantStatusActive {
		t.Fatalf("Legacy 租户应 active（slug=legacy）: %+v", ten)
	}
	ten2, err := ensureLegacyTenant(ctx, idStore, reg, "data/history.db", "data/wiki")
	if err != nil {
		t.Fatal(err)
	}
	if ten2.ID != ten.ID {
		t.Fatalf("幂等应返回同一租户，got %s != %s", ten2.ID, ten.ID)
	}

	// 4) 租户数据面：旧会话迁入 + 客户记忆进租户 wiki
	rt, err := reg.ForTenant(ctx, ten.ID)
	if err != nil {
		t.Fatal(err)
	}
	det, err := rt.History.GetSession(nil, "sess_legacy1", "")
	if err != nil {
		t.Fatalf("旧会话应迁入租户历史库: %v", err)
	}
	if len(det.Messages) != 1 {
		t.Fatalf("迁移后会话应保留消息，got %d", len(det.Messages))
	}
	cust, err := rt.Wiki.GetEntry("customer", "某集团")
	if err != nil {
		t.Fatalf("客户记忆应迁入租户 wiki: %v", err)
	}
	if !strings.Contains(cust.Content, "客户画像") {
		t.Fatalf("客户画像内容应保留: %q", cust.Content)
	}

	// 5) 系统知识：产品记忆进系统基线；系统层不携带客户记忆
	if _, err := sysWiki.GetEntry("product", "雷池"); err != nil {
		t.Fatalf("产品记忆应进系统基线: %v", err)
	}
	if _, err := sysWiki.GetEntry("customer", "某集团"); err == nil {
		t.Fatal("系统层不应有客户记忆")
	}
}

// TestExtractObserves 固化输入抽取：### 观察 标题段与其后续列表段一起取
// （曾只取标题段——列表注记被 \n\n 切走，LLM 固化读到空）。
func TestExtractObserves(t *testing.T) {
	c := "---\ntype: industry\n---\n\n正文。\n\n### 观察\n\n- [high] 银行问过大模型训练数据出境\n- [medium] 券商关心投研 LLM 生成合规\n\n### 其他\n\n别的段落"
	obs := extractObserves(c)
	if len(obs) != 1 {
		t.Fatalf("应 1 段观察，got %d", len(obs))
	}
	if !strings.Contains(obs[0], "银行") || !strings.Contains(obs[0], "券商") {
		t.Fatalf("观察列表项应包含在段内: %q", obs[0])
	}
	if strings.Contains(obs[0], "别的段落") {
		t.Fatal("不应吞掉下一标题的段落")
	}
}

func TestSeedSystemWikiRefreshesExistingVersionedKnowledge(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	seedPage := filepath.Join("wiki", "产品记忆", "端点安全", "谛听.md")
	systemPage := filepath.Join("system", "wiki", "产品记忆", "端点安全", "谛听.md")
	if err := os.MkdirAll(filepath.Dir(seedPage), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(systemPage), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(seedPage, []byte("当前版本：蜜罐与欺骗防御"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(systemPage, []byte("旧版本：NDR"), 0o644); err != nil {
		t.Fatal(err)
	}
	seedSystemWiki(filepath.Join(dir, "system", "wiki"), "")
	got, err := os.ReadFile(systemPage)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "当前版本：蜜罐与欺骗防御" {
		t.Fatalf("已有 system wiki 也应随版本种子刷新，got %q", got)
	}
}
