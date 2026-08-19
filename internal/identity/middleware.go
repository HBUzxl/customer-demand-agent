package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"customer-demand-agent/internal/auth"
	"customer-demand-agent/internal/domain"
)

// ctxKey 用于在请求 Context 中传递租户作用域。
type ctxKey int

const scopeKey ctxKey = iota

// ContextWithScope 把服务端校验过的 TenantScope 注入 Context。
func ContextWithScope(ctx context.Context, sc domain.TenantScope) context.Context {
	return context.WithValue(ctx, scopeKey, sc)
}

// ScopeFrom 从请求 Context 读取 TenantScope（未登录路径不得调用）。
func ScopeFrom(ctx context.Context) (domain.TenantScope, bool) {
	sc, ok := ctx.Value(scopeKey).(domain.TenantScope)
	return sc, ok
}

// Middleware 认证与安全中间件：登录态校验、CSRF、安全响应头。
// 位于 identity 包内以打破 auth→identity 依赖（identity 使用 auth 原语）。
type Middleware struct {
	svc     *Service
	cookies *auth.CookieManager
}

// NewMiddleware 创建认证中间件。
func NewMiddleware(svc *Service, cookies *auth.CookieManager) *Middleware {
	return &Middleware{svc: svc, cookies: cookies}
}

// RequireAuth 校验登录态并把 TenantScope 注入 Context；未登录/无效统一 401。
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := m.cookies.Read(r)
		sc, err := m.svc.CurrentTenantScope(raw)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "未登录或登录已过期")
			return
		}
		next.ServeHTTP(w, r.WithContext(ContextWithScope(r.Context(), *sc)))
	})
}

// RequireCSRF 校验修改类请求的 CSRF（X-CSRF-Token + Origin/Referer 同源，§7.3）。
// GET/HEAD/OPTIONS 不校验（幂等请求不携带状态变更）。
func (m *Middleware) RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if !sameOrigin(r) {
			writeErr(w, http.StatusForbidden, "跨站请求被拒绝")
			return
		}
		raw := m.cookies.Read(r)
		csrfRaw := r.Header.Get("X-CSRF-Token")
		if !m.svc.VerifyCSRF(raw, csrfRaw) {
			writeErr(w, http.StatusForbidden, "CSRF 校验失败")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole 限制访问者必须持有指定角色之一（403）。
func (m *Middleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sc, ok := ScopeFrom(r.Context())
			if !ok {
				writeErr(w, http.StatusUnauthorized, "未登录")
				return
			}
			for _, role := range roles {
				if sc.HasRole(role) {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeErr(w, http.StatusForbidden, "无权限")
		})
	}
}

// SecurityHeaders 设置安全响应头（点击劫持/XSS 纵深防护）。
func (m *Middleware) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// sameOrigin 校验 Origin/Referer 与请求目标同源（CSRF 纵深防护）。
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	if origin == "" {
		// 无 Origin/Referer 的请求（如 curl 测试）——配合 X-CSRF-Token 仍视为有效。
		return true
	}
	// 去掉 scheme 前缀，比较 host 部分。
	origin = strings.TrimPrefix(origin, "https://")
	origin = strings.TrimPrefix(origin, "http://")
	origin = strings.TrimSuffix(origin, "/")
	if i := strings.Index(origin, "/"); i >= 0 {
		origin = origin[:i]
	}
	return origin == r.Host
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ErrUnauthorized 供 handler 识别登录态失效（统一 401 处理）。
var ErrUnauthorized = errors.New("未登录或登录已过期")
