package model

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/llm"
)

// Manager 统一调度模型调用：按任务类型路由，失败时按回退链重试。
type Manager struct {
	client   *llm.Client
	registry *Registry
	router   *RouterConfig
	audit    []CallAudit // C2：最近 LLM 调用审计（观测台）
}

// CallAudit 一次 LLM 调用的审计记录。
type CallAudit struct {
	Task       string `json:"task"`     // analysis / background / …
	Model      string `json:"model"`    // 实际命中的模型名
	Fallback   bool   `json:"fallback"` // 是否走了回退
	DurationMs int64  `json:"duration_ms"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
	At         string `json:"at"`
}

// NewManager 创建模型管理器。
func NewManager(client *llm.Client, registry *Registry, router *RouterConfig) *Manager {
	if router == nil {
		router = defaultRouter()
	}
	if router.Fallback.MaxRetries <= 0 {
		router.Fallback.MaxRetries = 2
	}
	if router.Fallback.BackoffMs <= 0 {
		router.Fallback.BackoffMs = 200
	}
	return &Manager{client: client, registry: registry, router: router}
}

// Chat 按任务类型路由 + 回退链重试，调用 LLM。
func (m *Manager) Chat(ctx context.Context, task TaskType, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	// 解析模型链：primary + 备选
	primary := m.router.Route(task)
	chain := m.router.Fallback.Chain
	models := uniqueModels(append([]string{primary}, chain...))
	if len(models) == 0 {
		return nil, errors.New("没有可用的模型（既无 default 也无 fallback 链）")
	}

	var lastErr error
	for i, name := range models {
		cfg, err := m.registry.Get(name)
		if err != nil {
			lastErr = fmt.Errorf("模型 %s: %w", name, err)
			continue
		}
		start := time.Now()
		resp, err := m.callWithRetry(ctx, cfg, req)
		m.recordAudit(CallAudit{
			Task: string(task), Model: name, Fallback: i > 0,
			DurationMs: time.Since(start).Milliseconds(), OK: err == nil,
			Error: errStr(err), At: start.Format(time.RFC3339),
		})
		if err == nil {
			return resp, nil
		}
		lastErr = err
		// 不可重试的错误直接换下一个模型
		if !llm.IsRetryable(err) && i < len(models)-1 {
			continue
		}
	}
	return nil, fmt.Errorf("所有模型均失败，最后错误: %w", lastErr)
}

// recordAudit 记录一次调用（环形，保 100 条）。
func (m *Manager) recordAudit(a CallAudit) {
	m.audit = append(m.audit, a)
	if len(m.audit) > 100 {
		m.audit = m.audit[len(m.audit)-100:]
	}
}

// AuditRecent 返回最近审计记录（倒序）。
func (m *Manager) AuditRecent(limit int) []CallAudit {
	out := make([]CallAudit, 0, limit)
	for i := len(m.audit) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, m.audit[i])
	}
	return out
}

func errStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// callWithRetry 对单个模型做指数退避重试。
func (m *Manager) callWithRetry(ctx context.Context, cfg llm.Config, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= m.router.Fallback.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := m.backoff(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
		cctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
		resp, err := m.client.Chat(cctx, cfg, req)
		cancel()
		if err == nil {
			return resp, nil
		}
		lastErr = err
		// 不可重试（4xx/鉴权/内容违规）立即返回
		if !llm.IsRetryable(err) {
			return nil, err
		}
	}
	return nil, lastErr
}

// backoff 计算指数退避 + 全抖动。
func (m *Manager) backoff(attempt int) time.Duration {
	ms := m.router.Fallback.BackoffMs
	if ms <= 0 {
		ms = 200
	}
	base := time.Duration(ms) * time.Millisecond
	// base * 2^(attempt-1)，封顶 30s
	d := base * time.Duration(1<<(attempt-1))
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	// 全抖动：[0, d)
	jitter := time.Duration(rand.Int63n(int64(d)))
	return jitter
}

// ToolsForExport 返回工具定义（透传，便于 agent 注入）。
func (m *Manager) ToolsForExport(tools []domain.Tool) []domain.Tool { return tools }

// ChatStream 流式调用：按任务路由，失败时仅在「尚未推送任何 byte」时才换模型重试
// （一旦开始推送内容就无法撤销，不能回退）。
func (m *Manager) ChatStream(ctx context.Context, task TaskType, req *llm.ChatRequest, cb llm.DeltaCallbacks) (resp *llm.ChatResponse, err error) {
	start := time.Now()
	modelName := m.router.Route(task)
	defer func() {
		m.recordAudit(CallAudit{Task: string(task), Model: modelName, DurationMs: time.Since(start).Milliseconds(), OK: err == nil, Error: errStr(err), At: start.Format(time.RFC3339)})
	}()
	_ = modelName
	primary := m.router.Route(task)
	chain := m.router.Fallback.Chain
	models := uniqueModels(append([]string{primary}, chain...))
	if len(models) == 0 {
		return nil, errors.New("没有可用的模型")
	}
	var lastErr error
	for i, name := range models {
		cfg, err := m.registry.Get(name)
		if err != nil {
			lastErr = fmt.Errorf("模型 %s: %w", name, err)
			continue
		}
		emitted := false
		wrapped := llm.DeltaCallbacks{
			OnReasoning: func(s string) {
				emitted = true
				if cb.OnReasoning != nil {
					cb.OnReasoning(s)
				}
			},
			OnContent: func(s string) {
				emitted = true
				if cb.OnContent != nil {
					cb.OnContent(s)
				}
			},
		}
		cctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
		resp, err := m.client.ChatStream(cctx, cfg, req, wrapped)
		cancel()
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if emitted {
			return nil, fmt.Errorf("流式中断（已推送内容，无法回退）: %w", err)
		}
		if !llm.IsRetryable(err) && i < len(models)-1 {
			continue
		}
	}
	return nil, fmt.Errorf("所有模型均失败，最后错误: %w", lastErr)
}

func uniqueModels(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func defaultRouter() *RouterConfig {
	return &RouterConfig{
		Default:  "",
		Routes:   map[TaskType]string{},
		Fallback: FallbackPolicy{MaxRetries: 2, BackoffMs: 200},
	}
}

// SetRouter 更新路由 + 回退配置（前端 config API 调用）。
func (m *Manager) SetRouter(rc *RouterConfig) {
	if rc == nil {
		return
	}
	if rc.Fallback.MaxRetries <= 0 {
		rc.Fallback.MaxRetries = 2
	}
	if rc.Fallback.BackoffMs <= 0 {
		rc.Fallback.BackoffMs = 200
	}
	m.router = rc
}

// Router 返回当前路由配置（只读副本）。
func (m *Manager) Router() RouterConfig {
	rc := *m.router
	return rc
}

// NewManagerFromConfig 便捷构造：用单个默认配置注册一个模型并设为 default，适合一期。
