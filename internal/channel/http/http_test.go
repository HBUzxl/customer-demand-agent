package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"customer-demand-agent/internal/agent"
	httpapi "customer-demand-agent/internal/channel/http"
	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/history"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/assembler"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/memory/shortterm"
	"customer-demand-agent/internal/memory/tools"
	"customer-demand-agent/internal/model"
	"customer-demand-agent/internal/review"
)

// setupServer 构建一个完整可用的 HTTP server（用临时 wiki + sqlite + mock 模型）。
func setupServer(t *testing.T) (*httptest.Server, *longterm.WikiStore) {
	t.Helper()
	wikiDir := t.TempDir()
	store := longterm.NewWikiStore(wikiDir)
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}
	// seed 一个产品
	_ = os.MkdirAll(filepath.Join(wikiDir, "产品记忆"), 0o755)
	_ = os.WriteFile(filepath.Join(wikiDir, "产品记忆", "雷池.md"),
		[]byte("---\ntype: product\ntitle: 雷池\ntags: [\"WAF\"]\n---\nx"), 0o644)
	_ = store.Load()

	hist, err := history.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { hist.Close() })

	sessions := shortterm.NewSessionManager()
	toolReg := tools.NewRegistry(store)
	asm := assembler.New(store)

	// 用临时 config.json 加载（测试真实持久化路径）
	configFile := filepath.Join(t.TempDir(), "config.json")
	_ = os.WriteFile(configFile, []byte(`{"server":{"addr":":0"},"default_tenant":"default","models":[{"name":"default","endpoint":"http://localhost:1","api_key":"k","model":"m","temperature":0.3,"max_tokens":1024,"timeout_sec":5}],"router":{"default":"default","routes":{},"fallback":{"max_retries":1,"backoff_base_ms":10,"chain":null}}}`), 0o600)
	cfgStore, err := config.Load(configFile)
	if err != nil {
		t.Fatal(err)
	}
	registry := model.NewRegistry()
	for _, m := range cfgStore.Get().Models {
		_ = registry.Register(m)
	}
	rc := cfgStore.Get().Router
	mgr := model.NewManager(llm.NewClient(), registry, &rc)
	ag := agent.New(mgr, toolReg, asm, sessions)
	rv := review.New(store)

	srv := httpapi.New(cfgStore, ag, rv, store, hist, mgr, registry)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, store
}

func do(t *testing.T, ts *httptest.Server, method, path string, body any) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(method, ts.URL+path, nil)
	if body != nil {
		b, _ := json.Marshal(body)
		req.Body = nopCloser{bytes.NewReader(b)}
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func TestMemoryUpsertReviewFlow(t *testing.T) {
	ts, _ := setupServer(t)

	// 1. 人工 upsert 一条 threat（直接 verified，因为是人工接口）
	code, body := do(t, ts, "POST", "/api/memory", map[string]any{
		"type": "threat", "title": "测试威胁", "content": "内容", "status": "pending_review",
	})
	if code != 200 {
		t.Fatalf("upsert failed: %d %v", code, body)
	}

	// 2. 出现在审核队列
	code, body = do(t, ts, "GET", "/api/review/pending", nil)
	if code != 200 || int(body["count"].(float64)) != 1 {
		t.Fatalf("expected 1 pending, got %v", body)
	}

	// 3. 批准
	code, _ = do(t, ts, "POST", "/api/review/threat/测试威胁/approve", nil)
	if code != 200 {
		t.Fatalf("approve failed: %d", code)
	}

	// 4. 队列清空
	_, body = do(t, ts, "GET", "/api/review/pending", nil)
	if int(body["count"].(float64)) != 0 {
		t.Fatalf("pending should be empty after approve, got %v", body)
	}

	// 5. memory get 确认状态
	_, body = do(t, ts, "GET", "/api/memory/threat/测试威胁", nil)
	if body["status"] != "verified" {
		t.Fatalf("status should be verified, got %v", body["status"])
	}
}

func TestMemorySearchAndList(t *testing.T) {
	ts, _ := setupServer(t)

	// list products → 雷池
	_, body := do(t, ts, "GET", "/api/memory/list?type=product", nil)
	if int(body["count"].(float64)) != 1 {
		t.Fatalf("expected 1 product, got %v", body)
	}

	// search 雷池
	_, body = do(t, ts, "GET", "/api/memory/search?q=雷池&type=product", nil)
	if int(body["count"].(float64)) != 1 {
		t.Fatalf("search 雷池 expected 1, got %v", body)
	}
}

func TestConfigGetPut(t *testing.T) {
	ts, _ := setupServer(t)

	code, body := do(t, ts, "GET", "/api/config", nil)
	if code != 200 {
		t.Fatalf("config get failed: %d", code)
	}
	models := body["models"].([]any)
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	// api_key 应被脱敏（原始 'k' 不应泄露）
	m := models[0].(map[string]any)
	if m["api_key"] == "k" {
		t.Errorf("api_key should be masked, got raw value")
	}
}

func TestTenantIsolation(t *testing.T) {
	// 多租户隔离在 history 层生效：不同 tenant 看不到彼此的会话。
	ts, _ := setupServer(t)

	// tenant A 写一个会话
	reqA, _ := http.NewRequest("POST", ts.URL+"/api/analyze", nopCloser{bytes.NewReader([]byte(`{"text":"A的文本"}`))})
	reqA.Header.Set("Content-Type", "application/json")
	reqA.Header.Set("X-Tenant-ID", "tenantA")
	respA, errA := http.DefaultClient.Do(reqA)
	if errA != nil {
		t.Fatal(errA)
	}
	defer respA.Body.Close()
	// analyze 会因 LLM 不可用失败，但会话已创建（EnsureSession 在 LLM 调用前）

	// tenant A 能看到会话、tenant B 看不到
	reqLA, _ := http.NewRequest("GET", ts.URL+"/api/sessions", nil)
	reqLA.Header.Set("X-Tenant-ID", "tenantA")
	respLA, _ := http.DefaultClient.Do(reqLA)
	var bodyA map[string]any
	_ = json.NewDecoder(respLA.Body).Decode(&bodyA)
	respLA.Body.Close()

	reqLB, _ := http.NewRequest("GET", ts.URL+"/api/sessions", nil)
	reqLB.Header.Set("X-Tenant-ID", "tenantB")
	respLB, _ := http.DefaultClient.Do(reqLB)
	var bodyB map[string]any
	_ = json.NewDecoder(respLB.Body).Decode(&bodyB)
	respLB.Body.Close()

	countA, _ := bodyA["count"].(float64)
	countB, _ := bodyB["count"].(float64)
	if int(countA) != 1 {
		t.Errorf("tenantA should see 1 session, got %v", countA)
	}
	if int(countB) != 0 {
		t.Errorf("tenantB should see 0 sessions (isolation), got %v", countB)
	}
}
