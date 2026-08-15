package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestApplyDefaultsLegacyDataNoFallback P9 迁移语义：老 ./data 不再决定
// 数据根（迁移由 main.seedWiki 负责）——默认布局始终指向 XDG 数据根。
func TestApplyDefaultsLegacyDataNoFallback(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("data/history.db", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "xdg"))
	c := &Config{}
	applyDefaults(c)
	// 老布局存在但数据根仍是 XDG（迁移交给 seedWiki）
	want := filepath.Join(dir, "xdg", "customer-demand-agent")
	if filepath.Clean(c.DataDir) != want {
		t.Fatalf("数据根应 XDG %s，got %s", want, c.DataDir)
	}
}
