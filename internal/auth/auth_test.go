package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCSRFGenerateVerify raw/hash 往返 + 常量时间校验。
func TestCSRFGenerateVerify(t *testing.T) {
	c := NewCSRF()
	raw, hash := c.Generate()
	if raw == "" || hash == "" {
		t.Fatal("raw/hash 不应为空")
	}
	if !c.Verify(hash, raw) {
		t.Fatal("正确 raw 应通过")
	}
	if c.Verify(hash, "wrong-token") {
		t.Fatal("错误 raw 不应通过")
	}
	if c.Verify(hash, "") {
		t.Fatal("空 raw 不应通过")
	}
	if c.Verify("", raw) {
		t.Fatal("空 hash 不应通过")
	}
	// 两次 Generate 不应相同（每会话独立）。
	raw2, _ := c.Generate()
	if raw == raw2 {
		t.Fatal("CSRF 应每会话独立")
	}
}

// TestRateLimiterAllow 固定窗口：限内放行、超限拒绝、窗口重置后恢复。
func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(3, 50*time.Millisecond)
	key := "ip1|a@x.com"
	for i := 0; i < 3; i++ {
		if !rl.Allow(key) {
			t.Fatalf("第 %d 次应放行", i+1)
		}
	}
	if rl.Allow(key) {
		t.Fatal("超限应拒绝")
	}
	// 不同 Key 不受影响
	if !rl.Allow("ip2|b@x.com") {
		t.Fatal("不同 Key 应放行")
	}
	// 窗口过期后恢复
	time.Sleep(60 * time.Millisecond)
	if !rl.Allow(key) {
		t.Fatal("窗口过期后应恢复")
	}
}

// TestRateLimiterPrune 超量 Key 修剪：海量随机 Key 不撑爆内存。
func TestRateLimiterPrune(t *testing.T) {
	rl := NewRateLimiter(1, 5*time.Millisecond)
	// 制造大量过期桶触发修剪路径
	for i := 0; i < 12000; i++ {
		rl.Allow("rand-key-" + string(rune('a'+i%26)) + string(rune('0'+i%10)))
	}
	time.Sleep(10 * time.Millisecond)
	if len(rl.keys) > 10000 {
		t.Fatalf("修剪后仍超 10000 桶: %d", len(rl.keys))
	}
}

// TestCookieManager 读写清除 + Secure 标志 + 命名（__Host- 仅生产，dev 无前缀）。
func TestCookieManager(t *testing.T) {
	// secure=false：本地 http 联调 → 无前缀名 cda_session（__Host- 需 Secure）
	c := NewCookieManager(false)
	if c.Name() != DevSessionCookie {
		t.Fatalf("secure=false 应使用 dev 名 %s，got %s", DevSessionCookie, c.Name())
	}
	rec := httptest.NewRecorder()
	c.Set(rec, "rawtoken", time.Hour)
	res := rec.Result()
	var got *http.Cookie
	for _, ck := range res.Cookies() {
		if ck.Name == c.Name() {
			got = ck
		}
	}
	if got == nil {
		t.Fatal("应写入登录 Cookie")
	}
	if got.Name != "cda_session" {
		t.Fatalf("secure=false Cookie 名应为 cda_session（无 __Host- 前缀）: %s", got.Name)
	}
	if got.Value != "rawtoken" {
		t.Fatalf("Cookie 应存随机会话 Token: %q", got.Value)
	}
	if got.HttpOnly != true || got.SameSite != http.SameSiteLaxMode || got.Path != "/" {
		t.Fatalf("HttpOnly/SameSite=Lax/Path=/ 必须: %+v", got)
	}
	if got.Secure {
		t.Fatal("secure=false 时 Cookie 不应标 Secure")
	}

	// Read 回读
	req := httptest.NewRequest("GET", "http://x/", nil)
	req.AddCookie(got)
	if v := c.Read(req); v != "rawtoken" {
		t.Fatalf("Read 应返回 raw: %q", v)
	}
	// 无 Cookie
	if v := c.Read(httptest.NewRequest("GET", "http://x/", nil)); v != "" {
		t.Fatal("无 Cookie 应返回空")
	}
	// Clear 清 Cookie
	rec2 := httptest.NewRecorder()
	c.Clear(rec2)
	cleared := false
	for _, ck := range rec2.Result().Cookies() {
		if ck.Name == c.Name() && ck.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("Clear 应写 MaxAge<0 清除 Cookie")
	}
	// secure=true：生产必须 __Host- 前缀 + Secure
	c2 := NewCookieManager(true)
	if c2.Name() != SessionCookie {
		t.Fatalf("secure=true 应使用 __Host- 名 %s，got %s", SessionCookie, c2.Name())
	}
	rec3 := httptest.NewRecorder()
	c2.Set(rec3, "x", time.Hour)
	for _, ck := range rec3.Result().Cookies() {
		if ck.Name == c2.Name() && !ck.Secure {
			t.Fatal("secure=true 时 Cookie 必须标 Secure")
		}
		if ck.Name == c2.Name() && ck.Name != SessionCookie {
			t.Fatalf("secure=true 名应为 %s: %s", SessionCookie, ck.Name)
		}
	}
}
