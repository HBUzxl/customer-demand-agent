package identity

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"customer-demand-agent/internal/domain"
)

// TestGrantPlatformAdminMergesIntoScope 平台管理员是用户级标记（§4.2），与成员角色
// 叠加进入 TenantScope/Login 响应——但业务数据仍按成员租户隔离（不是"超级租户"）。
func TestGrantPlatformAdminMergesIntoScope(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })

	res, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "某集团", DisplayName: "张三", Email: "zhangsan@example.com", Password: testPassword,
	}, "127.0.0.1", "go-test")
	if err != nil {
		t.Fatal(err)
	}
	// 默认非平台管理员
	sc, err := svc.CurrentTenantScope(res.TokenRaw)
	if err != nil {
		t.Fatal(err)
	}
	if sc.HasRole(domain.RolePlatformAdmin) {
		t.Fatal("默认注册用户不应具备 platform_admin")
	}

	// 授予后（无需重新登录——会话校验实时读用户标记）
	if err := svc.GrantPlatformAdmin(res.User.ID, true); err != nil {
		t.Fatal(err)
	}
	sc, err = svc.CurrentTenantScope(res.TokenRaw)
	if err != nil {
		t.Fatal(err)
	}
	if !sc.HasRole(domain.RoleOwner) || !sc.HasRole(domain.RolePlatformAdmin) {
		t.Fatalf("scope 应同时含 owner+platform_admin: %+v", sc.Roles)
	}
	// 平台管理员仍受租户边界约束：TenantID 不变，scope 仍按当前租户数据面
	if sc.TenantID != res.Tenant.ID {
		t.Fatalf("平台管理员不应改变租户归属: %s", sc.TenantID)
	}

	// Login 响应 Roles 同样合并；Login 废止旧会话并签发新 Token（单活跃会话）。
	lres, err := svc.Login(context.Background(), "zhangsan@example.com", testPassword, "127.0.0.1", "go-test")
	if err != nil {
		t.Fatal(err)
	}
	if !hasRole(lres.Roles, domain.RolePlatformAdmin) {
		t.Fatalf("Login 响应应含 platform_admin: %v", lres.Roles)
	}

	// 撤销后恢复（用 Login 新签发的 Token 校验）
	if err := svc.GrantPlatformAdmin(res.User.ID, false); err != nil {
		t.Fatal(err)
	}
	sc, err = svc.CurrentTenantScope(lres.TokenRaw)
	if err != nil {
		t.Fatalf("校验撤销后 scope: %v", err)
	}
	if sc.HasRole(domain.RolePlatformAdmin) {
		t.Fatal("撤销后不应再有 platform_admin")
	}
}

func hasRole(roles []string, want string) bool {
	for _, r := range roles {
		if r == want {
			return true
		}
	}
	return false
}

// TestUsersPlatformAdminColumnMigration 旧库（无 platform_admin 列）打开时幂等补列：
// ALTER 迁移后列存在、旧行默认 0、可正常读写。
func TestUsersPlatformAdminColumnMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old.db")

	// 构造旧版 users 表（无 platform_admin）并写入一行
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE users (
	  id TEXT PRIMARY KEY, email_norm TEXT NOT NULL UNIQUE, display_name TEXT NOT NULL,
	  password_hash TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active',
	  last_login_at TIMESTAMP, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users (id, email_norm, display_name, password_hash, status, created_at, updated_at)
		VALUES ('u1','legacy@example.com','存量','h','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	// Open 触发幂等 ALTER 迁移
	s, err := Open(path)
	if err != nil {
		t.Fatalf("旧库打开应自动迁移: %v", err)
	}
	defer s.Close()

	u, err := s.GetUserByEmailNorm("legacy@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if u.PlatformAdmin {
		t.Fatal("旧行迁移后 platform_admin 应为默认 0")
	}
	if err := s.SetUserPlatformAdmin(u.ID, true); err != nil {
		t.Fatal(err)
	}
	u2, err := s.GetUserByID(u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !u2.PlatformAdmin {
		t.Fatal("迁移后的列应可正常写入/读取")
	}
}
