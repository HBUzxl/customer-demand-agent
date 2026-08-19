package identity

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"customer-demand-agent/internal/auth"
	"customer-demand-agent/internal/domain"
)

// 会话过期策略（设计文档 §7.3）：8 小时空闲过期、7 天绝对过期。
const (
	SessionIdleTimeout     = 8 * time.Hour
	SessionAbsoluteTimeout = 7 * 24 * time.Hour
)

// 密码策略（2026-08-19 修订）：8~16 字符，且至少包含
// 大写字母、小写字母、数字、符号中的三种。注册与改密统一走 ValidatePassword。
const (
	MinPasswordLen = 8
	MaxPasswordLen = 16
)

// 注册返回给客户端的错误语义（统一文案，避免账号枚举，§7.1/§7.2）。
var (
	ErrRegistrationDisabled = errors.New("当前未开放注册")
	ErrEmailTaken           = errors.New("该邮箱已注册")
	ErrTenantNameTaken      = errors.New("该工作空间名称已存在")
	ErrInvalidOrgName       = errors.New("组织/工作空间名称不能为空")
	ErrInvalidEmail         = errors.New("邮箱格式无效")
	ErrInvalidPassword      = errors.New("密码长度需为 8~16 字符，且至少包含大写字母、小写字母、数字、符号中的三种")
	ErrInvalidCredentials   = errors.New("邮箱或密码错误")
	ErrAccountUnavailable   = errors.New("账号或组织状态异常")
	ErrProvisionFailed      = errors.New("工作空间初始化失败")
	ErrInvalidOldPassword   = errors.New("旧密码错误")
	// 管理类错误（用户管理/成员管理，§4 权限矩阵）。
	ErrForbidden          = errors.New("没有权限执行该操作")
	ErrLastOwner          = errors.New("租户至少需要保留一位所有者")
	ErrLastPlatformAdmin  = errors.New("平台至少需要保留一位平台管理员")
	ErrSelfAction         = errors.New("不能对自己执行该操作")
	ErrMemberExists       = errors.New("该用户已是当前租户成员")
	ErrInvalidSlug        = errors.New("slug 只能包含小写字母、数字和单个连字符，长度不超过 40")
	ErrTenantDeleted      = errors.New("已删除的租户不能再修改或恢复")
	ErrInvalidDisplayName = errors.New("姓名不能为空")
	ErrTenantRequired     = errors.New("必须选择所属租户")
	ErrInvalidStatus      = errors.New("状态值无效")
	ErrInvalidRole        = errors.New("成员角色无效")
)

// RegisterRequest 注册入参（不允许传角色/tenant_id/状态/文件路径，§10.1）。
type RegisterRequest struct {
	OrgName     string
	DisplayName string
	Email       string
	Password    string
}

// MembershipView 用户在某个租户内的成员关系（平台用户视图用）。
type MembershipView struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	Slug       string `json:"slug"`
	Role       string `json:"role"`
	Status     string `json:"status"`
}

// WorkspaceView 是当前用户可切换的工作空间。只返回 active membership，租户
// 状态可为 active 或 provisioning_failed（后者用于进入修复页）。
type WorkspaceView struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	Slug       string `json:"slug"`
	Role       string `json:"role"`
	Status     string `json:"status"`
}

// AdminUserView 平台管理用户视图（不含密码 Hash/Token）。
type AdminUserView struct {
	UserID        string           `json:"user_id"`
	Email         string           `json:"email"`
	DisplayName   string           `json:"display_name"`
	Status        string           `json:"status"`
	PlatformAdmin bool             `json:"platform_admin"`
	LastLoginAt   *time.Time       `json:"last_login_at,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	Memberships   []MembershipView `json:"memberships"`
}

// AdminTenantView 平台管理租户视图。
type AdminTenantView struct {
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Status      string    `json:"status"`
	MemberCount int       `json:"member_count"`
	CreatedBy   string    `json:"created_by,omitempty"` // 创建人邮箱（无创建人留空）
	CreatedAt   time.Time `json:"created_at"`
}

// MemberView 租户成员视图（成员管理用）。
type MemberView struct {
	UserID      string     `json:"user_id"`
	Email       string     `json:"email"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// AuthResult 认证成功后的最小视图（不返回密码 Hash/Token Hash/内部目录）。
type AuthResult struct {
	User       *User
	Tenant     *Tenant
	Roles      []string
	Workspaces []WorkspaceView
	TokenRaw   string // 原始登录 Token，只在本响应出现一次（写入 Cookie）
	CSRFRaw    string // 原始 CSRF Token，前端内存持有，修改请求带 X-CSRF-Token
	// ProvisionFailed 会话所属租户数据面初始化失败（provisioning_failed）。
	// 允许登录但需先经 /api/auth/resume 修复；业务路由在数据面闭合前不可用。
	ProvisionFailed bool
}

// AdminCreateUserRequest 平台创建用户并直接加入一个租户。身份系统不创建“无租户
// 用户”，因为这类账号无法形成有效 TenantScope，也无法登录。
type AdminCreateUserRequest struct {
	Email         string
	DisplayName   string
	Password      string
	TenantID      string
	Role          string
	PlatformAdmin bool
}

// AdminCreateTenantRequest 平台创建租户，并指定已有或新建 owner。
type AdminCreateTenantRequest struct {
	Name             string
	Slug             string
	OwnerEmail       string
	OwnerDisplayName string
	OwnerPassword    string
}

// Service 身份控制面：注册/登录/会话校验/改密。依赖 Provisioner 与审计回调
// 通过注入解决（避免 identity→tenancy/audit 反向依赖）。
type Service struct {
	store       *Store
	provisioner func(ctx context.Context, tenantID string) error
	auditFn     func(tenantID, actorUserID, action, objType, objID, result string)
}

// NewService 创建身份服务。
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// Store 暴露底层存储（-setup/测试/运维工具直接建成员角色等控制面操作）。
func (s *Service) Store() *Store { return s.store }

// SetProvisioner 注入租户数据面初始化回调（main 装配 tenancy.Provisioner）。
func (s *Service) SetProvisioner(fn func(ctx context.Context, tenantID string) error) {
	s.provisioner = fn
}

// SetAuditLogger 注入安全审计回调（main 装配 audit.Service）。
func (s *Service) SetAuditLogger(fn func(tenantID, actorUserID, action, objType, objID, result string)) {
	s.auditFn = fn
}

func (s *Service) audit(tenantID, actorUserID, action, objType, objID, result string) {
	if s.auditFn != nil {
		s.auditFn(tenantID, actorUserID, action, objType, objID, result)
	}
}

// NormalizeEmail 规范化邮箱用于唯一性判断（小写 + 去空格）。
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail 校验邮箱格式（net/mail 宽松校验）。
func ValidateEmail(email string) bool {
	norm := NormalizeEmail(email)
	if len(norm) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(norm)
	return err == nil && addr.Address == norm
}

// NormalizeSlug 规范化租户展示 slug；空值由调用方基于租户名生成。
func NormalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

// ValidateSlug 限制 slug 为路径安全的展示标识。TenantID 才是数据面主键，slug
// 不参与目录拼接或鉴权。
func ValidateSlug(slug string) bool {
	if slug == "" || len(slug) > 40 || slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	lastDash := false
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			lastDash = false
		case r == '-' && !lastDash:
			lastDash = true
		default:
			return false
		}
	}
	return true
}

