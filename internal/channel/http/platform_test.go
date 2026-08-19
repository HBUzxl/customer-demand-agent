package http_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// 平台管理路由（/api/platform/**）授权：设计文档 §7.5 把全局模型配置/原始日志/
// 平台 LLM 审计从对所有租户开放迁移到 platform_admin 专用。platform_admin 是
// 用户级平台运维元数据权限（§4.2），不是"超级租户"——租户业务数据仍按租户隔离。

// TestPlatformRoutesTenantForbidden 普通租户 owner 访问平台管理路由一律 403；
// 租户 GET /api/config 只保留安全视图（不含 data_dir/wiki_dir/history_db）。
func TestPlatformRoutesTenantForbidden(t *testing.T) {
	ts, _, a, _ := setupTwoTenantServer(t)

	// 平台管理路由（只读 + 修改 + 探测 + 模型列表）→ 403
	for _, tc := range []struct {
		method, path string
		csrf         bool
	}{
		{"GET", "/api/platform/config", false},
		{"GET", "/api/platform/llm-audit", false},
		{"GET", "/api/platform/logs", false},
		{"PUT", "/api/platform/config", true},
		{"POST", "/api/platform/config/test", true},
		{"POST", "/api/platform/models", true},
		{"GET", "/api/platform/users", false},
		{"POST", "/api/platform/users", true},
		{"PATCH", "/api/platform/users/not-allowed", true},
		{"DELETE", "/api/platform/users/not-allowed", true},
		{"GET", "/api/platform/tenants", false},
		{"POST", "/api/platform/tenants", true},
		{"PATCH", "/api/platform/tenants/not-allowed", true},
		{"DELETE", "/api/platform/tenants/not-allowed", true},
		{"POST", "/api/platform/tenants/not-allowed/members", true},
	} {
		csrf := ""
		if tc.csrf {
			csrf = a.CSRF
		}
		if code, body, _ := rawDo(t, ts, tc.method, tc.path, a.Cookie, csrf, nil); code != http.StatusForbidden {
			t.Errorf("%s %s 普通租户应 403，got %d: %v", tc.method, tc.path, code, body)
		}
	}

	// 租户安全视图仍可用（model 别名 + 掩码 key），但不暴露平台敏感路径
	code, body, _ := rawDo(t, ts, "GET", "/api/config", a.Cookie, "", nil)
	if code != http.StatusOK {
		t.Fatalf("租户 GET /api/config 应 200，got %d: %v", code, body)
	}
	raw := jsonMarshal(body)
	for _, banned := range []string{"data_dir", "wiki_dir", "history_db", "default_user"} {
		if strings.Contains(string(raw), "\""+banned+"\"") {
			t.Errorf("租户安全视图不应泄露 %q 字段: %s", banned, raw)
		}
	}
}

// TestPlatformRoutesAdminAllowed 平台管理员可访问平台管理路由；授权在会话校验时
// 实时生效（无需重新登录——CurrentTenantScope 每次从身份库读用户标记）。
func TestPlatformRoutesAdminAllowed(t *testing.T) {
	ts, idSvc, a, b := setupTwoTenantServer(t)
	if err := idSvc.GrantPlatformAdmin(a.UserID, true); err != nil {
		t.Fatal(err)
	}

	if code, body, _ := rawDo(t, ts, "GET", "/api/platform/config", a.Cookie, "", nil); code != http.StatusOK {
		t.Fatalf("平台管理员 GET /api/platform/config 应 200，got %d: %v", code, body)
	} else if _, ok := body["data"]; !ok {
		t.Errorf("平台完整配置视图应含 data（data_dir 等），got %v", body)
	}
	if code, body, _ := rawDo(t, ts, "GET", "/api/platform/llm-audit", a.Cookie, "", nil); code != http.StatusOK {
		t.Errorf("平台管理员 GET /api/platform/llm-audit 应 200，got %d: %v", code, body)
	}
	// SSE 日志流：只验证头（200）即关闭，避免阻塞在读流。
	{
		req, _ := http.NewRequest("GET", ts.URL+"/api/platform/logs", nil)
		req.Header.Set("Cookie", a.Cookie)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("平台管理员 GET /api/platform/logs 应 200（SSE 已就绪），got %d", resp.StatusCode)
		}
	}

	// 另一普通租户 B 仍被拒绝（授权不扩散）
	if code, _, _ := rawDo(t, ts, "GET", "/api/platform/config", b.Cookie, "", nil); code != http.StatusForbidden {
		t.Errorf("普通租户 B 访问平台路由应 403，got %d", code)
	}

	// 平台管理员角色也反映在 /auth/me
	_, me, _ := rawDo(t, ts, "GET", "/api/auth/me", a.Cookie, "", nil)
	roles, _ := me["roles"].([]any)
	found := false
	for _, r := range roles {
		if r == "platform_admin" {
			found = true
		}
	}
	if !found {
		t.Errorf("/auth/me 应含 platform_admin 角色，got %v", roles)
	}
}

