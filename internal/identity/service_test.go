package identity

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"customer-demand-agent/internal/domain"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "identity.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

const testPassword = "password-123456"

// TestRegisterFlow 注册：user+tenant+owner 原子落库 + 签发会话 + owner 角色。
func TestRegisterFlow(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })

	res, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "某集团", DisplayName: "张三", Email: "zhangsan@example.com", Password: testPassword,
	}, "127.0.0.1", "go-test")
	if err != nil {
		t.Fatal(err)
	}
	if res.User == nil || res.Tenant == nil {
		t.Fatal("注册应返回 user+tenant")
	}
	if res.Tenant.Status != TenantStatusActive {
		t.Fatalf("tenant 应 active: %s", res.Tenant.Status)
	}
	if len(res.Roles) != 1 || res.Roles[0] != domain.RoleOwner {
		t.Fatalf("角色应为 owner: %v", res.Roles)
	}
	if res.TokenRaw == "" || res.CSRFRaw == "" {
		t.Fatal("应签发 Token+CSRF")
	}

	// 会话可校验
	as, u, ten, m, err := svc.ValidateSession(res.TokenRaw)
	if err != nil {
		t.Fatal(err)
	}
	if as.UserID != res.User.ID || u.ID != res.User.ID || ten.ID != res.Tenant.ID || m.Role != domain.RoleOwner {
		t.Fatalf("会话上下文不一致: as=%s u=%s ten=%s m.role=%s", as.UserID, u.ID, ten.ID, m.Role)
	}

	// CurrentTenantScope 不可伪造
	sc, err := svc.CurrentTenantScope(res.TokenRaw)
	if err != nil {
		t.Fatal(err)
	}
	if sc.TenantID != res.Tenant.ID || sc.UserID != res.User.ID || !sc.HasRole(domain.RoleOwner) || !sc.Valid() {
		t.Fatalf("scope 异常: %+v", sc)
	}
	if sc.Source != "http" {
		t.Fatalf("scope source 应为 http: %s", sc.Source)
	}
}

// TestRegisterRejectsDuplicateOrgName §4.1 严重缺陷修复：同名租户禁止注册（服务层哨兵错误）。
func TestRegisterRejectsDuplicateOrgName(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })

	if _, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "某集团", DisplayName: "张三", Email: "dup-a@example.com", Password: testPassword,
	}, "127.0.0.1", "go-test"); err != nil {
		t.Fatal(err)
	}
	// 不同邮箱、同一租户名 → ErrTenantNameTaken
	if _, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "某集团", DisplayName: "李四", Email: "dup-b@example.com", Password: testPassword,
	}, "127.0.0.1", "go-test"); !errors.Is(err, ErrTenantNameTaken) {
		t.Fatalf("应 ErrTenantNameTaken, got %v", err)
	}
	// 空组织名 → ErrInvalidOrgName
	if _, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "  ", DisplayName: "王五", Email: "dup-c@example.com", Password: testPassword,
	}, "127.0.0.1", "go-test"); !errors.Is(err, ErrInvalidOrgName) {
		t.Fatalf("应 ErrInvalidOrgName, got %v", err)
	}
}

// TestRegisterUniqueSlugOnCollision makeSlug 对纯中文名回退固定 "tenant"，
// 与英文名归一化结果冲突时追加「前 8 位租户 ID」消歧。
func TestRegisterUniqueSlugOnCollision(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })

	r1, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "某集团", DisplayName: "甲", Email: "slg-a@example.com", Password: testPassword,
	}, "127.0.0.1", "go-test")
	if err != nil {
		t.Fatal(err)
	}
	if r1.Tenant.Slug != "tenant" {
		t.Fatalf("纯中文名 slug 应回退 tenant, got %s", r1.Tenant.Slug)
	}
	// 英文名 "Tenant" 归一化后同为 "tenant" → 冲突 → 带 ID 后缀
	r2, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "Tenant", DisplayName: "乙", Email: "slg-b@example.com", Password: testPassword,
	}, "127.0.0.1", "go-test")
	if err != nil {
		t.Fatal(err)
	}
	if want := "tenant-" + r2.Tenant.ID[:8]; r2.Tenant.Slug != want {
		t.Fatalf("冲突 slug 应消歧为 %s, got %s", want, r2.Tenant.Slug)
	}
}

