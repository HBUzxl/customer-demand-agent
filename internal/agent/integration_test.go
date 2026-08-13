package agent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/assembler"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/memory/shortterm"
	"customer-demand-agent/internal/memory/tools"
	"customer-demand-agent/internal/model"
)

// seedWiki 写入一个最小 Wiki 用于集成测试。
func seedWiki(t *testing.T) *longterm.WikiStore {
	t.Helper()
	dir := t.TempDir()
	page := "---\n" +
		"type: product\n" +
		"title: 雷池\n" +
		"aliases: [\"WAF\"]\n" +
		"tags: [\"WAF\", \"CC防护\"]\n" +
		"capabilities:\n" +
		"  - name: CC 攻击防护\n" +
		"    confidence: 1.0\n" +
		"    keywords: [\"CC\", \"攻击\"]\n" +
		"description: 下一代 WAF\n" +
		"---\n雷池是 WAF。\n"
	writeFile(t, dir+"/产品记忆/雷池.md", page)

	store := longterm.NewWikiStore(dir)
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}
	return store
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := osMkdirAll(path); err != nil {
		t.Fatal(err)
	}
	if err := osWriteFile(path, []byte(content)); err != nil {
		t.Fatal(err)
	}
}

// osMkdirAll ensures the parent dir of path exists.
func osMkdirAll(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

// osWriteFile wraps os.WriteFile.
func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

// mockLLMServer 模拟一个 OpenAI 兼容网关（SSE 流式）：
// 第 1 次返回 tool_calls（memory_search），第 2 次返回最终 JSON 结果。
func mockLLMServer(t *testing.T) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		send := func(line string) {
			fmt.Fprint(w, "data: "+line+"\n\n")
			if flusher != nil {
				flusher.Flush()
			}
		}
		if n == 1 {
			send(`{"choices":[{"index":0,"delta":{"role":"assistant"}}]}`)
			send(`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"memory_search","arguments":"{\"query\":\"CC攻击\",\"type\":\"product\"}"}}]}}]}`)
			send(`{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`)
		} else {
			send(`{"choices":[{"index":0,"delta":{"role":"assistant"}}]}`)
			send(`{"choices":[{"index":0,"delta":{"content":"{\"demand_analysis\":\"客户遭遇CC攻击\",\"matched_products\":[{\"name\":\"雷池\",\"confidence\":0.95,\"reason\":\"CC防护\",\"suggestion\":\"上雷池\"}],\"feasibility\":\"direct\",\"feasibility_detail\":\"雷池直接覆盖\",\"missing_info\":[\"攻击规模\"]}"}}]}`)
			send(`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	return srv, &calls
}

func TestAgentFullLoopWithMockLLM(t *testing.T) {
	wiki := seedWiki(t)
	sessions := shortterm.NewSessionManager()
	toolReg := tools.NewRegistry(wiki)
	asm := assembler.New(wiki)

	srv, calls := mockLLMServer(t)
	defer srv.Close()

	registry := model.NewRegistry()
	_ = registry.Register(model.ModelConfig{
		Name: "default", Endpoint: srv.URL, APIKey: "test-key", Model: "mock",
		Temperature: 0.3, MaxTokens: 1024, TimeoutSec: 10,
	})
	router := &model.RouterConfig{Default: "default"}
	mgr := model.NewManager(llm.NewClient(), registry, router)

	ag := agent.New(mgr, toolReg, asm, sessions)

	result, trace, err := ag.Analyze(context.Background(), "sess-test", "我们网站被CC攻击了，怎么办？")
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// 验证最终结果
	if result.DemandAnalysis == "" {
		t.Error("demand_analysis empty")
	}
	if len(result.MatchedProducts) != 1 || result.MatchedProducts[0].Name != "雷池" {
		t.Errorf("expected matched 雷池, got %+v", result.MatchedProducts)
	}
	if result.Feasibility != domain.FeasibilityDirect {
		t.Errorf("feasibility = %v, want direct", result.Feasibility)
	}

	// 验证工具被调用过
	if atomic.LoadInt32(calls) < 2 {
		t.Errorf("expected >=2 LLM calls, got %d", *calls)
	}
	if len(trace.ToolCalls) != 1 || trace.ToolCalls[0].Tool != "memory_search" {
		t.Errorf("expected 1 tool call memory_search, got %+v", trace.ToolCalls)
	}

	// 验证 checkpoint 已创建
	if !sessions.Get("sess-test").HasHistory() {
		t.Error("checkpoint not created after analysis")
	}
}

func TestAgentDefinitionsAreValidJSON(t *testing.T) {
	// 确保工具定义可序列化为合法 JSON（给 LLM 用）
	wiki := seedWiki(t)
	toolReg := tools.NewRegistry(wiki)
	defs := toolReg.Definitions()
	b, err := json.Marshal(defs)
	if err != nil {
		t.Fatalf("tool defs not serializable: %v", err)
	}
	var decoded []domain.Tool
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("tool defs not round-trippable: %v", err)
	}
	if len(decoded) != 6 {
		t.Errorf("expected 6 tools, got %d", len(decoded))
	}
}
