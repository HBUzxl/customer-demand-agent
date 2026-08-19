package tenancy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/model"
)

// M3 跨租户 Agent 记忆隔离测试：每个租户的 Runtime 装配了绑定自己数据面的
// tools.Registry / agent.Agent（§6.4），同一段 agent 轨迹（七记忆工具 /
// history_search / 客户绑定）在 A/B 两个租户上执行，各自只见自己的数据面。
// 断言建立在 trace.ToolCalls 的工具结果上——结果里混入别租户数据即失败。

// mtMockLLM 返回一个脚本化 mock LLM：每个运行的前 1 次调用返回 tool_call，
// 后续调用返回固定 content（恰好两轮：工具轮 + 收尾轮，跨多次运行按奇偶轮替）。
func mtMockLLM(t *testing.T, toolCallJSON, content string) *httptest.Server {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(atomic.AddInt32(&calls, 1))
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fl, _ := w.(http.Flusher)
		send := func(line string) {
			fmt.Fprint(w, "data: "+line+"\n\n")
			if fl != nil {
				fl.Flush()
			}
		}
		send(`{"choices":[{"index":0,"delta":{"role":"assistant"}}]}`)
		if n%2 == 1 && toolCallJSON != "" {
			send(toolCallJSON)
			send(`{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`)
		} else {
			send(`{"choices":[{"index":0,"delta":{"content":` + mtJSON(content) + `}}]}`)
			send(`{"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if fl != nil {
			fl.Flush()
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func mtJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// mtToolCall 构造带指定工具名的单工具 tool_calls delta JSON。
func mtToolCall(name, argsJSON string) string {
	return `{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_0","type":"function","function":{"name":"` + name + `","arguments":` + mtJSON(argsJSON) + `}}]}}]}`
}

// mtNewRegistry 构造带 mock LLM（llmURL 非空时）的租户注册表。
func mtNewRegistry(t *testing.T, llmURL string) *Registry {
	t.Helper()
	dir := t.TempDir()
	sysWiki := longterm.NewWikiStore(filepath.Join(dir, "system", "wiki"))
	if err := sysWiki.Load(); err != nil {
		t.Fatal(err)
	}
	cfgStore, err := config.Load(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var mgr *model.Manager
	if llmURL != "" {
		reg := model.NewRegistry(10 * time.Second)
		_ = reg.Register(model.ModelConfig{
			Name: "default", Endpoint: llmURL, APIKey: "k", Model: "mock",
			Temperature: 0.3, MaxTokens: 1024,
		})
		mgr = model.NewManager(llm.NewClient(), reg, &model.RouterConfig{Default: "default"})
	}
	return NewRegistry(mgr, sysWiki, filepath.Join(dir, "tenants"), cfgStore, nil)
}

// mtProvisionTwo 建两个已 Provision 的租户运行时（A/B，合法 UUID 形态 ID）。
func mtProvisionTwo(t *testing.T, llmURL string) (*Runtime, *Runtime) {
	t.Helper()
	reg := mtNewRegistry(t, llmURL)
	ctx := context.Background()
	ta, tb := strings.Repeat("a", 32), strings.Repeat("b", 32)
	for _, id := range []string{ta, tb} {
		if err := reg.Provision(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	ra, err := reg.ForTenant(ctx, ta)
	if err != nil {
		t.Fatal(err)
	}
	rb, err := reg.ForTenant(ctx, tb)
	if err != nil {
		t.Fatal(err)
	}
	return ra, rb
}

// mtRun 驱动某租户的 Agent 跑一轮（script 含一个工具调用），返回轨迹。
func mtRun(t *testing.T, rt *Runtime, tenantID, sessionID, text string) *agent.Trace {
	t.Helper()
	scope := &domain.TenantScope{
		TenantID: tenantID,
		UserID:   "user-" + tenantID[:1],
		Roles:    []string{domain.RoleOwner},
	}
	_, _, trace, err := rt.Agent.Message(context.Background(), scope, sessionID, "", text, nil)
	if err != nil {
		t.Fatalf("Agent.Message(%s) 失败: %v", tenantID, err)
	}
	return trace
}

func mtUpsert(t *testing.T, s longterm.Store, e *longterm.Entry) {
	t.Helper()
	if err := s.UpsertEntry(e); err != nil {
		t.Fatal(err)
	}
}

func mtToolResult(t *testing.T, tr *agent.Trace, wantTool string) string {
	t.Helper()
	for _, tc := range tr.ToolCalls {
		if tc.Tool == wantTool {
			return tc.Result
		}
	}
	t.Fatalf("轨迹应含工具 %s，got %+v", wantTool, tr.ToolCalls)
	return ""
}

// TestAgentMemorySearchCrossTenant 七记忆工具之 search（读）：同名客户「集团」
// 在各租户有各自 summary；A 的检索只见 A 版本，B 只见 B 版本，绝不串读。
func TestAgentMemorySearchCrossTenant(t *testing.T) {
	ra, rb := mtProvisionTwo(t, mtMockLLM(t, mtToolCall("memory_search", `{"query":"集团","type":"customer"}`), "ok").URL)

	// 同名客户「集团」：内容/summary 各自独立（互不覆盖）
	mtUpsert(t, ra.Wiki, &longterm.Entry{Type: domain.MemoryCustomer, Title: "集团", Content: "A 的画像：总部在上海", Summary: "A 版集团", Status: longterm.StatusVerified})
	mtUpsert(t, rb.Wiki, &longterm.Entry{Type: domain.MemoryCustomer, Title: "集团", Content: "B 的画像：总部在北京", Summary: "B 版集团", Status: longterm.StatusVerified})
	// 各自独有客户
	mtUpsert(t, ra.Wiki, &longterm.Entry{Type: domain.MemoryCustomer, Title: "A独家客户", Content: "仅 A", Summary: "A 专属", Status: longterm.StatusVerified})
	mtUpsert(t, rb.Wiki, &longterm.Entry{Type: domain.MemoryCustomer, Title: "B独家客户", Content: "仅 B", Summary: "B 专属", Status: longterm.StatusVerified})

	// A 搜「集团」→ 命中自己的 集团（A 版 summary），绝不混入 B 版/B 独有客户
	resA := mtToolResult(t, mtRun(t, ra, strings.Repeat("a", 32), "s1", "查一下客户"), "memory_search")
	if !strings.Contains(resA, "集团") || !strings.Contains(resA, "A 版集团") {
		t.Errorf("A 检索应命中 A 版本同名客户: %s", resA)
	}
	for _, banned := range []string{"B 版集团", "B 专属", "B独家客户", "总部在北京"} {
		if strings.Contains(resA, banned) {
			t.Errorf("A 检索混入 B 的数据 %q: %s", banned, resA)
		}
	}

	// B 搜「集团」→ 命中 B 版本，绝不混入 A 版/A 独有客户
	resB := mtToolResult(t, mtRun(t, rb, strings.Repeat("b", 32), "s1", "查一下客户"), "memory_search")
	if !strings.Contains(resB, "集团") || !strings.Contains(resB, "B 版集团") {
		t.Errorf("B 检索应命中 B 版本同名客户: %s", resB)
	}
	for _, banned := range []string{"A 版集团", "A 专属", "A独家客户", "总部在上海"} {
		if strings.Contains(resB, banned) {
			t.Errorf("B 检索混入 A 的数据 %q: %s", banned, resB)
		}
	}
}

// TestAgentMemoryListCrossTenant 七记忆工具之 list：A 列举客户只见自己（集团 +
// A独家客户），B 只见自己（集团 + B独家客户）。
func TestAgentMemoryListCrossTenant(t *testing.T) {
	ra, rb := mtProvisionTwo(t, mtMockLLM(t, mtToolCall("memory_list", `{"type":"customer"}`), "ok").URL)
	mtUpsert(t, ra.Wiki, &longterm.Entry{Type: domain.MemoryCustomer, Title: "集团", Content: "A", Summary: "A 版集团", Status: longterm.StatusVerified})
	mtUpsert(t, rb.Wiki, &longterm.Entry{Type: domain.MemoryCustomer, Title: "集团", Content: "B", Summary: "B 版集团", Status: longterm.StatusVerified})
	mtUpsert(t, ra.Wiki, &longterm.Entry{Type: domain.MemoryCustomer, Title: "A独家客户", Content: "仅 A", Summary: "A 专属", Status: longterm.StatusVerified})
	mtUpsert(t, rb.Wiki, &longterm.Entry{Type: domain.MemoryCustomer, Title: "B独家客户", Content: "仅 B", Summary: "B 专属", Status: longterm.StatusVerified})

	resA := mtToolResult(t, mtRun(t, ra, strings.Repeat("a", 32), "s1", "列举客户"), "memory_list")
	if !strings.Contains(resA, "集团") || !strings.Contains(resA, "A独家客户") {
		t.Errorf("A 列举应含自己的客户: %s", resA)
	}
	if strings.Contains(resA, "B独家客户") || strings.Contains(resA, "B 专属") {
		t.Errorf("A 列举混入 B 的客户: %s", resA)
	}

	resB := mtToolResult(t, mtRun(t, rb, strings.Repeat("b", 32), "s1", "列举客户"), "memory_list")
	if !strings.Contains(resB, "集团") || !strings.Contains(resB, "B独家客户") {
		t.Errorf("B 列举应含自己的客户: %s", resB)
	}
	if strings.Contains(resB, "A独家客户") || strings.Contains(resB, "A 专属") {
		t.Errorf("B 列举混入 A 的客户: %s", resB)
	}
}

// TestAgentMemoryGetCrossTenant 同名客户精读（memory_get）内容隔离：A 读到自己
// 版本，B 读到 B 版本。
func TestAgentMemoryGetCrossTenant(t *testing.T) {
	ra, rb := mtProvisionTwo(t, mtMockLLM(t, mtToolCall("memory_get", `{"type":"customer","title":"集团"}`), "ok").URL)
	mtUpsert(t, ra.Wiki, &longterm.Entry{Type: domain.MemoryCustomer, Title: "集团", Content: "A 的画像：总部在上海", Status: longterm.StatusVerified})
	mtUpsert(t, rb.Wiki, &longterm.Entry{Type: domain.MemoryCustomer, Title: "集团", Content: "B 的画像：总部在北京", Status: longterm.StatusVerified})

	resA := mtToolResult(t, mtRun(t, ra, strings.Repeat("a", 32), "s2", "精读"), "memory_get")
	if !strings.Contains(resA, "总部在上海") || strings.Contains(resA, "总部在北京") {
		t.Errorf("A 精读应读到 A 版本: %s", resA)
	}
	resB := mtToolResult(t, mtRun(t, rb, strings.Repeat("b", 32), "s2", "精读"), "memory_get")
	if !strings.Contains(resB, "总部在北京") || strings.Contains(resB, "总部在上海") {
		t.Errorf("B 精读应读到 B 版本: %s", resB)
	}
}

// TestAgentMemoryEnsureWriteCrossTenant 七记忆工具之写（ensure）：A 建的客户
// 只落 A 的租户 wiki；B 的数据面不可见。Registry 每租户独立实例（结构边界）。
func TestAgentMemoryEnsureWriteCrossTenant(t *testing.T) {
	ra, rb := mtProvisionTwo(t, mtMockLLM(t, mtToolCall("memory_ensure", `{"type":"customer","title":"新建客户A","content":"A 建立"}`), "ok").URL)

	if ra.Tools == rb.Tools {
		t.Fatal("每租户应有独立的工具注册表实例（绑定各自复合记忆）")
	}
	if ra.Wiki == rb.Wiki {
		t.Fatal("每租户应有独立的记忆存储实例")
	}

	res := mtToolResult(t, mtRun(t, ra, strings.Repeat("a", 32), "s3", "记一下新客户"), "memory_ensure")
	if !strings.Contains(res, "新建客户A") {
		t.Errorf("ensure 结果应确认写入: %s", res)
	}
	if _, err := ra.Wiki.GetEntry("customer", "新建客户A"); err != nil {
		t.Fatalf("A 的客户应落入 A 租户 wiki: %v", err)
	}
	if _, err := rb.Wiki.GetEntry("customer", "新建客户A"); err == nil {
		t.Fatal("B 租户数据面不应出现 A 建立的客户")
	}
}

// TestAgentHistorySearchCrossTenant history_search 跨租户：同一 session_id 在
// A/B 各自 history.db 存不同消息；各租户 Agent 检索只见自己的历史原文。
func TestAgentHistorySearchCrossTenant(t *testing.T) {
	ra, rb := mtProvisionTwo(t, mtMockLLM(t, mtToolCall("history_search", `{"query":"预算","all_sessions":true}`), "ok").URL)

	_ = ra.History.EnsureSession("user-a", "sess-shared", "A标题", "")
	_, _ = ra.History.AppendMessage("sess-shared", "user", "A 的项目预算 500 万", "")
	_ = rb.History.EnsureSession("user-b", "sess-shared", "B标题", "")
	_, _ = rb.History.AppendMessage("sess-shared", "user", "B 的方案已定，无预算讨论", "")

	resA := mtToolResult(t, mtRun(t, ra, strings.Repeat("a", 32), "sess-shared", "之前聊过预算吗"), "history_search")
	if !strings.Contains(resA, "500 万") {
		t.Errorf("A 的 history_search 应命中 A 的历史: %s", resA)
	}
	if strings.Contains(resA, "方案已定") {
		t.Errorf("A 的 history_search 不应命中 B 的历史: %s", resA)
	}

	resB := mtToolResult(t, mtRun(t, rb, strings.Repeat("b", 32), "sess-shared", "之前聊过预算吗"), "history_search")
	if !strings.Contains(resB, "方案已定") {
		t.Errorf("B 的 history_search 应命中 B 的历史: %s", resB)
	}
	if strings.Contains(resB, "500 万") {
		t.Errorf("B 的 history_search 不应命中 A 的历史: %s", resB)
	}
}

// TestAgentCustomerBindCrossTenant 客户绑定兼容路径隔离：HTTP 入口会先创建
// 当前租户会话，历史模型即使返回旧的 session_bind_customer 调用，也只能给
// A 数据面中已存在的会话补绑定；B 数据面不会被隐式创建或写入。
func TestAgentCustomerBindCrossTenant(t *testing.T) {
	ra, rb := mtProvisionTwo(t, mtMockLLM(t, mtToolCall("session_bind_customer", `{"customer":"集团"}`), "ok").URL)

	if err := ra.History.EnsureSession("user-a", "bind-s1", "待绑定会话", ""); err != nil {
		t.Fatalf("创建 A 会话: %v", err)
	}
	mtRun(t, ra, strings.Repeat("a", 32), "bind-s1", "把当前会话绑到集团")
	det, err := ra.History.GetSessionInternal("bind-s1", "")
	if err != nil {
		t.Fatalf("A 会话应存在: %v", err)
	}
	if det.Session.Customer != "集团" {
		t.Fatalf("A 绑定应落库 Customer=集团，got %q", det.Session.Customer)
	}
	if _, err := rb.History.GetSession(nil, "bind-s1", ""); err == nil {
		t.Fatal("B 数据面不应存在该会话（跨租户同 session_id 也 404）")
	}
}
