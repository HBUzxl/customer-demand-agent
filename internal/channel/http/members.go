package http

import (
	"net/http"
)

// 租户成员管理（owner/admin，§4.2）handler。scope 由 s.sc(r) 从登录态解析，
// Service 层完成角色鉴权（canManageTenant/canManageRole）与最后 owner 保护。

// handleMembersList GET /api/members —— 当前租户全部成员（任何有效成员可见）。
func (s *Server) handleMembersList(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	members, err := s.auth.Identity.MembersList(&sc)
	if err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(members), "items": members})
}

type membersAddRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        string `json:"role"`
}

// handleMembersAdd POST /api/members —— 添加成员（已有邮箱复用；新邮箱建号）。
func (s *Server) handleMembersAdd(w http.ResponseWriter, r *http.Request) {
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
	if err := s.auth.Identity.MembersAdd(&sc, req.Email, req.DisplayName, req.Password, req.Role); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleCurrentTenant GET /api/tenant —— 当前租户安全资料视图。
func (s *Server) handleCurrentTenant(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	t, err := s.auth.Identity.CurrentTenant(&sc)
	if err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_id": t.ID, "name": t.Name, "slug": t.Slug, "status": t.Status,
		"created_at": t.CreatedAt,
	})
}

type currentTenantUpdateRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// handleCurrentTenantUpdate PATCH /api/tenant —— owner/admin 修改本租户资料。
func (s *Server) handleCurrentTenantUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	var req currentTenantUpdateRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体无效: %v", err)
		return
	}
	if err := s.auth.Identity.UpdateCurrentTenant(&sc, req.Name, req.Slug); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type membersRoleRequest struct {
	Role string `json:"role"`
}

// handleMembersSetRole PATCH /api/members/{user_id}/role —— 修改成员角色。
func (s *Server) handleMembersSetRole(w http.ResponseWriter, r *http.Request) {
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
	if err := s.auth.Identity.MembersSetRole(&sc, r.PathValue("user_id"), req.Role); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	if s.runs != nil {
		s.runs.CancelTenantUser(sc.TenantID, r.PathValue("user_id"))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleMembersRemove DELETE /api/members/{user_id} —— 移除成员。
func (s *Server) handleMembersRemove(w http.ResponseWriter, r *http.Request) {
	if !s.authReady() {
		writeError(w, http.StatusServiceUnavailable, "身份控制面未配置")
		return
	}
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	if err := s.auth.Identity.MembersRemove(&sc, r.PathValue("user_id")); err != nil {
		status, msg := manageErrStatus(err)
		writeError(w, status, "%s", msg)
		return
	}
	if s.runs != nil {
		s.runs.CancelTenantUser(sc.TenantID, r.PathValue("user_id"))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
