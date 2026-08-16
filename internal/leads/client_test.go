package leads

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// mockPlatform 起一个 mock 商机平台，按 script 逐次响应（记录收到的请求数）。
type mockPlatform struct {
	srv    *httptest.Server
	hits   atomic.Int32
	script []func(w http.ResponseWriter, r *http.Request) // 每次请求弹出一步（耗尽重复最后一步）
}

func newMockPlatform(t *testing.T, script ...func(w http.ResponseWriter, r *http.Request)) *mockPlatform {
	t.Helper()
	mp := &mockPlatform{script: script}
	mp.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(mp.hits.Add(1))
		step := script[len(script)-1]
		if n <= len(script) {
			step = script[n-1]
		}
		step(w, r)
	}))
	t.Cleanup(mp.srv.Close)
	return mp
}

func okJSON(body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}
}

func status(code int) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }
}

func statusWith429(retryAfter string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":"rate limited"}`)
	}
}

// fastLimiter 返回一个不实际限流的 Options 修正（burst/rate 极大）。
func fastOpts(base string) Options {
	return Options{BaseURL: base, APIKey: "lm_pat_test", RatePerMin: 1_000_000, Burst: 1000, Timeout: 2 * time.Second}
}

// ── 白名单：非白名单路径本地拒绝，零网络请求 ──────────────────

func TestWhitelistLocalReject(t *testing.T) {
	mp := newMockPlatform(t, okJSON(`{}`))
	c := NewClient(fastOpts(mp.srv.URL))

	for _, p := range []string{
		"/api/admin/users",     // admin 禁区
		"/api/leads/1/export",  // 导出（未记载路径）
		"/api/leads/1/zzz",     // 任意子路径
		"/api/unknown",         // 未知路径
		"/api/leads/",          // 空 id
		"/api/stats/secret",    // stats 白名单外
		"/../etc/passwd",       // 穿越尝试
		"/api/leads?x=1#frag/", // fragment 不影响 path 判定，但 path=/api/leads 合法——单独验证非法 path
	} {
		// 注：/api/leads?x=1#frag 的 path 部分是 /api/leads（合法），排除出本组
		if p == "/api/leads?x=1#frag/" {
			continue
		}
		before := mp.hits.Load()
		_, err := c.Get(context.Background(), p)
		if err == nil {
			t.Errorf("%s 应被白名单拒绝", p)
			continue
		}
		if !errors.Is(err, ErrNotWhitelisted) {
			t.Errorf("%s 拒绝原因应为 ErrNotWhitelisted，got %v", p, err)
		}
		if got := mp.hits.Load(); got != before {
			t.Errorf("%s 被拒绝后不应有网络请求（hits %d → %d）", p, before, got)
		}
	}

	// 白名单内路径放行（含 query 与 PathEscape 后的 id）
	for _, p := range []string{
		"/api/leads",
		"/api/leads?stage=mql&page=1&page_size=10",
		"/api/leads/abc-123",
		"/api/leads/abc-123/activities",
		"/api/leads/stats",
		"/api/stats/conversion",
		"/api/stats/conversion/trends",
		"/api/stats/ml-creators",
		"/api/stats/product-conversion",
		"/api/products",
		"/api/tags",
	} {
		if _, err := c.Get(context.Background(), p); err != nil {
			t.Errorf("%s 应放行，got %v", p, err)
		}
	}
	if n := mp.hits.Load(); n != 11 {
		t.Errorf("白名单路径应有 11 次请求，got %d", n)
	}
}

// Get 只发 GET：mock 收到的全部请求方法都应是 GET（客户端只有 Get 一个入口，
// 白名单+方法构造在请求层，无法通过 API 发出写请求——这里验证服务端视角）。
func TestGetOnlySendsGET(t *testing.T) {
	var methods []string
	mp := newMockPlatform(t, func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		fmt.Fprint(w, `{}`)
	})
	c := NewClient(fastOpts(mp.srv.URL))
	for range 3 {
		_, _ = c.Get(context.Background(), "/api/leads")
	}
	for _, m := range methods {
		if m != http.MethodGet {
			t.Errorf("客户端只应发 GET，got %s", m)
		}
	}
}

