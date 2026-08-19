package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreGetReturnsDeepSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	snap := s.Get()
	snap.DefaultUser = "外部修改"
	snap.Router.Routes["analysis"] = "外部模型"
	if len(snap.Models) > 0 {
		snap.Models[0].Name = "外部模型"
	}
	got := s.Get()
	if got.DefaultUser == "外部修改" || got.Router.Routes["analysis"] == "外部模型" ||
		(len(got.Models) > 0 && got.Models[0].Name == "外部模型") {
		t.Fatal("Get 返回值不应共享 Store 内部状态")
	}
}

func TestStoreFailedUpdateKeepsPreviousSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	badTarget := filepath.Join(t.TempDir(), "existing-dir")
	if err := os.MkdirAll(badTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	s.path = badTarget // write 的 rename 不能覆盖现有目录，稳定触发失败。
	before := s.Get().DefaultUser
	if err := s.Update(func(c *Config) { c.DefaultUser = "不应提交" }); err == nil {
		t.Fatal("写盘失败场景应返回错误")
	}
	if got := s.Get().DefaultUser; got != before {
		t.Fatalf("写盘失败后内存配置不应变化: got %q want %q", got, before)
	}
}

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

// TestLeadManagerDefaults：lead_manager 段缺省时填默认值（老 config.json
// 无此段 → 零值 → applyDefaults 补 base_url/timeout/rate/burst，enabled=false 不接入）。
func TestLeadManagerDefaults(t *testing.T) {
	c := &Config{}
	applyDefaults(c)
	if c.LeadManager.Enabled {
		t.Error("缺省不应启用")
	}
	if c.LeadManager.BaseURL != "http://api.in.chaitin.net/mql" {
		t.Errorf("base_url 默认值不对: %s", c.LeadManager.BaseURL)
	}
	if c.LeadManager.TimeoutSec != 10 || c.LeadManager.RatePerMin != 60 || c.LeadManager.Burst != 3 {
		t.Errorf("timeout/rate/burst 默认值不对: %+v", c.LeadManager)
	}

	// 模板包含 lead_manager 段（api_key 留空）
	tpl := template()
	if tpl.LeadManager.Enabled {
		t.Error("模板不应默认启用")
	}
	if tpl.LeadManager.APIKey != "" {
		t.Error("模板 api_key 必须留空")
	}
}
