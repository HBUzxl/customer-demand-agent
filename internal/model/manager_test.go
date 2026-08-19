package model

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"customer-demand-agent/internal/llm"
)

func streamOK(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"}}]}\n\n")
	fmt.Fprintf(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":%q}}]}\n\n", content)
	fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
	fmt.Fprint(w, "data: [DONE]\n\n")
}

func testModelConfig(name, endpoint string) ModelConfig {
	return ModelConfig{
		Name: name, Endpoint: endpoint, APIKey: "test", Model: "mock",
		TimeoutSec: 2,
	}
}

func TestRouterSnapshotIsDeepCopy(t *testing.T) {
	in := &RouterConfig{
		Default: "primary",
		Routes:  map[TaskType]string{TaskAnalysis: "analysis"},
		Fallback: FallbackPolicy{
			MaxRetries: 1, BackoffMs: 1, Chain: []string{"backup"},
		},
	}
	m := NewManager(nil, nil, in)

	// 构造后继续修改调用方对象，不得改变 Manager 内部路由。
	in.Default = "mutated"
	in.Routes[TaskAnalysis] = "mutated"
	in.Fallback.Chain[0] = "mutated"
	got := m.Router()
	if got.Default != "primary" || got.Routes[TaskAnalysis] != "analysis" || got.Fallback.Chain[0] != "backup" {
		t.Fatalf("Manager 应持有深拷贝快照: %+v", got)
	}

	// Router() 返回值也必须可安全修改，不反向污染 Manager。
	got.Routes[TaskAnalysis] = "outside"
	got.Fallback.Chain[0] = "outside"
	again := m.Router()
	if again.Routes[TaskAnalysis] != "analysis" || again.Fallback.Chain[0] != "backup" {
		t.Fatalf("Router 返回值不应共享 map/slice: %+v", again)
	}
}

func TestChatStreamRetriesBeforeAnyVisibleDelta(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		streamOK(w, "retry-ok")
	}))
	defer srv.Close()

	reg := NewRegistry(2 * time.Second)
	if err := reg.Register(testModelConfig("primary", srv.URL)); err != nil {
		t.Fatal(err)
	}
	m := NewManager(llm.NewClient(), reg, &RouterConfig{
		Default: "primary",
		Routes:  map[TaskType]string{},
		Fallback: FallbackPolicy{
			MaxRetries: 1, BackoffMs: 1,
		},
	})
	resp, err := m.ChatStream(context.Background(), TaskAnalysis, &llm.ChatRequest{}, llm.DeltaCallbacks{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message.Content != "retry-ok" || atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("流式调用应在未输出时重试一次: content=%q calls=%d", resp.Message.Content, calls)
	}
	audit := m.AuditRecent(10)
	if len(audit) != 2 || !audit[0].OK || audit[1].OK {
		t.Fatalf("每次流式尝试都应有准确审计: %+v", audit)
	}
}

func TestChatStreamFallbackAuditUsesActualModel(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer primary.Close()
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		streamOK(w, "fallback-ok")
	}))
	defer backup.Close()

	reg := NewRegistry(2 * time.Second)
	if err := reg.Register(testModelConfig("primary", primary.URL)); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(testModelConfig("backup", backup.URL)); err != nil {
		t.Fatal(err)
	}
	m := NewManager(llm.NewClient(), reg, &RouterConfig{
		Default: "primary", Routes: map[TaskType]string{},
		Fallback: FallbackPolicy{MaxRetries: 1, BackoffMs: 1, Chain: []string{"backup"}},
	})
	resp, err := m.ChatStream(context.Background(), TaskAnalysis, &llm.ChatRequest{}, llm.DeltaCallbacks{})
	if err != nil || resp.Message.Content != "fallback-ok" {
		t.Fatalf("fallback 应成功: resp=%+v err=%v", resp, err)
	}
	audit := m.AuditRecent(10)
	if len(audit) < 2 || audit[0].Model != "backup" || !audit[0].Fallback || !audit[0].OK {
		t.Fatalf("成功审计应记录实际 backup 模型和 fallback=true: %+v", audit)
	}
	if audit[1].Model != "primary" || audit[1].OK {
		t.Fatalf("失败审计应记录 primary: %+v", audit)
	}
}

func TestChatStreamSkipsDisabledModel(t *testing.T) {
	var primaryCalls int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&primaryCalls, 1)
		streamOK(w, "disabled-should-not-run")
	}))
	defer primary.Close()
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		streamOK(w, "backup-ok")
	}))
	defer backup.Close()

	reg := NewRegistry(2 * time.Second)
	disabled := testModelConfig("primary", primary.URL)
	off := false
	disabled.Enabled = &off
	if err := reg.Register(disabled); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(testModelConfig("backup", backup.URL)); err != nil {
		t.Fatal(err)
	}
	m := NewManager(llm.NewClient(), reg, &RouterConfig{
		Default: "primary", Routes: map[TaskType]string{},
		Fallback: FallbackPolicy{MaxRetries: 1, BackoffMs: 1, Chain: []string{"backup"}},
	})
	resp, err := m.ChatStream(context.Background(), TaskAnalysis, &llm.ChatRequest{}, llm.DeltaCallbacks{})
	if err != nil || resp.Message.Content != "backup-ok" {
		t.Fatalf("停用 primary 后应直接走 backup: resp=%+v err=%v", resp, err)
	}
	if atomic.LoadInt32(&primaryCalls) != 0 {
		t.Fatal("停用模型不应收到请求")
	}
}