// ValidatePassword 校验密码策略：8~16 字符，且至少包含大小写字母/数字/符号
// 中的三种。注册与改密共用（避免前后端规则漂移）。
func ValidatePassword(pw string) error {
	n := len([]rune(pw))
	if n < MinPasswordLen || n > MaxPasswordLen {
		return ErrInvalidPassword
	}
	var lower, upper, digit, sym bool
	for _, r := range pw {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		default:
			sym = true
		}
	}
	classes := 0
	for _, ok := range []bool{lower, upper, digit, sym} {
		if ok {
			classes++
		}
	}
	if classes < 3 {
		return ErrInvalidPassword
	}
	return nil
}

// Register 注册：原子建 user+tenant(provisioning)+membership(owner) → Provisioner
// 初始化租户数据面 → tenant active → 签发登录会话。失败不留半成品租户（§7.1）。
func (s *Service) Register(ctx context.Context, req RegisterRequest, ip, ua string) (*AuthResult, error) {
	now := time.Now().UTC()

	if !ValidateEmail(req.Email) {
		return nil, ErrInvalidEmail
	}
	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}
	norm := NormalizeEmail(req.Email)
	if _, err := s.store.GetUserByEmailNorm(norm); err == nil {
		s.audit("", "", "register_failed", "user", "", "email_taken")
		return nil, ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("检查邮箱: %w", err)
	}

	// 租户名唯一（§4.1 严重缺陷修复，2026-08-19）：同名租户会导致展示与鉴权混淆。
	// 预检覆盖常规重复；并发竞态由 tenants.name 唯一索引兜底（insertTenantTx 的
	// UNIQUE 约束错误在下方映射回 ErrTenantNameTaken）。
	orgName := strings.TrimSpace(req.OrgName)
	if orgName == "" {
		return nil, ErrInvalidOrgName
	}
	if exists, err := s.store.TenantNameExists(orgName); err != nil {
		return nil, fmt.Errorf("检查租户名: %w", err)
	} else if exists {
		s.audit("", "", "register_failed", "tenant", "", "tenant_name_taken")
		return nil, ErrTenantNameTaken
	}

	// 生成身份与租户（事务：user+tenant+membership 原子落库）。
	userID, tenantID := auth.NewRandomHex(16), auth.NewRandomHex(16)
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码哈希: %w", err)
	}
	u := &User{
		ID: userID, EmailNorm: norm, DisplayName: req.DisplayName,
		PasswordHash: hash, Status: UserStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	t := &Tenant{
		ID: tenantID, Name: orgName, Slug: makeSlug(orgName),
		Status: TenantStatusProvisioning, CreatedBy: userID, CreatedAt: now, UpdatedAt: now,
	}
	// slug 唯一：纯中文名 makeSlug 回退为固定 "tenant"，英文名也可能撞车（大小写/连字符归一），
	// 冲突时追加租户 ID 前 8 位消歧（slug 仅展示/URL，不参与鉴权，§10）。
	if collides, err := s.store.SlugExists(t.Slug); err != nil {
		return nil, fmt.Errorf("检查 slug: %w", err)
	} else if collides {
		t.Slug = t.Slug + "-" + tenantID[:8]
	}
	m := &Membership{
		TenantID: tenantID, UserID: userID, Role: domain.RoleOwner,
		Status: MembershipStatusActive, CreatedAt: now,
	}
	tx, err := s.store.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启注册事务: %w", err)
	}
	// 复用 Store 方法在事务连接上执行：直接对 tx 执行 SQL。
	if err := insertUserTx(tx, u); err != nil {
		_ = tx.Rollback()
		s.audit("", "", "register_failed", "user", "", "tx")
		return nil, fmt.Errorf("创建用户: %w", err)
	}
	if err := insertTenantTx(tx, t); err != nil {
		_ = tx.Rollback()
		// 并发注册同名租户：唯一索引兜底 → 语义化为租户名已存在（§4.1）。
		if strings.Contains(err.Error(), "tenants.name") {
			s.audit("", "", "register_failed", "tenant", "", "tenant_name_taken")
			return nil, ErrTenantNameTaken
		}
		return nil, fmt.Errorf("创建租户: %w", err)
	}
	if err := insertMembershipTx(tx, m); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("创建成员关系: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交注册事务: %w", err)
	}

	// 初始化租户数据面（history.db + wiki 覆盖层 + tenant.json），失败则标记 provisioning_failed。
	if s.provisioner != nil {
		if err := s.provisioner(ctx, tenantID); err != nil {
			_ = s.store.UpdateTenantStatus(tenantID, TenantStatusProvisioningFailed)
			s.audit(tenantID, userID, "tenant_provision_failed", "tenant", tenantID, "error")
			return nil, fmt.Errorf("%w: %v", ErrProvisionFailed, err)
		}
	}
	if err := s.store.UpdateTenantStatus(tenantID, TenantStatusActive); err != nil {
		return nil, fmt.Errorf("激活租户: %w", err)
	}
	t.Status = TenantStatusActive // 同步响应视图
	s.audit(tenantID, userID, "register_success", "tenant", tenantID, "ok")

	// 签发登录会话（全新 Token + CSRF）。
	res, err := s.issueSession(ctx, u, t, ip, ua)
	if err != nil {
		return nil, err
	}
	res.Roles = []string{domain.RoleOwner}
	return res, nil
}

// Login 登录：统一文案 + 校验状态 + 会话轮换（防会话固定）。
func (s *Service) Login(ctx context.Context, email, password, ip, ua string) (*AuthResult, error) {
	norm := NormalizeEmail(email)
	u, err := s.store.GetUserByEmailNorm(norm)
	if errors.Is(err, ErrNotFound) {
		s.audit("", "", "login_failed", "user", "", "unknown_email")
		// 统一文案 + 恒定时间开销近似（仍走一次 hash 校验）
		_, _ = auth.VerifyPassword(fakeHash(), password)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户: %w", err)
	}
	ok, err := auth.VerifyPassword(u.PasswordHash, password)
	if err != nil || !ok {
		s.audit("", u.ID, "login_failed", "user", u.ID, "bad_password")
		return nil, ErrInvalidCredentials
	}
	if u.Status != UserStatusActive {
		s.audit("", u.ID, "login_failed", "user", u.ID, "disabled")
		return nil, ErrAccountUnavailable
	}
	// 密码参数低于基线 → 本次成功后重新 Hash（成本可升级）。
	if auth.NeedsRehash(u.PasswordHash) {
		if newHash, err := auth.HashPassword(password); err == nil {
			_ = s.store.UpdateUserPassword(u.ID, newHash)
		}
	}

	// 选取活跃租户：当前用户第一个 active membership。
	tenantID, role, err := s.activeMembership(u.ID)
	if err != nil {
		s.audit("", u.ID, "login_failed", "user", u.ID, "no_membership")
		return nil, ErrAccountUnavailable
	}
	// 租户状态校验：provisioning_failed 允许登录（进入修复页重试初始化，P0-09）；
	// 其余非 active 状态仍拒绝。
	ten, err := s.store.GetTenant(tenantID)
	if err != nil {
		s.audit(tenantID, u.ID, "login_failed", "tenant", tenantID, "missing")
		return nil, ErrAccountUnavailable
	}
	if ten.Status != TenantStatusActive && ten.Status != TenantStatusProvisioningFailed {
		s.audit(tenantID, u.ID, "login_failed", "tenant", tenantID, "not_active")
		return nil, ErrAccountUnavailable
	}

	// 废止旧会话（防会话固定；单活跃会话语义）。
	_ = s.store.RevokeUserSessions(u.ID, "")
	_ = s.store.UpdateUserLastLogin(u.ID, time.Now().UTC())
	s.audit(tenantID, u.ID, "login_success", "user", u.ID, "ok")

	res, err := s.issueSession(ctx, u, ten, ip, ua)
	if err != nil {
		return nil, err
	}
	res.Roles = s.rolesFor(u, role)
	res.ProvisionFailed = ten.Status == TenantStatusProvisioningFailed
	return res, nil
}

