package http

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/identity"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/tenancy"
)

// ensureUserProfileLocked 保证当前租户成员拥有以 user_id 绑定的使用者画像。
// 调用方持有 Server.profileMu；显示名冲突时用邮箱消歧，避免两个成员覆盖画像。
func (s *Server) ensureUserProfileLocked(rt *tenancy.Runtime, sc domain.TenantScope) error {
	if rt == nil || sc.UserID == "" {
		return nil
	}
	if _, err := rt.Wiki.GetUserProfileByUserID(sc.UserID); err == nil {
		return nil
	}
	name := strings.TrimSpace(sc.DisplayName)
	if name == "" {
		name = strings.TrimSpace(sc.Email)
	}
	if name == "" {
		name = "用户-" + shortIdentity(sc.UserID)
	}
	if existing, err := rt.Wiki.GetEntry("user", name); err == nil && existing.UserID != sc.UserID {
		suffix := strings.TrimSpace(sc.Email)
		if suffix == "" {
			suffix = shortIdentity(sc.UserID)
		}
		name = fmt.Sprintf("%s（%s）", name, suffix)
		if collision, err := rt.Wiki.GetEntry("user", name); err == nil && collision.UserID != sc.UserID {
			name += "-" + shortIdentity(sc.UserID)
		}
	}
	role := tenantRoleLabel(sc.Roles)
	entry := longterm.NewUserEntry(longterm.UserProfile{
		Name: name, UserID: sc.UserID, Level: "中级", Tags: []string{"平台使用者", role},
	}, sc.Email, role)
	return rt.Wiki.UpsertEntry(entry)
}

func shortIdentity(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func tenantRoleLabel(roles []string) string {
	for _, role := range []struct{ value, label string }{
		{domain.RoleOwner, "所有者"},
		{domain.RoleAdmin, "管理员"},
		{domain.RoleReviewer, "审核人员"},
		{domain.RoleAnalyst, "分析人员"},
	} {
		for _, got := range roles {
			if got == role.value {
				return role.label
			}
		}
	}
	return "成员"
}

// syncTenantUserProfiles 把身份控制面的全部有效成员映射为本租户使用者画像。
// 旧演示画像没有 user_id，不会被认作任何成员，也不会参与前端列表或 Agent 解析。
func (s *Server) syncTenantUserProfiles(r *http.Request, rt *tenancy.Runtime) error {
	if !s.authReady() {
		return nil
	}
	sc, err := s.sc(r)
	if err != nil {
		return err
	}
	members, err := s.auth.Identity.MembersList(&sc)
	if err != nil {
		return err
	}
	s.profileMu.Lock()
	defer s.profileMu.Unlock()
	for _, member := range members {
		if member.Status != identity.MembershipStatusActive {
			continue
		}
		memberScope := domain.TenantScope{
			TenantID: sc.TenantID, UserID: member.UserID, DisplayName: member.DisplayName,
			Email: member.Email, Roles: []string{member.Role}, Source: "profile_sync",
		}
		if err := s.ensureUserProfileLocked(rt, memberScope); err != nil {
			return err
		}
	}
	return nil
}

func visibleMemoryEntries(entries []*longterm.Entry) []*longterm.Entry {
	out := make([]*longterm.Entry, 0, len(entries))
	for _, entry := range entries {
		if entry == nil || (entry.Type == domain.MemoryUser && entry.UserID == "") {
			continue
		}
		out = append(out, entry)
	}
	return out
}

type customerCreateReq struct {
	Name             string   `json:"name"`
	Industry         string   `json:"industry"`
	Scale            string   `json:"scale"`
	TechStack        []string `json:"tech_stack"`
	ExistingSecurity []string `json:"existing_security"`
	PainPoints       []string `json:"pain_points"`
	ProcurementPref  string   `json:"procurement_pref"`
	Notes            string   `json:"notes"`
}

// handleCustomerCreate 是新会话客户登记的最小权限入口：任意已认证租户成员
// 可以在自己的租户内创建客户画像，但不能覆盖同名客户，也不能借此写其他记忆
// 类型。租户隔离由 s.rt(r) 选择的 tenant runtime 保证。
func (s *Server) handleCustomerCreate(w http.ResponseWriter, r *http.Request) {
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	var req customerCreateReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name 不能为空")
		return
	}
	if len([]rune(req.Name)) > 120 {
		writeError(w, http.StatusBadRequest, "name 不能超过 120 个字符")
		return
	}
	if _, err := rt.Wiki.GetEntry("customer", req.Name); err == nil {
		writeError(w, http.StatusConflict, "客户 %q 已存在，请直接选择已有客户", req.Name)
		return
	}
	entry := longterm.NewCustomerEntry(longterm.CustomerProfile{
		Name: req.Name, Industry: strings.TrimSpace(req.Industry), Scale: strings.TrimSpace(req.Scale),
		TechStack: req.TechStack, ExistingSecurity: req.ExistingSecurity,
		PainPoints: req.PainPoints, ProcurementPref: strings.TrimSpace(req.ProcurementPref),
		Notes: strings.TrimSpace(req.Notes), Tags: compactCustomerTags(req.Industry),
	})
	if err := rt.Wiki.UpsertEntry(entry); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok", "name": req.Name})
}

func compactCustomerTags(industry string) []string {
	industry = strings.TrimSpace(industry)
	if industry == "" {
		return nil
	}
	return []string{industry}
}