// ── 认证头 ──────────────────────────────────────────────────

func TestAuthorizationHeader(t *testing.T) {
	var gotAuth atomic.Value
	mp := newMockPlatform(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth.Store(r.Header.Get("Authorization"))
		fmt.Fprint(w, `{}`)
	})
	c := NewClient(fastOpts(mp.srv.URL))
	_, _ = c.Get(context.Background(), "/api/leads")
	if gotAuth.Load() != "Bearer lm_pat_test" {
		t.Errorf("应带 Bearer Token，got %v", gotAuth.Load())
	}
}

// ── 限流：桶容量放行 + 等待语义 ─────────────────────────────

func TestRateLimitBurstAndWait(t *testing.T) {
	mp := newMockPlatform(t, okJSON(`{}`))
	// burst=2，600/min = 10/s：第 3+ 个请求需等待补充
	c := NewClient(Options{BaseURL: mp.srv.URL, APIKey: "k", RatePerMin: 600, Burst: 2, Timeout: 2 * time.Second})

	start := time.Now()
	for i := 0; i < 4; i++ {
		if _, err := c.Get(context.Background(), "/api/leads"); err != nil {
			t.Fatalf("第 %d 个请求失败: %v", i+1, err)
		}
	}
	elapsed := time.Since(start)
	// 前 2 个立即放行，第 3、4 个各等 ~100ms → 总耗时 ≥ 150ms（留容差）
	if elapsed < 150*time.Millisecond {
		t.Errorf("4 个请求（burst=2, 10/s）应至少等待 ~200ms，实际 %v", elapsed)
	}
	if n := mp.hits.Load(); n != 4 {
		t.Errorf("应有 4 次请求（等待而非丢弃），got %d", n)
	}
}

