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
