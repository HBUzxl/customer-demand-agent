package identity

import (
	"context"
	"errors"
	"testing"

	"customer-demand-agent/internal/domain"
)

// scopeFor 构造测试用作用域（Service 层鉴权只看 Roles，调用方成员关系由
// middleware 保证——这里直接构造，聚焦权限矩阵）。
func scopeFor(tenantID, userID, role string) *domain.TenantScope {
	return &domain.TenantScope{TenantID: tenantID, UserID: userID, Roles: []string{role}, Source: "test"}
}

// setupOwner 注册一个租户并返回 owner 作用域。
func setupOwner(t *testing.T, svc *Service) (*domain.TenantScope, string) {
	t.Helper()
	res, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "某集团", DisplayName: "张三", Email: "owner@example.com", Password: testPassword,
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := svc.CurrentTenantScope(res.TokenRaw)
	if err != nil {
		t.Fatal(err)
	}
	return sc, res.User.ID
}

// ── 平台管理（platform_admin 侧）──────────────────────────────────

func TestAdminLists(t *testing.T) {
	svc := NewService(newTestStore(t))
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	ownerSc, ownerUID := setupOwner(t, svc)
	if _, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "B公司", DisplayName: "王五", Email: "wang@example.com", Password: testPassword,
	}, "", ""); err != nil {
		t.Fatal(err)
	}

	users, err := svc.AdminUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("应列出 2 个用户，got %d", len(users))
	}
	var ownerV *AdminUserView
	for i := range users {
		if users[i].UserID == ownerUID {
			ownerV = &users[i]
		}
	}
	if ownerV == nil || len(ownerV.Memberships) != 1 || ownerV.Memberships[0].Role != domain.RoleOwner {
		t.Fatalf("owner 视图成员关系异常: %+v", ownerV)
	}

	tenants, err := svc.AdminTenants()
	if err != nil {
		t.Fatal(err)
	}
	if len(tenants) != 2 {
		t.Fatalf("应列出 2 个租户，got %d", len(tenants))
	}
	for _, tv := range tenants {
		if tv.MemberCount != 1 {
			t.Fatalf("租户 %s 成员数应为 1，got %d", tv.Name, tv.MemberCount)
		}
	}

	members, err := svc.AdminTenantMembers(ownerSc.TenantID)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].Role != domain.RoleOwner {
		t.Fatalf("租户成员应为 1 个 owner: %+v", members)
	}
	if _, err := svc.AdminTenantMembers("no-such-tenant"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("不存在租户应 ErrNotFound: %v", err)
	}
}

func TestAdminSetUserStatus(t *testing.T) {
	svc := NewService(newTestStore(t))
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	ownerSc, ownerUID := setupOwner(t, svc)
	other, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "B公司", DisplayName: "赵六", Email: "zhao@example.com", Password: testPassword,
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	otherScope := scopeFor(other.Tenant.ID, other.User.ID, domain.RoleOwner)
	if err := svc.MembersAdd(otherScope, "zhao-co@example.com", "联合所有者", testPassword, domain.RoleOwner); err != nil {
		t.Fatal(err)
	}

	// 禁用他人 → 登录被拒（账号状态异常）
	if err := svc.AdminSetUserStatus(ownerSc.UserID, other.User.ID, UserStatusDisabled); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(context.Background(), "zhao@example.com", testPassword, "", ""); !errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("禁用后登录应 ErrAccountUnavailable: %v", err)
	}
	// 重新启用 → 可登录
	if err := svc.AdminSetUserStatus(ownerSc.UserID, other.User.ID, UserStatusActive); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(context.Background(), "zhao@example.com", testPassword, "", ""); err != nil {
		t.Fatalf("启用后应可登录: %v", err)
	}
	// 禁用自己 → ErrSelfAction
	if err := svc.AdminSetUserStatus(ownerSc.UserID, ownerUID, UserStatusDisabled); !errors.Is(err, ErrSelfAction) {
		t.Fatalf("禁用自己应 ErrSelfAction: %v", err)
	}
	// 非法状态
	if err := svc.AdminSetUserStatus(ownerSc.UserID, other.User.ID, "bogus"); err == nil {
		t.Fatal("非法状态应报错")
	}
}

