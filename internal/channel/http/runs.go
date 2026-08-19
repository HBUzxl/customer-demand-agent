package http

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/domain"
)

// ── Run 任务模型（conversation-ux F0）──────────────────────────
//
// 「跑」的状态属于服务端，不属于连接：POST /api/message 创建 Run（goroutine
// 执行）立即返回 202；刷新/切换/断连只影响订阅，不影响执行。事件写入
// per-session 缓冲（带递增 seq），订阅式 SSE 按 seq replay 后续传 live。
//
// 多租户（§9.3）：所有 key 为 RunKey{TenantID, SessionID}——同一 session_id
// 在不同租户互不干扰；DropTenant 关闭整个租户的 Run 与缓冲。

// ErrRunBusy 同会话已有活跃 Run（简单优先：拒绝并发，不排队）。
var ErrRunBusy = errors.New("该会话已有正在进行的任务")

// ErrRunNotFound 指定的 Run 不存在或已结束。
var ErrRunNotFound = errors.New("任务不存在或已结束")

const eventBufferCap = 500 // 每会话事件缓冲上限（一轮完整轨迹 + 余量）

// BufferedEvent 是带序号的已发生事件（replay 用；RunID 标注产生它的 Run，
// replay 时按事件自身归属输出 x-run——跨 Run 不串台）。
type BufferedEvent struct {
	Seq   int         `json:"seq"`
	RunID string      `json:"run_id,omitempty"`
	Event agent.Event `json:"event"`
}

// Run 是一次服务端执行任务。
type Run struct {
	ID          string
	SessionID   string
	ActorUserID string
	cancel      context.CancelFunc
	done        chan struct{}
}

// RunManager 管理全部会话的 Run 与事件缓冲。
type RunManager struct {
	mu      sync.Mutex
	runs    map[domain.RunKey]*Run
	buffer  map[domain.RunKey][]BufferedEvent
	lastSeq map[domain.RunKey]int
	subs    map[domain.RunKey]map[chan BufferedEvent]struct{}
}

// NewRunManager 创建任务管理器。
func NewRunManager() *RunManager {
	return &RunManager{
		runs:    make(map[domain.RunKey]*Run),
		buffer:  make(map[domain.RunKey][]BufferedEvent),
		lastSeq: make(map[domain.RunKey]int),
		subs:    make(map[domain.RunKey]map[chan BufferedEvent]struct{}),
	}
}

// Start 为会话启动一个 Run（已有活跃 Run 返回 ErrBusy）。
// fn 在新 goroutine 中执行：ctx 取消 = 唯一停止途径；fn 返回即 Run 结束。
func (rm *RunManager) Start(key domain.RunKey, runID, actorUserID string, fn func(ctx context.Context, emit func(agent.Event))) error {
	rm.mu.Lock()
	if _, busy := rm.runs[key]; busy {
		rm.mu.Unlock()
		return ErrRunBusy
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &Run{
		ID: runID, SessionID: key.SessionID, ActorUserID: actorUserID,
		cancel: cancel, done: make(chan struct{}),
	}
	rm.runs[key] = run
	rm.mu.Unlock()

	emit := func(e agent.Event) {
		rm.mu.Lock()
		rm.lastSeq[key]++
		be := BufferedEvent{Seq: rm.lastSeq[key], RunID: run.ID, Event: e}
		buf := append(rm.buffer[key], be)
		if len(buf) > eventBufferCap {
			buf = buf[len(buf)-eventBufferCap:]
		}
		rm.buffer[key] = buf
		subs := make([]chan BufferedEvent, 0, len(rm.subs[key]))
		for ch := range rm.subs[key] {
			subs = append(subs, ch)
		}
		rm.mu.Unlock()
		for _, ch := range subs {
			select {
			case ch <- be:
			default: // 订阅者跟不上（慢连接）：丢弃，replay 会补
			}
		}
	}

	go func() {
		defer close(run.done)
		defer func() {
			rm.mu.Lock()
			delete(rm.runs, key)
			// 通知订阅者流可以结束（关闭由 subscriber 自己感知 done）
			rm.mu.Unlock()
		}()
		// P0-08：Run 执行体 panic 不得击穿整个进程（工具参数/存储层的任何越界
		// 都应被兜住）——recover 后向订阅者发 error/done，保证 SSE 正常关流。
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[recover] Run %s panic: %v", run.ID, rec)
				emit(agent.Event{Type: "error", Error: fmt.Sprintf("执行出错（内部异常）: %v", rec)})
			}
		}()
		fn(ctx, emit)
	}()
	return nil
}

