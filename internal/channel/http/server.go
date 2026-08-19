// Package http implements the HTTP channel: a REST API over the agent,
// memory, review, history, and config subsystems. Multi-tenant (设计文档 §6-§9):
// 认证控制面 + 租户独立数据面。所有业务 handler 经 s.rt(r) 解析当前登录
// 作用域（identity.ScopeFrom）的租户运行时，再访问其私有的 history/wiki/
// agent——同一 session_id/记忆标题在另一租户天然不存在（§9.3）。
package http

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"customer-demand-agent/internal/auth"
	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/identity"
	"customer-demand-agent/internal/leads"
	"customer-demand-agent/internal/model"
	"customer-demand-agent/internal/taskbg"
	"customer-demand-agent/internal/tenancy"
)

// AuthDeps 多租户身份控制面依赖（M1 起）。auth 为 nil 表示 legacy/测试最小装配
// （NewMinimal），此时业务路由不拦截、auth 端点返回 503——仅用于单元测试过渡。
type AuthDeps struct {
	Identity            *identity.Service
	Cookies             *auth.CookieManager
	Middleware          *identity.Middleware
	LoginLimiter        *auth.RateLimiter
	LoginIPLimiter      *auth.RateLimiter
	RegistrationLimiter *auth.RateLimiter
	RegistrationEnabled func() bool
}

// Server wires all subsystems behind an HTTP REST API.
// 不再持有全局 agent/review/wiki/history——每租户运行时由 runtimes 懒加载
// 解析（M2c）；平台级共享依赖（modelMgr/registry/tasks/logRing/leads）保留。
type Server struct {
	store     *config.Store
	runtimes  *tenancy.Registry // 租户运行时注册表（业务 handler 按作用域解析）
	modelMgr  *model.Manager
	registry  *model.Registry
	runs      *RunManager    // F0：服务端 Run 任务（执行与连接解耦，RunKey 租户隔离）
	tasks     *taskbg.Runner // P11：后台任务（观测台消费，按租户过滤）
	logRing   *LogRing       // C2：日志环形缓冲（观测台 tail）
	leads     *leads.Service // 商机面板（Dashboard）只读代理（nil=未接入；SetLeads 注入）
	auth      *AuthDeps      // M1：多租户身份控制面（nil=legacy/测试模式）
	profileMu sync.Mutex     // 使用者画像懒初始化/成员同步串行化，避免首次并发重复写文件
}

// Config 是 New 的装配参数（M2c 起 runtimes 取代进程级全局 agent/review/wiki/history）。
type Config struct {
	Store    *config.Store
	Runtimes *tenancy.Registry
	ModelMgr *model.Manager
	Registry *model.Registry
	Tasks    *taskbg.Runner
	LogRing  *LogRing
	Leads    *leads.Service
}

// New creates the HTTP server with all dependencies injected.
func New(cfg Config) *Server {
	return &Server{
		store: cfg.Store, runtimes: cfg.Runtimes,
		modelMgr: cfg.ModelMgr, registry: cfg.Registry,
		runs: NewRunManager(), tasks: cfg.Tasks, logRing: cfg.LogRing,
		leads: cfg.Leads,
	}
}

// NewMinimal 构造只装配后台任务/记忆/审核路由的最小服务（cmd 层端到端
// 测试用——需传入真实 runtimes 才能服务业务路由；仅 leads 路由可无 runtime）。
func NewMinimal(runtimes *tenancy.Registry, tasks *taskbg.Runner) *Server {
	return &Server{
		runtimes: runtimes,
		runs:     NewRunManager(), tasks: tasks,
	}
}

// SetAuth 注入多租户身份控制面依赖（M1：main 装配 identity/auth/audit 后调用）。
// auth 为 nil 时业务路由不拦截（legacy/测试过渡态），M2 起由 New 统一装配。
func (s *Server) SetAuth(d *AuthDeps) { s.auth = d }

// ErrUnauthenticated 请求无有效作用域（未登录）。业务路由在 requireAuth 后
// 不应出现；防御性兜底（防未来新增路由漏包）。
var ErrUnauthenticated = errors.New("未登录")