// rolesFor 合并成员角色与平台管理员标记（platform_admin 是用户级平台元数据权限，
// 与租户成员角色叠加，§4.2——不是"超级租户"，业务数据仍按租户隔离）。
func (s *Service) rolesFor(u *User, membershipRole string) []string {
	roles := []string{membershipRole}
	if u.PlatformAdmin {
		roles = append(roles, domain.RolePlatformAdmin)
	}
	return roles
}

// Logout 废止当前会话。
func (s *Service) Logout(tokenRaw string) error {
	as, _, _, _, err := s.validate(tokenRaw)
	if err != nil {
		return nil // 已失效也幂等成功
	}
	s.audit(as.ActiveTenantID, as.UserID, "logout", "session", as.ID, "ok")
	return s.store.RevokeSession(as.ID)
}

// ChangePassword 校验旧密码后改密，废止其他会话（保留当前）。
func (s *Service) ChangePassword(ctx context.Context, tokenRaw, oldPw, newPw string) error {
	as, u, _, _, err := s.validate(tokenRaw)
	if err != nil {
		return err
	}
	ok, err := auth.VerifyPassword(u.PasswordHash, oldPw)
	if err != nil || !ok {
		return ErrInvalidOldPassword
	}
	if err := ValidatePassword(newPw); err != nil {
		return err
	}
	hash, err := auth.HashPassword(newPw)
	if err != nil {
		return fmt.Errorf("密码哈希: %w", err)
	}
	if err := s.store.UpdateUserPassword(u.ID, hash); err != nil {
		return fmt.Errorf("更新密码: %w", err)
	}
	_ = s.store.RevokeUserSessions(u.ID, as.ID)
	s.audit(as.ActiveTenantID, u.ID, "password_change", "user", u.ID, "ok")
	return nil
}

// ValidateSession 校验登录态并返回完整上下文（Session/User/Tenant/Membership）。
func (s *Service) ValidateSession(tokenRaw string) (*AuthSession, *User, *Tenant, *Membership, error) {
	return s.validate(tokenRaw)
}

// CurrentTenantScope 从登录态构造不可伪造的 TenantScope（§9.1）。
func (s *Service) CurrentTenantScope(tokenRaw string) (*domain.TenantScope, error) {
	as, u, ten, m, err := s.validate(tokenRaw)
	if err != nil {
		return nil, err
	}
	return &domain.TenantScope{
		TenantID:    ten.ID,
		UserID:      as.UserID,
		DisplayName: u.DisplayName,
		Email:       u.EmailNorm,
		Roles:       s.rolesFor(u, m.Role),
		Source:      "http",
	}, nil
}

// GrantPlatformAdmin 授予/撤销平台管理员标记（-setup 引导；生产环境受审计操作）。
func (s *Service) GrantPlatformAdmin(userID string, on bool) error {
	return s.store.SetUserPlatformAdmin(userID, on)
}

