package agent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/leads"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/assembler"
	"customer-demand-agent/internal/memory/shortterm"
	"customer-demand-agent/internal/memory/tools"
	"customer-demand-agent/internal/model"
)

// mockPlatformSSE 是一个 mock 商机平台（记录请求并返回线索列表）。
func mockPlatformSSE(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.RequestURI())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items":[
			{"id":"L001","customer":"某电商集团","stage":"mql","product":"雷池"},
			{"id":"L002","customer":"某银行","stage":"mql","product":"万象"}],"total":2}`)
	}))
	t.Cleanup(srv.Close)
	return srv, &paths
}

// mockLLMCapturing 模拟 OpenAI 兼容网关（SSE 流式），逐轮按 script 响应，
// 并捕获每个请求体的 tools 定义（供「未启用不注册」断言）。
func mockLLMCapturing(t *testing.T, script []step) (*httptest.Server, *[][]domain2Tool) {
	t.Helper()
	var mu sync.Mutex
	var bodies [][]domain2Tool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req struct {
			Tools []domain2Tool `json:"tools"`
		}
		_ = json.Unmarshal(raw, &req)
		mu.Lock()
		n := len(bodies)
		bodies = append(bodies, req.Tools)
		mu.Unlock()
		st := script[len(script)-1]
		if n < len(script) {
			st = script[n]
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "data: "+`{"choices":[{"index":0,"delta":{"role":"assistant"}}]}`+"\n\n")
		if st.toolCallJSON != "" {
			fmt.Fprint(w, "data: "+st.toolCallJSON+"\n\n")
			fmt.Fprint(w, "data: "+`{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`+"\n\n")
		} else {
			fmt.Fprint(w, "data: "+`{"choices":[{"index":0,"delta":{"content":`+jsonString(st.content)+`}}]}`+"\n\n")
			fmt.Fprint(w, "data: "+`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`+"\n\n")
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &bodies
}

// domain2Tool 是捕获工具列表用的最小结构（对齐 domain.Tool）。
type domain2Tool struct {
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

// newLeadsAgent 组装连着 mock LLM 与 mock 商机平台的 Agent。
func newLeadsAgent(t *testing.T, script []step, platformURL string) (*agent.Agent, *[][]domain2Tool) {
	t.Helper()
	wiki := seedWiki(t)
	sessions := shortterm.NewSessionManager()
	toolReg := tools.NewRegistry(wiki)
	asm := assembler.New(wiki, "")

	srv, bodies := mockLLMCapturing(t, script)
	registry := model.NewRegistry(10 * time.Second)
	_ = registry.Register(model.ModelConfig{
		Name: "default", Endpoint: srv.URL, APIKey: "test-key", Model: "mock",
		Temperature: 0.3, MaxTokens: 1024,
	})
	mgr := model.NewManager(llm.NewClient(), registry, &model.RouterConfig{Default: "default"})
	ag := agent.New(mgr, toolReg, asm, sessions)
	if platformURL != "" {
		ag.SetLeads(leads.New(leads.Options{
			BaseURL: platformURL, APIKey: "lm_pat_test",
			RatePerMin: 1_000_000, Burst: 100, Timeout: 2 * time.Second,
		}))
	}
	return ag, bodies
}

// TestAgentLeadsSearchHighValue（验收场景 1）：用户问「高价值线索」→
// Agent 调 leads_search(stage=mql) → 结果进入 LLM 上下文 → 最终回答包含列表。
func TestAgentLeadsSearchHighValue(t *testing.T) {
	platform, paths := mockPlatformSSE(t)
	ag, _ := newLeadsAgent(t, []step{
		// 第 1 轮：检索商机（「高价值」→ stage=mql，文档口径）
		{toolCallJSON: `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_0","type":"function","function":{"name":"leads_search","arguments":"{\"stage\":\"mql\"}"}}]}}]}`},
		// 第 2 轮：基于检索结果自然语言回答
		{content: "当前 MQL 线索 2 条：某电商集团（雷池）、某银行（万象）。注：「高价值」按 MQL 阶段筛选，平台无排序能力，以上为默认顺序。"},
	}, platform.URL)

	content, _, trace, err := ag.Message(context.Background(), nil, "sess-leads", "", "当前有哪些高价值的线索？", nil)
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	if !strings.Contains(content, "某电商集团") {
		t.Errorf("最终回答应包含检索到的线索列表，got: %s", content)
	}
	// 平台收到 stage=mql 检索
	if len(*paths) == 0 || !strings.Contains((*paths)[0], "stage=mql") {
		t.Errorf("平台应收到 stage=mql 查询，got %v", *paths)
	}
	// 轨迹含 leads_search 且结果进入上下文
	var sawLeads bool
	for _, tc := range trace.ToolCalls {
		if tc.Tool == "leads_search" {
			sawLeads = true
			if !strings.Contains(tc.Result, "某电商集团") {
				t.Errorf("工具结果应含线索数据: %s", tc.Result)
			}
		}
	}
	if !sawLeads {
		t.Fatal("trace 应含 leads_search")
	}
	// system prompt 注入数据源身份行（assembler 动态注入）
	if !strings.Contains(trace.SystemPrompt, "商机平台（Lead Manager）已接入") {
		t.Error("system prompt 应含商机数据源身份行")
	}
}

// TestAgentLeadsDisabledToolsHidden（启停语义）：未启用时工具定义根本不注册
// ——模型工具列表里没有 leads_*，prompt 也不注入数据源身份行。
func TestAgentLeadsDisabledToolsHidden(t *testing.T) {
	ag, bodies := newLeadsAgent(t, []step{
		{content: "商机平台未接入，请在设置里配置。"},
	}, "") // platformURL 为空 → SetLeads 不调用

	content, _, trace, err := ag.Message(context.Background(), nil, "sess-noleads", "", "当前有哪些高价值的线索？", nil)
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	if content == "" {
		t.Error("未启用时也应自然语言回答")
	}
	if len(*bodies) == 0 {
		t.Fatal("mock LLM 未收到请求")
	}
	for i, toolsList := range *bodies {
		for _, tl := range toolsList {
			if strings.HasPrefix(tl.Function.Name, "leads_") {
				t.Errorf("第 %d 轮工具列表不应含 leads_*（未启用不注册）: %s", i+1, tl.Function.Name)
			}
		}
	}
	if strings.Contains(trace.SystemPrompt, "Lead Manager") {
		t.Error("未启用时 prompt 不应注入数据源身份行")
	}
}

// TestAgentLeadsToolsRegistered：启用时工具列表包含 3 个 leads_*。
func TestAgentLeadsToolsRegistered(t *testing.T) {
	platform, _ := mockPlatformSSE(t)
	ag, bodies := newLeadsAgent(t, []step{
		{content: "好的。"},
	}, platform.URL)
	if _, _, _, err := ag.Message(context.Background(), nil, "sess-reg", "", "你好", nil); err != nil {
		t.Fatal(err)
	}
	if len(*bodies) == 0 {
		t.Fatal("mock LLM 未收到请求")
	}
	names := map[string]bool{}
	for _, tl := range (*bodies)[0] {
		names[tl.Function.Name] = true
	}
	for _, want := range []string{"leads_search", "leads_get", "leads_stats"} {
		if !names[want] {
			t.Errorf("工具列表应含 %s", want)
		}
	}
	// 既有工具不受影响
	if !names["memory_search"] || !names["analysis_submit"] {
		t.Error("既有工具应保持注册")
	}
}
