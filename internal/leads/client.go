// Package leads 实现商机平台（Lead Manager / MQL）只读查询客户端与
// Agent 工具（leads_search / leads_get / leads_stats）。
//
// 平台契约（docs/lead-managers.md）：需公司内网或 VPN；Authorization:
// Bearer lm_pat_...；仅 GET + 白名单路径；限流 10 次/秒、生产 60 次/分钟；
// 统计接口按 Token 创建人角色判权（403）；401=Token 失效 / 429=频率 /
// 500=服务端（Redis 抖动）/ 超时=未连 VPN。
//
// 客户端硬化三原则：
//  1. GET-only + 路径白名单在请求构造时本地校验——白名单外直接报错，零网络请求；
//  2. 进程内令牌桶限流（跨会话全局生效），等待语义而非快速失败；
//  3. 重试只在幂等 GET 的可重试错误（429/5xx/网络错误）上做，至多 1 次，
//     全抖动退避（沿用 llm 层纪律，但错误类型在包内自持——不 import llm）。
package leads

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Options 是客户端构造参数（由 config.LeadManagerCfg 翻译而来，包内不依赖 config）。
type Options struct {
	BaseURL    string        // 平台地址（如 http://api.in.chaitin.net/mql）
	APIKey     string        // 系统级 Token（lm_pat_...）
	Timeout    time.Duration // 单请求超时（默认 10s）
	RatePerMin int           // 令牌桶速率：次/分钟（默认 60）
	Burst      int           // 令牌桶容量：瞬时并发（默认 3；≤ 平台 10 次/秒上限）
}

// Client 是商机平台只读 API 客户端。进程级共享（跨会话全局限流）。
type Client struct {
	base    string // 无尾斜杠
	apiKey  string
	hc      *http.Client
	limiter *tokenBucket
}

// NewClient 创建客户端（GET-only + 白名单 + 限流 + 重试）。
func NewClient(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}
	return &Client{
		base:   strings.TrimRight(opts.BaseURL, "/"),
		apiKey: opts.APIKey,
		hc: &http.Client{
			Timeout: opts.Timeout,
			// 重定向防护：仅允许同 host（防 Authorization 头被引到别处）。
			CheckRedirect: sameHostRedirect,
		},
		limiter: newTokenBucket(opts.RatePerMin, opts.Burst),
	}
}

// ── 错误类型（参照 llm.TransportError/APIError 模式，包内自持） ──

// TransportError 是网络层错误（超时/连接失败/被拒的重定向，可重试）。
type TransportError struct{ Cause error }

func (e *TransportError) Error() string { return "网络错误: " + e.Cause.Error() }

func (e *TransportError) Unwrap() error { return e.Cause }

// APIError 是平台返回的非 2xx。
type APIError struct {
	StatusCode int
	Body       string
	Retryable  bool
	RetryAfter time.Duration // 429 时来自 Retry-After 头（可能为 0）
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API 错误 %d: %.200s", e.StatusCode, e.Body)
}

// retryable 判断错误是否值得重试（429 / 5xx / 网络错误；401/403/404 不重试）。
func retryable(err error) bool {
	var te *TransportError
	if errors.As(err, &te) {
		return true
	}
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.Retryable
	}
	return false
}

// ── 路径白名单（只读硬边界：schema 即边界，绝不实现写请求） ──

// readOnlyPaths 是无参数的固定白名单路径（lead-managers.md 只读接口清单）。
var readOnlyPaths = map[string]bool{
	"/api/leads":                    true,
	"/api/leads/stats":              true,
	"/api/stats/conversion":         true,
	"/api/stats/conversion/trends":  true,
	"/api/stats/ml-creators":        true,
	"/api/stats/product-conversion": true,
	"/api/products":                 true, // 验证期探活用
	"/api/tags":                     true,
}

// allowedPath 校验路径（不含 query）是否落在只读白名单内。
// 参数化路径仅限 /api/leads/{id}、/api/leads/{id}/activities、/api/products/{id}。
// 空段（//、尾斜杠）直接拒绝，保持白名单形状严格。
func allowedPath(p string) bool {
	if p == "" {
		return false
	}
	if strings.Contains(p, "//") || strings.HasSuffix(p, "/") {
		return false
	}
	p = path.Clean("/" + p)
	if readOnlyPaths[p] {
		return true
	}
	segs := strings.Split(strings.Trim(p, "/"), "/")
	switch {
	case len(segs) == 3 && segs[0] == "api" && segs[1] == "leads" && segs[2] != "":
		return true // /api/leads/{id}
	case len(segs) == 4 && segs[0] == "api" && segs[1] == "leads" && segs[2] != "" && segs[3] == "activities":
		return true // /api/leads/{id}/activities
	case len(segs) == 3 && segs[0] == "api" && segs[1] == "products" && segs[2] != "":
		return true // /api/products/{id}
	}
	return false
}