// Running 该会话是否有活跃 Run。
func (rm *RunManager) Running(key domain.RunKey) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	_, ok := rm.runs[key]
	return ok
}

// RunID 返回会话当前活跃 Run 的 id（无则空）。
func (rm *RunManager) RunID(key domain.RunKey) string {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if r, ok := rm.runs[key]; ok {
		return r.ID
	}
	return ""
}

// Cancel 取消会话的活跃 Run（ctx cancel）。返回是否成功。
func (rm *RunManager) Cancel(key domain.RunKey, runID string) error {
	rm.mu.Lock()
	r, ok := rm.runs[key]
	rm.mu.Unlock()
	if !ok || r.ID != runID {
		return ErrRunNotFound
	}
	r.cancel()
	return nil
}

// Subscribe 订阅会话事件：先拿 since 之后的 replay（快照），再返回 live
// 通道与结束信号。unsubscribe 必须调用（释放订阅者槽位）。
func (rm *RunManager) Subscribe(key domain.RunKey, since int) (replay []BufferedEvent, live <-chan BufferedEvent, done <-chan struct{}, unsubscribe func()) {
	rm.mu.Lock()
	buf := rm.buffer[key]
	for _, be := range buf {
		if be.Seq > since {
			replay = append(replay, be)
		}
	}
	ch := make(chan BufferedEvent, 64)
	if rm.subs[key] == nil {
		rm.subs[key] = make(map[chan BufferedEvent]struct{})
	}
	rm.subs[key][ch] = struct{}{}
	var doneCh <-chan struct{}
	if r, ok := rm.runs[key]; ok {
		doneCh = r.done
	}
	rm.mu.Unlock()

	unsub := func() {
		rm.mu.Lock()
		delete(rm.subs[key], ch)
		rm.mu.Unlock()
	}
	// 无活跃 Run 且缓冲无新事件：doneCh 为 nil 表示可立即结束（replay 完即关流）
	return replay, ch, doneCh, unsub
}

// DropSession 清理会话的缓冲与订阅槽位（会话删除时调用）。
func (rm *RunManager) DropSession(key domain.RunKey) {
	rm.mu.Lock()
	delete(rm.buffer, key)
	delete(rm.lastSeq, key)
	delete(rm.subs, key)
	rm.mu.Unlock()
}

// DropTenant 清理租户的全部 Run 与缓冲（租户关闭/退出时调用）。
func (rm *RunManager) DropTenant(tenantID string) {
	rm.mu.Lock()
	for key := range rm.runs {
		if key.TenantID == tenantID {
			rm.runs[key].cancel()
		}
	}
	for key := range rm.buffer {
		if key.TenantID == tenantID {
			delete(rm.buffer, key)
			delete(rm.lastSeq, key)
			delete(rm.subs, key)
		}
	}
	rm.mu.Unlock()
}

// CancelTenantUser 在成员角色变化或移除时取消该用户在目标租户仍在执行的 Run。
// 角色授权是 Run 启动时的快照；不取消会让已降级/移除用户继续用旧角色执行写工具。
func (rm *RunManager) CancelTenantUser(tenantID, userID string) {
	rm.mu.Lock()
	for key, run := range rm.runs {
		if key.TenantID == tenantID && run.ActorUserID == userID {
			run.cancel()
		}
	}
	rm.mu.Unlock()
}

// CancelUser 在全局停用用户或重置密码时取消其跨工作空间的全部活动 Run。
func (rm *RunManager) CancelUser(userID string) {
	rm.mu.Lock()
	for _, run := range rm.runs {
		if run.ActorUserID == userID {
			run.cancel()
		}
	}
	rm.mu.Unlock()
}

// ReplayAfter 返回缓冲中 seq 大于 after 的所有事件（慢订阅者补尾用）。
func (rm *RunManager) ReplayAfter(key domain.RunKey, after int) []BufferedEvent {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	var out []BufferedEvent
	for _, be := range rm.buffer[key] {
		if be.Seq > after {
			out = append(out, be)
		}
	}
	return out
}
