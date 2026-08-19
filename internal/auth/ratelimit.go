package auth

import (
	"sync"
	"time"
)

// RateLimiter 固定窗口限流器：按 Key（IP+账号组合）限制频率。
// 内存有界：窗口过期条目惰性清理，超阈值时全量修剪。
type RateLimiter struct {
	mu     sync.Mutex
	window time.Duration
	max    int
	keys   map[string]*rlBucket
}

type rlBucket struct {
	count       int
	windowStart time.Time
}

// NewRateLimiter 创建限流器（window 内最多 max 次）。
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	if max <= 0 {
		max = 1
	}
	return &RateLimiter{window: window, max: max, keys: make(map[string]*rlBucket)}
}

// Allow 尝试放行；返回 false 表示已超限。
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	if len(rl.keys) > 10000 {
		// 修剪过期桶，防止攻击者用海量随机 Key 撑爆内存。
		for k, b := range rl.keys {
			if now.Sub(b.windowStart) > rl.window {
				delete(rl.keys, k)
			}
		}
	}
	b, ok := rl.keys[key]
	if !ok || now.Sub(b.windowStart) > rl.window {
		rl.keys[key] = &rlBucket{count: 1, windowStart: now}
		return true
	}
	if b.count >= rl.max {
		return false
	}
	b.count++
	return true
}