// ErrNotWhitelisted 表示路径不在只读白名单内（本地拒绝，零网络请求）。
var ErrNotWhitelisted = errors.New("路径不在商机平台只读白名单内")

// Get 发起一次白名单 GET（含限流与至多 1 次的重试）。
// pathWithQuery 形如 "/api/leads?stage=mql&page=1"。
func (c *Client) Get(ctx context.Context, pathWithQuery string) ([]byte, error) {
	// 白名单校验：本地拒绝，零网络请求（只读硬边界的第一道闸）。
	p := pathWithQuery
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if !allowedPath(p) {
		return nil, fmt.Errorf("%w: %s", ErrNotWhitelisted, p)
	}
	full := c.base + pathWithQuery

	for attempt := 0; ; attempt++ {
		if err := c.limiter.wait(ctx); err != nil {
			return nil, fmt.Errorf("限流等待: %w", err)
		}
		body, err := c.doOnce(ctx, full)
		if err == nil {
			return body, nil
		}
		if attempt >= 1 || !retryable(err) {
			return nil, err
		}
		if err := sleepBackoff(ctx, err, attempt); err != nil {
			return nil, err
		}
	}
}

// doOnce 执行单次 HTTP GET（不重试）。
func (c *Client) doOnce(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, &TransportError{Cause: err}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 上限 10MB
	if err != nil {
		return nil, &TransportError{Cause: err}
	}
	if resp.StatusCode >= 400 {
		ae := &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(raw),
			Retryable:  resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500,
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			if secs, perr := strconv.Atoi(strings.TrimSpace(resp.Header.Get("Retry-After"))); perr == nil && secs >= 0 {
				ae.RetryAfter = time.Duration(secs) * time.Second
			}
		}
		return nil, ae
	}
	return raw, nil
}

// ── 重试退避（全抖动，沿用 llm 层纪律） ──

const (
	backoffBase   = 300 * time.Millisecond // 第 1 次重试的退避基数
	backoffCap    = 30 * time.Second
	retryAfterCap = 10 * time.Second // Retry-After 上限（防单次工具调用阻塞过久）
)

// sleepBackoff 按错误类型退避：429 尊重 Retry-After（封顶），其余全抖动
// base * 2^attempt；ctx 取消即中断。
func sleepBackoff(ctx context.Context, err error, attempt int) error {
	var sleep time.Duration
	var ae *APIError
	if errors.As(err, &ae) && ae.StatusCode == http.StatusTooManyRequests && ae.RetryAfter > 0 {
		sleep = min(ae.RetryAfter, retryAfterCap)
	} else {
		d := min(backoffBase*time.Duration(1<<attempt), backoffCap)
		sleep = time.Duration(rand.Int63n(int64(d) + 1)) // 全抖动 [0, d]
	}
	if sleep <= 0 {
		return nil
	}
	t := time.NewTimer(sleep)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// ── 重定向防护 ──

// sameHostRedirect 仅允许同 host 重定向（链上每一跳对照原始请求），
// 跨 host 视为错误——防 Authorization 头被引到别处。
func sameHostRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 3 {
		return fmt.Errorf("过多重定向")
	}
	if len(via) == 0 {
		return nil
	}
	if !strings.EqualFold(req.URL.Host, via[0].URL.Host) {
		return fmt.Errorf("拒绝跨 host 重定向（防 Authorization 外泄）: %s → %s", via[0].URL.Host, req.URL.Host)
	}
	return nil
}

// ── 令牌桶限流（跨会话全局；等待语义） ──

// tokenBucket 是进程内令牌桶：速率 ratePerMin/60 每秒补充、桶容量 burst。
// burst=3 + rate=60/min 同时满足平台两条限流（瞬时 ≤10/s、持续 ≤60/min）。
type tokenBucket struct {
	mu     sync.Mutex
	tokens float64
	max    float64
	perSec float64
	last   time.Time
}

func newTokenBucket(ratePerMin, burst int) *tokenBucket {
	if ratePerMin <= 0 {
		ratePerMin = 60
	}
	if burst <= 0 {
		burst = 3
	}
	return &tokenBucket{
		tokens: float64(burst),
		max:    float64(burst),
		perSec: float64(ratePerMin) / 60.0,
		last:   time.Now(),
	}
}

// wait 阻塞直到取到一个令牌或 ctx 结束。
func (b *tokenBucket) wait(ctx context.Context) error {
	for {
		b.mu.Lock()
		now := time.Now()
		b.tokens = min(b.tokens+now.Sub(b.last).Seconds()*b.perSec, b.max)
		b.last = now
		if b.tokens >= 1 {
			b.tokens--
			b.mu.Unlock()
			return nil
		}
		need := (1 - b.tokens) / b.perSec // 距下一个令牌的秒数
		b.mu.Unlock()

		t := time.NewTimer(time.Duration(need * float64(time.Second)))
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
}