// TestRegisterValidations 注册输入校验：重复邮箱/非法邮箱/弱密码。
func TestRegisterValidations(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })

	// 非法邮箱
	if _, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "A", DisplayName: "B", Email: "not-an-email", Password: testPassword,
	}, "", ""); !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("非法邮箱应 ErrInvalidEmail: %v", err)
	}
	// 弱密码
	if _, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "A", DisplayName: "B", Email: "x@example.com", Password: "short",
	}, "", ""); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("弱密码应 ErrInvalidPassword: %v", err)
	}
	// 成功注册一次
	if _, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "A", DisplayName: "B", Email: "dup@example.com", Password: testPassword,
	}, "", ""); err != nil {
		t.Fatal(err)
	}
	// 重复邮箱（大小写/空格规范化）
	if _, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "A2", DisplayName: "B2", Email: "  DUP@Example.COM ", Password: testPassword,
	}, "", ""); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("重复邮箱应 ErrEmailTaken: %v", err)
	}
}

// TestRegisterProvisionFailure 数据面初始化失败 → provisioning_failed + ErrProvisionFailed。
func TestRegisterProvisionFailure(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error {
		return errors.New("磁盘写满")
	})
	_, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "P", DisplayName: "P", Email: "p@example.com", Password: testPassword,
	}, "", "")
	if !errors.Is(err, ErrProvisionFailed) {
		t.Fatalf("应 ErrProvisionFailed: %v", err)
	}
	// 半成品租户标记失败、不 active
	u, _ := s.GetUserByEmailNorm("p@example.com")
	ms, _ := s.ListMembershipsByUser(u.ID)
	ten, _ := s.GetTenant(ms[0].TenantID)
	if ten.Status != TenantStatusProvisioningFailed {
		t.Fatalf("失败租户应为 provisioning_failed: %s", ten.Status)
	}
}

// TestResumeProvisionAfterFailure P0-09 验收：注册 provision 失败 → 邮箱被占、
// 但同邮箱可真正重试——登录（provisioning_failed，带失败标记）→ resume 重跑
// provisioner → 租户 active → 会话可用、再次登录无标记。故障注入磁盘满/恢复。
func TestResumeProvisionAfterFailure(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	fail := true
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error {
		if fail {
			return errors.New("磁盘写满") // 故障注入
		}
		return nil
	})

	// 1. 注册失败（磁盘满）→ ErrProvisionFailed，租户半成品。
	if _, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "R", DisplayName: "R", Email: "resume@example.com", Password: testPassword,
	}, "", ""); !errors.Is(err, ErrProvisionFailed) {
		t.Fatalf("应 ErrProvisionFailed: %v", err)
	}
	// 2. 同邮箱无法重新注册（用户/租户/成员关系已提交，邮箱已占用）。
	if _, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "R2", DisplayName: "R2", Email: "resume@example.com", Password: testPassword,
	}, "", ""); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("重复邮箱应 ErrEmailTaken: %v", err)
	}
	// 3. 故障恢复前登录：允许，但标记 ProvisionFailed（前端据此进修复页）。
	fail = false
	res, err := svc.Login(context.Background(), "resume@example.com", testPassword, "", "")
	if err != nil {
		t.Fatalf("provisioning_failed 应可登录（进入修复页）: %v", err)
	}
	if !res.ProvisionFailed {
		t.Fatal("修复前登录应带 ProvisionFailed=true")
	}
	// 4. resume：重跑 provisioner → 租户 active，标记清除，CSRF 轮换。
	rr, err := svc.ResumeProvision(context.Background(), res.TokenRaw)
	if err != nil {
		t.Fatalf("resume 应成功: %v", err)
	}
	if rr.ProvisionFailed {
		t.Fatal("resume 成功后应清除 ProvisionFailed")
	}
	ten, _ := s.GetTenant(res.Tenant.ID)
	if ten.Status != TenantStatusActive {
		t.Fatalf("resume 后租户应 active: %s", ten.Status)
	}
	// 5. 原会话仍有效（resume 复用会话，未新签发），旧 CSRF 失效、新 CSRF 有效。
	if _, _, _, _, err := svc.ValidateSession(res.TokenRaw); err != nil {
		t.Fatalf("resume 后会话应可用: %v", err)
	}
	if svc.VerifyCSRF(res.TokenRaw, res.CSRFRaw) {
		t.Fatal("resume 后旧 CSRF 应失效")
	}
	if !svc.VerifyCSRF(res.TokenRaw, rr.CSRFRaw) {
		t.Fatal("resume 返回的新 CSRF 应有效")
	}
	// 6. 再次登录：正常，无失败标记。
	res2, err := svc.Login(context.Background(), "resume@example.com", testPassword, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if res2.ProvisionFailed {
		t.Fatal("修复后登录不应再有 ProvisionFailed")
	}
}