func TestRateLimitCtxCancel(t *testing.T) {
	mp := newMockPlatform(t, okJSON(`{}`))
	// burst=1，60/min = 1/s：第 2 个请求需等 ~1s——50ms 超时的 ctx 应先取消
	c := NewClient(Options{BaseURL: mp.srv.URL, APIKey: "k", RatePerMin: 60, Burst: 1, Timeout: 2 * time.Second})
	if _, err := c.Get(context.Background(), "/api/leads"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Get(ctx, "/api/leads")
	if err == nil {
		t.Fatal("限流等待应被 ctx 取消")
	}
	if n := mp.hits.Load(); n != 1 {
		t.Errorf("取消后不应有第 2 次请求，got %d", n)
	}
}

// ── 错误分类 ────────────────────────────────────────────────

func TestErrorTypes(t *testing.T) {
	// 401 → APIError 不可重试
	mp := newMockPlatform(t, status(401))
	c := NewClient(fastOpts(mp.srv.URL))
	_, err := c.Get(context.Background(), "/api/leads")
	var ae *APIError
	if !errors.As(err, &ae) || ae.StatusCode != 401 {
		t.Fatalf("应为 APIError 401，got %v", err)
	}
	if retryable(err) {
		t.Error("401 不应可重试")
	}
	if n := mp.hits.Load(); n != 1 {
		t.Errorf("401 不应重试（1 次请求），got %d", n)
	}

	// 403/404 同样不重试
	for _, code := range []int{403, 404} {
		mp := newMockPlatform(t, status(code))
		c := NewClient(fastOpts(mp.srv.URL))
		_, err := c.Get(context.Background(), "/api/leads")
		var ae *APIError
		if !errors.As(err, &ae) || ae.StatusCode != code {
			t.Fatalf("code %d: 应为 APIError，got %v", code, err)
		}
		if retryable(err) {
			t.Errorf("code %d 不应可重试", code)
		}
		if n := mp.hits.Load(); n != 1 {
			t.Errorf("code %d 不应重试，got %d hits", code, n)
		}
	}

	// 429/500 标记可重试
	for _, code := range []int{429, 500, 503} {
		mp := newMockPlatform(t, status(code))
		c := NewClient(fastOpts(mp.srv.URL))
		_, err := c.Get(context.Background(), "/api/leads")
		var ae *APIError
		if !errors.As(err, &ae) || ae.StatusCode != code {
			t.Fatalf("code %d: got %v", code, err)
		}
		if !retryable(err) {
			t.Errorf("code %d 应可重试", code)
		}
	}
}

// ── 重试：429/500 至多重试 1 次后成功 ────────────────────────

func TestRetryOn429ThenSuccess(t *testing.T) {
	mp := newMockPlatform(t,
		statusWith429("0"), // 第 1 次 429（Retry-After: 0）
		okJSON(`{"ok":1}`), // 重试后成功
	)
	c := NewClient(fastOpts(mp.srv.URL))
	body, err := c.Get(context.Background(), "/api/leads")
	if err != nil {
		t.Fatalf("429 重试一次后应成功: %v", err)
	}
	if string(body) != `{"ok":1}` {
		t.Errorf("重试后应拿到第 2 次响应，got %s", body)
	}
	if n := mp.hits.Load(); n != 2 {
		t.Errorf("应有 2 次请求（1 次 429 + 1 次成功），got %d", n)
	}
}

func TestRetryOn500ThenSuccess(t *testing.T) {
	mp := newMockPlatform(t,
		status(500), // Redis 抖动
		okJSON(`{"ok":1}`),
	)
	c := NewClient(fastOpts(mp.srv.URL))
	if _, err := c.Get(context.Background(), "/api/leads"); err != nil {
		t.Fatalf("500 重试一次后应成功: %v", err)
	}
	if n := mp.hits.Load(); n != 2 {
		t.Errorf("应有 2 次请求，got %d", n)
	}
}

func TestRetryOnlyOnce(t *testing.T) {
	// 持续 429：重试 1 次后放弃（不轰炸）
	mp := newMockPlatform(t, statusWith429("0"))
	c := NewClient(fastOpts(mp.srv.URL))
	_, err := c.Get(context.Background(), "/api/leads")
	var ae *APIError
	if !errors.As(err, &ae) || ae.StatusCode != 429 {
		t.Fatalf("应透传 429，got %v", err)
	}
	if n := mp.hits.Load(); n != 2 {
		t.Errorf("至多 1 次重试（共 2 次请求），got %d", n)
	}
}

// ── 超时 → TransportError ───────────────────────────────────

func TestTimeoutTransportError(t *testing.T) {
	mp := newMockPlatform(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		fmt.Fprint(w, `{}`)
	})
	c := NewClient(Options{BaseURL: mp.srv.URL, APIKey: "k", Timeout: 50 * time.Millisecond, RatePerMin: 1_000_000, Burst: 100})
	_, err := c.Get(context.Background(), "/api/leads")
	var te *TransportError
	if !errors.As(err, &te) {
		t.Fatalf("超时应为 TransportError，got %v (%T)", err, err)
	}
	if !retryable(err) {
		t.Error("网络错误应标记可重试")
	}
	// 重试也超时（mock 恒慢）→ 最终仍 TransportError
	_, err = c.Get(context.Background(), "/api/leads")
	if !errors.As(err, &te) {
		t.Fatalf("重试后仍超时应为 TransportError，got %v", err)
	}
}

// ── 重定向防护：跨 host 拒绝 ─────────────────────────────────

func TestCrossHostRedirectRejected(t *testing.T) {
	// 平台重定向到 evil.example（不同 host）→ 应被拒（TransportError）
	mp := newMockPlatform(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "http://evil.example/steal")
		w.WriteHeader(302)
	})
	c := NewClient(fastOpts(mp.srv.URL))
	_, err := c.Get(context.Background(), "/api/leads")
	if err == nil {
		t.Fatal("跨 host 重定向应被拒")
	}
	var te *TransportError
	if !errors.As(err, &te) {
		t.Errorf("应为 TransportError，got %v", err)
	}
}