// ResumeProvision 重试租户数据面初始化（P0-09）。注册时 provision 失败会把租户
// 标记为 provisioning_failed——用户/租户/成员关系已提交，邮箱已占用，但数据面
// 缺失。用户登录（带 ProvisionFailed 标记）后前端进修复页，调用本方法重跑
// provisioner 并把租户置回 active。已 active → 幂等成功。复用当前会话（Token
// 仍在 Cookie），轮换 CSRF 并返回。
func (s *Service) ResumeProvision(ctx context.Context, tokenRaw string) (*AuthResult, error) {
	as, u, ten, m, err := s.validate(tokenRaw)
	if err != nil {
		return nil, err
	}
	if ten.Status == TenantStatusProvisioningFailed {
		if s.provisioner == nil {
			s.audit(ten.ID, u.ID, "tenant_provision_failed", "tenant", ten.ID, "no_provisioner")
			return nil, ErrProvisionFailed
		}
		if err := s.provisioner(ctx, ten.ID); err != nil {
			s.audit(ten.ID, u.ID, "tenant_provision_failed", "tenant", ten.ID, "error")
			return nil, fmt.Errorf("%w: %v", ErrProvisionFailed, err)
		}
		if err := s.store.UpdateTenantStatus(ten.ID, TenantStatusActive); err != nil {
			return nil, fmt.Errorf("激活租户: %w", err)
		}
		ten.Status = TenantStatusActive
		s.audit(ten.ID, u.ID, "tenant_provision_resumed", "tenant", ten.ID, "ok")
	} else if ten.Status != TenantStatusActive {
		s.audit(ten.ID, u.ID, "resume_provision_failed", "tenant", ten.ID, ten.Status)
		return nil, fmt.Errorf("租户状态 %s 不可恢复", ten.Status)
	}
	res := s.sessionResult(as, u, ten, m)
	res.CSRFRaw, err = s.rotateCSRF(as.ID, tokenRaw)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// sessionResult 由已验证会话构造响应视图（复用现存会话，不新签发 Token）。
func (s *Service) sessionResult(as *AuthSession, u *User, ten *Tenant, m *Membership) *AuthResult {
	res := &AuthResult{User: u, Tenant: ten, Roles: s.rolesFor(u, m.Role)}
	res.Workspaces, _ = s.workspacesFor(u.ID)
	if ten.Status != TenantStatusActive {
		res.ProvisionFailed = true
	}
	return res
}

// CurrentCSRF 返回当前会话稳定的 CSRF Token。读取 /auth/me 不轮换，避免一个
// 浏览器标签页刷新后让同一登录会话的其他标签页全部失效。
func (s *Service) CurrentCSRF(tokenRaw string) (string, error) {
	as, _, _, _, err := s.validate(tokenRaw)
	if err != nil {
		return "", err
	}
	return auth.DeriveCSRF(tokenRaw, as.CSRFHash), nil
}

// RotateCSRF 轮换服务端 Secret 并返回新的派生 Token（租户切换/恢复时使用）。
func (s *Service) RotateCSRF(tokenRaw string) (string, error) {
	as, _, _, _, err := s.validate(tokenRaw)
	if err != nil {
		return "", err
	}
	return s.rotateCSRF(as.ID, tokenRaw)
}

// rotateCSRF 以会话 ID 轮换服务端 Secret（Resume 等复用现存会话的场景）。
func (s *Service) rotateCSRF(sessionID, tokenRaw string) (string, error) {
	secret := auth.NewRandomHex(32)
	if err := s.store.UpdateSessionCSRF(sessionID, secret); err != nil {
		return "", fmt.Errorf("更新 CSRF: %w", err)
	}
	return auth.DeriveCSRF(tokenRaw, secret), nil
}

// VerifyCSRF 校验 X-CSRF-Token 与当前会话派生值是否匹配（常量时间）。
func (s *Service) VerifyCSRF(tokenRaw, csrfRaw string) bool {
	as, _, _, _, err := s.validate(tokenRaw)
	if err != nil {
		return false
	}
	if auth.VerifyDerivedCSRF(tokenRaw, as.CSRFHash, csrfRaw) {
		return true
	}
	// 滚动升级兼容：旧版本会话的 csrf_hash 保存 raw Token 的 Hash。允许旧标签页
	// 在会话自然过期前继续提交；新签发会话使用不可逆派生值，不走到此分支。
	return auth.NewCSRF().Verify(as.CSRFHash, csrfRaw)
}

// validate 核心校验：token → 会话未吊销/未过期 → 用户有效 → 租户有效 → 成员关系有效。
// SQLite 负责并发读；last_seen 节流更新。最后 owner/platform_admin 等跨语句
// 管理约束由对应写方法在 Store 临界区内完成，不在普通会话校验路径加全局锁。
func (s *Service) validate(tokenRaw string) (*AuthSession, *User, *Tenant, *Membership, error) {
	if tokenRaw == "" {
		return nil, nil, nil, nil, errors.New("未登录")
	}
	as, err := s.store.GetAuthSessionByTokenHash(auth.TokenHash(tokenRaw))
	if errors.Is(err, ErrNotFound) {
		return nil, nil, nil, nil, errors.New("会话无效")
	}
	if err != nil {
		return nil, nil, nil, nil, err
	}
	now := time.Now().UTC()
	if as.RevokedAt != nil {
		return nil, nil, nil, nil, errors.New("会话已废止")
	}
	if now.After(as.ExpiresAt) {
		_ = s.store.RevokeSession(as.ID)
		return nil, nil, nil, nil, errors.New("会话已过期")
	}
	if now.Sub(as.LastSeenAt) > SessionIdleTimeout {
		_ = s.store.RevokeSession(as.ID)
		return nil, nil, nil, nil, errors.New("会话空闲过期")
	}
	u, err := s.store.GetUserByID(as.UserID)
	if err != nil || u.Status != UserStatusActive {
		return nil, nil, nil, nil, ErrAccountUnavailable
	}
	// P0-09：provisioning_failed 允许校验（登录/me/resume 都要用到会话），
	// 业务数据面在 tenancy 层按未初始化闭合（ForTenant → ErrNotProvisioned）。
	ten, err := s.store.GetTenant(as.ActiveTenantID)
	if err != nil || (ten.Status != TenantStatusActive && ten.Status != TenantStatusProvisioningFailed) {
		return nil, nil, nil, nil, ErrAccountUnavailable
	}
	m, err := s.store.GetMembership(as.ActiveTenantID, as.UserID)
	if err != nil || m.Status != MembershipStatusActive {
		return nil, nil, nil, nil, ErrAccountUnavailable
	}
	// last_seen 节流：至少 5 分钟才写一次库（§7.3）。
	if now.Sub(as.LastSeenAt) > 5*time.Minute {
		_ = s.store.TouchSession(as.ID, now)
	}
	return as, u, ten, m, nil
}

// issueSession 生成全新登录会话（Token 只存 Hash）+ CSRF。
func (s *Service) issueSession(ctx context.Context, u *User, tenant *Tenant, ip, ua string) (*AuthResult, error) {
	rawToken, tokenHash, err := auth.NewSessionToken()
	if err != nil {
		return nil, err
	}
	csrfSecret := auth.NewRandomHex(32)
	csrfRaw := auth.DeriveCSRF(rawToken, csrfSecret)
	now := time.Now().UTC()
	as := &AuthSession{
		ID: auth.NewRandomHex(16), TokenHash: tokenHash, UserID: u.ID,
		ActiveTenantID: tenant.ID, CSRFHash: csrfSecret,
		ExpiresAt: now.Add(SessionAbsoluteTimeout), LastSeenAt: now,
	}
	if err := s.store.CreateAuthSession(as); err != nil {
		return nil, fmt.Errorf("创建登录会话: %w", err)
	}
	workspaces, err := s.workspacesFor(u.ID)
	if err != nil {
		_ = s.store.RevokeSession(as.ID)
		return nil, fmt.Errorf("加载工作空间: %w", err)
	}
	return &AuthResult{
		User: u, Tenant: tenant, TokenRaw: rawToken, CSRFRaw: csrfRaw,
		Workspaces: workspaces,
	}, nil
}

// activeMembership 返回用户最早加入的 active 成员关系。多租户账号登录后可通过
// SwitchTenant 显式切换；确定性排序避免数据库返回顺序造成随机落入工作空间。
func (s *Service) activeMembership(userID string) (tenantID, role string, err error) {
	ms, err := s.store.ListMembershipsByUser(userID)
	if err != nil {
		return "", "", err
	}
	for _, m := range ms {
		if m.Status != MembershipStatusActive {
			continue
		}
		t, err := s.store.GetTenant(m.TenantID)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return "", "", err
		}
		if t.Status == TenantStatusActive || t.Status == TenantStatusProvisioningFailed {
			return m.TenantID, m.Role, nil
		}
	}
	return "", "", ErrNotFound
}

// workspacesFor 返回用户可进入的全部工作空间，不包含 disabled membership、已停用
// 或已删除租户。provisioning_failed 保留，以便 owner 进入修复页。
func (s *Service) workspacesFor(userID string) ([]WorkspaceView, error) {
	ms, err := s.store.ListMembershipsByUser(userID)
	if err != nil {
		return nil, err
	}
	out := make([]WorkspaceView, 0, len(ms))
	for _, m := range ms {
		if m.Status != MembershipStatusActive {
			continue
		}
		t, err := s.store.GetTenant(m.TenantID)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if t.Status != TenantStatusActive && t.Status != TenantStatusProvisioningFailed {
			continue
		}
		out = append(out, WorkspaceView{
			TenantID: t.ID, TenantName: t.Name, Slug: t.Slug, Role: m.Role, Status: t.Status,
		})
	}
	return out, nil
}

// WorkspacesForUser 暴露认证响应所需的安全工作空间视图。
func (s *Service) WorkspacesForUser(userID string) ([]WorkspaceView, error) {
	return s.workspacesFor(userID)
}

// SwitchTenant 把当前会话切换到用户另一个 active membership。tenantID 只由服务端
// 再次查询控制面确认，不能通过伪造请求进入未授权数据面；切换同时轮换 CSRF。
func (s *Service) SwitchTenant(tokenRaw, tenantID string) (*AuthResult, error) {
	as, u, _, _, err := s.validate(tokenRaw)
	if err != nil {
		return nil, err
	}
	m, err := s.store.GetMembership(tenantID, u.ID)
	if err != nil || m.Status != MembershipStatusActive {
		return nil, ErrForbidden
	}
	ten, err := s.store.GetTenant(tenantID)
	if err != nil {
		return nil, err
	}
	if ten.Status != TenantStatusActive && ten.Status != TenantStatusProvisioningFailed {
		return nil, ErrAccountUnavailable
	}
	csrfSecret := auth.NewRandomHex(32)
	csrfRaw := auth.DeriveCSRF(tokenRaw, csrfSecret)
	if err := s.store.UpdateSessionTenant(as.ID, tenantID, csrfSecret); err != nil {
		return nil, err
	}
	s.audit(tenantID, u.ID, "tenant_switch", "tenant", tenantID, "ok")
	res := s.sessionResult(as, u, ten, m)
	res.CSRFRaw = csrfRaw
	return res, nil
}