// TestResumeProvisionIdempotent 已 active 租户调 resume 幂等成功（不重跑 provisioner）。
func TestResumeProvisionIdempotent(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	calls := 0
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error {
		calls++
		return nil
	})
	res, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "I", DisplayName: "I", Email: "idem@example.com", Password: testPassword,
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	before := calls
	rr, err := svc.ResumeProvision(context.Background(), res.TokenRaw)
	if err != nil {
		t.Fatalf("active 租户 resume 应幂等成功: %v", err)
	}
	if rr.ProvisionFailed {
		t.Fatal("active 租户不应标记 ProvisionFailed")
	}
	if calls != before {
		t.Fatalf("active 租户 resume 不应重跑 provisioner: calls=%d before=%d", calls, before)
	}
}

// TestResumeProvisionStillFails 修复重试再次失败 → 保持 provisioning_failed，
// 再次登录仍是失败标记，可继续重试（不是死路）。
func TestResumeProvisionStillFails(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error {
		return errors.New("权限不足") // 故障注入：目录不可写
	})
	_, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "F", DisplayName: "F", Email: "fail2@example.com", Password: testPassword,
	}, "", "")
	if !errors.Is(err, ErrProvisionFailed) {
		t.Fatalf("应 ErrProvisionFailed: %v", err)
	}
	res, err := svc.Login(context.Background(), "fail2@example.com", testPassword, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResumeProvision(context.Background(), res.TokenRaw); !errors.Is(err, ErrProvisionFailed) {
		t.Fatalf("再次失败应 ErrProvisionFailed: %v", err)
	}
	ten, _ := s.GetTenant(res.Tenant.ID)
	if ten.Status != TenantStatusProvisioningFailed {
		t.Fatalf("再次失败后应保持 provisioning_failed: %s", ten.Status)
	}
	// 会话仍有效（还能再点重试）。
	if _, _, _, _, err := svc.ValidateSession(res.TokenRaw); err != nil {
		t.Fatalf("再次失败后会话应仍可重试: %v", err)
	}
}

// TestLoginFlow 登录：统一错误文案、会话轮换、状态校验。
func TestLoginFlow(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })

	res, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "L", DisplayName: "L", Email: "login@example.com", Password: testPassword,
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}

	// 错误密码 → 统一 ErrInvalidCredentials
	if _, err := svc.Login(context.Background(), "login@example.com", "wrong-password-1", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("错误密码应 ErrInvalidCredentials: %v", err)
	}
	// 未知邮箱 → 同样 ErrInvalidCredentials（防枚举）
	if _, err := svc.Login(context.Background(), "nobody@example.com", testPassword, "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("未知邮箱应 ErrInvalidCredentials: %v", err)
	}
	// 正确登录 → 新会话；旧会话被废止（防会话固定）
	res2, err := svc.Login(context.Background(), "login@example.com", testPassword, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if res2.TokenRaw == res.TokenRaw {
		t.Fatal("登录应轮换 Token")
	}
	if _, _, _, _, err := svc.ValidateSession(res.TokenRaw); err == nil {
		t.Fatal("旧会话应已废止")
	}
	if _, _, _, _, err := svc.ValidateSession(res2.TokenRaw); err != nil {
		t.Fatalf("新会话应有效: %v", err)
	}
}

// TestLogoutAndCSRF 退出废止会话 + CSRF 轮换/校验。
func TestLogoutAndCSRF(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	res, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "C", DisplayName: "C", Email: "csrf@example.com", Password: testPassword,
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}

	// CSRF 校验：正确 raw 通过
	if !svc.VerifyCSRF(res.TokenRaw, res.CSRFRaw) {
		t.Fatal("正确 CSRF 应通过")
	}
	if svc.VerifyCSRF(res.TokenRaw, "bad-token") {
		t.Fatal("错误 CSRF 不应通过")
	}
	// 轮换后旧 raw 失效、新 raw 有效
	newRaw, err := svc.RotateCSRF(res.TokenRaw)
	if err != nil {
		t.Fatal(err)
	}
	if svc.VerifyCSRF(res.TokenRaw, res.CSRFRaw) {
		t.Fatal("轮换后旧 CSRF 应失效")
	}
	if !svc.VerifyCSRF(res.TokenRaw, newRaw) {
		t.Fatal("轮换后新 CSRF 应有效")
	}

	// 退出：幂等 + 会话废止
	if err := svc.Logout(res.TokenRaw); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := svc.ValidateSession(res.TokenRaw); err == nil {
		t.Fatal("退出后会话应失效")
	}
	if err := svc.Logout(res.TokenRaw); err != nil {
		t.Fatal("重复退出应幂等成功")
	}
}