// sc 返回当前请求的服务端校验作用域（identity.Middleware 注入）。
// TenantID/UserID 只来自登录态，客户端无法伪造（设计文档 §9.1）。
func (s *Server) sc(r *http.Request) (domain.TenantScope, error) {
	sc, ok := identity.ScopeFrom(r.Context())
	if !ok || !sc.Valid() {
		return domain.TenantScope{}, ErrUnauthenticated
	}
	return sc, nil
}

// rt 解析当前请求所属租户的运行时（懒加载）。跨租户访问在这里闭合：
// sc.TenantID 决定了打开哪个租户的数据面，之后 session_id/记忆标题自然
// 属于该租户——A 用 B 的资源 id 在 B 的库中不存在 → 404。
func (s *Server) rt(r *http.Request) (*tenancy.Runtime, error) {
	sc, err := s.sc(r)
	if err != nil {
		return nil, err
	}
	if s.runtimes == nil {
		return nil, errors.New("租户运行时未配置")
	}
	rt, err := s.runtimes.ForTenant(r.Context(), sc.TenantID)
	if err != nil {
		return nil, err
	}
	// 每个平台成员都是一个使用者。首次访问租户数据面时按服务端登录身份懒建
	// 画像；失败只记日志，不让辅助画像阻断会话/知识读取主流程。
	s.profileMu.Lock()
	profileErr := s.ensureUserProfileLocked(rt, sc)
	s.profileMu.Unlock()
	if profileErr != nil {
		log.Printf("同步使用者画像 tenant=%s user=%s: %v", sc.TenantID, sc.UserID, profileErr)
	}
	return rt, nil
}

// authReady 身份控制面是否可服务。
func (s *Server) authReady() bool {
	return s.auth != nil && s.auth.Identity != nil && s.auth.Cookies != nil
}

// requireAuth 包裹需登录的路由（legacy 无身份控制面时不拦截）。
func (s *Server) requireAuth(h http.Handler) http.Handler {
	if s.auth == nil || s.auth.Middleware == nil {
		return h
	}
	return s.auth.Middleware.RequireAuth(h)
}

// requireCSRF 包裹修改类路由（legacy 无身份控制面时不校验）。
func (s *Server) requireCSRF(h http.Handler) http.Handler {
	if s.auth == nil || s.auth.Middleware == nil {
		return h
	}
	return s.auth.Middleware.RequireCSRF(h)
}

// protected 只读业务路由：登录即可（未登录 401）。
func (s *Server) protected(h http.HandlerFunc) http.Handler {
	return s.requireAuth(http.HandlerFunc(h))
}

// protectedCSRF 修改类业务路由：登录 + CSRF 校验（§7.3）。
func (s *Server) protectedCSRF(h http.HandlerFunc) http.Handler {
	return s.requireAuth(s.requireCSRF(http.HandlerFunc(h)))
}

// platformRole 平台管理员角色闸门：作用域须含 platform_admin 否则 403。
// 平台管理员只拥有平台级运维元数据权限，不是"超级租户"（§4.2/§4.3）。
func (s *Server) platformRole(h http.Handler) http.Handler {
	if s.auth == nil || s.auth.Middleware == nil {
		return h // legacy/测试装配无身份控制面时不拦截
	}
	return s.auth.Middleware.RequireRole(domain.RolePlatformAdmin)(h)
}

// platformAdmin 平台管理只读路由：登录 + platform_admin。
func (s *Server) platformAdmin(h http.HandlerFunc) http.Handler {
	return s.requireAuth(s.platformRole(http.HandlerFunc(h)))
}

// platformAdminCSRF 平台管理修改类路由：登录 + CSRF + platform_admin。
func (s *Server) platformAdminCSRF(h http.HandlerFunc) http.Handler {
	return s.requireAuth(s.requireCSRF(s.platformRole(http.HandlerFunc(h))))
}