// makeSlug 从组织名生成展示用 slug（不做文件路径，仅展示/URL）。
func makeSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		default:
			// 跳过中文/标点
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "tenant"
	}
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

// fakeHash 登录失败时也执行一次哈希比较，使耗时近似、防时序侧信道（§7.2）。
func fakeHash() string {
	// 预计算一个合法 PHC（永不匹配任意明文，只用于统一耗时）。
	return "$argon2id$v=19$m=65536,t=3,p=2$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
}

// ── 平台管理（platform_admin 专用，§4.2 控制面）────────────────────────

// AdminUsers 平台管理：全量用户及各自成员关系视图（不含密码/Token）。
func (s *Service) AdminUsers() ([]AdminUserView, error) {
	users, err := s.store.ListUsers()
	if err != nil {
		return nil, err
	}
	tenants := map[string]Tenant{}
	if ts, err := s.store.ListTenants(); err == nil {
		for _, t := range ts {
			tenants[t.ID] = t
		}
	}
	views := make([]AdminUserView, 0, len(users))
	for i := range users {
		u := &users[i]
		v := AdminUserView{
			UserID: u.ID, Email: u.EmailNorm, DisplayName: u.DisplayName,
			Status: u.Status, PlatformAdmin: u.PlatformAdmin,
			LastLoginAt: u.LastLoginAt, CreatedAt: u.CreatedAt,
			Memberships: []MembershipView{}, // 恒为 []（非 null）：无成员用户前端不崩
		}
		ms, err := s.store.ListMembershipsByUser(u.ID)
		if err != nil {
			return nil, err
		}
		for _, m := range ms {
			mv := MembershipView{TenantID: m.TenantID, Role: m.Role, Status: m.Status}
			if ten, ok := tenants[m.TenantID]; ok {
				mv.TenantName, mv.Slug = ten.Name, ten.Slug
			}
			v.Memberships = append(v.Memberships, mv)
		}
		views = append(views, v)
	}
	return views, nil
}

// AdminTenants 平台管理：全量租户视图（含成员数、创建人邮箱）。
func (s *Service) AdminTenants() ([]AdminTenantView, error) {
	ts, err := s.store.ListTenants()
	if err != nil {
		return nil, err
	}
	userEmail := map[string]string{}
	if us, err := s.store.ListUsers(); err == nil {
		for _, u := range us {
			userEmail[u.ID] = u.EmailNorm
		}
	}
	views := make([]AdminTenantView, 0, len(ts))
	for i := range ts {
		t := &ts[i]
		ms, err := s.store.ListMembershipsByTenant(t.ID)
		if err != nil {
			return nil, err
		}
		memberCount := 0
		for _, m := range ms {
			if m.Status != MembershipStatusActive {
				continue
			}
			if u, err := s.store.GetUserByID(m.UserID); err == nil && u.Status == UserStatusActive {
				memberCount++
			}
		}
		views = append(views, AdminTenantView{
			TenantID: t.ID, Name: t.Name, Slug: t.Slug, Status: t.Status,
			MemberCount: memberCount, CreatedBy: userEmail[t.CreatedBy], CreatedAt: t.CreatedAt,
		})
	}
	return views, nil
}

// AdminTenantMembers 平台管理：某租户的全部成员（租户不存在 → ErrNotFound）。
func (s *Service) AdminTenantMembers(tenantID string) ([]MemberView, error) {
	if _, err := s.store.GetTenant(tenantID); err != nil {
		return nil, err
	}
	ms, err := s.store.ListMembershipsByTenant(tenantID)
	if err != nil {
		return nil, err
	}
	return s.memberViews(ms)
}

// AdminSetUserStatus 禁用/启用账号。禁用自己会被拒绝；禁用即撤销其全部会话。
func (s *Service) AdminSetUserStatus(actorUserID, userID, status string) error {
	if userID == actorUserID {
		return ErrSelfAction
	}
	if _, err := s.store.GetUserByID(userID); err != nil {
		return err
	}
	if status != UserStatusActive && status != UserStatusDisabled {
		return fmt.Errorf("%w: %s", ErrInvalidStatus, status)
	}
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	if status == UserStatusDisabled {
		if err := s.ensureUserCanBeDisabled(userID); err != nil {
			return err
		}
	}
	if err := s.store.UpdateUserStatus(userID, status); err != nil {
		return err
	}
	if status == UserStatusDisabled {
		_ = s.store.RevokeUserSessions(userID, "")
	}
	s.audit("", actorUserID, "admin_set_user_status", "user", userID, status)
	return nil
}

// AdminResetPassword 平台重置密码：校验策略 → 重哈希 → 撤销其会话强制重登。
func (s *Service) AdminResetPassword(actorUserID, userID, newPw string) error {
	if _, err := s.store.GetUserByID(userID); err != nil {
		return err
	}
	if err := ValidatePassword(newPw); err != nil {
		return err
	}
	hash, err := auth.HashPassword(newPw)
	if err != nil {
		return fmt.Errorf("密码哈希: %w", err)
	}
	if err := s.store.UpdateUserPassword(userID, hash); err != nil {
		return err
	}
	_ = s.store.RevokeUserSessions(userID, "")
	s.audit("", actorUserID, "admin_reset_password", "user", userID, "ok")
	return nil
}

// AdminSetPlatformAdmin 授予/撤销平台管理员。禁止撤销自己的平台管理员身份。
func (s *Service) AdminSetPlatformAdmin(actorUserID, userID string, on bool) error {
	if userID == actorUserID && !on {
		return ErrSelfAction
	}
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	u, err := s.store.GetUserByID(userID)
	if err != nil {
		return err
	}
	if !on && u.PlatformAdmin {
		n, err := s.store.CountPlatformAdmins()
		if err != nil {
			return err
		}
		if n <= 1 {
			return ErrLastPlatformAdmin
		}
	}
	if err := s.store.SetUserPlatformAdmin(userID, on); err != nil {
		return err
	}
	s.audit("", actorUserID, "admin_set_platform_admin", "user", userID, fmt.Sprintf("%t", on))
	return nil
}

// ensureUserCanBeDisabled 防止停用任一租户最后一位 owner，或最后一位平台管理员。
func (s *Service) ensureUserCanBeDisabled(userID string) error {
	u, err := s.store.GetUserByID(userID)
	if err != nil {
		return err
	}
	if u.PlatformAdmin {
		n, err := s.store.CountPlatformAdmins()
		if err != nil {
			return err
		}
		if n <= 1 {
			return ErrLastPlatformAdmin
		}
	}
	ms, err := s.store.ListMembershipsByUser(userID)
	if err != nil {
		return err
	}
	for _, m := range ms {
		if m.Status != MembershipStatusActive || m.Role != domain.RoleOwner {
			continue
		}
		t, err := s.store.GetTenant(m.TenantID)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if t.Status == TenantStatusDeleted || t.Status == TenantStatusDeleting {
			continue
		}
		n, err := s.ownerCount(m.TenantID)
		if err != nil {
			return err
		}
		if n <= 1 {
			return ErrLastOwner
		}
	}
	return nil
}