func TestPlatformUserTenantCRUDHTTP(t *testing.T) {
	ts, idSvc, a, b := setupTwoTenantServer(t)
	if err := idSvc.GrantPlatformAdmin(a.UserID, true); err != nil {
		t.Fatal(err)
	}

	// 新建租户，复用 B 用户作为 owner。
	code, body, _ := rawDo(t, ts, "POST", "/api/platform/tenants", a.Cookie, a.CSRF, map[string]any{
		"name": "C公司", "slug": "company-c", "owner_email": "b@example.com",
	})
	if code != http.StatusCreated {
		t.Fatalf("平台新建租户应 201，got %d: %v", code, body)
	}
	tenantID, _ := body["tenant_id"].(string)
	if tenantID == "" {
		t.Fatal("新建租户应返回 tenant_id")
	}
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/platform/tenants/"+tenantID+"/status", a.Cookie, a.CSRF,
		map[string]any{"status": "bogus"}); code != http.StatusBadRequest {
		t.Fatalf("非法租户状态应 400，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/platform/tenants/"+tenantID, a.Cookie, a.CSRF,
		map[string]any{"name": "C公司更新", "slug": "company-c-2"}); code != http.StatusOK {
		t.Fatalf("平台更新租户应 200，got %d", code)
	}

	// 在 C 新建成员、改角色、移除。
	if code, _, _ := rawDo(t, ts, "POST", "/api/platform/tenants/"+tenantID+"/members", a.Cookie, a.CSRF,
		map[string]any{"email": "invalid-role@example.com", "display_name": "非法角色", "password": "password-123456", "role": "superuser"}); code != http.StatusBadRequest {
		t.Fatalf("非法成员角色应 400，got %d", code)
	}
	code, _, _ = rawDo(t, ts, "POST", "/api/platform/tenants/"+tenantID+"/members", a.Cookie, a.CSRF,
		map[string]any{"email": "c-member@example.com", "display_name": "C成员", "password": "password-123456", "role": "analyst"})
	if code != http.StatusCreated {
		t.Fatalf("平台添加租户成员应 201，got %d", code)
	}
	_, membersBody, _ := rawDo(t, ts, "GET", "/api/platform/tenants/"+tenantID+"/members", a.Cookie, "", nil)
	var memberID string
	for _, raw := range membersBody["items"].([]any) {
		m := raw.(map[string]any)
		if m["email"] == "c-member@example.com" {
			memberID, _ = m["user_id"].(string)
		}
	}
	if memberID == "" {
		t.Fatal("应找到新建成员")
	}
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/platform/tenants/"+tenantID+"/members/"+memberID+"/role",
		a.Cookie, a.CSRF, map[string]any{"role": "reviewer"}); code != http.StatusOK {
		t.Fatalf("平台改成员角色应 200，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "DELETE", "/api/platform/tenants/"+tenantID+"/members/"+memberID,
		a.Cookie, a.CSRF, nil); code != http.StatusOK {
		t.Fatalf("平台移除成员应 200，got %d", code)
	}

	// 平台用户 CRUD：新建并加入 A、修改、软删除。
	code, body, _ = rawDo(t, ts, "POST", "/api/platform/users", a.Cookie, a.CSRF, map[string]any{
		"email": "platform-created@example.com", "display_name": "平台创建", "password": "password-123456",
		"tenant_id": a.TenantID, "role": "analyst",
	})
	if code != http.StatusCreated {
		t.Fatalf("平台新建用户应 201，got %d: %v", code, body)
	}
	userID, _ := body["user_id"].(string)
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/platform/users/"+userID, a.Cookie, a.CSRF,
		map[string]any{"email": "platform-updated@example.com", "display_name": "平台更新"}); code != http.StatusOK {
		t.Fatalf("平台更新用户应 200，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "DELETE", "/api/platform/users/"+userID, a.Cookie, a.CSRF, nil); code != http.StatusOK {
		t.Fatalf("平台软删除用户应 200，got %d", code)
	}

	// 租户停用/启用/软删除；平台管理员当前 A 租户不能被误操作。
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/platform/tenants/"+tenantID+"/status", a.Cookie, a.CSRF,
		map[string]any{"status": "disabled"}); code != http.StatusOK {
		t.Fatalf("停用租户应 200，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/platform/tenants/"+tenantID+"/status", a.Cookie, a.CSRF,
		map[string]any{"status": "active"}); code != http.StatusOK {
		t.Fatalf("启用租户应 200，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "DELETE", "/api/platform/tenants/"+tenantID, a.Cookie, a.CSRF, nil); code != http.StatusOK {
		t.Fatalf("软删除租户应 200，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "PATCH", "/api/platform/tenants/"+tenantID+"/members/"+b.UserID+"/role",
		a.Cookie, a.CSRF, map[string]any{"role": "analyst"}); code != http.StatusConflict {
		t.Fatalf("已删除租户不应继续修改成员，got %d", code)
	}
	if code, _, _ := rawDo(t, ts, "DELETE", "/api/platform/tenants/"+a.TenantID, a.Cookie, a.CSRF, nil); code != http.StatusBadRequest {
		t.Fatalf("删除当前租户应被保护为 400，got %d", code)
	}

	// B 仍只能访问自己的业务数据，平台 CRUD 不授予其平台权限。
	if code, _, _ := rawDo(t, ts, "GET", "/api/platform/users", b.Cookie, "", nil); code != http.StatusForbidden {
		t.Fatalf("普通 B 用户仍应 403，got %d", code)
	}
}

// jsonMarshal 是断言辅助：把响应 map 序列化为 JSON 文本（便于字符串检查）。
func jsonMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
