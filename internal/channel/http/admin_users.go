package http

import (
	"errors"
	"net/http"

	"customer-demand-agent/internal/identity"
)

// 平台管理（platform_admin）用户/租户管理 handler。平台管理员只有平台级运维元数据
// 权限（§4.2），不越入任何租户业务数据；审计 actor 取当前请求作用域 sc.UserID。

// manageErrStatus 管理类错误 → HTTP 状态码与文案（用户/成员管理共用）。
func manageErrStatus(err error) (int, string) {
	switch {
	case errors.Is(err, identity.ErrNotFound):
		return http.StatusNotFound, "资源不存在"
	case errors.Is(err, identity.ErrForbidden):
		return http.StatusForbidden, err.Error()
	case errors.Is(err, identity.ErrMemberExists):
		return http.StatusConflict, err.Error()
	case errors.Is(err, identity.ErrEmailTaken), errors.Is(err, identity.ErrTenantNameTaken),
		errors.Is(err, identity.ErrTenantDeleted), errors.Is(err, identity.ErrAccountUnavailable):
		return http.StatusConflict, err.Error()
	case errors.Is(err, identity.ErrLastOwner), errors.Is(err, identity.ErrLastPlatformAdmin),
		errors.Is(err, identity.ErrSelfAction), errors.Is(err, identity.ErrInvalidEmail),
		errors.Is(err, identity.ErrInvalidPassword), errors.Is(err, identity.ErrInvalidOrgName),
		errors.Is(err, identity.ErrInvalidSlug), errors.Is(err, identity.ErrInvalidDisplayName),
		errors.Is(err, identity.ErrTenantRequired), errors.Is(err, identity.ErrInvalidStatus),
		errors.Is(err, identity.ErrInvalidRole):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, identity.ErrProvisionFailed):
		return http.StatusServiceUnavailable, err.Error()
	default:
		return http.StatusInternalServerError, "内部错误"
	}
}

// handleAdminUsers GET /api/platform/users —— 平台全量用户及成员关系。
func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	users, err := s.auth.Identity.AdminUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询用户失败: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(users), "items": users})
}

// handleAdminTenants GET /api/platform/tenants —— 平台全量租户。
func (s *Server) handleAdminTenants(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	tenants, err := s.auth.Identity.AdminTenants()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询租户失败: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(tenants), "items": tenants})
}

// handleAdminTenantMembers GET /api/platform/tenants/{id}/members —— 某租户全部成员。
func (s *Server) handleAdminTenantMembers(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	members, err := s.auth.Identity.AdminTenantMembers(r.PathValue("id"))
	if err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(members), "items": members})
}

type adminUserStatusRequest struct {
	Status string `json:"status"`
}