// AdminCreateUser 平台创建一个新用户并直接加入指定租户。user、membership 与
// platform_admin 标记在同一个事务中落库，避免并发下把其他流程创建的同邮箱用户
// 错当成本次结果，也避免“用户已创建但管理员标记失败”的半完成状态。
func (s *Service) AdminCreateUser(actorUserID string, req AdminCreateUserRequest) (*User, error) {
	if req.TenantID == "" {
		return nil, ErrTenantRequired
	}
	tenant, err := s.store.GetTenant(req.TenantID)
	if err != nil {
		return nil, err
	}
	if tenant.Status == TenantStatusDeleted || tenant.Status == TenantStatusDeleting {
		return nil, ErrTenantDeleted
	}
	if !ValidMemberRoles(req.Role) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRole, req.Role)
	}
	if !ValidateEmail(req.Email) {
		return nil, ErrInvalidEmail
	}
	name := strings.TrimSpace(req.DisplayName)
	if name == "" {
		return nil, ErrInvalidDisplayName
	}
	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码哈希: %w", err)
	}
	now := time.Now().UTC()
	u := &User{
		ID: auth.NewRandomHex(16), EmailNorm: NormalizeEmail(req.Email), DisplayName: name,
		PasswordHash: hash, Status: UserStatusActive, PlatformAdmin: req.PlatformAdmin,
		CreatedAt: now, UpdatedAt: now,
	}
	m := &Membership{
		TenantID: req.TenantID, UserID: u.ID, Role: req.Role,
		Status: MembershipStatusActive, CreatedAt: now,
	}
	if err := s.store.CreateUserAndMembership(u, m); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	s.audit(req.TenantID, actorUserID, "admin_create_user", "user", u.ID, req.Role)
	return u, nil
}

// AdminUpdateUser 修改用户全局姓名/登录邮箱。邮箱变更后既有会话仍然有效，下一次
// 登录使用新邮箱；租户管理员无权修改全局身份。
func (s *Service) AdminUpdateUser(actorUserID, userID, email, displayName string) error {
	if !ValidateEmail(email) {
		return ErrInvalidEmail
	}
	name := strings.TrimSpace(displayName)
	if name == "" {
		return ErrInvalidDisplayName
	}
	u, err := s.store.GetUserByID(userID)
	if err != nil {
		return err
	}
	norm := NormalizeEmail(email)
	if norm != u.EmailNorm {
		if other, err := s.store.GetUserByEmailNorm(norm); err == nil && other.ID != userID {
			return ErrEmailTaken
		} else if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	if err := s.store.UpdateUserProfile(userID, norm, name); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return ErrEmailTaken
		}
		return err
	}
	s.audit("", actorUserID, "admin_update_user", "user", userID, "ok")
	return nil
}

// AdminDeleteUser 使用停用语义实现可恢复删除：不硬删身份、成员关系、会话历史或
// 审计记录。最后 owner/最后平台管理员保护与禁用操作一致。
func (s *Service) AdminDeleteUser(actorUserID, userID string) error {
	if err := s.AdminSetUserStatus(actorUserID, userID, UserStatusDisabled); err != nil {
		return err
	}
	s.audit("", actorUserID, "admin_delete_user", "user", userID, "soft_deleted")
	return nil
}

// AdminCreateTenant 创建租户并指定 owner；owner 邮箱已存在则新增 membership，
// 不存在则连同账号原子创建。数据面 Provision 失败时保留 provisioning_failed，
// 可由平台或 owner 重试，避免静默产生半初始化的 active 租户。
func (s *Service) AdminCreateTenant(ctx context.Context, actorUserID string, req AdminCreateTenantRequest) (*Tenant, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrInvalidOrgName
	}
	if exists, err := s.store.TenantNameExists(name); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrTenantNameTaken
	}
	slug := NormalizeSlug(req.Slug)
	if slug == "" {
		slug = makeSlug(name)
	}
	if !ValidateSlug(slug) {
		return nil, ErrInvalidSlug
	}
	if exists, err := s.store.SlugExists(slug); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrTenantNameTaken
	}
	if !ValidateEmail(req.OwnerEmail) {
		return nil, ErrInvalidEmail
	}

	now := time.Now().UTC()
	tenantID := auth.NewRandomHex(16)
	t := &Tenant{
		ID: tenantID, Name: name, Slug: slug, Status: TenantStatusProvisioning,
		CreatedBy: actorUserID, CreatedAt: now, UpdatedAt: now,
	}
	var owner *User
	var newOwner *User
	owner, err := s.store.GetUserByEmailNorm(NormalizeEmail(req.OwnerEmail))
	if errors.Is(err, ErrNotFound) {
		if strings.TrimSpace(req.OwnerDisplayName) == "" {
			return nil, ErrInvalidDisplayName
		}
		if err := ValidatePassword(req.OwnerPassword); err != nil {
			return nil, err
		}
		hash, err := auth.HashPassword(req.OwnerPassword)
		if err != nil {
			return nil, err
		}
		owner = &User{
			ID: auth.NewRandomHex(16), EmailNorm: NormalizeEmail(req.OwnerEmail),
			DisplayName: strings.TrimSpace(req.OwnerDisplayName), PasswordHash: hash,
			Status: UserStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		newOwner = owner
	} else if err != nil {
		return nil, err
	} else if owner.Status != UserStatusActive {
		return nil, ErrAccountUnavailable
	}
	m := &Membership{
		TenantID: tenantID, UserID: owner.ID, Role: domain.RoleOwner,
		Status: MembershipStatusActive, CreatedAt: now,
	}
	if err := s.store.CreateTenantOwner(t, newOwner, m); err != nil {
		if strings.Contains(err.Error(), "tenants.name") || strings.Contains(err.Error(), "tenants.slug") {
			return nil, ErrTenantNameTaken
		}
		if strings.Contains(err.Error(), "users.email_norm") {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	if s.provisioner != nil {
		if err := s.provisioner(ctx, tenantID); err != nil {
			_ = s.store.UpdateTenantStatus(tenantID, TenantStatusProvisioningFailed)
			t.Status = TenantStatusProvisioningFailed
			s.audit(tenantID, actorUserID, "admin_create_tenant_failed", "tenant", tenantID, "provision")
			return t, fmt.Errorf("%w: %v", ErrProvisionFailed, err)
		}
	}
	if err := s.store.UpdateTenantStatus(tenantID, TenantStatusActive); err != nil {
		return nil, err
	}
	t.Status = TenantStatusActive
	s.audit(tenantID, actorUserID, "admin_create_tenant", "tenant", tenantID, "ok")
	return t, nil
}

// updateTenantProfile 是平台与本租户管理共用的资料更新原语。
func (s *Service) updateTenantProfile(actorUserID, tenantID, name, slug string) error {
	t, err := s.store.GetTenant(tenantID)
	if err != nil {
		return err
	}
	if t.Status == TenantStatusDeleted {
		return ErrTenantDeleted
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidOrgName
	}
	slug = NormalizeSlug(slug)
	if slug == "" {
		slug = makeSlug(name)
	}
	if !ValidateSlug(slug) {
		return ErrInvalidSlug
	}
	if name != t.Name {
		if exists, err := s.store.TenantNameExists(name); err != nil {
			return err
		} else if exists {
			return ErrTenantNameTaken
		}
	}
	if slug != t.Slug {
		if exists, err := s.store.SlugExists(slug); err != nil {
			return err
		} else if exists {
			return ErrTenantNameTaken
		}
	}
	if err := s.store.UpdateTenantProfile(tenantID, name, slug); err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return ErrTenantNameTaken
		}
		return err
	}
	s.audit(tenantID, actorUserID, "tenant_update", "tenant", tenantID, "ok")
	return nil
}