func TestAdminResetPassword(t *testing.T) {
	svc := NewService(newTestStore(t))
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	ownerSc, _ := setupOwner(t, svc)
	other, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "B公司", DisplayName: "钱七", Email: "qian@example.com", Password: testPassword,
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	newPw := "Newpass-123456"
	if err := svc.AdminResetPassword(ownerSc.UserID, other.User.ID, newPw); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(context.Background(), "qian@example.com", newPw, "", ""); err != nil {
		t.Fatalf("新密码应可登录: %v", err)
	}
	if _, err := svc.Login(context.Background(), "qian@example.com", testPassword, "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("旧密码应失效: %v", err)
	}
	if err := svc.AdminResetPassword(ownerSc.UserID, other.User.ID, "weak"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("弱密码应 ErrInvalidPassword: %v", err)
	}
}

func TestAdminSetPlatformAdmin(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	ownerSc, ownerUID := setupOwner(t, svc)
	other, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "B公司", DisplayName: "孙八", Email: "sun@example.com", Password: testPassword,
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.GrantPlatformAdmin(ownerUID, true); err != nil {
		t.Fatal(err)
	}
	if err := svc.AdminSetPlatformAdmin(ownerSc.UserID, other.User.ID, true); err != nil {
		t.Fatal(err)
	}
	u, err := s.GetUserByID(other.User.ID)
	if err != nil || !u.PlatformAdmin {
		t.Fatalf("应授予平台管理员: %+v %v", u, err)
	}
	// 撤销自己 → ErrSelfAction
	if err := svc.AdminSetPlatformAdmin(ownerSc.UserID, ownerUID, false); !errors.Is(err, ErrSelfAction) {
		t.Fatalf("撤销自己应 ErrSelfAction: %v", err)
	}
	if err := svc.AdminSetPlatformAdmin(ownerSc.UserID, other.User.ID, false); err != nil {
		t.Fatalf("撤销第二位平台管理员应成功: %v", err)
	}
	// Service 层不能依赖“只能由本人调用”来保护最后管理员；后台运维 actor
	// 或未来服务账号尝试撤销/停用最后一位管理员也必须被拒绝。
	if err := svc.AdminSetPlatformAdmin("system-operator", ownerUID, false); !errors.Is(err, ErrLastPlatformAdmin) {
		t.Fatalf("撤销最后平台管理员应 ErrLastPlatformAdmin: %v", err)
	}
	if err := svc.AdminSetUserStatus("system-operator", ownerUID, UserStatusDisabled); !errors.Is(err, ErrLastPlatformAdmin) {
		t.Fatalf("停用最后平台管理员应 ErrLastPlatformAdmin: %v", err)
	}
}

// ── 租户成员管理（owner/admin 侧）──────────────────────────────────

