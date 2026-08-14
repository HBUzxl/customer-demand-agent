package tools

import (
	"encoding/json"
	"os"
	"testing"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/memory/longterm"
)

func newTestRegistry(t *testing.T) (*Registry, *longterm.WikiStore) {
	t.Helper()
	store := longterm.NewWikiStore(t.TempDir())
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}
	return NewRegistry(store), store
}

func TestPermissionMatrix(t *testing.T) {
	r, _ := newTestRegistry(t)

	// product 不可写
	_, err := r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "product", "title": "雷池", "content": "x",
	}))
	if err == nil {
		t.Fatal("writing product should be rejected")
	}

	// threat 可写但打待审核
	_, err = r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "threat", "title": "新型威胁", "content": "测试",
	}))
	if err != nil {
		t.Fatalf("writing threat should succeed: %v", err)
	}
	e, _ := r.store.GetEntry("threat", "新型威胁")
	if e.Status != longterm.StatusPendingReview {
		t.Fatalf("threat should be pending_review, got %s", e.Status)
	}

	// customer 可写且直接 verified
	_, err = r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "customer", "title": "客户A", "content": "x",
	}))
	if err != nil {
		t.Fatalf("writing customer should succeed: %v", err)
	}
	e, _ = r.store.GetEntry("customer", "客户A")
	if e.Status != longterm.StatusVerified {
		t.Fatalf("customer should be verified, got %s", e.Status)
	}

	// user 不可写
	_, err = r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "user", "title": "张三", "content": "x",
	}))
	if err == nil {
		t.Fatal("writing user should be rejected")
	}
}

func TestDeletePermission(t *testing.T) {
	r, _ := newTestRegistry(t)
	// 先写一个 customer
	_, _ = r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "customer", "title": "客户X", "content": "x",
	}))
	// customer 可删
	_, err := r.Execute("memory_delete", rawJSON(map[string]any{
		"type": "customer", "title": "客户X",
	}))
	if err != nil {
		t.Fatalf("deleting customer should succeed: %v", err)
	}
	// threat 不可删
	_, _ = r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "threat", "title": "威胁Y", "content": "x",
	}))
	_, err = r.Execute("memory_delete", rawJSON(map[string]any{
		"type": "threat", "title": "威胁Y",
	}))
	if err == nil {
		t.Fatal("deleting threat should be rejected")
	}
}

func TestUnknownTool(t *testing.T) {
	r, _ := newTestRegistry(t)
	_, err := r.Execute("memory_nonexistent", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("unknown tool should error")
	}
}

func TestSearchAndList(t *testing.T) {
	r, _ := newTestRegistry(t)
	// 写几条
	_, _ = r.Execute("memory_ensure", rawJSON(map[string]any{"type": "customer", "title": "电商客户", "content": "做电商", "tags": []string{"电商"}}))
	_, _ = r.Execute("memory_ensure", rawJSON(map[string]any{"type": "customer", "title": "金融客户", "content": "做金融"}))

	// list
	out, err := r.Execute("memory_list", rawJSON(map[string]any{"type": "customer"}))
	if err != nil {
		t.Fatal(err)
	}
	var lr struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal([]byte(out), &lr)
	if lr.Count != 2 {
		t.Fatalf("expected 2 customers, got %d", lr.Count)
	}

	// search
	out, err = r.Execute("memory_search", rawJSON(map[string]any{"query": "电商", "type": "customer"}))
	if err != nil {
		t.Fatal(err)
	}
	var sr struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal([]byte(out), &sr)
	if sr.Count != 1 {
		t.Fatalf("search 电商 expected 1, got %d", sr.Count)
	}
}

func TestDefinitions(t *testing.T) {
	r, _ := newTestRegistry(t)
	defs := r.Definitions()
	if len(defs) != 6 {
		t.Fatalf("expected 6 tool definitions, got %d", len(defs))
	}
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Function.Name] = true
	}
	for _, want := range []string{"memory_search", "memory_ensure", "memory_observe", "memory_delete", "memory_recall", "memory_list"} {
		if !names[want] {
			t.Errorf("missing tool definition: %s", want)
		}
	}
	_ = os.Stdout
}

func rawJSON(m map[string]any) json.RawMessage {
	b, _ := json.Marshal(m)
	return b
}

var _ domain.MemoryType // keep import

// TestEnsureOverwriteVerifiedDemotes P10：AI ensure 覆盖已 verified 条目时，
// threat/compliance/industry 必须降级回 pending（改动重新过审，防审核被绕过）。
func TestEnsureOverwriteVerifiedDemotes(t *testing.T) {
	r, store := newTestRegistry(t)
	// 人工先建一条 verified 威胁（模拟已审核通过）
	if err := store.UpsertEntry(&longterm.Entry{
		Type: domain.MemoryThreat, Title: "已审威胁", Status: longterm.StatusVerified,
		Content: "原始内容",
	}); err != nil {
		t.Fatal(err)
	}
	// AI ensure 覆盖
	if _, err := r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "threat", "title": "已审威胁", "content": "AI 改写的内容",
	})); err != nil {
		t.Fatal(err)
	}
	e, err := store.GetEntry("threat", "已审威胁")
	if err != nil {
		t.Fatal(err)
	}
	if e.Status != longterm.StatusPendingReview {
		t.Fatalf("AI 覆盖 verified 条目必须降级 pending，got %s", e.Status)
	}
}