// tenantRole 租户内角色闸门（P0-06）：作用域须含给定角色之一否则 403。
// 与 platformRole 同款 legacy 兼容模式：无身份控制面时不拦截（旧单租户测试）。
func (s *Server) tenantRole(roles ...string) func(http.Handler) http.Handler {
	if s.auth == nil || s.auth.Middleware == nil {
		return func(h http.Handler) http.Handler { return h }
	}
	return s.auth.Middleware.RequireRole(roles...)
}

// tenantManager 租户管理类修改路由：登录 + CSRF + owner/admin（记忆写/删除、任务）。
func (s *Server) tenantManager(h http.HandlerFunc) http.Handler {
	return s.requireAuth(s.requireCSRF(s.tenantRole(domain.RoleOwner, domain.RoleAdmin)(http.HandlerFunc(h))))
}

// tenantManagerRead 租户管理只读路由：登录 + owner/admin，不需要 CSRF。
func (s *Server) tenantManagerRead(h http.HandlerFunc) http.Handler {
	return s.requireAuth(s.tenantRole(domain.RoleOwner, domain.RoleAdmin)(http.HandlerFunc(h)))
}

// tenantReviewer 租户审核类修改路由：登录 + CSRF + owner/admin/reviewer（审核通过/驳回）。
func (s *Server) tenantReviewer(h http.HandlerFunc) http.Handler {
	return s.requireAuth(s.requireCSRF(s.tenantRole(domain.RoleOwner, domain.RoleAdmin, domain.RoleReviewer)(http.HandlerFunc(h))))
}

