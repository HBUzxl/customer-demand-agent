package http

import (
	"context"
	"errors"
	"sync"

	"customer-demand-agent/internal/agent"
)

// ── Run 任务模型（conversation-ux F0）──────────────────────────
//
// 「跑」的状态属于服务端，不属于连接：POST /api/message 创建 Run（goroutine
// 执行）立即返回 202；刷新/切换/断连只影响订阅，不影响执行。事件写入
// per-session 缓冲（带递增 seq），订阅式 SSE 按 seq replay 后续传 live。

// ErrRunBusy 同会话已有活跃 Run（简单优先：拒绝并发，不排队）。
var ErrRunBusy = errors.New("该会话已有正在进行的任务")

// ErrRunNotFound 指定的 Run 不存在或已结束。
var ErrRunNotFound = errors.New("任务不存在或已结束")

const eventBufferCap = 500 // 每会话事件缓冲上限（一轮完整轨迹 + 余量）

// BufferedEvent 是带序号的已发生事件（replay 用）。
type BufferedEvent struct {
	Seq   int         `json:"seq"`
	Event agent.Event `json:"event"`
}

// Run 是一次服务端执行任务。
type Run struct {
	ID        string
	SessionID string
	cancel    context.CancelFunc
	done      chan struct{}
}

// RunManager 管理全部会话的 Run 与事件缓冲。
type RunManager struct {
	mu      sync.Mutex
	runs    map[string]*Run // sessionID → 活跃 Run
	buffer  map[string][]BufferedEvent
	lastSeq map[string]int
	subs    map[string]map[chan BufferedEvent]struct{} // sessionID → 订阅者
}

// NewRunManager 创建任务管理器。
func NewRunManager() *RunManager {
	return &RunManager{
		runs:    make(map[string]*Run),
		buffer:  make(map[string][]BufferedEvent),
		lastSeq: make(map[string]int),
		subs:    make(map[string]map[chan BufferedEvent]struct{}),
	}
}

// Start 为会话启动一个 Run（已有活跃 Run 返回 ErrBusy）。
// fn 在新 goroutine 中执行：ctx 取消 = 唯一停止途径；fn 返回即 Run 结束。
func (rm *RunManager) Start(sessionID, runID string, fn func(ctx context.Context, emit func(agent.Event))) error {
	rm.mu.Lock()
	if _, busy := rm.runs[sessionID]; busy {
		rm.mu.Unlock()
		return ErrRunBusy
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &Run{ID: runID, SessionID: sessionID, cancel: cancel, done: make(chan struct{})}
	rm.runs[sessionID] = run
	rm.mu.Unlock()

	emit := func(e agent.Event) {
		rm.mu.Lock()
		rm.lastSeq[sessionID]++
		be := BufferedEvent{Seq: rm.lastSeq[sessionID], Event: e}
		buf := append(rm.buffer[sessionID], be)
		if len(buf) > eventBufferCap {
			buf = buf[len(buf)-eventBufferCap:]
		}
		rm.buffer[sessionID] = buf
		subs := make([]chan BufferedEvent, 0, len(rm.subs[sessionID]))
		for ch := range rm.subs[sessionID] {
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
			delete(rm.runs, sessionID)
			// 通知订阅者流可以结束（关闭由 subscriber 自己感知 done）
			rm.mu.Unlock()
		}()
		fn(ctx, emit)
	}()
	return nil
}

// Running 该会话是否有活跃 Run。
func (rm *RunManager) Running(sessionID string) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	_, ok := rm.runs[sessionID]
	return ok
}

// RunID 返回会话当前活跃 Run 的 id（无则空）。
func (rm *RunManager) RunID(sessionID string) string {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if r, ok := rm.runs[sessionID]; ok {
		return r.ID
	}
	return ""
}

// Cancel 取消会话的活跃 Run（ctx cancel）。返回是否成功。
func (rm *RunManager) Cancel(sessionID, runID string) error {
	rm.mu.Lock()
	r, ok := rm.runs[sessionID]
	rm.mu.Unlock()
	if !ok || r.ID != runID {
		return ErrRunNotFound
	}
	r.cancel()
	return nil
}

// Subscribe 订阅会话事件：先拿 since 之后的 replay（快照），再返回 live
// 通道与结束信号。unsubscribe 必须调用（释放订阅者槽位）。
func (rm *RunManager) Subscribe(sessionID string, since int) (replay []BufferedEvent, live <-chan BufferedEvent, done <-chan struct{}, unsubscribe func()) {
	rm.mu.Lock()
	buf := rm.buffer[sessionID]
	for _, be := range buf {
		if be.Seq > since {
			replay = append(replay, be)
		}
	}
	ch := make(chan BufferedEvent, 64)
	if rm.subs[sessionID] == nil {
		rm.subs[sessionID] = make(map[chan BufferedEvent]struct{})
	}
	rm.subs[sessionID][ch] = struct{}{}
	var doneCh <-chan struct{}
	if r, ok := rm.runs[sessionID]; ok {
		doneCh = r.done
	}
	rm.mu.Unlock()

	unsub := func() {
		rm.mu.Lock()
		delete(rm.subs[sessionID], ch)
		rm.mu.Unlock()
	}
	// 无活跃 Run 且缓冲无新事件：doneCh 为 nil 表示可立即结束（replay 完即关流）
	return replay, ch, doneCh, unsub
}

// DropSession 清理会话的缓冲与订阅槽位（会话删除时调用）。
func (rm *RunManager) DropSession(sessionID string) {
	rm.mu.Lock()
	delete(rm.buffer, sessionID)
	delete(rm.lastSeq, sessionID)
	delete(rm.subs, sessionID)
	rm.mu.Unlock()
}

// ReplayAfter 返回缓冲中 seq 大于 after 的所有事件（慢订阅者补尾用）。
func (rm *RunManager) ReplayAfter(sessionID string, after int) []BufferedEvent {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	var out []BufferedEvent
	for _, be := range rm.buffer[sessionID] {
		if be.Seq > after {
			out = append(out, be)
		}
	}
	return out
}
