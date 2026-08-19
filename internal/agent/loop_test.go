package agent

import (
	"strings"
	"testing"
	"unicode/utf8"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/memory/longterm"
	memorytools "customer-demand-agent/internal/memory/tools"
)

// TestDetectLoop 死循环判定：
// 同一「工具+参数」签名重复 ≥3 → 命中；analysis_submit 提交 ≥3 次未成功 → 命中；
// 正常多工具调用 / 少量重复 → 不命中。
func TestDetectLoop(t *testing.T) {
	// 1) 同一签名重复 3 次 → 命中（返回带工具名与次数）
	reason := detectLoop(map[string]int{"memory_search\x00{\"query\":\"x\"}": 3}, 0, nil)
	if reason == "" || !strings.Contains(reason, "memory_search") {
		t.Fatalf("同一签名重复 3 次应命中，got %q", reason)
	}
	// 2) 重复 2 次 → 不命中
	if reason := detectLoop(map[string]int{"memory_search\x00{\"query\":\"x\"}": 2}, 0, nil); reason != "" {
		t.Fatalf("重复 2 次不应命中，got %q", reason)
	}
	// 3) 不同签名各 2 次 → 不命中（多工具协同，非循环）
	if reason := detectLoop(map[string]int{
		"memory_search\x00{\"query\":\"x\"}": 2,
		"memory_get\x00{\"title\":\"雷池\"}":   2,
	}, 0, nil); reason != "" {
		t.Fatalf("多工具协同不应命中，got %q", reason)
	}
	// 4) analysis_submit 提交 3 次未成功（st.analysis==nil）→ 命中
	if reason := detectLoop(map[string]int{}, 3, &turnState{}); reason == "" || !strings.Contains(reason, "analysis_submit") {
		t.Fatalf("analysis_submit 连续被拒应命中，got %q", reason)
	}
	// 5) analysis_submit 提交 3 次但已成功（st.analysis!=nil）→ 不命中
	if reason := detectLoop(map[string]int{}, 3, &turnState{analysis: &AnalysisSubmission{}}); reason != "" {
		t.Fatalf("提交成功不应命中，got %q", reason)
	}
	// 6) analysis_submit 只提交 2 次 → 不命中
	if reason := detectLoop(map[string]int{}, 2, &turnState{}); reason != "" {
		t.Fatalf("提交 2 次不应命中，got %q", reason)
	}
	// 7) 空 → 不命中
	if reason := detectLoop(map[string]int{}, 0, nil); reason != "" {
		t.Fatalf("空调用不应命中，got %q", reason)
	}
}

func TestValidateToolBatchBudgetAndProtocol(t *testing.T) {
	call := func(id, name, args string) domain.ToolCall {
		return domain.ToolCall{ID: id, Type: "function", Function: domain.ToolCallFunction{Name: name, Arguments: args}}
	}
	if err := validateToolBatch([]domain.ToolCall{call("c1", "memory_search", `{}`)}, 0); err != nil {
		t.Fatalf("合法工具批次不应拒绝: %v", err)
	}
	tooMany := make([]domain.ToolCall, maxToolCallsPerRound+1)
	for i := range tooMany {
		tooMany[i] = call(string(rune('a'+i)), "memory_search", `{}`)
	}
	if err := validateToolBatch(tooMany, 0); err == nil || !strings.Contains(err.Error(), "单轮") {
		t.Fatalf("超单轮预算应拒绝: %v", err)
	}
	if err := validateToolBatch([]domain.ToolCall{call("c1", "memory_search", `{}`)}, maxToolCallsPerTurn); err == nil {
		t.Fatal("超累计预算应拒绝")
	}
	if err := validateToolBatch([]domain.ToolCall{
		call("dup", "memory_search", `{}`), call("dup", "memory_get", `{}`),
	}, 0); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("重复 tool_call id 应拒绝: %v", err)
	}
	if err := validateToolBatch([]domain.ToolCall{call("c1", "memory_search", strings.Repeat("x", maxToolArgumentBytes+1))}, 0); err == nil {
		t.Fatal("超大工具参数应拒绝")
	}
}

func TestSafeReasoningStatusDoesNotExposeRawReasoning(t *testing.T) {
	for _, tc := range []struct {
		iter int
		mode string
	}{
		{0, ""}, {1, ""}, {2, "analysis"}, {3, "recovery"},
	} {
		got := safeReasoningStatus(tc.iter, tc.mode)
		if strings.TrimSpace(got) == "" || strings.Contains(got, "prompt") {
			t.Fatalf("阶段状态应是非空安全文案: %q", got)
		}
	}
}