// handleMemorySearch: GET /api/memory/search?q=&type=&limit=
func (s *Server) handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	q := r.URL.Query().Get("q")
	typ := r.URL.Query().Get("type")
	if typ == "" || typ == string(domain.MemoryUser) {
		if err := s.syncTenantUserProfiles(r, rt); err != nil {
			writeError(w, http.StatusInternalServerError, "同步使用者画像失败")
			return
		}
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	hits := visibleMemoryEntries(rt.Wiki.SearchEntry(q, typ, limit))
	writeJSON(w, http.StatusOK, map[string]any{"count": len(hits), "items": hits})
}

// handleMemoryList: GET /api/memory/list?type=&offset=&limit=
func (s *Server) handleMemoryList(w http.ResponseWriter, r *http.Request) {
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	typ := r.URL.Query().Get("type")
	if typ == "" || typ == string(domain.MemoryUser) {
		if err := s.syncTenantUserProfiles(r, rt); err != nil {
			writeError(w, http.StatusInternalServerError, "同步使用者画像失败")
			return
		}
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries := visibleMemoryEntries(rt.Wiki.ListEntry(typ, offset, limit))
	writeJSON(w, http.StatusOK, map[string]any{"count": len(entries), "items": entries})
}

// handleMemoryGet: GET /api/memory/{type}/{title}
func (s *Server) handleMemoryGet(w http.ResponseWriter, r *http.Request) {
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	typ := r.PathValue("type")
	title := r.PathValue("title")
	if typ == string(domain.MemoryUser) {
		if err := s.syncTenantUserProfiles(r, rt); err != nil {
			writeError(w, http.StatusInternalServerError, "同步使用者画像失败")
			return
		}
	}
	entry, err := rt.Wiki.GetEntry(typ, title)
	if err != nil || (entry.Type == domain.MemoryUser && entry.UserID == "") {
		writeError(w, http.StatusNotFound, "记忆条目不存在")
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

// memoryUpsertReq is the body for POST /api/memory (human-facing, bypasses AI permission).
type memoryUpsertReq struct {
	Type    string   `json:"type"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Aliases []string `json:"aliases"`
	Summary string   `json:"summary"`
}

// handleMemoryUpsert: POST /api/memory
// 这是人工管理入口；产品始终只读，普通成员只能编辑自己的身份绑定画像，
// 其他租户记忆仅 owner/admin 可写。类型权限必须由后端校验，不能只依赖前端隐藏。
// 人工=可信源：一律 verified，不接受 status 传参（pending 是 AI 写入专属语义，
// 审核状态只能通过审核流改变——P10）。
func (s *Server) handleMemoryUpsert(w http.ResponseWriter, r *http.Request) {
	sc, err := s.sc(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "未登录")
		return
	}
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	var req memoryUpsertReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	mt, ok := parseType(req.Type)
	if !ok {
		writeError(w, http.StatusBadRequest, "未知 type: %q", req.Type)
		return
	}
	if mt == domain.MemoryProduct {
		writeError(w, http.StatusForbidden, "产品知识属于平台只读基线，不能在租户内修改")
		return
	}
	existing, existingErr := rt.Wiki.GetEntry(req.Type, req.Title)
	if mt == domain.MemoryUser {
		// 使用者由身份成员自动创建；本人只能维护自己的画像，租户管理员可维护
		// 本租户全部画像。禁止通过通用接口伪造任意 user_id 或新建“幽灵用户”。
		if existingErr != nil || existing.UserID == "" {
			writeError(w, http.StatusBadRequest, "使用者由租户成员自动生成，不能手工新建")
			return
		}
		if !sc.IsTenantAdmin() && existing.UserID != sc.UserID {
			writeError(w, http.StatusForbidden, "只能编辑自己的使用者画像")
			return
		}
	} else if !sc.IsTenantAdmin() {
		writeError(w, http.StatusForbidden, "普通成员对平台知识和租户记忆仅有查看权限")
		return
	}
	status := longterm.StatusVerified
	// 若已存在，以旧条目为基底覆盖可编辑字段，保留其结构化 frontmatter（如产品 capabilities）
	entry := &longterm.Entry{
		Type:    mt,
		Title:   req.Title,
		Content: req.Content,
		Tags:    req.Tags,
		Aliases: req.Aliases,
		Summary: req.Summary,
		Status:  status,
	}
	if existingErr == nil {
		entry = existing // existing.typed 已填充（结构化字段保留）
		entry.Content = req.Content
		if req.Tags != nil {
			entry.Tags = req.Tags
		}
		if req.Aliases != nil {
			entry.Aliases = req.Aliases
		}
		entry.Summary = req.Summary
		entry.Status = status
	}
	if err := rt.Wiki.UpsertEntry(entry); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "type": req.Type, "title": req.Title})
}

// handleMemoryDelete: DELETE /api/memory/{type}/{title}?archive=true
func (s *Server) handleMemoryDelete(w http.ResponseWriter, r *http.Request) {
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	typ := r.PathValue("type")
	title := r.PathValue("title")
	if typ == string(domain.MemoryProduct) || typ == string(domain.MemoryUser) {
		writeError(w, http.StatusForbidden, "产品知识和使用者身份画像不能删除")
		return
	}
	archive := r.URL.Query().Get("archive") != "false"
	if err := rt.Wiki.DeleteEntry(typ, title, archive); err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	action := "已归档"
	if !archive {
		action = "已删除"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": action})
}

// handleMemoryHistory P6 时效查询：条目历史版本链（superseded 归档）。
func (s *Server) handleMemoryHistory(w http.ResponseWriter, r *http.Request) {
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	typeStr := r.PathValue("type")
	title := r.PathValue("title")
	hist := rt.Wiki.GetEntryHistory(typeStr, title)
	if hist == nil {
		writeJSON(w, http.StatusOK, map[string]any{"history": []any{}, "count": 0})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": hist, "count": len(hist)})
}
