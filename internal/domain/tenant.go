package domain

// 角色常量（设计文档 §4.2）。注册用户默认为 owner；owner/admin 可在本租户
// 管理成员，platform_admin 只管理平台控制面，不天然获得租户业务数据权限。
const (
	RolePlatformAdmin = "platform_admin"
	RoleOwner         = "owner"
	RoleAdmin         = "admin"
	RoleAnalyst       = "analyst"
	RoleReviewer      = "reviewer"
)

// TenantScope 是服务端校验过的租户作用域：由登录会话 + Membership 构造，
// 绝不从 JSON Body / Query / X-Tenant-ID / 前端状态恢复（设计文档 §9.1）。
// 叶子包类型，禁止依赖业务包（防循环依赖）。
type TenantScope struct {
	TenantID    string
	UserID      string
	DisplayName string // 服务端身份控制面解析出的当前用户姓名（不可由请求体覆盖）
	Email       string // 服务端身份控制面解析出的当前用户邮箱（画像展示/消歧）
	Roles       []string
	Source      string // "http" / "taskbg" / "setup" / ...
}

// Valid 校验作用域是否可用于数据面访问（TenantID 与 UserID 都非空）。
func (s TenantScope) Valid() bool {
	return s.TenantID != "" && s.UserID != ""
}

// HasRole 校验当前作用域是否具备指定角色。
func (s TenantScope) HasRole(role string) bool {
	for _, r := range s.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsTenantAdmin 是否租户管理员及以上（owner/admin 可维护本租户全部会话）。
func (s TenantScope) IsTenantAdmin() bool {
	return s.HasRole(RoleOwner) || s.HasRole(RoleAdmin)
}

// RunKey 是不可混淆的复合 Key：同一 session_id 在不同租户互不干扰（§9.3）。
type RunKey struct {
	TenantID  string
	SessionID string
}