// Handler returns the configured ServeMux with all routes registered.
// 路由分组（设计文档 §7.5）：公开 / 需登录 / 平台管理（M4）。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// ── 公开路由（无需登录）──────────────────────────────────
	mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// ── 身份（需登录）────────────────────────────────────────
	mux.Handle("POST /api/auth/logout", s.protectedCSRF(s.handleLogout))
	mux.Handle("GET /api/auth/me", s.protected(s.handleMe))
	mux.Handle("PUT /api/auth/password", s.protectedCSRF(s.handlePassword))
	mux.Handle("POST /api/auth/resume", s.protectedCSRF(s.handleResume)) // P0-09：provision 失败后重试初始化
	mux.Handle("POST /api/auth/switch-tenant", s.protectedCSRF(s.handleSwitchTenant))

	// 租户资料/成员管理（owner/admin）。前端隐藏只是体验，后端路由和 Service 双闸门。
	mux.Handle("GET /api/tenant", s.tenantManagerRead(s.handleCurrentTenant))
	mux.Handle("PATCH /api/tenant", s.tenantManager(s.handleCurrentTenantUpdate))
	mux.Handle("GET /api/members", s.tenantManagerRead(s.handleMembersList))
	mux.Handle("POST /api/members", s.tenantManager(s.handleMembersAdd))
	mux.Handle("PATCH /api/members/{user_id}/role", s.tenantManager(s.handleMembersSetRole))
	mux.Handle("DELETE /api/members/{user_id}", s.tenantManager(s.handleMembersRemove))

	// ── 分析对话（统一入口 + deprecated shims，ADR-014；F0：202 + Run 任务）
	mux.Handle("POST /api/message", s.protectedCSRF(s.handleMessage))
	mux.Handle("POST /api/analyze", s.protectedCSRF(s.handleAnalyze))
	mux.Handle("POST /api/chat", s.protectedCSRF(s.handleChat))
	mux.Handle("GET /api/sessions/{id}/stream", s.protected(s.handleSessionStream))
	mux.Handle("GET /api/sessions/{id}/running", s.protected(s.handleSessionRunning))
	mux.Handle("POST /api/sessions/{id}/runs/{run_id}/cancel", s.protectedCSRF(s.handleRunCancel))
	mux.Handle("DELETE /api/sessions/{id}/messages/after", s.protectedCSRF(s.handleMessagesTruncate))

	// 记忆管理
	mux.Handle("GET /api/memory/search", s.protected(s.handleMemorySearch))
	mux.Handle("GET /api/memory/list", s.protected(s.handleMemoryList))
	mux.Handle("GET /api/memory/{type}/{title}", s.protected(s.handleMemoryGet))
	mux.Handle("GET /api/memory/{type}/{title}/history", s.protected(s.handleMemoryHistory))
	mux.Handle("POST /api/customers", s.protectedCSRF(s.handleCustomerCreate)) // 租户成员可登记客户，不可覆盖
	mux.Handle("POST /api/memory", s.protectedCSRF(s.handleMemoryUpsert))      // handler 按类型授权：本人画像或 owner/admin
	mux.Handle("DELETE /api/memory/{type}/{title}", s.tenantManager(s.handleMemoryDelete))

	// 审核
	mux.Handle("GET /api/review/pending", s.protected(s.handleReviewPending))
	mux.Handle("POST /api/review/{type}/{title}/approve", s.tenantReviewer(s.handleReviewApprove)) // P0-06：审核=reviewer/owner/admin
	mux.Handle("POST /api/review/{type}/{title}/reject", s.tenantReviewer(s.handleReviewReject))

	// 配置
	// 租户安全视图（model 别名 + 掩码 key，§10.1）；完整配置/探测/模型列表在 /api/platform/**。
	mux.Handle("GET /api/config", s.protected(s.handleConfigGet))

	// ── 平台管理（/api/platform/**：platform_admin 专用，§7.5）────────────
	// 全局模型配置、原始日志、平台 LLM 审计不再对所有租户开放——迁移到该分组。
	mux.Handle("PUT /api/platform/config", s.platformAdminCSRF(s.handleConfigPut))
	mux.Handle("POST /api/platform/config/test", s.platformAdminCSRF(s.handleConfigTest))
	mux.Handle("POST /api/platform/models", s.platformAdminCSRF(s.handleListModels))
	mux.Handle("GET /api/platform/config", s.platformAdmin(s.handleConsoleConfig))      // 含 data_dir/wiki_dir 的完整视图
	mux.Handle("GET /api/platform/logs", s.platformAdmin(s.handleConsoleLogs))          // 原始日志（SSE）
	mux.Handle("GET /api/platform/llm-audit", s.platformAdmin(s.handleConsoleLLMAudit)) // 平台 LLM 调用审计
	// 用户/租户管理（平台控制面，§4.2）
	mux.Handle("GET /api/platform/users", s.platformAdmin(s.handleAdminUsers))
	mux.Handle("POST /api/platform/users", s.platformAdminCSRF(s.handleAdminCreateUser))
	mux.Handle("PATCH /api/platform/users/{user_id}", s.platformAdminCSRF(s.handleAdminUpdateUser))
	mux.Handle("DELETE /api/platform/users/{user_id}", s.platformAdminCSRF(s.handleAdminDeleteUser))
	mux.Handle("GET /api/platform/tenants", s.platformAdmin(s.handleAdminTenants))
	mux.Handle("POST /api/platform/tenants", s.platformAdminCSRF(s.handleAdminCreateTenant))
	mux.Handle("PATCH /api/platform/tenants/{id}", s.platformAdminCSRF(s.handleAdminUpdateTenant))
	mux.Handle("PATCH /api/platform/tenants/{id}/status", s.platformAdminCSRF(s.handleAdminTenantStatus))
	mux.Handle("DELETE /api/platform/tenants/{id}", s.platformAdminCSRF(s.handleAdminDeleteTenant))
	mux.Handle("GET /api/platform/tenants/{id}/members", s.platformAdmin(s.handleAdminTenantMembers))
	mux.Handle("POST /api/platform/tenants/{id}/members", s.platformAdminCSRF(s.handleAdminTenantMemberAdd))
	mux.Handle("PATCH /api/platform/tenants/{id}/members/{user_id}/role", s.platformAdminCSRF(s.handleAdminTenantMemberSetRole))
	mux.Handle("DELETE /api/platform/tenants/{id}/members/{user_id}", s.platformAdminCSRF(s.handleAdminTenantMemberRemove))
	mux.Handle("PATCH /api/platform/users/{user_id}/status", s.platformAdminCSRF(s.handleAdminUserStatus))
	mux.Handle("POST /api/platform/users/{user_id}/password", s.platformAdminCSRF(s.handleAdminResetPassword))
	mux.Handle("POST /api/platform/users/{user_id}/platform-admin", s.platformAdminCSRF(s.handleAdminSetPlatformAdmin))

	// 商机面板（Dashboard，只读代理：与 Agent 工具共享同一 leads 实例与限流）
	mux.Handle("GET /api/leads/dashboard", s.protected(s.handleLeadsDashboard))
	mux.Handle("GET /api/leads/stats", s.protected(s.handleLeadsStats))

	// 会话历史（租户内视图）
	mux.Handle("GET /api/console/prompts", s.protected(s.handleConsolePrompts))
	mux.Handle("GET /api/console/memory", s.protected(s.handleConsoleMemoryStats))
	mux.Handle("GET /api/console/health", s.protected(s.handleConsoleHealth))
	mux.Handle("GET /api/tasks", s.protected(s.handleTasksList))
	mux.Handle("POST /api/tasks/consolidate", s.tenantManager(s.handleTaskConsolidate)) // P0-06：任务触发=租户管理
	mux.Handle("POST /api/tasks/lint", s.tenantManager(s.handleTaskLint))

	mux.Handle("GET /api/sessions/search", s.protected(s.handleSessionsSearch))
	mux.Handle("GET /api/sessions", s.protected(s.handleSessionList))
	mux.Handle("GET /api/sessions/{id}", s.protected(s.handleSessionGet))
	mux.Handle("PATCH /api/sessions/{id}", s.protectedCSRF(s.handleSessionRename)) // 手动重命名
	mux.Handle("DELETE /api/sessions/{id}", s.protectedCSRF(s.handleSessionDelete))

	// 中间件链：安全响应头 → 日志 → recover → （各路由自身的 requireAuth/requireCSRF）。
	// 当前版本的写接口均为 JSON；统一限制请求体，避免开放注册/Agent 消息用
	// 超大 Body 消耗内存。文件上传尚未实现，未来应使用独立路由和独立限额。
	const maxRequestBodyBytes = 1 << 20 // 1 MiB
	return logging(recoverPanic(securityHeaders(http.MaxBytesHandler(mux, maxRequestBodyBytes))))
}