func TestTruncateForTracePreservesUTF8(t *testing.T) {
	in := strings.Repeat("客", 600)
	got := truncateForTrace(in)
	if !strings.HasSuffix(got, "…") || !utf8.ValidString(got) {
		t.Fatalf("截断应保持 UTF-8 完整: %q", got[len(got)-8:])
	}
}

// TestCompactJSON 签名归一化：键序/空白差异不影响签名；非法 JSON 退化为原串。
func TestCompactJSON(t *testing.T) {
	a := compactJSON(`{"query":"x","type":"product"}`)
	b := compactJSON(`{ "type" : "product",  "query"  : "x" }`)
	if a != b {
		t.Fatalf("键序/空白不同的参数应同签名：%q vs %q", a, b)
	}
	if a != `{"query":"x","type":"product"}` {
		t.Fatalf("compact 结果异常: %s", a)
	}
	if c := compactJSON(`not-json{`); c != "not-json{" {
		t.Fatalf("非法 JSON 应退化为原串，got %q", c)
	}
}

// TestCanonicalProductFromResult 从 memory_get 结果首行解析规范产品名：
// 含括号的标题不被误切、错误 JSON/不存在响应返回空。
func TestCanonicalProductFromResult(t *testing.T) {
	cases := []struct{ in, want string }{
		{"# 雷池（product｜verified）\n> summary\n正文", "雷池"},
		{"# 雷池（SafeLine）WAF（product｜verified）\n正文", "雷池（SafeLine）WAF"},
		{`{"error":"product/雷池 不存在或不可见"}`, ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := canonicalProductFromResult(c.in); got != c.want {
			t.Fatalf("canonicalProductFromResult(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestReadProductLenient 证据门禁名称容忍：归一化/互相包含都算已读，
// 避免产品名拼写偏差把模型卡在「读详情→重提被拒」死循环。
func TestReadProductLenient(t *testing.T) {
	st := &turnState{readProducts: map[string]bool{"雷池（SafeLine）WAF": true}}
	if !st.readProduct("雷池（SafeLine）WAF") {
		t.Fatal("全名应匹配")
	}
	if !st.readProduct("雷池 WAF") {
		t.Fatal("归一化后互相包含应匹配（防拼写偏差死循环）")
	}
	if st.readProduct("WAF") {
		// 「WAF」是雷池的别名，包含于已读产品名 → 命中合理（读的就是该产品）
		t.Log("WAF 别名命中：合理")
	}
	if st.readProduct("日志审计") {
		t.Fatal("无包含关系的无关名不应匹配")
	}
	// 大小写与空白归一化
	st2 := &turnState{readProducts: map[string]bool{"SafeLine": true}}
	if !st2.readProduct("  safeline  ") {
		t.Fatal("应做小写/去空白归一化")
	}
}

func TestMemoryGetManyMarksEveryProductAsRead(t *testing.T) {
	store := longterm.NewWikiStore(t.TempDir())
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"全悉", "谛听"} {
		if err := store.UpsertEntry(&longterm.Entry{
			Type: domain.MemoryProduct, Title: name, Content: name + "产品正文",
			Status: longterm.StatusVerified,
		}); err != nil {
			t.Fatal(err)
		}
	}
	ag := &Agent{tools: memorytools.NewRegistry(store)}
	st := &turnState{readProducts: map[string]bool{}}
	out := ag.execMemoryGetMany(`{"items":[{"type":"product","title":"全悉"},{"type":"product","title":"谛听"}]}`, st)
	if strings.Contains(out, `"error"`) {
		t.Fatalf("批量读取不应失败: %s", out)
	}
	if !st.readProduct("全悉") || !st.readProduct("谛听") {
		t.Fatalf("批量读取的每个产品都应通过证据门禁，got %v", st.readProducts)
	}
}

func TestWithoutToolsEnforcesHarnessBudget(t *testing.T) {
	defs := []domain.Tool{
		{Type: "function", Function: domain.ToolFunction{Name: "memory_search"}},
		{Type: "function", Function: domain.ToolFunction{Name: "memory_get"}},
		{Type: "function", Function: domain.ToolFunction{Name: ToolAnalysisSubmit}},
	}
	got := withoutTools(defs, "memory_search", "memory_get")
	if len(got) != 1 || got[0].Function.Name != ToolAnalysisSubmit {
		t.Fatalf("预算耗尽后应只保留未禁用工具，got %+v", got)
	}
	if len(defs) != 3 {
		t.Fatal("过滤工具定义不得修改原切片")
	}
}
