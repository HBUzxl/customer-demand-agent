// Package taskbg 实现后台任务域（ADR-015 / memory-v2 P11）。
//
// 与前台（Agent 对话，全记忆系统成套）相对：后台任务是无记忆依赖的
// 一次性 LLM 调用——不挂 checkpoint、不做 assembler 注入、不进会话。
// 典型任务：记忆固化（observe 注记→结构化草案→人审）、Wiki Lint、
// 会话标题生成。
package taskbg

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// TaskType 任务类型（注册新任务时扩展）。
type TaskType string

const (
	TaskConsolidate TaskType = "consolidate" // 记忆固化：观察→结构化草案→pending 人审
	TaskLint        TaskType = "lint"        // Wiki Lint：矛盾/孤儿/残缺→审核建议
	TaskTitle       TaskType = "title"       // 会话标题生成
)

// Task 是一次后台任务的执行记录（观测台消费）。
type Task struct {
	ID        string     `json:"id"`
	Type      TaskType   `json:"type"`
	Detail    string     `json:"detail"` // 任务描述（如固化的条目名）
	Status    string     `json:"status"` // running / done / failed
	Result    string     `json:"result"` // 完成摘要（观测台展示）
	CreatedAt time.Time  `json:"created_at"`
	DoneAt    *time.Time `json:"done_at,omitempty"`
}

// Runner 是后台任务执行器：串行执行（单 worker 消费队列——同一时刻至多
// 一个任务在跑，防 LLM 并发打爆网关）+ 任务列表（观测台消费）。
type Runner struct {
	mu      sync.Mutex
	tasks   []*Task
	history int
	queue   chan *Task
	exec    func(ctx context.Context, t *Task) error
}

// NewRunner 创建执行器（启动单个 worker goroutine 串行消费）。
func NewRunner(exec func(ctx context.Context, t *Task) error) *Runner {
	r := &Runner{history: 100, queue: make(chan *Task, 64), exec: exec}
	go r.worker()
	return r
}

// worker 串行消费队列。
func (r *Runner) worker() {
	for t := range r.queue {
		r.run(t)
	}
}

// Submit 提交任务（入队，立即返回任务记录——状态 running 表示排队+执行中）。
func (r *Runner) Submit(id string, typ TaskType, detail string) *Task {
	t := &Task{ID: id, Type: typ, Detail: detail, Status: "running", CreatedAt: time.Now()}
	r.mu.Lock()
	r.tasks = append(r.tasks, t)
	if len(r.tasks) > r.history {
		r.tasks = r.tasks[len(r.tasks)-r.history:]
	}
	r.mu.Unlock()
	r.queue <- t
	return t
}

func (r *Runner) run(t *Task) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var err error
	func() {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("[taskbg] 任务 %s panic: %v", t.ID, p)
				err = fmt.Errorf("panic: %v", p) // panic 记失败
			}
		}()
		err = r.exec(ctx, t)
	}()
	now := time.Now()
	r.mu.Lock()
	t.DoneAt = &now
	if err != nil {
		t.Status = "failed"
		t.Result = err.Error()
		log.Printf("[taskbg] 任务 %s(%s) 失败: %v", t.ID, t.Type, err)
	} else {
		t.Status = "done"
	}
	r.mu.Unlock()
}

// SetResult 任务执行中写结果（带锁——exec 回调在 worker goroutine，
// 观测台在 HTTP goroutine 读）。
func (r *Runner) SetResult(t *Task, result string) {
	r.mu.Lock()
	t.Result = result
	r.mu.Unlock()
}

// Status 返回任务当前状态（带锁——Task 字段跨 goroutine 读写）。
func (r *Runner) Status(t *Task) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return t.Status
}

// Result 返回任务结果（带锁）。
func (r *Runner) Result(t *Task) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return t.Result
}

// List 返回任务列表（倒序：最新在前）。
func (r *Runner) List(limit int) []*Task {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > len(r.tasks) {
		limit = len(r.tasks)
	}
	out := make([]*Task, limit)
	for i := 0; i < limit; i++ {
		out[i] = r.tasks[len(r.tasks)-1-i]
	}
	return out
}