// handleAdminUserStatus PATCH /api/platform/users/{user_id}/status —— 禁用/启用账号。
func (s *Server) handleAdminUserStatus(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var req adminUserStatusRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	if err := s.auth.Identity.AdminSetUserStatus(sc.UserID, r.PathValue("user_id"), req.Status); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	if req.Status == identity.UserStatusDisabled && s.runs != nil {
		s.runs.CancelUser(r.PathValue("user_id"))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type adminResetPasswordRequest struct {
	Password string `json:"password"`
}

// handleAdminResetPassword POST /api/platform/users/{user_id}/password —— 重置密码。
func (s *Server) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var req adminResetPasswordRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	if err := s.auth.Identity.AdminResetPassword(sc.UserID, r.PathValue("user_id"), req.Password); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	if s.runs != nil {
		s.runs.CancelUser(r.PathValue("user_id"))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type adminSetPlatformAdminRequest struct {
	On bool `json:"on"`
}

// handleAdminSetPlatformAdmin POST /api/platform/users/{user_id}/platform-admin —— 授予/撤销平台管理员。
func (s *Server) handleAdminSetPlatformAdmin(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var req adminSetPlatformAdminRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	if err := s.auth.Identity.AdminSetPlatformAdmin(sc.UserID, r.PathValue("user_id"), req.On); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type adminCreateUserRequest struct {
	Email         string `json:"email"`
	DisplayName   string `json:"display_name"`
	Password      string `json:"password"`
	TenantID      string `json:"tenant_id"`
	Role          string `json:"role"`
	PlatformAdmin bool   `json:"platform_admin"`
}

// handleAdminCreateUser POST /api/platform/users —— 新建用户并加入指定租户。
func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var req adminCreateUserRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	u, err := s.auth.Identity.AdminCreateUser(sc.UserID, identity.AdminCreateUserRequest{
		Email: req.Email, DisplayName: req.DisplayName, Password: req.Password,
		TenantID: req.TenantID, Role: req.Role, PlatformAdmin: req.PlatformAdmin,
	})
	if err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok", "user_id": u.ID})
}

type adminUpdateUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// handleAdminUpdateUser PATCH /api/platform/users/{user_id} —— 修改用户全局资料。
func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var req adminUpdateUserRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	if err := s.auth.Identity.AdminUpdateUser(sc.UserID, r.PathValue("user_id"), req.Email, req.DisplayName); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminDeleteUser DELETE /api/platform/users/{user_id} —— 软删除（停用）用户。
func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	if err := s.auth.Identity.AdminDeleteUser(sc.UserID, r.PathValue("user_id")); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	if s.runs != nil {
		s.runs.CancelUser(r.PathValue("user_id"))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

type adminCreateTenantRequest struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	OwnerEmail       string `json:"owner_email"`
	OwnerDisplayName string `json:"owner_display_name"`
	OwnerPassword    string `json:"owner_password"`
}

// handleAdminCreateTenant POST /api/platform/tenants —— 新建租户与 owner。
func (s *Server) handleAdminCreateTenant(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var req adminCreateTenantRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	t, err := s.auth.Identity.AdminCreateTenant(r.Context(), sc.UserID, identity.AdminCreateTenantRequest{
		Name: req.Name, Slug: req.Slug, OwnerEmail: req.OwnerEmail,
		OwnerDisplayName: req.OwnerDisplayName, OwnerPassword: req.OwnerPassword,
	})
	if err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": t.Status, "tenant_id": t.ID})
}

type adminUpdateTenantRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Server) handleAdminUpdateTenant(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var req adminUpdateTenantRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	if err := s.auth.Identity.AdminUpdateTenant(sc.UserID, r.PathValue("id"), req.Name, req.Slug); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type adminTenantStatusRequest struct {
	Status string `json:"status"`
}

func (s *Server) handleAdminTenantStatus(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	tenantID := r.PathValue("id")
	var req adminTenantStatusRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	if err := s.auth.Identity.AdminSetTenantStatus(r.Context(), sc.UserID, sc.TenantID, tenantID, req.Status); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	if req.Status == identity.TenantStatusDisabled {
		if s.runs != nil {
			s.runs.DropTenant(tenantID)
		}
		if s.runtimes != nil {
			_ = s.runtimes.CloseTenant(tenantID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (s *Server) handleAdminDeleteTenant(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	tenantID := r.PathValue("id")
	if err := s.auth.Identity.AdminDeleteTenant(sc.UserID, sc.TenantID, tenantID); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	if s.runs != nil {
		s.runs.DropTenant(tenantID)
	}
	if s.runtimes != nil {
		_ = s.runtimes.CloseTenant(tenantID)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": identity.TenantStatusDeleted})
}

// 平台管理员在指定租户中增/改/删 membership。只改控制面，不创建目标租户
// TenantScope，因此不会绕过业务数据隔离。
func (s *Server) handleAdminTenantMemberAdd(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var req membersAddRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	if err := s.auth.Identity.AdminTenantMemberAdd(sc.UserID, r.PathValue("id"),
		req.Email, req.DisplayName, req.Password, req.Role); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (s *Server) handleAdminTenantMemberSetRole(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var req membersRoleRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	if err := s.auth.Identity.AdminTenantMemberSetRole(sc.UserID, r.PathValue("id"),
		r.PathValue("user_id"), req.Role); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	if s.runs != nil {
		s.runs.CancelTenantUser(r.PathValue("id"), r.PathValue("user_id"))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAdminTenantMemberRemove(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	if err := s.auth.Identity.AdminTenantMemberRemove(sc.UserID, r.PathValue("id"),
		r.PathValue("user_id")); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	if s.runs != nil {
		s.runs.CancelTenantUser(r.PathValue("id"), r.PathValue("user_id"))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
