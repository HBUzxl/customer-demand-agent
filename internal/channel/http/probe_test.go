package http_test

import (
	"bytes"
	"net/http"
	"testing"
)

// postProbeJSON 向 path 发一个 JSON POST，返回状态码。
func postProbeJSON(t *testing.T, tsURL, path, jsonBody string) int {
	t.Helper()
	req, _ := http.NewRequest("POST", tsURL+path, nopCloser{bytes.NewReader([]byte(jsonBody))})
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", path, err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// TestProbeConfigTestRejectsSSRF 验证 /api/config/test 拒绝云元数据/回环地址（C1 SSRF）。
func TestProbeConfigTestRejectsSSRF(t *testing.T) {
	ts, _ := setupServer(t)
	for _, endpoint := range []string{
		"http://169.254.169.254/latest/meta-data/", // 云元数据
		"http://127.0.0.1:6379/",                   // 回环
	} {
		body := `{"endpoint":"` + endpoint + `","api_key":"x","model":"m","protocol":"openai-chat"}`
		if code := postProbeJSON(t, ts.URL, "/api/config/test", body); code != http.StatusBadRequest {
			t.Errorf("/api/config/test endpoint %s 应被拒（400），got %d", endpoint, code)
		}
	}
}

// TestProbeListModelsRejectsSSRF 验证 /api/models 同样拒绝元数据/回环。
func TestProbeListModelsRejectsSSRF(t *testing.T) {
	ts, _ := setupServer(t)
	for _, endpoint := range []string{
		"http://169.254.169.254/",
		"http://127.0.0.1:80/",
	} {
		body := `{"endpoint":"` + endpoint + `","api_key":"x","protocol":"openai-chat"}`
		if code := postProbeJSON(t, ts.URL, "/api/models", body); code != http.StatusBadRequest {
			t.Errorf("/api/models endpoint %s 应被拒（400），got %d", endpoint, code)
		}
	}
}

// TestProbeNoCredExfilOnEndpointMismatch 验证凭据外泄防护（C2）：
// name 命中已注册模型但请求 endpoint 与之不符、未传 api_key → 拒绝代填，不发送已存明文 key。
func TestProbeNoCredExfilOnEndpointMismatch(t *testing.T) {
	ts, _ := setupServer(t)
	// setupServer 注册了 default{endpoint:http://localhost:1, api_key:k}
	// 用 name=default 但 endpoint 指向别处、不传 key → 应 400（resolveKey 拒绝代填，无任何出站请求）
	body := `{"name":"default","endpoint":"http://attacker.invalid/v1","protocol":"openai-chat","model":"m"}`
	if code := postProbeJSON(t, ts.URL, "/api/config/test", body); code != http.StatusBadRequest {
		t.Errorf("endpoint 不符时应拒绝代填已存 key（400），got %d（凭据可能外泄）", code)
	}
}

// TestConfigPutNoKeyCarryoverOnEndpointChange 验证 PUT /api/config 的空 key 合并防护（C2 关联）：
// 把已存模型 repoint 到新 endpoint、api_key 留空 → 旧 key 不应被代填到新 endpoint。
func TestConfigPutNoKeyCarryoverOnEndpointChange(t *testing.T) {
	ts, _ := setupServer(t)
	// 1. PUT 把 default repoint 到 attacker.invalid，api_key 留空
	do(t, ts, "PUT", "/api/config", map[string]any{
		"models": []map[string]any{{
			"name": "default", "endpoint": "http://attacker.invalid/v1", "api_key": "",
			"model": "m", "temperature": 0.3, "max_tokens": 1024,
		}},
		"router": map[string]any{
			"default": "default", "routes": map[string]any{},
			"fallback": map[string]any{"max_retries": 1, "backoff_base_ms": 10},
		},
	})
	// 2. 探测：name=default、endpoint 与新 endpoint 一致、不传 key
	//    若旧 key 被错误代填 → resolveKey 命中 → 进入出站（200 ok:false 或 502）；
	//    若正确丢弃 → GetModel 返回空 key → 400「缺少 api_key」
	body := `{"name":"default","endpoint":"http://attacker.invalid/v1","protocol":"openai-chat","model":"m"}`
	if code := postProbeJSON(t, ts.URL, "/api/config/test", body); code != http.StatusBadRequest {
		t.Errorf("repoint endpoint 后旧 key 不应被代填（应 400），got %d（凭据可能外泄到新 endpoint）", code)
	}
}
