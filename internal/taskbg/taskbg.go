// Package taskbg 实现后台任务域（ADR-015 / memory-v2 P11）。
//
// 与前台（Agent 对话，全记忆系统成套）相对：后台任务是无记忆依赖的
// 一次性 LLM 调用——不挂 checkpoint、不做 assembler 注入、不进会话。
// 典型任务：记忆固化（observe 注记→结构化草案→人审）、Wiki Lint、
// 会话标题生成。
package taskbg

import (
	"context"
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

// Runner 是后台任务执行器：任务列表（观测）+ 串行执行（简单优先）。
type Runner struct {
	mu      sync.Mutex
	tasks   []*Task
	history int // 保留最近多少条记录
	// exec 具体执行函数（按类型分发，由 main 注入——依赖 llm/wiki 但本包不 import）
	exec func(ctx context.Context, t *Task) error
}

// NewRunner 创建执行器。exec 为实际执行函数。
func NewRunner(exec func(ctx context.Context, t *Task) error) *Runner {
	return &Runner{history: 100, exec: exec}
}

// Submit 提交任务（异步执行，立即返回任务记录）。
func (r *Runner) Submit(id string, typ TaskType, detail string) *Task {
	t := &Task{ID: id, Type: typ, Detail: detail, Status: "running", CreatedAt: time.Now()}
	r.mu.Lock()
	r.tasks = append(r.tasks, t)
	if len(r.tasks) > r.history {
		r.tasks = r.tasks[len(r.tasks)-r.history:]
	}
	r.mu.Unlock()
	go r.run(t)
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
				err = nil // panic 视为完成（防拖垮进程）
			}
		}()
		err = r.exec(ctx, t)
	}()
	now := time.Now()
	t.DoneAt = &now
	if err != nil {
		t.Status = "failed"
		t.Result = err.Error()
		log.Printf("[taskbg] 任务 %s(%s) 失败: %v", t.ID, t.Type, err)
	} else {
		t.Status = "done"
	}
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
