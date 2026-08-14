package config

import (
	"testing"
)

func TestApplyDefaultsDataDir(t *testing.T) {
	t.Setenv("CDA_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", "")

	t.Run("env override wins", func(t *testing.T) {
		t.Setenv("CDA_DATA_DIR", "/tmp/cda-env-test")
		cfg := &Config{}
		applyDefaults(cfg)
		if cfg.DataDir != "/tmp/cda-env-test" {
			t.Fatalf("env 应最优先，got %s", cfg.DataDir)
		}
		if cfg.WikiDir != "/tmp/cda-env-test/wiki" || cfg.HistoryDB != "/tmp/cda-env-test/history.db" {
			t.Fatalf("默认布局应重定向到数据根: %s %s", cfg.WikiDir, cfg.HistoryDB)
		}
	})

	t.Run("explicit config respected", func(t *testing.T) {
		t.Setenv("CDA_DATA_DIR", "/tmp/cda-env-ignored")
		cfg := &Config{WikiDir: "/my/wiki", HistoryDB: "/my/h.db"}
		applyDefaults(cfg)
		if cfg.WikiDir != "/my/wiki" || cfg.HistoryDB != "/my/h.db" {
			t.Fatalf("显式配置应被尊重: %s %s", cfg.WikiDir, cfg.HistoryDB)
		}
	})

	t.Run("xdg default", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "/tmp/xdg-root")
		cfg := &Config{}
		applyDefaults(cfg)
		if cfg.DataDir != "/tmp/xdg-root/customer-demand-agent" {
			t.Fatalf("XDG 默认数据根不对: %s", cfg.DataDir)
		}
	})
}