func (s *Service) AdminUpdateTenant(actorUserID, tenantID, name, slug string) error {
	return s.updateTenantProfile(actorUserID, tenantID, name, slug)
}

// CurrentTenant 返回当前租户控制面资料；不包含其他租户或数据目录。
func (s *Service) CurrentTenant(scope *domain.TenantScope) (*Tenant, error) {
	if scope == nil || !scope.Valid() {
		return nil, ErrForbidden
	}
	return s.store.GetTenant(scope.TenantID)
}

// UpdateCurrentTenant 允许本租户 owner/admin 修改名称与 slug。
func (s *Service) UpdateCurrentTenant(scope *domain.TenantScope, name, slug string) error {
	if !s.canManageTenant(scope) {
		return ErrForbidden
	}
	return s.updateTenantProfile(scope.UserID, scope.TenantID, name, slug)
}

// AdminSetTenantStatus 支持 active/disabled 双向切换。provisioning_failed 激活前
// 会重跑 Provision；deleted 是终态。停用会立即废止该租户活动会话。
func (s *Service) AdminSetTenantStatus(ctx context.Context, actorUserID, actorTenantID, tenantID, status string) error {
	if tenantID == actorTenantID && status != TenantStatusActive {
		return ErrSelfAction
	}
	t, err := s.store.GetTenant(tenantID)
	if err != nil {
		return err
	}
	if t.Status == TenantStatusDeleted {
		return ErrTenantDeleted
	}
	if status != TenantStatusActive && status != TenantStatusDisabled {
		return fmt.Errorf("%w: %s", ErrInvalidStatus, status)
	}
	if status == TenantStatusActive && t.Status == TenantStatusProvisioningFailed {
		if s.provisioner == nil {
			return ErrProvisionFailed
		}
		if err := s.provisioner(ctx, tenantID); err != nil {
			return fmt.Errorf("%w: %v", ErrProvisionFailed, err)
		}
	}
	if err := s.store.UpdateTenantStatus(tenantID, status); err != nil {
		return err
	}
	if status == TenantStatusDisabled {
		_ = s.store.RevokeTenantSessions(tenantID)
	}
	s.audit(tenantID, actorUserID, "admin_set_tenant_status", "tenant", tenantID, status)
	return nil
}

// AdminDeleteTenant 软删除租户：控制面标记 deleted、废止活动会话，不删除数据目录
// 和业务历史。禁止删除管理员当前所在租户，避免把自己锁在失效会话中。
func (s *Service) AdminDeleteTenant(actorUserID, actorTenantID, tenantID string) error {
	if tenantID == actorTenantID {
		return ErrSelfAction
	}
	t, err := s.store.GetTenant(tenantID)
	if err != nil {
		return err
	}
	if t.Status == TenantStatusDeleted {
		return nil
	}
	if err := s.store.UpdateTenantStatus(tenantID, TenantStatusDeleted); err != nil {
		return err
	}
	_ = s.store.RevokeTenantSessions(tenantID)
	s.audit(tenantID, actorUserID, "admin_delete_tenant", "tenant", tenantID, "soft_deleted")
	return nil
}

// 平台管理员按指定租户管理成员；完整 owner 权限仅作用于目标 membership，不会
// 产生目标租户 TenantScope，也不能借此读取该租户业务数据。
func (s *Service) AdminTenantMemberAdd(actorUserID, tenantID, email, displayName, password, role string) error {
	_, err := s.addMemberToTenant(tenantID, actorUserID, email, displayName, password, role)
	return err
}

func (s *Service) AdminTenantMemberSetRole(actorUserID, tenantID, userID, role string) error {
	if err := s.ensureTenantMembershipMutable(tenantID); err != nil {
		return err
	}
	sc := &domain.TenantScope{
		TenantID: tenantID, UserID: actorUserID, Roles: []string{domain.RoleOwner}, Source: "platform",
	}
	return s.MembersSetRole(sc, userID, role)
}

func (s *Service) AdminTenantMemberRemove(actorUserID, tenantID, userID string) error {
	if err := s.ensureTenantMembershipMutable(tenantID); err != nil {
		return err
	}
	sc := &domain.TenantScope{
		TenantID: tenantID, UserID: actorUserID, Roles: []string{domain.RoleOwner}, Source: "platform",
	}
	return s.MembersRemove(sc, userID)
}

func (s *Service) ensureTenantMembershipMutable(tenantID string) error {
	t, err := s.store.GetTenant(tenantID)
	if err != nil {
		return err
	}
	if t.Status == TenantStatusDeleted || t.Status == TenantStatusDeleting {
		return ErrTenantDeleted
	}
	return nil
}

// ── 租户成员管理（owner/admin 专用，§4.2 租户内）─────────────────────

// canManageTenant 调用方须为本租户 owner 或 admin。
func (s *Service) canManageTenant(scope *domain.TenantScope) bool {
	return scope != nil && scope.Valid() &&
		(scope.HasRole(domain.RoleOwner) || scope.HasRole(domain.RoleAdmin))
}

// canManageRole owner 可管理任意角色；admin 只能管理 analyst/reviewer（§4.2）。
func canManageRole(scope *domain.TenantScope, targetRole string) bool {
	if scope.HasRole(domain.RoleOwner) {
		return true
	}
	if scope.HasRole(domain.RoleAdmin) {
		return targetRole == domain.RoleAnalyst || targetRole == domain.RoleReviewer
	}
	return false
}

// ValidMemberRoles 成员可分配角色（platform_admin 是平台级标记，不可作为成员角色）。
func ValidMemberRoles(role string) bool {
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleAnalyst, domain.RoleReviewer:
		return true
	}
	return false
}

// MembersList 当前租户全部成员（任何有效成员可见）。
func (s *Service) MembersList(scope *domain.TenantScope) ([]MemberView, error) {
	if scope == nil || !scope.Valid() {
		return nil, ErrForbidden
	}
	ms, err := s.store.ListMembershipsByTenant(scope.TenantID)
	if err != nil {
		return nil, err
	}
	return s.memberViews(ms)
}

// MembersAdd 添加成员：已有邮箱直接复用全局身份并新增本租户 membership；新邮箱
// 同时创建账号。一个用户可加入多个租户，登录后通过 SwitchTenant 显式切换。
// owner 可加任意角色；admin 只能加 analyst/reviewer。
func (s *Service) MembersAdd(scope *domain.TenantScope, email, displayName, password, role string) error {
	if !s.canManageTenant(scope) {
		return ErrForbidden
	}
	if !ValidMemberRoles(role) {
		return fmt.Errorf("%w: %s", ErrInvalidRole, role)
	}
	if !canManageRole(scope, role) {
		return ErrForbidden
	}
	_, err := s.addMemberToTenant(scope.TenantID, scope.UserID, email, displayName, password, role)
	return err
}

