package http_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigLeadManagerGetMasked：GET /api/config 返回 lead_manager 段，
// api_key 只出掩码（已设置 "********"，未设置为空）——与 LLM key 纪律一致。
func TestConfigLeadManagerGetMasked(t *testing.T) {
	ts, _ := setupServer(t)

	// 初始未配置 key → 空
	code, out := do(t, ts, "GET", "/api/config", nil)
	if code != 200 {
		t.Fatalf("GET /api/config: %d", code)
	}
	lm, ok := out["lead_manager"].(map[string]any)
	if !ok {
		t.Fatalf("响应应含 lead_manager 段: %v", out)
	}
	if lm["api_key"] != "" {
		t.Errorf("未配置 key 应为空，got %v", lm["api_key"])
	}
	if lm["base_url"] == "" {
		t.Error("base_url 应有默认值")
	}

	// PUT 配置 key → GET 掩码
	_, _ = do(t, ts, "PUT", "/api/config", map[string]any{
		"lead_manager": map[string]any{
			"enabled": true, "base_url": "http://mql.internal/mql", "api_key": "lm_pat_secret",
		},
	})
	code, out = do(t, ts, "GET", "/api/config", nil)
	if code != 200 {
		t.Fatalf("GET: %d", code)
	}
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "lm_pat_secret") {
		t.Fatal("真实 api_key 泄漏到 GET /api/config")
	}
	lm = out["lead_manager"].(map[string]any)
	if lm["api_key"] != "********" {
		t.Errorf("已设置 key 应显示掩码，got %v", lm["api_key"])
	}
	if lm["enabled"] != true {
		t.Error("enabled 应持久化")
	}
}

// TestConfigLeadManagerPutPartial：仅 PUT lead_manager 不动现有模型
// （部分更新语义），且回写 config.json、key 留空保留旧值。
func TestConfigLeadManagerPutPartial(t *testing.T) {
	ts, _ := setupServer(t)

	// 1. 仅 PUT lead_manager（不带 models）→ 模型应保留
	code, out := do(t, ts, "PUT", "/api/config", map[string]any{
		"lead_manager": map[string]any{"enabled": true, "api_key": "lm_pat_k1"},
	})
	if code != 200 {
		t.Fatalf("PUT lead_manager: %d %v", code, out)
	}
	if out["models"] != nil && len(out["models"].([]any)) == 0 {
		t.Fatal("部分更新不应清空模型（models=nil 语义）")
	}
	// registry 仍可探测（模型未被 reset）
	code, _ = do(t, ts, "GET", "/api/config", nil)
	if code != 200 {
		t.Fatalf("GET after partial PUT: %d", code)
	}

	// 2. 再 PUT：key 留空 → 保留已存 key；enabled 关闭
	_, out = do(t, ts, "PUT", "/api/config", map[string]any{
		"lead_manager": map[string]any{"enabled": false, "api_key": ""},
	})
	lm := out["lead_manager"].(map[string]any)
	if lm["api_key"] != "********" {
		t.Errorf("留空 key 应保留旧值（仍显示掩码），got %v", lm["api_key"])
	}
	if lm["enabled"] != false {
		t.Error("enabled=false 应生效")
	}

	// 3. 回写文件确实包含 lead_manager（api_key 落盘明文——文件本就是 0600 私密）
	// setupServer 的 config.json 在临时目录；通过 GET console 再验证 has_key
	_, out = do(t, ts, "GET", "/api/console/config", nil)
	clm, ok := out["lead_manager"].(map[string]any)
	if !ok {
		t.Fatal("console config 应含 lead_manager")
	}
	if clm["has_key"] != true {
		t.Errorf("console has_key 应为 true，got %v", clm["has_key"])
	}
	if _, exists := clm["api_key"]; exists {
		t.Error("console 视图不应含 api_key 字段（只出 has_key）")
	}
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "lm_pat") {
		t.Fatal("console 视图泄漏 api_key")
	}
}

// TestConfigLeadManagerBadURL：非法 base_url → 400 且不落盘。
func TestConfigLeadManagerBadURL(t *testing.T) {
	ts, _ := setupServer(t)
	code, _ := do(t, ts, "PUT", "/api/config", map[string]any{
		"lead_manager": map[string]any{"enabled": true, "base_url": "ftp://bad"},
	})
	if code != 400 {
		t.Errorf("非法 base_url 应 400，got %d", code)
	}
	// 未落盘：GET 仍是默认 base_url
	_, out := do(t, ts, "GET", "/api/config", nil)
	lm := out["lead_manager"].(map[string]any)
	if lm["base_url"] == "ftp://bad" {
		t.Error("非法配置不应落盘")
	}
}

// TestConfigLeadManagerPersistsToFile：PUT 后 config.json 文件含明文 key
// （回写持久化；文件权限 0600 由 config.write 保证）。
func TestConfigLeadManagerPersistsToFile(t *testing.T) {
	// setupServer 不暴露 config 路径，这里独立建服务验证文件回写
	cfgFile := filepath.Join(t.TempDir(), "config.json")
	_ = os.WriteFile(cfgFile, []byte(`{"models":[{"name":"default","endpoint":"http://localhost:1","api_key":"k","model":"m","temperature":0.3,"max_tokens":8}],"router":{"default":"default","routes":{},"fallback":{"max_retries":1,"backoff_base_ms":10,"chain":null}}}`), 0o600)

	ts2, _ := setupServerWithConfig(t, cfgFile)
	_, out := do(t, ts2, "PUT", "/api/config", map[string]any{
		"lead_manager": map[string]any{"enabled": true, "api_key": "lm_pat_file"},
	})
	if out["lead_manager"] == nil {
		t.Fatal("PUT 响应应含 lead_manager")
	}
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "lm_pat_file") {
		t.Error("lead_manager.api_key 应回写 config.json")
	}
	if !strings.Contains(string(data), "\"lead_manager\"") {
		t.Error("config.json 应含 lead_manager 段")
	}
}
