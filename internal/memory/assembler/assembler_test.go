package assembler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/memory/longterm"
)

func TestGradedOutputByUserLevel(t *testing.T) {
	dir := t.TempDir()
	// 写一个高级使用者画像
	_ = os.MkdirAll(filepath.Join(dir, "用户记忆", "使用者"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "用户记忆", "使用者", "张三.md"),
		[]byte("---\ntype: user\ntitle: 张三\nlevel: 高级\n---\nx"), 0o644)

	store := longterm.NewWikiStore(dir)
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}

	a := New(store, "张三")
	msgs := a.Assemble(domain.OpInitial, nil, "测试输入")
	sys := msgs[0].Content

	if !strings.Contains(sys, "输出风格") {
		t.Fatalf("system prompt 应含输出风格段落，got:\n%s", sys)
	}
	if !strings.Contains(sys, "竞品") {
		t.Fatalf("高级销售应含竞品话术提示，got:\n%s", sys)
	}
}

func TestNoUserProfileNoGradedOutput(t *testing.T) {
	store := longterm.NewWikiStore(t.TempDir())
	_ = store.Load()
	a := New(store, "不存在的人")
	msgs := a.Assemble(domain.OpInitial, nil, "x")
	if strings.Contains(msgs[0].Content, "输出风格") {
		t.Fatal("无使用者画像时不应有输出风格段落")
	}
}

// TestLoadTemplateExternalBodyEffective C3 契约：外置文件四段正文直接驱动
// prompt——修改文件正文后 loadTemplate 输出随之变化（编辑重启生效）。
func TestLoadTemplateExternalBodyEffective(t *testing.T) {
	dir := t.TempDir()
	// 标准四段模板
	tmpl1 := "# C3 模板（## 段名组织；编辑后重启生效）\n\n## 角色\n\n你是原版角色。\n\n## 自主性指引\n\n原版自主性。\n\n## 目标\n\n原版目标。\n\n## 约束\n\n原版约束。\n"
	if err := os.WriteFile(filepath.Join(dir, "system.md"), []byte(tmpl1), 0o644); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	defer os.Chdir(old)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("prompts", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("prompts/system.md", []byte(tmpl1), 0o644); err != nil {
		t.Fatal(err)
	}
	got1 := loadTemplate()
	if !strings.Contains(got1, "你是原版角色。") || !strings.Contains(got1, "原版自主性。") {
		t.Fatalf("外置正文应直接生效: %q", got1)
	}
	if !strings.Contains(got1, "## 你是自主的") {
		t.Fatalf("已知段名应映射为标准标题锚: %q", got1)
	}
	// 编辑正文 → 输出变化
	tmpl2 := "# C3 模板（## 段名组织；编辑后重启生效）\n\n## 角色\n\n你是定制角色：金融行业专家。\n\n## 自主性指引\n\n定制自主性。\n\n## 目标\n\n原版目标。\n\n## 约束\n\n原版约束。\n"
	if err := os.WriteFile("prompts/system.md", []byte(tmpl2), 0o644); err != nil {
		t.Fatal(err)
	}
	got2 := loadTemplate()
	if !strings.Contains(got2, "你是定制角色：金融行业专家。") {
		t.Fatal("编辑外置正文后 loadTemplate 输出应变化")
	}
	if strings.Contains(got2, "你是原版角色。") {
		t.Fatal("旧正文不应残留")
	}
}
