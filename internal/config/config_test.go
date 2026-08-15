package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestApplyDefaultsLegacyDataCompat P9 迁移：老 ./data/history.db 存在时
// 就地兼容（不搬家、不打扰——未配置 data_dir 的默认布局回落 ./data）。
func TestApplyDefaultsLegacyDataCompat(t *testing.T) {
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
	c := &Config{}
	applyDefaults(c)
	// 老布局存在：数据根回落 ./data（就地加载，无迁移）
	want := filepath.Join("data", "history.db")
	if filepath.Clean(c.HistoryDB) != want {
		t.Fatalf("老 data/ 存在应回落就地路径 %s，got %s", want, c.HistoryDB)
	}
	if filepath.Clean(c.WikiDir) != filepath.Join("data", "wiki") {
		t.Fatalf("wiki 应随数据根，got %s", c.WikiDir)
	}
}