// TestChangePassword 改密：旧密码校验 + 废止其他会话（保留当前）。
func TestChangePassword(t *testing.T) {
	s := newTestStore(t)
	svc := NewService(s)
	svc.SetProvisioner(func(ctx context.Context, tenantID string) error { return nil })
	res, err := svc.Register(context.Background(), RegisterRequest{
		OrgName: "P", DisplayName: "P", Email: "pw@example.com", Password: testPassword,
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	// 错误旧密码
	if err := svc.ChangePassword(context.Background(), res.TokenRaw, "wrong-old-1", "new-password-1"); !errors.Is(err, ErrInvalidOldPassword) {
		t.Fatalf("错误旧密码应 ErrInvalidOldPassword: %v", err)
	}
	// 弱新密码
	if err := svc.ChangePassword(context.Background(), res.TokenRaw, testPassword, "x"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("弱新密码应 ErrInvalidPassword: %v", err)
	}
	// 正确改密：当前会话仍有效，旧密码不再可登录，新密码可登录
	if err := svc.ChangePassword(context.Background(), res.TokenRaw, testPassword, "new-password-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := svc.ValidateSession(res.TokenRaw); err != nil {
		t.Fatalf("当前会话应保留: %v", err)
	}
	if _, err := svc.Login(context.Background(), "pw@example.com", testPassword, "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("旧密码不应再可登录")
	}
	if _, err := svc.Login(context.Background(), "pw@example.com", "new-password-1", "", ""); err != nil {
		t.Fatalf("新密码应可登录: %v", err)
	}
}

// TestNormalizeValidateEmail 邮箱规范化与格式校验。
func TestNormalizeValidateEmail(t *testing.T) {
	if NormalizeEmail("  A@Example.COM ") != "a@example.com" {
		t.Fatal("规范化应小写去空格")
	}
	if !ValidateEmail("a@example.com") {
		t.Fatal("合法邮箱应通过")
	}
	if ValidateEmail("not-an-email") || ValidateEmail("") || ValidateEmail("@x.com") {
		t.Fatal("非法邮箱应拒绝")
	}
}

// TestValidatePasswordPolicy 密码策略（2026-08-19 修订）：8~16 字符 +
// 至少三种字符类别（大小写/数字/符号）。
func TestValidatePasswordPolicy(t *testing.T) {
	// 通过用例（≥3 类，长度 8~16）
	for _, pw := range []string{
		"password-123456", // 小写+数字+符号
		"Password123",     // 大写+小写+数字
		"Pass word 123",   // 三类以上含空格符号
		"12345678Aa",      // 数字+大小写
	} {
		if err := ValidatePassword(pw); err != nil {
			t.Errorf("应通过: %q err=%v", pw, err)
		}
	}
	// 拒绝用例
	for pw, why := range map[string]string{
		"short":      "过短",
		"abcdEFGH":   "仅两类（大小写）",
		"12345678":   "仅数字一类",
		"abcdefgh":   "仅小写一类",
		"ABCDEFG H":  "两类（大写+符号）",
		"abcdefghij": "十位但仅小写一类",
	} {
		if err := ValidatePassword(pw); err == nil {
			t.Errorf("应拒绝 %q（%s）", pw, why)
		}
	}
	// 边界长度：7 拒、8 通过（三类）、16 通过、17 拒
	if ValidatePassword("Aa1!234") == nil {
		t.Error("7 位应拒绝")
	}
	if ValidatePassword("Aa1!2345") != nil {
		t.Error("8 位三类应通过")
	}
	if ValidatePassword("Aa1!abcdEFGH1234") != nil {
		t.Error("16 位三类应通过")
	}
	if ValidatePassword("Aa1!abcdEFGH12345") == nil {
		t.Error("17 位应拒绝")
	}
}