func TestMembersAddNewAndReuseExistingCrossTenantUser(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	ownerSc, _ := setupOwner(t, svc)

	// 新邮箱 → 建号 + 加成员（analyst 可登录）
	if err := svc.MembersAdd(ownerSc, "new@example.com", "新成员", testPassword, domain.RoleAnalyst); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Login(context.Background(), "new@example.com", testPassword, "", "")
	if err != nil {
		t.Fatalf("新成员应可登录: %v", err)
	}
	if len(res.Roles) != 1 || res.Roles[0] != domain.RoleAnalyst {
		t.Fatalf("新成员角色应为 analyst: %v", res.Roles)
	}

	// 邮箱已存在（另一租户用户）→ 复用全局身份并新增 membership；无需再次传密码。
	other, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "B公司", DisplayName: "周九", Email: "zhou@example.com", Password: testPassword,
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.MembersAdd(ownerSc, "zhou@example.com", "", "", domain.RoleReviewer); err != nil {
		t.Fatalf("已有用户加入第二租户应成功: %v", err)
	}
	existing, _ := s.GetUserByEmailNorm("zhou@example.com")
	if m, err := s.GetMembership(ownerSc.TenantID, existing.ID); err != nil || m.Role != domain.RoleReviewer {
		t.Fatalf("应创建第二租户 reviewer membership: %+v %v", m, err)
	}
	// 原登录会话仍在 B；显式切换后进入 A，工作空间列表含两项。
	switched, err := svc.SwitchTenant(other.TokenRaw, ownerSc.TenantID)
	if err != nil {
		t.Fatalf("切换到新增租户应成功: %v", err)
	}
	if switched.Tenant.ID != ownerSc.TenantID || len(switched.Workspaces) != 2 {
		t.Fatalf("切换结果异常: tenant=%s workspaces=%+v", switched.Tenant.ID, switched.Workspaces)
	}
	intent, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "C公司", DisplayName: "已存在用户", Email: "intent@example.com", Password: testPassword,
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.MembersAdd(ownerSc, "intent@example.com", "误选新建", testPassword, domain.RoleAnalyst); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("新建模式遇到已有邮箱应明确冲突，不能静默复用: %v", err)
	}
	if _, err := s.GetMembership(ownerSc.TenantID, intent.User.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("新建模式冲突后不应写入 membership: %v", err)
	}

	// 重复添加 → ErrMemberExists
	if err := svc.MembersAdd(ownerSc, "new@example.com", "X", testPassword, domain.RoleAnalyst); !errors.Is(err, ErrMemberExists) {
		t.Fatalf("重复成员应 ErrMemberExists: %v", err)
	}
	// 弱密码 → ErrInvalidPassword
	if err := svc.MembersAdd(ownerSc, "y@example.com", "Y", "weak", domain.RoleAnalyst); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("弱密码应 ErrInvalidPassword: %v", err)
	}
}

