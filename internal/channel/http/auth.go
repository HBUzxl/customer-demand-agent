package http

import (
	"errors"
	"net"
	"net/http"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/identity"
)

// 登录限流（§7.2）：同一 IP+账号组合 1 分钟最多 10 次尝试，在 main 装配时传入
// auth.NewRateLimiter(10, time.Minute)。

// authView 认证接口最小响应视图（不返回密码 Hash、Token、内部目录、平台配置，§10.1）。
type authView struct {
	UserID      string                   `json:"user_id"`
	Email       string                   `json:"email"`
	DisplayName string                   `json:"display_name"`
	TenantID    string                   `json:"tenant_id"`
	TenantName  string                   `json:"tenant_name"`
	Slug        string                   `json:"slug"`
	Roles       []string                 `json:"roles"`
	Workspaces  []identity.WorkspaceView `json:"workspaces"`
	CSRFToken   string                   `json:"csrf"`
	// ProvisionFailed 租户数据面初始化失败（注册时 provision 中断）。前端据此
	// 进修复页（P0-09），业务路由在数据面闭合前不可用。
	ProvisionFailed bool `json:"provision_failed,omitempty"`
}

type registerRequest struct {
	OrgName     string `json:"org_name"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type passwordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type switchTenantRequest struct {
	TenantID string `json:"tenant_id"`
}

// handleRegister POST /api/auth/register —— 创建用户+租户+owner Membership 并签发登录态。
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	if !s.registrationEnabled() {
		writeError(w, http.StatusForbidden, "当前未开放注册")
		return
	}
	if s.auth.RegistrationLimiter != nil && !s.auth.RegistrationLimiter.Allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "注册尝试过于频繁，请稍后再试")
		return
	}
	var req registerRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	res, err := s.auth.Identity.Register(r.Context(), identity.RegisterRequest{
		OrgName: req.OrgName, DisplayName: req.DisplayName,
		Email: req.Email, Password: req.Password,
	}, clientIP(r), r.UserAgent())
	if err != nil {
		status, msg := authErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeAuthResult(w, s, res, http.StatusCreated)
}

// handleLogin POST /api/auth/login —— 登录并签发服务端 Session Cookie（§7.2 限流）。
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	var req loginRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	ip := clientIP(r)
	key := ip + "|" + identity.NormalizeEmail(req.Email)
	if s.auth.LoginIPLimiter != nil && !s.auth.LoginIPLimiter.Allow(ip) {
		writeError(w, http.StatusTooManyRequests, "尝试过于频繁，请稍后再试")
		return
	}
	if s.auth.LoginLimiter != nil && !s.auth.LoginLimiter.Allow(key) {
		writeError(w, http.StatusTooManyRequests, "尝试过于频繁，请稍后再试")
		return
	}
	res, err := s.auth.Identity.Login(r.Context(), req.Email, req.Password, clientIP(r), r.UserAgent())
	if err != nil {
		status, msg := authErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeAuthResult(w, s, res, http.StatusOK)
}

// handleLogout POST /api/auth/logout —— 废止当前登录会话并清 Cookie（幂等）。
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	raw := s.auth.Cookies.Read(r)
	_ = s.auth.Identity.Logout(raw)
	s.auth.Cookies.Clear(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleMe GET /api/auth/me —— 返回当前用户/租户/角色 + 当前稳定的 CSRF raw（§10.3）。
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	raw := s.auth.Cookies.Read(r)
	_, u, ten, m, err := s.auth.Identity.ValidateSession(raw)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录或登录已过期")
		return
	}
	workspaces, err := s.auth.Identity.WorkspacesForUser(u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "加载工作空间失败")
		return
	}
	csrfRaw, err := s.auth.Identity.CurrentCSRF(raw)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "会话已失效")
		return
	}
	roles := []string{m.Role}
	if u.PlatformAdmin {
		roles = append(roles, domain.RolePlatformAdmin)
	}
	writeJSON(w, http.StatusOK, authView{
		UserID: u.ID, Email: u.EmailNorm, DisplayName: u.DisplayName,
		TenantID: ten.ID, TenantName: ten.Name, Slug: ten.Slug,
		Roles: roles, CSRFToken: csrfRaw,
		Workspaces:      workspaces,
		ProvisionFailed: ten.Status == identity.TenantStatusProvisioningFailed,
	})
}

// handleSwitchTenant POST /api/auth/switch-tenant —— 校验 membership 后切换当前
// 会话的数据面并轮换 CSRF。客户端 tenant_id 只是目标候选，不直接形成 TenantScope。
func (s *Server) handleSwitchTenant(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	var req switchTenantRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	res, err := s.auth.Identity.SwitchTenant(s.auth.Cookies.Read(r), req.TenantID)
	if err != nil {
		status, msg := manageErrStatus(err)
		if errors.Is(err, identity.ErrAccountUnavailable) {
			status, msg = http.StatusForbidden, "目标工作空间不可用"
		}
		writeError(w, status, "%s", msg)
		return
	}
	writeAuthResult(w, s, res, http.StatusOK)
}

// handleResume POST /api/auth/resume —— 重试工作空间初始化（P0-09）。
// 注册时 provision 失败 → 租户 provisioning_failed → 登录后前端进修复页，
// 用户点「重试初始化」调用本接口：重跑 provisioner，成功后租户 active，可正常使用。
func (s *Server) handleResume(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	raw := s.auth.Cookies.Read(r)
	res, err := s.auth.Identity.ResumeProvision(r.Context(), raw)
	if err != nil {
		status, msg := authErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeAuthResult(w, s, res, http.StatusOK)
}

// handlePassword PUT /api/auth/password —— 校验旧密码后改密并废止其他会话。
func (s *Server) handlePassword(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	var req passwordRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	raw := s.auth.Cookies.Read(r)
	if err := s.auth.Identity.ChangePassword(r.Context(), raw, req.OldPassword, req.NewPassword); err != nil {
		status, msg := authErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// writeAuthResult 写入登录 Cookie + 返回最小视图。Token raw 只经 Set-Cookie 下发一次。
func writeAuthResult(w http.ResponseWriter, s *Server, res *identity.AuthResult, status int) {
	if res.TokenRaw != "" {
		s.auth.Cookies.Set(w, res.TokenRaw, identity.SessionAbsoluteTimeout)
	}
	writeJSON(w, status, authView{
		UserID: res.User.ID, Email: res.User.EmailNorm, DisplayName: res.User.DisplayName,
		TenantID: res.Tenant.ID, TenantName: res.Tenant.Name, Slug: res.Tenant.Slug,
		Roles: res.Roles, Workspaces: res.Workspaces, CSRFToken: res.CSRFRaw,
		ProvisionFailed: res.ProvisionFailed,
	})
}

// authErrStatus 身份错误 → HTTP 状态码与文案（统一文案防枚举，§7.1/§7.2）。
func authErrStatus(err error) (int, string) {
	switch {
	case errors.Is(err, identity.ErrRegistrationDisabled):
		return http.StatusForbidden, "当前未开放注册"
	case errors.Is(err, identity.ErrInvalidEmail), errors.Is(err, identity.ErrInvalidPassword),
		errors.Is(err, identity.ErrInvalidOldPassword), errors.Is(err, identity.ErrInvalidOrgName):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, identity.ErrInvalidCredentials):
		return http.StatusUnauthorized, "邮箱或密码错误"
	case errors.Is(err, identity.ErrAccountUnavailable):
		return http.StatusForbidden, "账号或组织状态异常，请联系管理员"
	case errors.Is(err, identity.ErrProvisionFailed):
		return http.StatusServiceUnavailable, "工作空间初始化失败，请重试"
	case errors.Is(err, identity.ErrEmailTaken):
		return http.StatusConflict, "该邮箱已注册"
	case errors.Is(err, identity.ErrTenantNameTaken):
		return http.StatusConflict, "该工作空间名称已存在"
	default:
		return http.StatusInternalServerError, "内部错误"
	}
}

// clientIP 从 RemoteAddr 提取客户端 IP（无端口形式；代理场景由反向代理覆盖）。
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