// registrationEnabled 是否允许开放注册（未配置身份控制面时默认 false——仅 legacy 测试）。
func (s *Server) registrationEnabled() bool {
	if s.auth == nil || s.auth.RegistrationEnabled == nil {
		return false
	}
	return s.auth.RegistrationEnabled()
}

// securityHeaders 设置安全响应头（点击劫持/XSS 纵深防护，与 auth.Middleware 一致；
// 此处独立实现以兼容 legacy 无身份控制面装配）。
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// recoverPanic 兜底：panic 转 500，避免进程崩溃（设计文档 §6.3 纵深防御）。
func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[recover] panic: %v（%s %s）", rec, r.Method, r.URL.Path)
				writeError(w, http.StatusInternalServerError, "内部错误")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ── 中间件 ────────────────────────────────────────────────────

// logging wraps a handler with request logging.
func logging(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// ── 响应辅助 ──────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, format string, args ...any) {
	// MaxBytesHandler 通过请求体 Read 返回 *http.MaxBytesError；各 JSON handler
	// 原本统一把 decode 错误映射为 400，这里集中提升为准确的 413，避免每个
	// 路由重复判断或遗漏。
	if status == http.StatusBadRequest {
		for _, arg := range args {
			err, ok := arg.(error)
			if !ok {
				continue
			}
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				status = http.StatusRequestEntityTooLarge
				format = "请求体超过大小限制"
				args = nil
				break
			}
		}
	}
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf(format, args...)})
}

// newSessionID 生成一个会话 id。
func newSessionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "sess_" + hex.EncodeToString(b)
}

func decodeBody(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("请求体只能包含一个 JSON 对象")
		}
		return fmt.Errorf("请求体包含多余内容: %w", err)
	}
	return nil
}