// 多租户用户的首个工作空间被停用后，登录必须跳过它并落到仍可用的租户，
// 不能因为最早 membership 失效而把整个全局账号锁死。
func TestLoginSkipsDisabledTenantMembership(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	firstScope, _ := setupOwner(t, svc)

	second, err := svc.AdminCreateTenant(context.Background(), firstScope.UserID, AdminCreateTenantRequest{
		Name:       "第二工作空间",
		Slug:       "second-workspace",
		OwnerEmail: "owner@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateTenantStatus(firstScope.TenantID, TenantStatusDisabled); err != nil {
		t.Fatal(err)
	}

	res, err := svc.Login(context.Background(), "owner@example.com", testPassword, "", "")
	if err != nil {
		t.Fatalf("仍有可用工作空间时登录应成功: %v", err)
	}
	if res.Tenant.ID != second.ID {
		t.Fatalf("应跳过停用租户并进入 %s，实际 %s", second.ID, res.Tenant.ID)
	}
	if len(res.Workspaces) != 1 || res.Workspaces[0].TenantID != second.ID {
		t.Fatalf("工作空间列表不应包含停用租户: %+v", res.Workspaces)
	}
}

func TestMembersAddRoleGuard(t *testing.T) {
	svc := NewService(newTestStore(t))
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	ownerSc, _ := setupOwner(t, svc)
	adminSc := scopeFor(ownerSc.TenantID, "admin-u", domain.RoleAdmin)
	analystSc := scopeFor(ownerSc.TenantID, "analyst-u", domain.RoleAnalyst)

	// analyst 不能添加任何成员
	if err := svc.MembersAdd(analystSc, "x@example.com", "X", testPassword, domain.RoleAnalyst); !errors.Is(err, ErrForbidden) {
		t.Fatalf("analyst 添加应 ErrForbidden: %v", err)
	}
	// admin 不能加 owner/admin
	if err := svc.MembersAdd(adminSc, "o@example.com", "O", testPassword, domain.RoleOwner); !errors.Is(err, ErrForbidden) {
		t.Fatalf("admin 加 owner 应 ErrForbidden: %v", err)
	}
	if err := svc.MembersAdd(adminSc, "ad@example.com", "AD", testPassword, domain.RoleAdmin); !errors.Is(err, ErrForbidden) {
		t.Fatalf("admin 加 admin 应 ErrForbidden: %v", err)
	}
	// admin 可加 analyst/reviewer
	if err := svc.MembersAdd(adminSc, "an@example.com", "分析员", testPassword, domain.RoleAnalyst); err != nil {
		t.Fatalf("admin 加 analyst 应成功: %v", err)
	}
	// owner 可加 owner
	if err := svc.MembersAdd(ownerSc, "co@example.com", "联合owner", testPassword, domain.RoleOwner); err != nil {
		t.Fatalf("owner 加 owner 应成功: %v", err)
	}
	// 非法角色
	if err := svc.MembersAdd(ownerSc, "z@example.com", "Z", testPassword, "superuser"); err == nil {
		t.Fatal("非法角色应报错")
	}
}

func TestMembersSetRoleAndRemove(t *testing.T) {
	svc := NewService(newTestStore(t))
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	ownerSc, _ := setupOwner(t, svc)
	adminSc := scopeFor(ownerSc.TenantID, "admin-u", domain.RoleAdmin)

	// 添加两个 analyst，一个 admin
	for _, m := range []struct {
		email string
		role  string
	}{
		{"an1@example.com", domain.RoleAnalyst},
		{"an2@example.com", domain.RoleAnalyst},
		{"ad@example.com", domain.RoleAdmin},
	} {
		if err := svc.MembersAdd(ownerSc, m.email, m.email, testPassword, m.role); err != nil {
			t.Fatal(err)
		}
	}
	an1, _ := svc.store.GetUserByEmailNorm("an1@example.com")
	an2, _ := svc.store.GetUserByEmailNorm("an2@example.com")
	ad, _ := svc.store.GetUserByEmailNorm("ad@example.com")

	// admin 改 analyst 角色 → 成功
	if err := svc.MembersSetRole(adminSc, an1.ID, domain.RoleReviewer); err != nil {
		t.Fatalf("admin 改 analyst 角色应成功: %v", err)
	}
	if m, _ := svc.store.GetMembership(ownerSc.TenantID, an1.ID); m.Role != domain.RoleReviewer {
		t.Fatalf("角色应为 reviewer: %s", m.Role)
	}
	// admin 提升 reviewer 为 admin → ErrForbidden（admin 不能授予 admin，仅 owner 可）
	if err := svc.MembersSetRole(adminSc, an1.ID, domain.RoleAdmin); !errors.Is(err, ErrForbidden) {
		t.Fatalf("admin 提升他人为 admin 应 ErrForbidden: %v", err)
	}
	// owner 可提升到 admin
	if err := svc.MembersSetRole(ownerSc, an1.ID, domain.RoleAdmin); err != nil {
		t.Fatalf("owner 提升到 admin 应成功: %v", err)
	}
	// admin 改另一位 admin 的角色 → ErrForbidden（admin 不能动 admin）
	if err := svc.MembersSetRole(adminSc, ad.ID, domain.RoleAnalyst); !errors.Is(err, ErrForbidden) {
		t.Fatalf("admin 动 admin 应 ErrForbidden: %v", err)
	}
	// admin 移除 owner → ErrForbidden
	if err := svc.MembersRemove(adminSc, ownerSc.UserID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("admin 移除 owner 应 ErrForbidden: %v", err)
	}
	// owner 移除成员 → 成功；成员登录失效
	if err := svc.MembersRemove(ownerSc, an2.ID); err != nil {
		t.Fatalf("owner 移除成员应成功: %v", err)
	}
	if _, err := svc.store.GetMembership(ownerSc.TenantID, an2.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("移除后成员关系应不存在: %v", err)
	}
}

func TestMembersLastOwnerGuard(t *testing.T) {
	svc := NewService(newTestStore(t))
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	ownerSc, ownerUID := setupOwner(t, svc)

	// 唯一 owner 移除自己 → ErrSelfAction
	if err := svc.MembersRemove(ownerSc, ownerUID); !errors.Is(err, ErrSelfAction) {
		t.Fatalf("移除自己应 ErrSelfAction: %v", err)
	}
	// 唯一 owner 降级自己 → ErrLastOwner
	if err := svc.MembersSetRole(ownerSc, ownerUID, domain.RoleAnalyst); !errors.Is(err, ErrLastOwner) {
		t.Fatalf("降级唯一 owner 应 ErrLastOwner: %v", err)
	}
	// 增加第二个 owner 后可降级自己
	if err := svc.MembersAdd(ownerSc, "co@example.com", "联合owner", testPassword, domain.RoleOwner); err != nil {
		t.Fatal(err)
	}
	if err := svc.MembersSetRole(ownerSc, ownerUID, domain.RoleAnalyst); err != nil {
		t.Fatalf("有第二 owner 时降级自己应成功: %v", err)
	}
}

func TestDisabledOwnerDoesNotSatisfyLastOwnerGuard(t *testing.T) {
	svc := NewService(newTestStore(t))
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	ownerSc, ownerUID := setupOwner(t, svc)
	if err := svc.MembersAdd(ownerSc, "co-disabled@example.com", "停用所有者", testPassword, domain.RoleOwner); err != nil {
		t.Fatal(err)
	}
	co, err := svc.store.GetUserByEmailNorm("co-disabled@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AdminSetUserStatus(ownerUID, co.ID, UserStatusDisabled); err != nil {
		t.Fatal(err)
	}

	platformScope := scopeFor(ownerSc.TenantID, "platform-operator", domain.RoleOwner)
	if err := svc.MembersSetRole(platformScope, ownerUID, domain.RoleAnalyst); !errors.Is(err, ErrLastOwner) {
		t.Fatalf("停用 owner 不应允许最后可用 owner 降级: %v", err)
	}
	if err := svc.MembersRemove(platformScope, ownerUID); !errors.Is(err, ErrLastOwner) {
		t.Fatalf("停用 owner 不应允许最后可用 owner 被移除: %v", err)
	}
}

func TestMembersListScope(t *testing.T) {
	svc := NewService(newTestStore(t))
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	ownerSc, _ := setupOwner(t, svc)
	otherSc := scopeFor("other-tenant", "u", domain.RoleOwner)
	if err := svc.MembersAdd(ownerSc, "m@example.com", "成员", testPassword, domain.RoleAnalyst); err != nil {
		t.Fatal(err)
	}

	list, err := svc.MembersList(ownerSc)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("本租户应 2 个成员，got %d", len(list))
	}
	listOther, err := svc.MembersList(otherSc)
	if err != nil {
		t.Fatal(err)
	}
	if len(listOther) != 0 {
		t.Fatalf("他租户 scope 应看到 0 个成员，got %d", len(listOther))
	}
	// 无效 scope → ErrForbidden
	if _, err := svc.MembersList(nil); !errors.Is(err, ErrForbidden) {
		t.Fatalf("空 scope 应 ErrForbidden: %v", err)
	}
}

