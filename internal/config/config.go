// Package config loads runtime configuration from a JSON config file — NOT from
// environment variables. config.json is the single source of truth for server
// settings, model registry, router, and (crucially) API keys.
//
// 启动时加载（不存在则写一份模板）；/api/config 的改动会回写文件。这样 API key
// 与模型配置集中在一个文件里管理，不依赖环境变量。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"customer-demand-agent/internal/model"
)

// DefaultPath 是默认配置文件路径。
const DefaultPath = "./config.json"

// Config 是整个服务的运行时配置（config.json 的结构）。
type Config struct {
	Server        ServerCfg           `json:"server"`
	WikiDir       string              `json:"wiki_dir"`
	HistoryDB     string              `json:"history_db"`
	DefaultUser   string              `json:"default_user"`    // 当前使用者（销售），影响分级输出
	LLMTimeoutSec int                 `json:"llm_timeout_sec"` // 全局 LLM 调用超时（非模型属性）
	Models        []model.ModelConfig `json:"models"`
	Router        model.RouterConfig  `json:"router"`
}

// ServerCfg 是服务监听与前端托管配置。
type ServerCfg struct {
	Addr         string `json:"addr"`
	FrontendDist string `json:"frontend_dist"`
}

// Store 在 Config 之上加了文件路径与写锁，支持回写持久化。
type Store struct {
	mu   sync.Mutex
	path string
	cfg  *Config
}

// template 是 config.json 不存在时写入的模板（API key 留空，待用户填写）。
func template() *Config {
	return &Config{
		Server: ServerCfg{
			Addr:         ":8080",
			FrontendDist: "",
		},
		WikiDir:       "./wiki",
		HistoryDB:     "./data/history.db",
		DefaultUser:   "张三",
		LLMTimeoutSec: 120,
		Models: []model.ModelConfig{
			{
				Name:        "default",
				Endpoint:    "https://api.deepseek.com/v1",
				APIKey:      "", // ← 在这里填 API key
				Protocol:    "openai-chat",
				Model:       "deepseek-chat",
				Temperature: 0.3,
				MaxTokens:   4096,
			},
		},
		Router: model.RouterConfig{
			Default: "default",
			Routes:  map[model.TaskType]string{},
			Fallback: model.FallbackPolicy{
				MaxRetries: 2,
				BackoffMs:  200,
				Chain:      nil,
			},
		},
	}
}

// Load 从 path 读取配置。文件不存在时写入模板并返回它（便于用户照着改）。
func Load(path string) (*Store, error) {
	if path == "" {
		path = DefaultPath
	}
	cfg, err := read(path)
	if err != nil {
		if os.IsNotExist(err) {
			t := template()
			if werr := write(path, t); werr != nil {
				return nil, fmt.Errorf("写入配置模板 %s: %w", path, werr)
			}
			fmt.Fprintf(os.Stderr, "[config] 未找到 %s，已写入模板，请编辑后重启（填入 api_key）\n", path)
			cfg = t
		} else {
			return nil, fmt.Errorf("读取配置 %s: %w", path, err)
		}
	}
	return &Store{path: path, cfg: cfg}, nil
}

// Get 返回当前配置（只读视图，调用方不应修改）。
func (s *Store) Get() *Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

// Update 在锁内修改配置并回写文件。mutate 操作 cfg 后返回写盘错误。
func (s *Store) Update(mutate func(cfg *Config)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	mutate(s.cfg)
	return write(s.path, s.cfg)
}

// Save 将给定配置整体替换并写回文件。
func (s *Store) Save(cfg *Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	return write(s.path, cfg)
}

func read(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 JSON: %w", err)
	}
	applyDefaults(&cfg)
	return &cfg, nil
}

// write 原子写盘（临时文件 + rename，避免写一半）。
func write(path string, cfg *Config) error {
	applyDefaults(cfg)
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil { // 0600：含 api_key，限权限
		return err
	}
	return os.Rename(tmp, path)
}

// applyDefaults 填充缺失的默认值。
func applyDefaults(cfg *Config) {
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.WikiDir == "" {
		cfg.WikiDir = "./wiki"
	}
	if cfg.HistoryDB == "" {
		cfg.HistoryDB = "./data/history.db"
	}
	if cfg.DefaultUser == "" {
		cfg.DefaultUser = "张三"
	}
	if cfg.LLMTimeoutSec <= 0 {
		cfg.LLMTimeoutSec = 120
	}
	if cfg.Router.Fallback.MaxRetries <= 0 {
		cfg.Router.Fallback.MaxRetries = 2
	}
	if cfg.Router.Fallback.BackoffMs <= 0 {
		cfg.Router.Fallback.BackoffMs = 200
	}
}
