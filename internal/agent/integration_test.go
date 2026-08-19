package agent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/history"
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
		"    description: 下一代 WAF\n" +
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// step 是 mock LLM 的一轮响应：toolCallJSON 非空则返回 tool_calls，否则返回 content。
type step struct {
	toolCallJSON string // 完整 tool_calls 的 delta JSON
	content      string // 自然语言内容
	reasoning    string // 模型私有 reasoning_content（Agent 不应原样外发）
}

// mockLLMServer 模拟一个 OpenAI 兼容网关（SSE 流式），按 script 逐轮响应。
// script 耗尽后重复最后一个 step（防越界）。
func mockLLMServer(t *testing.T, script []step) (*httptest.Server, *int32) {
	return mockLLMServerInspect(t, script, nil)
}

// mockLLMServerInspect 允许测试观察每轮请求是否仍携带工具定义。
func mockLLMServerInspect(t *testing.T, script []step, inspect func(call, toolCount int)) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(atomic.AddInt32(&calls, 1))
		var req struct {
			Tools []json.RawMessage `json:"tools"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if inspect != nil {
			inspect(n, len(req.Tools))
		}
		st := script[len(script)-1]
		if n <= len(script) {
			st = script[n-1]
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		send := func(line string) {
			fmt.Fprint(w, "data: "+line+"\n\n")
			if flusher != nil {
				flusher.Flush()
			}
		}
		send(`{"choices":[{"index":0,"delta":{"role":"assistant"}}]}`)
		if st.reasoning != "" {
			send(`{"choices":[{"index":0,"delta":{"reasoning_content":` + jsonString(st.reasoning) + `}}]}`)
		}
		if st.content != "" {
			send(`{"choices":[{"index":0,"delta":{"content":` + jsonString(st.content) + `}}]}`)
		}
		if st.toolCallJSON != "" {
			send(st.toolCallJSON)
			send(`{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`)
		} else {
			send(`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	return srv, &calls
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// newTestAgent 组装一个连着 mock LLM 的 Agent。
func newTestAgent(t *testing.T, script []step) (*agent.Agent, *shortterm.SessionManager, *int32) {
	return newTestAgentInspect(t, script, nil)
}

func newTestAgentInspect(t *testing.T, script []step, inspect func(call, toolCount int)) (*agent.Agent, *shortterm.SessionManager, *int32) {
	t.Helper()
	wiki := seedWiki(t)
	sessions := shortterm.NewSessionManager()
	toolReg := tools.NewRegistry(wiki)
	asm := assembler.New(wiki, "")

	srv, calls := mockLLMServerInspect(t, script, inspect)
	t.Cleanup(srv.Close)

	registry := model.NewRegistry(10 * time.Second)
	_ = registry.Register(model.ModelConfig{
		Name: "default", Endpoint: srv.URL, APIKey: "test-key", Model: "mock",
		Temperature: 0.3, MaxTokens: 1024,
	})
	router := &model.RouterConfig{Default: "default"}
	mgr := model.NewManager(llm.NewClient(), registry, router)
	return agent.New(mgr, toolReg, asm, sessions), sessions, calls
}

const submitCall = `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"analysis_submit","arguments":"{\"demand_analysis\":\"客户遭遇CC攻击，需要Web防护\",\"matched_products\":[{\"name\":\"雷池\",\"confidence\":0.95,\"reason\":\"CC防护核心场景\",\"suggestion\":\"推荐雷池\"}],\"feasibility\":\"direct\",\"feasibility_detail\":\"直接覆盖\",\"missing_info\":[\"攻击规模\"]}"}}]}}]}`

// getLeiChiCall 是 analysis_submit 前的必要证据步骤（P0-02 门禁：推荐产品前
// 当轮必须 memory_get 读产品主页核验能力）。
const getLeiChiCall = `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_get","type":"function","function":{"name":"memory_get","arguments":"{\"type\":\"product\",\"title\":\"雷池\"}"}}]}}]}`

// TestAgentGreetingChat（分支①寒暄）：直接自然语言回答——无 analysis_submit、
// done 无 analysis、纯聊天轮次也建轻量 followup checkpoint（ADR-015/Q1）。
func TestAgentGreetingChat(t *testing.T) {
	ag, sessions, _ := newTestAgent(t, []step{
		{content: "你好！我是售前需求分析助手，把客户沟通原文发我就行。"},
	})

	content, analysis, trace, err := ag.Message(context.Background(), nil, "sess-chat", "", "你好", nil)
	if err != nil {
		t.Fatalf("Message failed: %v", err)
	}
	if content == "" {
		t.Error("寒暄应返回自然语言 content")
	}
	if analysis != nil {
		t.Errorf("寒暄不应产生 analysis，got %+v", analysis)
	}
	for _, tc := range trace.ToolCalls {
		if tc.Tool == agent.ToolAnalysisSubmit {
			t.Error("寒暄不应调用 analysis_submit")
		}
	}
	// Q1：纯聊天轮次也建 checkpoint（followup，Analysis 空）
	m := sessions.Get("sess-chat")
	if !m.HasHistory() {
		t.Fatal("纯聊天轮次也应创建 checkpoint（ADR-015/Q1）")
	}
	cp := m.CurrentCheckpoint()
	if cp == nil || cp.Type != domain.CheckpointFollowup {
		t.Errorf("纯聊天 checkpoint 应为 followup，got %+v", cp)
	}
	if cp != nil && cp.Analysis != nil {
		t.Error("纯聊天 followup checkpoint 的 Analysis 应为空")
	}
	if cp == nil || cp.Question != "你好" || cp.Answer == "" {
		t.Errorf("followup checkpoint 应记 Question/Answer，got %+v", cp)
	}
}

func TestAgentRetriesOneEmptyModelResponse(t *testing.T) {
	ag, _, calls := newTestAgent(t, []step{
		{},
		{content: "已恢复并给出有效回答。"},
	})
	content, _, _, err := ag.Message(context.Background(), nil, "sess-empty-retry", "", "你好", nil)
	if err != nil {
		t.Fatalf("单次空响应应自动重试而不是暴露错误: %v", err)
	}
	if content != "已恢复并给出有效回答。" {
		t.Fatalf("应返回重试后的内容，got %q", content)
	}
	if atomic.LoadInt32(calls) != 2 {
		t.Fatalf("应恰好重试一次，calls=%d", atomic.LoadInt32(calls))
	}
}

// TestAgentDemandAnalysis（分支②真实需求）：自主检索 → analysis_submit → 自然语言总结。
func TestAgentDemandAnalysis(t *testing.T) {
	ag, sessions, calls := newTestAgent(t, []step{
		// 第 1 轮：先检索产品记忆（体现自主性）
		{toolCallJSON: `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_0","type":"function","function":{"name":"memory_search","arguments":"{\"query\":\"CC攻击\",\"type\":\"product\"}"}}]}}]}`},
		// 第 1.5 轮：读产品详情（P0-02 证据门禁：推荐前必须 memory_get）
		{toolCallJSON: getLeiChiCall},
		// 第 2 轮：提交结构化分析
		{toolCallJSON: submitCall},
		// 第 3 轮：自然语言总结
		{content: "结论：客户网站被 CC 攻击，**雷池**可直接覆盖（置信度 0.95），建议先确认攻击规模。"},
	})

	content, analysis, trace, err := ag.Message(context.Background(), nil, "sess-req", "", "我们电商网站大促被CC攻击，要过等保二级", nil)
	if err != nil {
		t.Fatalf("Message failed: %v", err)
	}
	if content == "" {
		t.Error("需求分析也应返回自然语言总结 content")
	}
	if analysis == nil {
		t.Fatal("真实需求应通过 analysis_submit 产生 analysis")
	}
	if analysis.DemandAnalysis == "" || analysis.Feasibility != domain.FeasibilityDirect {
		t.Errorf("analysis 内容不对: %+v", analysis)
	}
	if len(analysis.MatchedProducts) != 1 || analysis.MatchedProducts[0].Name != "雷池" {
		t.Errorf("expected matched 雷池, got %+v", analysis.MatchedProducts)
	}

	// trace：memory_search + analysis_submit 都在
	var sawSearch, sawSubmit bool
	for _, tc := range trace.ToolCalls {
		switch tc.Tool {
		case "memory_search":
			sawSearch = true
		case agent.ToolAnalysisSubmit:
			sawSubmit = true
		}
	}
	if !sawSearch || !sawSubmit {
		t.Errorf("trace 应含 memory_search 与 analysis_submit，got %+v", trace.ToolCalls)
	}
	// trace：本轮 system prompt 被捕获（回放要能看到整个调用状态）
	if trace.SystemPrompt == "" || !strings.Contains(trace.SystemPrompt, "产品") {
		t.Errorf("trace.SystemPrompt 应为拼装后的系统提示词（含产品知识库），got %d 字", len(trace.SystemPrompt))
	}
	if atomic.LoadInt32(calls) < 3 {
		t.Errorf("expected >=3 LLM calls, got %d", *calls)
	}

	// checkpoint：提交过分析 → initial，带 Analysis
	m := sessions.Get("sess-req")
	cp := m.CurrentCheckpoint()
	if cp == nil || cp.Type != domain.CheckpointInitial {
		t.Fatalf("提交分析后 checkpoint 应为 initial，got %+v", cp)
	}
	if cp.Analysis == nil || cp.Analysis.DemandAnalysis == "" {
		t.Errorf("initial checkpoint 应携带 Analysis，got %+v", cp)
	}
}

// TestAgentFollowupAfterAnalysis（分支③追问）：基于上下文回答，不重复提交。
func TestAgentFollowupAfterAnalysis(t *testing.T) {
	ag, sessions, _ := newTestAgent(t, []step{
		{toolCallJSON: getLeiChiCall},
		{toolCallJSON: submitCall},
		{content: "结论：推荐雷池。"},
		// 追问轮：直接回答
		{content: "雷池支持旁路部署，无需改动现有网络架构。"},
	})

	ctx := context.Background()
	_, analysis, _, err := ag.Message(ctx, nil, "sess-fu", "", "我们网站被CC攻击了", nil)
	if err != nil || analysis == nil {
		t.Fatalf("首轮分析失败: %v / analysis=%v", err, analysis)
	}

	content2, analysis2, _, err := ag.Message(ctx, nil, "sess-fu", "", "那部署方式呢？", nil)
	if err != nil {
		t.Fatalf("追问失败: %v", err)
	}
	if content2 == "" {
		t.Error("追问应返回自然语言回答")
	}
	if analysis2 != nil {
		t.Errorf("简单追问不应重新提交 analysis（需求未变），got %+v", analysis2)
	}
	// checkpoint 链：initial + followup
	m := sessions.Get("sess-fu")
	chain := m.CheckpointChain()
	if len(chain) != 2 {
		t.Fatalf("应有两级 checkpoint（initial+followup），got %d", len(chain))
	}
	if chain[0].Type != domain.CheckpointInitial || chain[1].Type != domain.CheckpointFollowup {
		t.Errorf("checkpoint 类型错: %s → %s", chain[0].Type, chain[1].Type)
	}
}

// TestAgentReanalysisCheckpoint（is_reanalysis 优先）：Agent 置 is_reanalysis=true
// 时，即使 DetermineOp 启发式判定为 initial（无历史），也建 reanalysis checkpoint。
func TestAgentReanalysisCheckpoint(t *testing.T) {
	submitRe := `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"analysis_submit","arguments":"{\"demand_analysis\":\"客户换了方向：数据泄露防护\",\"feasibility\":\"custom\",\"is_reanalysis\":true}"}}]}}]}`
	ag, sessions, _ := newTestAgent(t, []step{
		{toolCallJSON: submitRe},
		{content: "需求实质变化，重新分析如下。"},
	})

	_, analysis, _, err := ag.Message(context.Background(), nil, "sess-re", "", "完全新的数据泄露场景", nil)
	if err != nil {
		t.Fatalf("Message failed: %v", err)
	}
	if analysis == nil {
		t.Fatal("应产生 analysis")
	}
	cp := sessions.Get("sess-re").CurrentCheckpoint()
	if cp == nil || cp.Type != domain.CheckpointReanalysis {
		t.Fatalf("is_reanalysis=true 应建 reanalysis checkpoint，got %+v", cp)
	}
}

// TestAgentCheckpointRestore 断点续传：checkpoint 持久化（含纯聊天 followup）后，
// 新 Agent 实例（模拟重启）从历史 restore，链完整、追问正常。
func TestAgentCheckpointRestore(t *testing.T) {
	ag1, _, _ := newTestAgent(t, []step{
		{toolCallJSON: getLeiChiCall},
		{toolCallJSON: submitCall},
		{content: "结论：推荐雷池。"},
		// 纯聊天轮
		{content: "你好呀！随时把客户材料发我。"},
		// 重启后的追问轮
		{content: "雷池支持旁路部署。"},
	})
	ctx := context.Background()

	// 模拟持久化：sink 收集 → source 供 restore。
	var saved []*domain.Checkpoint
	ag1.SetCheckpointSink(func(_ string, _ string, cp *domain.Checkpoint) { saved = append(saved, cp) })
	var gotErr error
	ag1.SetCheckpointSource(func(_ string, _ string) ([]*domain.Checkpoint, error) { return saved, gotErr })

	// 轮 1：真实需求（initial）
	if _, a, _, err := ag1.Message(ctx, nil, "sess-rs", "", "我们网站被CC攻击", nil); err != nil || a == nil {
		t.Fatalf("首轮失败: %v", err)
	}
	// 轮 2：纯聊天（轻量 followup）
	if _, a2, _, err := ag1.Message(ctx, nil, "sess-rs", "", "你好", nil); err != nil {
		t.Fatalf("纯聊天失败: %v", err)
	} else if a2 != nil {
		t.Fatal("纯聊天不应有 analysis")
	}
	if len(saved) != 2 {
		t.Fatalf("应持久化 2 个 checkpoint（initial+followup），got %d", len(saved))
	}
	if saved[0].Type != domain.CheckpointInitial || saved[1].Type != domain.CheckpointFollowup {
		t.Fatalf("checkpoint 类型: %s, %s", saved[0].Type, saved[1].Type)
	}

	// 模拟重启：全新 Agent（同一 mock 脚本继续服务追问轮）+ 全新 SessionManager，
	// 用同一份持久化 checkpoint restore。
	ag2, _, _ := newTestAgent(t, []step{
		{toolCallJSON: getLeiChiCall},
		{toolCallJSON: submitCall},
		{content: "结论：推荐雷池。"},
		{content: "你好呀！"},
		{content: "雷池支持旁路部署。"},
	})
	ag2.SetCheckpointSource(func(_ string, _ string) ([]*domain.Checkpoint, error) { return saved, gotErr })

	// 重启后追问：restore 应生效（内存无链 → 从 saved 载入 → DetermineOp 走 followup）。
	content2, _, _, err := ag2.Message(ctx, nil, "sess-rs", "", "那部署方式呢？", nil)
	if err != nil {
		t.Fatalf("重启后追问失败: %v", err)
	}
	if content2 == "" {
		t.Error("重启后追问应正常回答")
	}
}

// TestAgentChatUsesMemoryTools（ADR-015 澄清）：聊天不是裸 chat——涉及事实时
// Agent 调 memory_search 查证后再回答。
func TestAgentChatUsesMemoryTools(t *testing.T) {
	ag, _, _ := newTestAgent(t, []step{
		// 第 1 轮：聊天涉及产品事实 → 查证
		{toolCallJSON: `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_0","type":"function","function":{"name":"memory_search","arguments":"{\"query\":\"雷池 WAF\",\"type\":\"product\"}"}}]}}]}`},
		// 第 2 轮：基于查证结果回答
		{content: "雷池是长亭的下一代 WAF，支持 CC 攻击防护等场景。"},
	})

	content, analysis, trace, err := ag.Message(context.Background(), nil, "sess-mc", "", "顺便问下雷池是什么产品？", nil)
	if err != nil {
		t.Fatalf("Message failed: %v", err)
	}
	if content == "" {
		t.Error("聊天应返回自然语言回答")
	}
	if analysis != nil {
		t.Error("聊天咨询不应触发 analysis_submit")
	}
	var sawSearch bool
	for _, tc := range trace.ToolCalls {
		if tc.Tool == "memory_search" {
			sawSearch = true
		}
		if tc.Tool == agent.ToolAnalysisSubmit {
			t.Error("聊天不应调用 analysis_submit")
		}
	}
	if !sawSearch {
		t.Error("涉及产品事实的聊天应调用 memory_search 查证（ADR-015：不是裸 chat）")
	}
}

// TestLastAnalysisSurvivesChats（审计修复验证）：连续纯聊天 followup 不会把
// 分析上下文挤出下一轮 prompt——BuildContext.LastAnalysis 回溯全链注入。
func TestLastAnalysisSurvivesChats(t *testing.T) {
	sm := shortterm.NewSessionManager()
	m := sm.Get("s")

	// 轮1：带分析 initial
	m.CreateInitialCheckpoint("客户文档", &domain.AnalysisResult{DemandAnalysis: "CC攻击防护需求"})
	// 轮2-4：连续三轮纯聊天 followup（挤掉 current/previous）
	m.CreateFollowupCheckpoint("你好", "你好！")
	m.CreateFollowupCheckpoint("在吗", "在")
	m.CreateFollowupCheckpoint("哈哈", "嗯")

	ctx := m.BuildContext(domain.OpFollowup)
	if ctx.Current.Type != domain.CheckpointFollowup {
		t.Fatalf("current 应为 followup，got %s", ctx.Current.Type)
	}
	if ctx.LastAnalysis == nil || ctx.LastAnalysis.Analysis == nil {
		t.Fatal("LastAnalysis 应回溯到链上的 initial 分析（不被聊天轮挤掉）")
	}
	if ctx.LastAnalysis.Analysis.DemandAnalysis != "CC攻击防护需求" {
		t.Errorf("LastAnalysis 内容错: %+v", ctx.LastAnalysis.Analysis)
	}
}

// TestRestorePromptKeepsAnalysis（审计要求：断言重启 restore 后 prompt 仍含分析上下文）。
func TestRestorePromptKeepsAnalysis(t *testing.T) {
	ag1, _, _ := newTestAgent(t, []step{
		{toolCallJSON: getLeiChiCall},
		{toolCallJSON: submitCall},
		{content: "结论：推荐雷池。"},
		{content: "聊天回复"},
		{content: "重启后的回答"},
	})
	ctx := context.Background()
	var saved []*domain.Checkpoint
	ag1.SetCheckpointSink(func(_ string, _ string, cp *domain.Checkpoint) { saved = append(saved, cp) })
	if _, a, _, err := ag1.Message(ctx, nil, "s", "", "我们网站被CC攻击", nil); err != nil || a == nil {
		t.Fatalf("首轮: %v", err)
	}
	if _, _, _, err := ag1.Message(ctx, nil, "s", "", "谢谢", nil); err != nil {
		t.Fatalf("聊天轮: %v", err)
	}

	// 模拟重启：新 Agent + 捕获实际发给 LLM 的 messages，断言含分析上下文。
	var sawAnalysisContext bool
	ag2, _, _ := newTestAgent(t, []step{
		{toolCallJSON: getLeiChiCall},
		{toolCallJSON: submitCall},
		{content: "结论：推荐雷池。"},
		{content: "聊天回复"},
		{content: "重启后的回答"},
	})
	ag2.SetCheckpointSource(func(_ string, _ string) ([]*domain.Checkpoint, error) { return saved, nil })
	// 用 emit 捕获不了 prompt；改为直接检查 restore 后的 BuildContext 经 Assemble 的结果——
	// 通过一个变通：restore 后再发一轮，检查 Agent 行为路径中 LastAnalysis 被注入。
	// 这里直接构造同链 assembler 验证（restore 的本质 = 同一条链）。
	sm2 := shortterm.NewSessionManager()
	sm2.Restore("s", saved)
	asm := assemblerForTest(t)
	msgs := asm.Assemble(nil, domain.OpFollowup, sm2.Get("s").BuildContext(domain.OpFollowup), "那部署呢")
	for _, m := range msgs {
		if m.Role == domain.RoleSystem && strings.Contains(m.Content, "客户遭遇CC攻击，需要Web防护") {
			sawAnalysisContext = true
		}
	}
	if !sawAnalysisContext {
		t.Fatal("restore 后 prompt 应包含分析上下文（LastAnalysis 注入）")
	}
	// ag2 重启后追问仍正常（restore 生效）
	if _, _, _, err := ag2.Message(ctx, nil, "s", "", "那部署方式呢", nil); err != nil {
		t.Fatalf("重启后追问: %v", err)
	}
}

// assemblerForTest 构造一个带种子知识库的拼装器。
func assemblerForTest(t *testing.T) *assembler.Assembler {
	return assembler.New(seedWiki(t), "")
}

// TestAgentDefinitionsAreValidJSON 确保工具定义可序列化为合法 JSON（给 LLM 用）。
func TestAgentDefinitionsAreValidJSON(t *testing.T) {
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
	if len(decoded) != 7 {
		t.Errorf("expected 7 memory tools, got %d", len(decoded))
	}
}

// TestAgentMissingAnswerLoop 追问闭环：missing_answer 工具调用 → checkpoint
// 记录 Answered → 后续轮次 prompt 注入「已回答的追问」。
func TestAgentMissingAnswerLoop(t *testing.T) {
	ag, sessions, _ := newTestAgent(t, []step{
		// 第 1 轮：记录两条追问答案
		{toolCallJSON: `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_0","type":"function","function":{"name":"missing_answer","arguments":"{\"answers\":[{\"item\":\"预算多少\",\"answer\":\"50万\"},{\"item\":\"部署环境\",\"answer\":\"私有云\"}]}"}}]}}]}`},
		// 第 2 轮：自然语言确认
		{content: "已记录：预算 50 万、私有云部署。"},
	})
	_, _, _, err := ag.Message(context.Background(), nil, "sess-ma", "", "预算50万，部署在私有云", nil)
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	// checkpoint 应带 Answered
	stm := sessions.Get("sess-ma")
	ctx := stm.BuildContext(domain.OpFollowup)
	var saw1, saw2 bool
	for _, a := range ctx.Answered {
		if a.Item == "预算多少" && a.Answer == "50万" {
			saw1 = true
		}
		if a.Item == "部署环境" && a.Answer == "私有云" {
			saw2 = true
		}
	}
	if !saw1 || !saw2 {
		t.Fatalf("BuildContext 应聚合两条 Answered，got %+v", ctx.Answered)
	}
	// 下一轮 prompt 注入「已回答的追问」
	asm := assemblerForTest(t)
	msgs := asm.Assemble(nil, domain.OpFollowup, ctx, "还有什么要确认的")
	injected := false
	for _, m := range msgs {
		if m.Role == domain.RoleSystem && strings.Contains(m.Content, "已回答的追问") &&
			strings.Contains(m.Content, "预算多少 → 50万") {
			injected = true
		}
	}
	if !injected {
		t.Fatal("prompt 应注入已回答的追问（追问闭环）")
	}
}

// TestAssemblerCustomerInjection 客户身份行（ADR-016 L2）：ctx.Customer
// 非空时注入一行身份（指引工具取画像），画像内容绝不注入。
func TestAssemblerCustomerInjection(t *testing.T) {
	wiki := seedWiki(t)
	// 建一个客户画像（含鲜明内容字段，若被注入必然断言失败）
	if err := wiki.UpsertEntry(&longterm.Entry{
		Type: domain.MemoryCustomer, Title: "某电商集团", Status: longterm.StatusVerified,
		Summary: "头部电商，大促流量峰值高",
		Content: "---\ntype: customer\nstatus: verified\nindustry: 电商\nscale: 大型\nsummary: 头部电商，大促流量峰值高\n---\n大促期间关注 CC 与库存接口防刷",
	}); err != nil {
		t.Fatal(err)
	}
	asm := assembler.New(wiki, "")
	ctx := &domain.SessionContext{Op: domain.OpInitial, Customer: "某电商集团"}
	msgs := asm.Assemble(nil, domain.OpInitial, ctx, "我们网站被CC攻击")
	foundIdentity := false
	for _, m := range msgs {
		if m.Role == domain.RoleSystem {
			if strings.Contains(m.Content, "【当前会话客户】某电商集团") {
				foundIdentity = true
			}
			// 画像内容字段绝不进 prompt（ADR-016：知识不注入）
			for _, banned := range []string{"行业：电商", "规模：大型", "【当前客户画像", "采购偏好"} {
				if strings.Contains(m.Content, banned) {
					t.Errorf("客户画像内容被注入 prompt（禁止）：%q 出现", banned)
				}
			}
		}
	}
	if !foundIdentity {
		t.Fatal("ctx.Customer 非空时应注入【当前会话客户】身份行")
	}
	// 无客户时不注入
	msgs2 := asm.Assemble(nil, domain.OpInitial, &domain.SessionContext{Op: domain.OpInitial}, "你好")
	for _, m := range msgs2 {
		if m.Role == domain.RoleSystem && strings.Contains(m.Content, "当前会话客户") {
			t.Fatal("无客户关联时不应注入客户身份行")
		}
	}
}

// TestAssemblerProductCatalogOnly 产品知识只注入目录索引（ADR-016 L3a）：
// 名字+一句话+别名进 prompt；能力/场景/竞品/威胁/合规/行业内容不进。
func TestAssemblerProductCatalogOnly(t *testing.T) {
	wiki := seedWiki(t)
	asm := assembler.New(wiki, "")
	msgs := asm.Assemble(nil, domain.OpInitial, &domain.SessionContext{Op: domain.OpInitial}, "你好")
	var sys string
	for _, m := range msgs {
		if m.Role == domain.RoleSystem {
			sys += m.Content
		}
	}
	if !strings.Contains(sys, "长亭产品目录") {
		t.Fatal("应注入产品目录索引")
	}
	if !strings.Contains(sys, "雷池") {
		t.Fatal("目录应含产品名")
	}
	// 全量知识注入的痕迹必须消失
	for _, banned := range []string{
		"CC 攻击防护 [1.0]", // capabilities 明细
		"已知威胁类型",        // 威胁全量段
		"合规要求",          // 合规全量段
		"行业场景",          // 行业全量段
	} {
		if strings.Contains(sys, banned) {
			t.Errorf("知识全量注入仍存在（ADR-016 禁止）：%q 出现", banned)
		}
	}
}

// TestAgentHistorySearch（P1）：Agent 能通过 history_search 工具回溯
// 历史消息原文（当前会话 + 跨会话两个范围）。
func TestAgentHistorySearch(t *testing.T) {
	// 真实 history.Store（临时库）
	hist, err := history.Open(filepath.Join(t.TempDir(), "h.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { hist.Close() })
	_ = hist.EnsureSession("", "sess-a", "会话A", "")
	_ = hist.EnsureSession("", "sess-b", "会话B", "")
	_, _ = hist.AppendMessage("sess-a", "user", "我们官网被挂马了，要过等保三级", "")
	_, _ = hist.AppendMessage("sess-b", "user", "上次说过的那个预算50万的项目", "")

	ag, sessions, calls := newTestAgent(t, []step{
		// 第 1 轮：跨会话搜"预算"
		{toolCallJSON: `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_0","type":"function","function":{"name":"history_search","arguments":"{\"query\":\"预算\",\"all_sessions\":true}"}}]}}]}`},
		// 第 2 轮：自然语言回答
		{content: "找到了：上次有个预算 50 万的项目（会话B）。"},
	})
	ag.SetHistorySearcher(func(scope *domain.TenantScope, sessionID, query string, limit int) ([]history.MessageHit, error) {
		// P0-03：作用域透传——测试单租户 legacy 传 nil（不按 owner 过滤）
		return hist.SearchMessages(scope, sessionID, query, limit)
	})
	_ = sessions
	content, _, _, err := ag.Message(context.Background(), nil, "sess-a", "", "之前有没有聊过预算的事", nil)
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	if !strings.Contains(content, "50 万") {
		t.Fatalf("应基于历史检索回答，got: %s", content)
	}
	if atomic.LoadInt32(calls) < 2 {
		t.Errorf("expected >=2 LLM calls, got %d", *calls)
	}
}

// TestAgentAskUser（F3）：Agent 调 ask_user → 轮次结束前 SSE 收到
// ask_user 事件（question+options），done 正常收尾。
func TestAgentAskUser(t *testing.T) {
	ag, _, _ := newTestAgent(t, []step{
		{toolCallJSON: `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_0","type":"function","function":{"name":"ask_user","arguments":"{\"question\":\"部署环境是？\",\"options\":[{\"label\":\"公有云\"},{\"label\":\"私有云\"}]}"}}]}}]}`},
		{content: "好的，请选择部署环境。"},
	})
	var events []agent.Event
	_, _, _, err := ag.Message(context.Background(), nil, "sess-ask", "", "客户想部署，不知道环境", func(e agent.Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatal(err)
	}
	var askEv, doneEv *agent.Event
	for i := range events {
		if events[i].Type == agent.EventAskUser && askEv == nil {
			askEv = &events[i]
		}
		if events[i].Type == agent.EventDone {
			doneEv = &events[i]
		}
	}
	if askEv == nil {
		t.Fatal("应发出 ask_user 事件")
	}
	if askEv.Question != "部署环境是？" || len(askEv.Options) != 2 || askEv.Options[0].Label != "公有云" {
		t.Fatalf("ask_user 事件内容不对: %+v", askEv)
	}
	if doneEv == nil {
		t.Fatal("done 应正常收尾")
	}
	// 事件顺序：ask_user 在 done 之前
	askIdx, doneIdx := -1, -1
	for i, e := range events {
		if e.Type == agent.EventAskUser && askIdx < 0 {
			askIdx = i
		}
		if e.Type == agent.EventDone {
			doneIdx = i
		}
	}
	if askIdx > doneIdx {
		t.Fatal("ask_user 应在 done 之前")
	}
}

// TestAgentBindCustomer 客户绑定 Agent 自主化（2026-08-15 用户裁决）：
// mock LLM 流 memory_list(customer) → ask_user（问客户，轮次终止式）→
// 用户点选后下一轮 session_bind_customer → 绑定落库回调。
// ask_user 是轮次终止式（2026-08-19 加固）：调用后本轮即结束等待用户选择，
// 绑定发生在用户回答后的下一轮，不再同一轮内继续空转。
func TestAgentBindCustomer(t *testing.T) {
	tc := func(i int, name, args string) string {
		return `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":` + fmt.Sprint(i) + `,"id":"call_` + fmt.Sprint(i) + `","type":"function","function":{"name":"` + name + `","arguments":` + jsonString(args) + `}}]}}]}`
	}
	ag, _, _ := newTestAgent(t, []step{
		{toolCallJSON: tc(0, "memory_list", `{"type":"customer"}`)},
		{toolCallJSON: tc(1, "ask_user", `{"question":"这是谁家客户？","options":[{"label":"某跨境电商"},{"label":"新客户"}]}`)},
		{toolCallJSON: tc(2, "session_bind_customer", `{"customer":"某跨境电商"}`)},
		{content: "已绑定某跨境电商，继续聊。"},
	})
	var bound []string
	ag.SetCustomerBinder(func(sessionID, customer string) error {
		bound = append(bound, sessionID, customer)
		return nil
	})
	// 第 1 轮：memory_list → ask_user，轮次终止（用户未回答前不得绑定）。
	if _, _, _, err := ag.Message(context.Background(), nil, "bind-s1", "", "客户想做大促防护", nil); err != nil {
		t.Fatal(err)
	}
	if len(bound) != 0 {
		t.Fatalf("用户未回答前不应绑定: %v", bound)
	}
	// 第 2 轮：用户点选「某跨境电商」→ 模型绑定客户并确认。
	if _, _, _, err := ag.Message(context.Background(), nil, "bind-s1", "", "某跨境电商", nil); err != nil {
		t.Fatal(err)
	}
	if len(bound) != 2 || bound[0] != "bind-s1" || bound[1] != "某跨境电商" {
		t.Fatalf("绑定回调应收到 (bind-s1, 某跨境电商): %v", bound)
	}
}

// TestAgentMaxIterationsConfigurable console-config：maxIter 可配置——
// 设为 2 时工具循环第 3 次前被截断（报超过迭代上限）。
func TestAgentMaxIterationsConfigurable(t *testing.T) {
	// 每步都返回 tool_call（无限循环脚本）
	tc := func(i int) string {
		return `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_` + fmt.Sprint(i) + `","type":"function","function":{"name":"memory_search","arguments":"{\"query\":\"x\",\"type\":\"product\"}"}}]}}]}`
	}
	steps := make([]step, 0, 10)
	for i := 0; i < 10; i++ {
		steps = append(steps, step{toolCallJSON: tc(i)})
	}
	ag, _, _ := newTestAgent(t, steps)
	ag.SetMaxIterations(2)
	_, _, _, err := ag.Message(context.Background(), nil, "iter-s1", "", "触发循环", nil)
	if err == nil || !strings.Contains(err.Error(), "超过最大迭代次数 2") {
		t.Fatalf("应报超过迭代上限 2: %v", err)
	}
}

// memSearchCall 返回一轮 memory_search 相同参数的 tool_call delta。
func memSearchCall(i int, query string) string {
	return `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_` + fmt.Sprint(i) + `","type":"function","function":{"name":"memory_search","arguments":"{\"query\":\"` + query + `\",\"type\":\"product\"}"}}]}}]}`
}

// TestAgentLoopDetectionRecovers 死循环防御：同一工具+参数重复 3 次即判定死循环，
// 注入纠正指令后模型收敛出最终答案 → 本轮成功返回（不再空转耗光 15 次上限）。
func TestAgentLoopDetectionRecovers(t *testing.T) {
	ag, _, calls := newTestAgent(t, []step{
		{toolCallJSON: memSearchCall(0, "WAF")},
		{toolCallJSON: memSearchCall(1, "WAF")},
		{toolCallJSON: memSearchCall(2, "WAF")}, // 第 3 次重复 → 判定死循环 → 纠正
		{content: "已根据检索结果给出最终结论。"},
	})
	content, _, _, err := ag.Message(context.Background(), nil, "loop-rec-s1", "", "客户需要Web防护", nil)
	if err != nil {
		t.Fatalf("纠正后应收敛成功: %v", err)
	}
	if !strings.Contains(content, "最终结论") {
		t.Fatalf("应返回纠正后的最终答案: %q", content)
	}
	// 共 4 次 LLM 调用（3 次重复 + 1 次纠正后的最终答案），远小于默认 15。
	if got := atomic.LoadInt32(calls); got != 4 {
		t.Fatalf("应恰好 4 次 LLM 调用（及时收敛），got %d", got)
	}
}

// TestAgentLoopDetectionRecoveryFails 死循环防御兜底：纠正指令后模型仍调同一工具 →
// 优雅降级报错（含卡点与重试建议），而非裸报「超过最大迭代次数」。
func TestAgentLoopDetectionRecoveryFails(t *testing.T) {
	ag, _, _ := newTestAgent(t, []step{
		{toolCallJSON: memSearchCall(0, "WAF")},
		{toolCallJSON: memSearchCall(1, "WAF")},
		{toolCallJSON: memSearchCall(2, "WAF")},
		{toolCallJSON: memSearchCall(3, "WAF")}, // 纠正后仍重复
	})
	_, _, _, err := ag.Message(context.Background(), nil, "loop-stuck-s1", "", "客户需要Web防护", nil)
	if err == nil {
		t.Fatal("纠正后仍循环应报错")
	}
	if !strings.Contains(err.Error(), "死循环") || !strings.Contains(err.Error(), "memory_search") {
		t.Fatalf("应报优雅降级错误（含卡点），got: %v", err)
	}
	if strings.Contains(err.Error(), "超过最大迭代次数") {
		t.Fatalf("不应再裸报超过迭代上限: %v", err)
	}
}

// TestAgentAskUserTerminatesRound ask_user 是轮次终止式（2026-08-19 加固）：
// 调用后本轮立即结束、等待用户选择，不再继续调 LLM 空转。
func TestAgentAskUserTerminatesRound(t *testing.T) {
	ag, _, calls := newTestAgent(t, []step{
		{toolCallJSON: `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_0","type":"function","function":{"name":"ask_user","arguments":"{\"question\":\"部署环境是？\",\"options\":[{\"label\":\"公有云\"},{\"label\":\"私有云\"}]}"}}]}}]}`},
		{content: "这步不应被触达（ask_user 应终止本轮）"},
	})
	content, _, _, err := ag.Message(context.Background(), nil, "ask-term-s1", "", "客户想部署", nil)
	if err != nil {
		t.Fatal(err)
	}
	// 仅 1 次 LLM 调用即收尾；脚本第 2 步（content）不应被消费。
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("ask_user 应恰好 1 次 LLM 调用即收尾，got %d", got)
	}
	if content != "" {
		t.Fatalf("ask_user 轮次返回内容应为空（问题随 ask_user 事件带出），got %q", content)
	}
}

// TestAgentAskUserStopsRemainingToolBatch ask_user 后同一模型响应里夹带的后续
// 工具也不得执行，防止提问后继续发生写入副作用。
func TestAgentAskUserStopsRemainingToolBatch(t *testing.T) {
	batch := `{"choices":[{"index":0,"delta":{"tool_calls":[` +
		`{"index":0,"id":"call_ask","type":"function","function":{"name":"ask_user","arguments":"{\"question\":\"是否确认？\",\"options\":[{\"label\":\"确认\"},{\"label\":\"取消\"}]}"}},` +
		`{"index":1,"id":"call_write","type":"function","function":{"name":"memory_ensure","arguments":"{\"type\":\"customer\",\"title\":\"不应写入\",\"content\":\"x\"}"}}` +
		`]}}]}`
	ag, _, calls := newTestAgent(t, []step{{toolCallJSON: batch}})
	_, _, trace, err := ag.Message(context.Background(), nil, "ask-batch-s1", "", "请确认", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Fatalf("ask_user 应在当前批次终止，LLM calls=%d", got)
	}
	if len(trace.ToolCalls) != 1 || trace.ToolCalls[0].Tool != agent.ToolAskUser {
		t.Fatalf("ask_user 后剩余工具不应执行: %+v", trace.ToolCalls)
	}
}

// TestAgentAnalysisSubmitForcesToolFreeFinal 结构化提交一旦成功，下一轮请求
// 必须 tools=空，只允许生成最终回答。
func TestAgentAnalysisSubmitForcesToolFreeFinal(t *testing.T) {
	var finalToolCount int32 = -1
	ag, _, calls := newTestAgentInspect(t, []step{
		{toolCallJSON: getLeiChiCall},
		{toolCallJSON: submitCall},
		{content: "最终结论：推荐雷池，并确认攻击规模。"},
	}, func(call, toolCount int) {
		if call == 3 {
			atomic.StoreInt32(&finalToolCount, int32(toolCount))
		}
	})
	content, analysis, _, err := ag.Message(context.Background(), nil, "final-only-s1", "", "网站被CC攻击", nil)
	if err != nil {
		t.Fatal(err)
	}
	if analysis == nil || !strings.Contains(content, "最终结论") {
		t.Fatalf("分析与最终回答应正常返回: analysis=%v content=%q", analysis, content)
	}
	if got := atomic.LoadInt32(calls); got != 3 {
		t.Fatalf("应在提交后恰好再调用一次收敛，got %d", got)
	}
	if got := atomic.LoadInt32(&finalToolCount); got != 0 {
		t.Fatalf("最终收敛请求不得携带工具定义，got %d", got)
	}
}

func TestAgentCachesRepeatedReadTool(t *testing.T) {
	ag, _, _ := newTestAgent(t, []step{
		{toolCallJSON: memSearchCall(0, "WAF")},
		{toolCallJSON: memSearchCall(1, "WAF")},
		{content: "已完成检索。"},
	})
	_, _, trace, err := ag.Message(context.Background(), nil, "cache-s1", "", "查一下WAF", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(trace.ToolCalls) != 2 || trace.ToolCalls[0].Cached || !trace.ToolCalls[1].Cached {
		t.Fatalf("第二次相同只读调用应命中本轮缓存: %+v", trace.ToolCalls)
	}
}

func TestAgentHidesRawReasoningAndIntermediateDraft(t *testing.T) {
	ag, _, _ := newTestAgent(t, []step{
		{
			reasoning:    "SECRET_CHAIN_OF_THOUGHT",
			content:      "这是未经核验的中间草稿",
			toolCallJSON: memSearchCall(0, "WAF"),
		},
		{content: "这是最终回答。"},
	})
	var events []agent.Event
	_, _, _, err := ag.Message(context.Background(), nil, "reasoning-safe-s1", "", "查询WAF", func(e agent.Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatal(err)
	}
	var visible strings.Builder
	for _, e := range events {
		if e.Type == agent.EventReasoning || e.Type == agent.EventContent {
			visible.WriteString(e.Text)
		}
	}
	got := visible.String()
	if strings.Contains(got, "SECRET_CHAIN_OF_THOUGHT") || strings.Contains(got, "未经核验的中间草稿") {
		t.Fatalf("原始思维链/中间草稿不应外发: %q", got)
	}
	if !strings.Contains(got, "正在理解需求") || !strings.Contains(got, "最终回答") {
		t.Fatalf("应保留安全阶段状态和最终回答: %q", got)
	}
}

// TestAgentAnalysisGateLoopRecovers analysis_submit 连续被拒（证据门禁）死循环：
// 不同参数连续提交 3 次均未成功 → 判定死循环 → 纠正后收敛成功。
// 用不同参数隔离「analysis_submit 被拒」规则（非「同签名重复」规则）。
func TestAgentAnalysisGateLoopRecovers(t *testing.T) {
	sub := func(i int, demand string) string {
		args := `{"demand_analysis":"` + demand + `","matched_products":[{"name":"雷池","confidence":0.9,"reason":"CC防护"}],"feasibility":"direct"}`
		return `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_` + fmt.Sprint(i) + `","type":"function","function":{"name":"analysis_submit","arguments":` + jsonString(args) + `}}]}}]}`
	}
	ag, _, calls := newTestAgent(t, []step{
		{toolCallJSON: sub(0, "第一版")}, // 未 memory_get → 证据门禁拒绝
		{toolCallJSON: sub(1, "第二版")}, // 未 memory_get → 拒绝
		{toolCallJSON: sub(2, "第三版")}, // 未 memory_get → 拒绝（第 3 次 → 死循环判定）
		{content: "信息不足，请补充客户部署环境。"},
	})
	content, _, _, err := ag.Message(context.Background(), nil, "gate-loop-s1", "", "客户遭遇CC攻击", nil)
	if err != nil {
		t.Fatalf("纠正后应收敛成功: %v", err)
	}
	if content == "" {
		t.Fatal("应返回纠正后的最终答案")
	}
	// 4 次调用即收敛（3 次被拒 + 1 次纠正后答案），不耗光默认 15 次。
	if got := atomic.LoadInt32(calls); got != 4 {
		t.Fatalf("应恰好 4 次 LLM 调用即收敛，got %d", got)
	}
}
