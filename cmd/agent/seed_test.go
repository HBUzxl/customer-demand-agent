package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSeedWikiMigratesLegacyData P9 真迁移：老 ./data 存在 + 数据根空 →
// history.db 与 wiki/ 全量搬入数据根；原位保留（可回退）；幂等（第二次不重复搬）。
func TestSeedWikiMigratesLegacyData(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	dataRoot := filepath.Join(dir, "xdgroot")
	// 老布局
	if err := os.MkdirAll("data/wiki/行业记忆", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("data/history.db", []byte("olddb"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("data/wiki/行业记忆/x.md", []byte("---\ntype: industry\ntitle: x\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	seedWiki(dataRoot, filepath.Join(dataRoot, "wiki"), false)
	// 迁移产物
	if _, err := os.Stat(filepath.Join(dataRoot, "history.db")); err != nil {
		t.Fatal("history.db 应迁入数据根")
	}
	b, _ := os.ReadFile(filepath.Join(dataRoot, "wiki/行业记忆/x.md"))
	if string(b) == "" {
		t.Fatal("wiki 内容应随迁")
	}
	// 原位保留
	if _, err := os.Stat("data/history.db"); err != nil {
		t.Fatal("原位应保留（回退路径）")
	}
	// 幂等：数据根已有 history.db 后再跑不覆盖（内容不变）
	if err := os.WriteFile(filepath.Join(dataRoot, "history.db"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	seedWiki(dataRoot, filepath.Join(dataRoot, "wiki"), false)
	b2, _ := os.ReadFile(filepath.Join(dataRoot, "history.db"))
	if string(b2) != "new" {
		t.Fatal("已初始化的数据根不应被旧数据覆盖")
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