// addMemberToTenant 是本租户管理员与平台管理员共享的成员写入原语；调用方必须先
// 完成角色鉴权。返回 true 表示创建了新用户，false 表示复用已有账号。
func (s *Service) addMemberToTenant(tenantID, actorUserID, email, displayName, password, role string) (bool, error) {
	tenant, err := s.store.GetTenant(tenantID)
	if err != nil {
		return false, err
	}
	if tenant.Status == TenantStatusDeleted || tenant.Status == TenantStatusDeleting {
		return false, ErrTenantDeleted
	}
	if !ValidMemberRoles(role) {
		return false, fmt.Errorf("%w: %s", ErrInvalidRole, role)
	}
	if !ValidateEmail(email) {
		return false, ErrInvalidEmail
	}
	norm := NormalizeEmail(email)
	now := time.Now().UTC()

	existing, err := s.store.GetUserByEmailNorm(norm)
	switch {
	case err == nil:
		if existing.Status != UserStatusActive {
			return false, ErrAccountUnavailable
		}
		if _, err := s.store.GetMembership(tenantID, existing.ID); err == nil {
			s.audit(tenantID, actorUserID, "member_add_failed", "user", existing.ID, "already_member")
			return false, ErrMemberExists
		} else if !errors.Is(err, ErrNotFound) {
			return false, err
		}
		// displayName/password 非空表示调用方明确选择“新建账号”。邮箱已经存在时
		// 不静默降级成“添加已有账号”，避免 UI 显示已新建、实际却复用全局身份。
		if strings.TrimSpace(displayName) != "" || password != "" {
			return false, ErrEmailTaken
		}
		m := &Membership{TenantID: tenantID, UserID: existing.ID, Role: role,
			Status: MembershipStatusActive, CreatedAt: now}
		if err := s.store.CreateMembership(m); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return false, ErrMemberExists
			}
			return false, err
		}
		s.audit(tenantID, actorUserID, "member_add_existing", "user", existing.ID, role)
		return false, nil
	case errors.Is(err, ErrNotFound):
		if strings.TrimSpace(displayName) == "" {
			return false, ErrInvalidDisplayName
		}
		if err := ValidatePassword(password); err != nil {
			return false, err
		}
		hash, err := auth.HashPassword(password)
		if err != nil {
			return false, fmt.Errorf("密码哈希: %w", err)
		}
		u := &User{
			ID: auth.NewRandomHex(16), EmailNorm: norm, DisplayName: strings.TrimSpace(displayName),
			PasswordHash: hash, Status: UserStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		m := &Membership{TenantID: tenantID, UserID: u.ID, Role: role,
			Status: MembershipStatusActive, CreatedAt: now}
		if err := s.store.CreateUserAndMembership(u, m); err != nil {
			// 并发注册/添加竞态由 users.email_norm 唯一约束兜底。
			if strings.Contains(err.Error(), "UNIQUE") {
				return false, ErrEmailTaken
			}
			return false, err
		}
		s.audit(tenantID, actorUserID, "member_create", "user", u.ID, role)
		return true, nil
	default:
		return false, err
	}
}

// MembersSetRole 修改成员角色。admin 只能改 analyst/reviewer；最后一位 owner 不可降级。
func (s *Service) MembersSetRole(scope *domain.TenantScope, userID, role string) error {
	if !s.canManageTenant(scope) {
		return ErrForbidden
	}
	if !ValidMemberRoles(role) {
		return fmt.Errorf("%w: %s", ErrInvalidRole, role)
	}
	// ownerCount 与更新必须处于同一进程临界区，防止两位 owner 并发降级后
	// 都通过“还剩一位”检查，最终留下零 owner。
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	m, err := s.store.GetMembership(scope.TenantID, userID)
	if err != nil {
		return err // ErrNotFound → 404
	}
	if !canManageRole(scope, m.Role) || !canManageRole(scope, role) {
		return ErrForbidden
	}
	if m.Role == domain.RoleOwner && role != domain.RoleOwner {
		cnt, err := s.ownerCount(scope.TenantID)
		if err != nil {
			return err
		}
		if cnt <= 1 {
			s.audit(scope.TenantID, scope.UserID, "member_role_failed", "user", userID, "last_owner")
			return ErrLastOwner
		}
	}
	if err := s.store.UpdateMembershipRole(scope.TenantID, userID, role); err != nil {
		return err
	}
	s.audit(scope.TenantID, scope.UserID, "member_role_change", "user", userID, role)
	return nil
}

// MembersRemove 移除成员。owner 可移除任意非自身成员；admin 只能移除 analyst/reviewer；
// 最后一位 owner 不可移除。
func (s *Service) MembersRemove(scope *domain.TenantScope, userID string) error {
	if !s.canManageTenant(scope) {
		return ErrForbidden
	}
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	m, err := s.store.GetMembership(scope.TenantID, userID)
	if err != nil {
		return err // ErrNotFound → 404
	}
	if !canManageRole(scope, m.Role) {
		return ErrForbidden
	}
	if userID == scope.UserID {
		return ErrSelfAction
	}
	if m.Role == domain.RoleOwner {
		cnt, err := s.ownerCount(scope.TenantID)
		if err != nil {
			return err
		}
		if cnt <= 1 {
			s.audit(scope.TenantID, scope.UserID, "member_remove_failed", "user", userID, "last_owner")
			return ErrLastOwner
		}
	}
	if err := s.store.DeleteMembership(scope.TenantID, userID); err != nil {
		return err
	}
	_ = s.store.RevokeTenantUserSessions(scope.TenantID, userID)
	s.audit(scope.TenantID, scope.UserID, "member_remove", "user", userID, "ok")
	return nil
}

// memberViews 由成员关系构造视图（补充用户信息）。
func (s *Service) memberViews(ms []Membership) ([]MemberView, error) {
	views := make([]MemberView, 0, len(ms))
	for _, m := range ms {
		u, err := s.store.GetUserByID(m.UserID)
		if err != nil {
			return nil, err
		}
		status := m.Status
		if u.Status != UserStatusActive {
			status = u.Status
		}
		views = append(views, MemberView{
			UserID: u.ID, Email: u.EmailNorm, DisplayName: u.DisplayName,
			Role: m.Role, Status: status, LastLoginAt: u.LastLoginAt, CreatedAt: m.CreatedAt,
		})
	}
	return views, nil
}

// ownerCount 租户内真正可登录的 active owner 数量（最后所有者保护）。
// 不能只数 membership：若一个 owner 的全局账号已停用，再把最后一个可用
// owner 降级/移除，会让租户只剩“名义 owner”而无人能够进入管理。
func (s *Service) ownerCount(tenantID string) (int, error) {
	ms, err := s.store.ListMembershipsByTenant(tenantID)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, m := range ms {
		if m.Role != domain.RoleOwner || m.Status != MembershipStatusActive {
			continue
		}
		u, err := s.store.GetUserByID(m.UserID)
		if err != nil {
			return 0, err
		}
		if u.Status == UserStatusActive {
			n++
		}
	}
	return n, nil
}