func TestAdminUserTenantCRUD(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	actorScope, _ := setupOwner(t, svc)

	// 新建租户，复用 actor 身份作为新租户 owner（形成多 membership）。
	tenant, err := svc.AdminCreateTenant(context.Background(), actorScope.UserID, AdminCreateTenantRequest{
		Name: "新租户", Slug: "new-tenant", OwnerEmail: "owner@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tenant.Status != TenantStatusActive {
		t.Fatalf("新租户应 active: %+v", tenant)
	}
	if ms, _ := s.ListMembershipsByUser(actorScope.UserID); len(ms) != 2 {
		t.Fatalf("复用 owner 后应有 2 个 membership: %+v", ms)
	}
	if err := svc.AdminUpdateTenant(actorScope.UserID, tenant.ID, "新租户二", "new-tenant-2"); err != nil {
		t.Fatal(err)
	}
	gotTenant, _ := s.GetTenant(tenant.ID)
	if gotTenant.Name != "新租户二" || gotTenant.Slug != "new-tenant-2" {
		t.Fatalf("租户更新未生效: %+v", gotTenant)
	}

	// 平台创建用户并加入目标租户，再修改资料和角色。
	u, err := svc.AdminCreateUser(actorScope.UserID, AdminCreateUserRequest{
		Email: "crud-user@example.com", DisplayName: "CRUD 用户", Password: testPassword,
		TenantID: tenant.ID, Role: domain.RoleAnalyst,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AdminUpdateUser(actorScope.UserID, u.ID, "crud-user-2@example.com", "更新用户"); err != nil {
		t.Fatal(err)
	}
	if err := svc.AdminTenantMemberSetRole(actorScope.UserID, tenant.ID, u.ID, domain.RoleReviewer); err != nil {
		t.Fatal(err)
	}
	if m, _ := s.GetMembership(tenant.ID, u.ID); m.Role != domain.RoleReviewer {
		t.Fatalf("成员角色未更新: %+v", m)
	}
	if err := svc.AdminDeleteUser(actorScope.UserID, u.ID); err != nil {
		t.Fatal(err)
	}
	if disabled, _ := s.GetUserByID(u.ID); disabled.Status != UserStatusDisabled {
		t.Fatalf("用户删除应为软停用: %+v", disabled)
	}

	// 租户停用/启用/软删除；当前 actor 所在原租户不受影响。
	if err := svc.AdminSetTenantStatus(context.Background(), actorScope.UserID, actorScope.TenantID,
		tenant.ID, TenantStatusDisabled); err != nil {
		t.Fatal(err)
	}
	if err := svc.AdminSetTenantStatus(context.Background(), actorScope.UserID, actorScope.TenantID,
		tenant.ID, TenantStatusActive); err != nil {
		t.Fatal(err)
	}
	if err := svc.AdminDeleteTenant(actorScope.UserID, actorScope.TenantID, tenant.ID); err != nil {
		t.Fatal(err)
	}
	if deleted, _ := s.GetTenant(tenant.ID); deleted.Status != TenantStatusDeleted {
		t.Fatalf("租户删除应保留记录并标 deleted: %+v", deleted)
	}
}

func TestDeletedTenantDoesNotBlockUserDisableOrAllowMembershipWrites(t *testing.T) {
	svc := NewService(newTestStore(t))
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	actorScope, _ := setupOwner(t, svc)
	deleted, err := svc.AdminCreateTenant(context.Background(), actorScope.UserID, AdminCreateTenantRequest{
		Name: "待删除租户", Slug: "deleted-tenant", OwnerEmail: "deleted-owner@example.com",
		OwnerDisplayName: "待删除所有者", OwnerPassword: testPassword,
	})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := svc.store.GetUserByEmailNorm("deleted-owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AdminDeleteTenant(actorScope.UserID, actorScope.TenantID, deleted.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.AdminTenantMemberSetRole(actorScope.UserID, deleted.ID, owner.ID, domain.RoleAnalyst); !errors.Is(err, ErrTenantDeleted) {
		t.Fatalf("已删除租户不应再改成员: %v", err)
	}
	if err := svc.AdminTenantMemberRemove(actorScope.UserID, deleted.ID, owner.ID); !errors.Is(err, ErrTenantDeleted) {
		t.Fatalf("已删除租户不应再移除成员: %v", err)
	}
	if err := svc.AdminSetUserStatus(actorScope.UserID, owner.ID, UserStatusDisabled); err != nil {
		t.Fatalf("终态租户的名义 owner 不应阻止用户停用: %v", err)
	}
}
