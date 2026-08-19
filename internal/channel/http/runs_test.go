package http

import (
	"context"
	"testing"
	"time"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/domain"
)

// TestRunPanicRecovery P0-08：Run 执行体 panic 不得击穿进程（Run goroutine
// recover），向订阅者发 error 事件、正常关流，且活跃 Run 表不残留。
func TestRunPanicRecovery(t *testing.T) {
	rm := NewRunManager()
	key := domain.RunKey{TenantID: "t1", SessionID: "s1"}

	if err := rm.Start(key, "r1", "user-a", func(ctx context.Context, emit func(agent.Event)) {
		panic("工具参数越界")
	}); err != nil {
		t.Fatal(err)
	}

	// 订阅：error 事件要么在 replay（已 emit 后订阅），要么在 live 通道。
	replay, live, done, unsub := rm.Subscribe(key, 0)
	defer unsub()
	gotError := false
	for _, be := range replay {
		if be.Event.Type == "error" {
			gotError = true
		}
	}
	if !gotError {
		select {
		case be := <-live:
			if be.Event.Type != "error" {
				t.Fatalf("期望 error 事件，got %+v", be.Event)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("超时未收到 error 事件（panic 未 recover 或缓冲未写入）")
		}
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run done 通道未关闭（goroutine 卡死）")
	}
	// 活跃表不残留：Running 应为 false
	if rm.Running(key) {
		t.Fatal("panic 后活跃 Run 表应已清理")
	}
}

func TestRunManagerCancelsOnRBACAndTenantLifecycle(t *testing.T) {
	rm := NewRunManager()
	keys := []domain.RunKey{
		{TenantID: "tenant-a", SessionID: "session-a1"},
		{TenantID: "tenant-a", SessionID: "session-b1"},
		{TenantID: "tenant-b", SessionID: "session-a2"},
	}
	actors := []string{"user-a", "user-b", "user-a"}
	dones := make([]<-chan struct{}, len(keys))
	for i, key := range keys {
		if err := rm.Start(key, "run-"+key.SessionID, actors[i], func(ctx context.Context, _ func(agent.Event)) {
			<-ctx.Done()
		}); err != nil {
			t.Fatal(err)
		}
		_, _, dones[i], _ = rm.Subscribe(key, 0)
	}

	waitDone := func(label string, done <-chan struct{}) {
		t.Helper()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("%s 未及时取消", label)
		}
	}

	rm.CancelTenantUser("tenant-a", "user-a")
	waitDone("成员角色变化后的 tenant-a/user-a Run", dones[0])
	if !rm.Running(keys[1]) || !rm.Running(keys[2]) {
		t.Fatal("租户成员级取消不应影响其他用户或其他租户")
	}

	rm.CancelUser("user-a")
	waitDone("全局停用后的 user-a Run", dones[2])
	if !rm.Running(keys[1]) {
		t.Fatal("用户级取消不应影响其他用户")
	}

	rm.DropTenant("tenant-a")
	waitDone("租户停用后的剩余 Run", dones[1])
}
