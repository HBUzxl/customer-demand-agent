package auth

import (
	"net/http"
	"time"
)

// Cookie 命名（§7.3）：生产用 __Host- 前缀（强制 Secure + Path=/ + 无 Domain）；
// 本地 http 联调无法携带 Secure，__Host- 前缀会被浏览器拒存，故退化为无前缀名。
const (
	// SessionCookie 生产登录 Cookie 名。
	SessionCookie = "__Host-cda_session"
	// DevSessionCookie 本地 http 联调 Cookie 名（secure=false 时使用）。
	DevSessionCookie = "cda_session"
)

// CookieManager 管理登录 Cookie 的读写。
type CookieManager struct {
	secure bool
}

// NewCookieManager 创建 Cookie 管理器。secure=false 仅用于本地 http 测试/联调。
func NewCookieManager(secure bool) *CookieManager {
	return &CookieManager{secure: secure}
}

// Name 返回当前安全模式下的 Cookie 名（secure=true → __Host- 前缀）。
func (c *CookieManager) Name() string {
	if c.secure {
		return SessionCookie
	}
	return DevSessionCookie
}

// Set 写入登录 Cookie（HttpOnly + SameSite=Lax；Secure 与命名由构造决定）。
func (c *CookieManager) Set(w http.ResponseWriter, rawToken string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name(),
		Value:    rawToken,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Clear 清除登录 Cookie（退出登录用）。
func (c *CookieManager) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// Read 读取请求中的登录 Token。
func (c *CookieManager) Read(r *http.Request) string {
	cookie, err := r.Cookie(c.Name())
	if err != nil {
		return ""
	}
	return cookie.Value
}
