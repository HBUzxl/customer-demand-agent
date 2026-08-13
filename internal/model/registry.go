package model

import (
	"fmt"
	"sort"
	"sync"

	"customer-demand-agent/internal/llm"
)

// Registry 是模型配置的注册表（内存，可被前端 config API 增删改）。
type Registry struct {
	mu     sync.RWMutex
	models map[string]ModelConfig
}

// NewRegistry 创建空注册表。
func NewRegistry() *Registry {
	return &Registry{models: make(map[string]ModelConfig)}
}

// Register 注册或更新一个模型配置。
func (r *Registry) Register(m ModelConfig) error {
	if m.Name == "" {
		return fmt.Errorf("模型 name 不能为空")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[m.Name] = m
	return nil
}

// Get 获取一个模型配置。
func (r *Registry) Get(name string) (llm.Config, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[name]
	if !ok {
		return llm.Config{}, fmt.Errorf("模型 %q 未注册", name)
	}
	return m.ToLLMConfig(), nil
}

// All 返回全部模型配置（按名排序，密钥脱敏）。
func (r *Registry) All() []ModelConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ModelConfig, 0, len(r.models))
	for _, m := range r.models {
		if m.APIKey != "" {
			m.APIKey = maskKey(m.APIKey)
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Delete 删除一个模型。
func (r *Registry) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.models[name]; !ok {
		return fmt.Errorf("模型 %q 不存在", name)
	}
	delete(r.models, name)
	return nil
}

// Names 返回全部模型名。
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.models))
	for n := range r.models {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Route 按任务类型解析模型名：优先 routes 表，否则 default。
func (rc *RouterConfig) Route(task TaskType) string {
	if name, ok := rc.Routes[task]; ok && name != "" {
		return name
	}
	return rc.Default
}

// SetRouter 更新路由配置（可被前端 config API 调用）。
type ManagerSetter interface {
	SetRouter(rc *RouterConfig)
}

func maskKey(k string) string {
	if len(k) <= 8 {
		return "****"
	}
	return k[:4] + "****" + k[len(k)-4:]
}
