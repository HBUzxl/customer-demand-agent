package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"customer-demand-agent/internal/agent"
	httpapi "customer-demand-agent/internal/channel/http"
	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/domain"
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
	// 用临时 config.json 加载（测试真实持久化路径）
	configFile := filepath.Join(t.TempDir(), "config.json")
	_ = os.WriteFile(configFile, []byte(`{"server":{"addr":":0"},"default_tenant":"default","models":[{"name":"default","endpoint":"http://localhost:1","api_key":"k","model":"m","temperature":0.3,"max_tokens":1024,"timeout_sec":5}],"router":{"default":"default","routes":{},"fallback":{"max_retries":1,"backoff_base_ms":10,"chain":null}}}`), 0o600)
	return setupServerWithConfig(t, configFile)
}

// setupServerWithConfig 用指定 config.json 构造完整服务（验证回写文件用）。
func setupServerWithConfig(t *testing.T, configFile string) (*httptest.Server, *longterm.WikiStore) {
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
	asm := assembler.New(store, "")

	cfgStore, err := config.Load(configFile)
	if err != nil {
		t.Fatal(err)
	}
	registry := model.NewRegistry(10 * time.Second)
	for _, m := range cfgStore.Get().Models {
		_ = registry.Register(m)
	}
	rc := cfgStore.Get().Router
	mgr := model.NewManager(llm.NewClient(), registry, &rc)
	ag := agent.New(mgr, toolReg, asm, sessions)
	rv := review.New(store)

	srv := httpapi.New(cfgStore, ag, rv, store, hist, mgr, registry, nil, nil)
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
	ts, store := setupServer(t)

	// 1. AI 路径写入 threat（经 wiki store 打 pending——人工通道已不能传
	// status，P10；此测试锁定审核流本身）
	if err := store.UpsertEntry(&longterm.Entry{
		Type: domain.MemoryThreat, Title: "测试威胁", Content: "内容",
		Status: longterm.StatusPendingReview,
	}); err != nil {
		t.Fatal(err)
	}

	// 2. 出现在审核队列
	code, body := do(t, ts, "GET", "/api/review/pending", nil)
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

// TestMessageCustomerFieldPersisted 客户身份端到端（ADR-016 L2）：
// /api/message 带 customer → EnsureSession 落库 → 会话详情读回。
// LLM 不可达会失败，但 customer 落库发生在 LLM 调用前——验证不受影响。
func TestMessageCustomerFieldPersisted(t *testing.T) {
	ts, _ := setupServer(t)
	// 带 customer 建会话（F0：202 + Run）
	code, body := do(t, ts, "POST", "/api/message", map[string]any{
		"text": "你好", "session_id": "sess-cust", "customer": "某跨境电商",
	})
	if code != 202 {
		t.Fatalf("消息应 202，got %d: %v", code, body)
	}
	code2, body2 := do(t, ts, "GET", "/api/sessions/sess-cust", nil)
	if code2 != 200 {
		t.Fatalf("读取会话应 200，got %d", code2)
	}
	cust, _ := body2["session"].(map[string]any)["customer"].(string)
	if cust != "某跨境电商" {
		t.Fatalf("customer 应已落库为 某跨境电商，got %q", cust)
	}
}

// TestMemoryUpsertIgnoresStatusParam P10：人工通道不接受 status 传参——
// 传 pending_review 也必须落 verified（pending 是 AI 写入专属语义）。
func TestMemoryUpsertIgnoresStatusParam(t *testing.T) {
	ts, store := setupServer(t)
	code, _ := do(t, ts, "POST", "/api/memory", map[string]any{
		"type": "threat", "title": "人工条目", "content": "x", "status": "pending_review",
	})
	if code != 200 {
		t.Fatalf("upsert 应 200，got %d", code)
	}
	e, err := store.GetEntry("threat", "人工条目")
	if err != nil {
		t.Fatal(err)
	}
	if e.Status != longterm.StatusVerified {
		t.Fatalf("人工写入必须 verified（忽略 status 传参），got %s", e.Status)
	}
}
