// Package identity 实现中央身份控制面：users/tenants/memberships/auth_sessions。
//
// 数据模型见设计文档 §8.1。所有写操作用事务；连接显式启用 SQLite Foreign Key；
// 敏感字段（password_hash、token_hash、csrf_hash）严格服务层访问，绝不外泄。
package identity

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store 包装中央身份库（control/identity.db）。
type Store struct {
	mu sync.Mutex
	db *sql.DB
}

// Open 打开（或创建）身份库并建表。目录 0700，显式启用 foreign_keys 并自检。
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("创建控制面目录: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("打开身份库: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping 身份库: %w", err)
	}
	var fk int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("检查 foreign_keys: %w", err)
	}
	if fk != 1 {
		_ = db.Close()
		return nil, fmt.Errorf("身份库 foreign_keys 未启用（fk=%d）", fk)
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("身份库建表: %w", err)
	}
	if err := migrateUsersPlatformAdmin(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("身份库迁移 users.platform_admin: %w", err)
	}
	if err := migrateTenantsUnique(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("身份库迁移 tenants 唯一约束: %w", err)
	}
	return &Store{db: db}, nil
}

// migrateUsersPlatformAdmin 幂等迁移：为已存在的 users 表补 platform_admin 列
// （新库由 schema 直接建出；旧库 CREATE TABLE IF NOT EXISTS 不生效，需 ALTER）。
func migrateUsersPlatformAdmin(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(users)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "platform_admin" {
			return nil // 已存在
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(`ALTER TABLE users ADD COLUMN platform_admin INTEGER NOT NULL DEFAULT 0`)
	return err
}

// migrateTenantsUnique 幂等迁移（§4.1 严重缺陷修复，2026-08-19）：
//  1. 唯一索引已存在 → 直接返回。
//  2. 解析存量重复 name/slug：保留最早创建者（created_at 升序、id 升序取首），
//     其余追加「前 8 位租户 ID」后缀消歧（name 用 " · "，slug 用 "-"）。
//  3. 建 tenants.name / tenants.slug 唯一索引，杜绝注册并发竞态落重复。
//
// 存量数据唯一后才能建索引，故必须先解析再建索引。
func migrateTenantsUnique(db *sql.DB) error {
	exists, err := indexExists(db, "idx_tenants_name")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if err := dedupeTenantColumn(db, "name", " · "); err != nil {
		return fmt.Errorf("解析重复租户名: %w", err)
	}
	if err := dedupeTenantColumn(db, "slug", "-"); err != nil {
		return fmt.Errorf("解析重复 slug: %w", err)
	}
	// 旧 schema 曾建过非唯一 idx_tenants_slug（仅存量库存在；新库 schema 已不再创建）。
	if _, err := db.Exec(`DROP INDEX IF EXISTS idx_tenants_slug`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX idx_tenants_name ON tenants(name)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX idx_tenants_slug ON tenants(slug)`); err != nil {
		return err
	}
	return nil
}

// indexExists 判断索引是否存在（幂等迁移守卫）。
func indexExists(db *sql.DB, name string) (bool, error) {
	var one int
	err := db.QueryRow(`SELECT 1 FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// dedupeTenantColumn 将某列中的重复值消歧：保留最早创建者原值，其余改为
// 「原值+分隔符+前8位ID」（拼接后仍冲突则退到完整 ID）。仅对重复组生效。
func dedupeTenantColumn(db *sql.DB, col, sep string) error {
	rows, err := db.Query(`SELECT ` + col + ` FROM tenants GROUP BY ` + col + ` HAVING COUNT(*) > 1`)
	if err != nil {
		return err
	}
	var dupVals []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		dupVals = append(dupVals, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, v := range dupVals {
		var keeper string
		if err := db.QueryRow(
			`SELECT id FROM tenants WHERE `+col+`=? ORDER BY created_at ASC, id ASC LIMIT 1`, v,
		).Scan(&keeper); err != nil {
			return err
		}
		idRows, err := db.Query(`SELECT id FROM tenants WHERE `+col+`=? AND id!=?`, v, keeper)
		if err != nil {
			return err
		}
		var others []string
		for idRows.Next() {
			var id string
			if err := idRows.Scan(&id); err != nil {
				idRows.Close()
				return err
			}
			others = append(others, id)
		}
		idRows.Close()
		if err := idRows.Err(); err != nil {
			return err
		}
		for _, id := range others {
			newVal := v + sep + id[:8]
			var clash int
			if qerr := db.QueryRow(`SELECT 1 FROM tenants WHERE `+col+`=? LIMIT 1`, newVal).Scan(&clash); qerr == nil {
				newVal = v + sep + id // 拼接仍冲突 → 退到完整 ID
			} else if !errors.Is(qerr, sql.ErrNoRows) {
				return qerr
			}
			if _, err := db.Exec(`UPDATE tenants SET `+col+`=?, updated_at=? WHERE id=?`,
				newVal, time.Now().UTC(), id); err != nil {
				return err
			}
		}
	}
	return nil
}

// Close 关闭身份库。
func (s *Store) Close() error { return s.db.Close() }

const schema = `
CREATE TABLE IF NOT EXISTS users (
  id            TEXT PRIMARY KEY,
  email_norm    TEXT NOT NULL UNIQUE,
  display_name  TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'active',
  last_login_at TIMESTAMP,
  created_at    TIMESTAMP NOT NULL,
  updated_at    TIMESTAMP NOT NULL,
  platform_admin INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS tenants (
  id         TEXT PRIMARY KEY,
  name       TEXT NOT NULL,
  slug       TEXT NOT NULL,
  status     TEXT NOT NULL DEFAULT 'provisioning',
  created_by TEXT REFERENCES users(id), -- 可 NULL：系统引导租户（-setup/存量迁移）
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
-- 租户名/slug 唯一索引由 migrateTenantsUnique 幂等创建（需先解析存量重复，§4.1）。

CREATE TABLE IF NOT EXISTS memberships (
  tenant_id  TEXT NOT NULL REFERENCES tenants(id),
  user_id    TEXT NOT NULL REFERENCES users(id),
  role       TEXT NOT NULL,
  status     TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL,
  PRIMARY KEY (tenant_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_memberships_user ON memberships(user_id);

CREATE TABLE IF NOT EXISTS auth_sessions (
  id               TEXT PRIMARY KEY,
  token_hash       TEXT NOT NULL UNIQUE,
  user_id          TEXT NOT NULL REFERENCES users(id),
  active_tenant_id TEXT NOT NULL REFERENCES tenants(id),
  csrf_hash        TEXT NOT NULL,
  expires_at       TIMESTAMP NOT NULL,
  last_seen_at     TIMESTAMP NOT NULL,
  revoked_at       TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user ON auth_sessions(user_id);

-- 第二阶段：即应/钉钉渠道可信映射（首版留表，不启用）。
CREATE TABLE IF NOT EXISTS channel_identities (
  tenant_id        TEXT NOT NULL REFERENCES tenants(id),
  channel          TEXT NOT NULL,
  installation_id  TEXT NOT NULL,
  external_user_id TEXT NOT NULL,
  user_id          TEXT,
  PRIMARY KEY (channel, installation_id, external_user_id)
);

CREATE TABLE IF NOT EXISTS security_audit_logs (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  tenant_id     TEXT,
  actor_user_id TEXT,
  action        TEXT NOT NULL,
  object_type   TEXT,
  object_id     TEXT,
  result        TEXT,
  ip            TEXT,
  user_agent    TEXT,
  created_at    TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_tenant ON security_audit_logs(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON security_audit_logs(actor_user_id, created_at);

CREATE TABLE IF NOT EXISTS schema_migrations (
  version    INTEGER PRIMARY KEY,
  applied_at TIMESTAMP NOT NULL,
  checksum   TEXT NOT NULL
);
`

// ── 数据模型 ───────────────────────────────────────────────────────

// 用户状态。
const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
	UserStatusLocked   = "locked"
)

// 租户状态（设计文档 §8.4 生命周期状态机）。
const (
	TenantStatusProvisioning       = "provisioning"
	TenantStatusActive             = "active"
	TenantStatusDisabled           = "disabled"
	TenantStatusProvisioningFailed = "provisioning_failed"
	TenantStatusDeleting           = "deleting"
	TenantStatusDeleted            = "deleted"
)

// 成员关系状态。
const (
	MembershipStatusActive   = "active"
	MembershipStatusDisabled = "disabled"
)

// User 可登录身份（邮箱平台唯一）。
type User struct {
	ID            string
	EmailNorm     string
	DisplayName   string
	PasswordHash  string
	Status        string
	LastLoginAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PlatformAdmin bool // 平台管理员（平台级元数据权限，非租户超级用户，§4.2）
}

// Tenant 组织/工作空间（数据隔离根）。
type Tenant struct {
	ID        string
	Name      string
	Slug      string
	Status    string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Membership 用户与租户的成员关系及角色。
type Membership struct {
	TenantID  string
	UserID    string
	Role      string
	Status    string
	CreatedAt time.Time
}

// AuthSession Web 登录态（只存 Token Hash）。CSRFHash 是历史列名，当前保存用于
// 派生 CSRF Token 的服务端 Secret，不保存浏览器实际持有的 Token。
type AuthSession struct {
	ID             string
	TokenHash      string
	UserID         string
	ActiveTenantID string
	CSRFHash       string
	ExpiresAt      time.Time
	LastSeenAt     time.Time
	RevokedAt      *time.Time
}

// ErrNotFound 统一"不存在"错误（映射 404，避免泄露资源存在性）。
var ErrNotFound = errors.New("identity: 记录不存在")

// ── 用户 ───────────────────────────────────────────────────────────

func (s *Store) CreateUser(u *User) error {
	_, err := s.db.Exec(`INSERT INTO users
		(id, email_norm, display_name, password_hash, status, last_login_at, created_at, updated_at, platform_admin)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		u.ID, u.EmailNorm, u.DisplayName, u.PasswordHash, u.Status, u.LastLoginAt, u.CreatedAt, u.UpdatedAt, u.PlatformAdmin)
	return err
}

func (s *Store) GetUserByEmailNorm(emailNorm string) (*User, error) {
	row := s.db.QueryRow(`SELECT id, email_norm, display_name, password_hash, status, last_login_at, created_at, updated_at, platform_admin
		FROM users WHERE email_norm=?`, emailNorm)
	return scanUser(row)
}

func (s *Store) GetUserByID(id string) (*User, error) {
	row := s.db.QueryRow(`SELECT id, email_norm, display_name, password_hash, status, last_login_at, created_at, updated_at, platform_admin
		FROM users WHERE id=?`, id)
	return scanUser(row)
}

func (s *Store) UpdateUserPassword(id, hash string) error {
	_, err := s.db.Exec(`UPDATE users SET password_hash=?, updated_at=? WHERE id=?`, hash, time.Now().UTC(), id)
	return err
}

func (s *Store) UpdateUserLastLogin(id string, at time.Time) error {
	_, err := s.db.Exec(`UPDATE users SET last_login_at=?, updated_at=? WHERE id=?`, at, at, id)
	return err
}

func (s *Store) UpdateUserStatus(id, status string) error {
	_, err := s.db.Exec(`UPDATE users SET status=?, updated_at=? WHERE id=?`, status, time.Now().UTC(), id)
	return err
}

// UpdateUserProfile 修改用户的全局身份资料。邮箱是平台唯一登录标识；租户管理员
// 不能调用本方法，避免一个租户修改用户在其他租户中的身份。
func (s *Store) UpdateUserProfile(id, emailNorm, displayName string) error {
	res, err := s.db.Exec(`UPDATE users SET email_norm=?, display_name=?, updated_at=? WHERE id=?`,
		emailNorm, displayName, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

// SetUserPlatformAdmin 授予/撤销平台管理员标记（-setup 引导 + 平台级受审计操作）。
func (s *Store) SetUserPlatformAdmin(id string, on bool) error {
	_, err := s.db.Exec(`UPDATE users SET platform_admin=?, updated_at=? WHERE id=?`, on, time.Now().UTC(), id)
	return err
}

// ListUsers 全量用户（平台管理视图），按创建时间倒序。
func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT id, email_norm, display_name, password_hash, status, last_login_at, created_at, updated_at, platform_admin
		FROM users ORDER BY created_at DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.EmailNorm, &u.DisplayName, &u.PasswordHash, &u.Status,
			&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.PlatformAdmin); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CountPlatformAdmins 统计仍可登录的平台管理员，用于防止撤销最后一位平台管理员。
func (s *Store) CountPlatformAdmins() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE platform_admin=1 AND status=?`, UserStatusActive).Scan(&n)
	return n, err
}

// ── 租户与成员关系 ─────────────────────────────────────────────────

func (s *Store) CreateTenant(t *Tenant) error {
	_, err := s.db.Exec(`INSERT INTO tenants (id, name, slug, status, created_by, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?)`,
		t.ID, t.Name, t.Slug, t.Status, t.CreatedBy, t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *Store) GetTenant(id string) (*Tenant, error) {
	row := s.db.QueryRow(`SELECT id, name, slug, status, COALESCE(created_by, ''), created_at, updated_at FROM tenants WHERE id=?`, id)
	var t Tenant
	err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &t, err
}

func (s *Store) UpdateTenantStatus(id, status string) error {
	res, err := s.db.Exec(`UPDATE tenants SET status=?, updated_at=? WHERE id=?`, status, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

// UpdateTenantProfile 修改租户展示资料；name/slug 的唯一性由数据库索引兜底。
func (s *Store) UpdateTenantProfile(id, name, slug string) error {
	res, err := s.db.Exec(`UPDATE tenants SET name=?, slug=?, updated_at=? WHERE id=?`,
		name, slug, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

// GetTenantBySlug 按 slug 查询租户（-setup/legacy 引导按约定 slug 幂等查找）。
func (s *Store) GetTenantBySlug(slug string) (*Tenant, error) {
	row := s.db.QueryRow(`SELECT id, name, slug, status, COALESCE(created_by, ''), created_at, updated_at FROM tenants WHERE slug=?`, slug)
	var t Tenant
	err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &t, err
}

// TenantNameExists 判断租户名是否已被占用（注册唯一性预检，§4.1）。
func (s *Store) TenantNameExists(name string) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM tenants WHERE name=? LIMIT 1`, name).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// SlugExists 判断 slug 是否已被占用（注册时 slug 冲突消歧，§4.1）。
func (s *Store) SlugExists(slug string) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM tenants WHERE slug=? LIMIT 1`, slug).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CreateSystemTenant 创建系统引导租户（-setup/存量迁移）：created_by 为 NULL
// （启动引导先于任何用户存在，无创建人；-setup 随后补 owner membership）。
func (s *Store) CreateSystemTenant(t *Tenant) error {
	_, err := s.db.Exec(`INSERT INTO tenants (id, name, slug, status, created_by, created_at, updated_at)
		VALUES (?,?,?,?,NULL,?,?)`,
		t.ID, t.Name, t.Slug, t.Status, t.CreatedAt, t.UpdatedAt)
	return err
}

// ListTenants 全量租户（平台管理视图），按创建时间倒序。
func (s *Store) ListTenants() ([]Tenant, error) {
	rows, err := s.db.Query(`SELECT id, name, slug, status, COALESCE(created_by, ''), created_at, updated_at
		FROM tenants ORDER BY created_at DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CreateMembership(m *Membership) error {
	_, err := s.db.Exec(`INSERT INTO memberships (tenant_id, user_id, role, status, created_at) VALUES (?,?,?,?,?)`,
		m.TenantID, m.UserID, m.Role, m.Status, m.CreatedAt)
	return err
}

func (s *Store) GetMembership(tenantID, userID string) (*Membership, error) {
	row := s.db.QueryRow(`SELECT tenant_id, user_id, role, status, created_at FROM memberships WHERE tenant_id=? AND user_id=?`,
		tenantID, userID)
	var m Membership
	err := row.Scan(&m.TenantID, &m.UserID, &m.Role, &m.Status, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &m, err
}

func (s *Store) ListMembershipsByUser(userID string) ([]Membership, error) {
	rows, err := s.db.Query(`SELECT tenant_id, user_id, role, status, created_at FROM memberships
		WHERE user_id=? ORDER BY created_at ASC, tenant_id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Membership
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.TenantID, &m.UserID, &m.Role, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) UpdateMembershipStatus(tenantID, userID, status string) error {
	_, err := s.db.Exec(`UPDATE memberships SET status=? WHERE tenant_id=? AND user_id=?`, status, tenantID, userID)
	return err
}

// ListMembershipsByTenant 某租户的全部成员关系（成员管理/平台视图）。
func (s *Store) ListMembershipsByTenant(tenantID string) ([]Membership, error) {
	rows, err := s.db.Query(`SELECT tenant_id, user_id, role, status, created_at FROM memberships WHERE tenant_id=?`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Membership
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.TenantID, &m.UserID, &m.Role, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// UpdateMembershipRole 修改某成员在租户内的角色。
func (s *Store) UpdateMembershipRole(tenantID, userID, role string) error {
	_, err := s.db.Exec(`UPDATE memberships SET role=? WHERE tenant_id=? AND user_id=?`, role, tenantID, userID)
	return err
}

// DeleteMembership 删除成员关系（移除成员）。
func (s *Store) DeleteMembership(tenantID, userID string) error {
	_, err := s.db.Exec(`DELETE FROM memberships WHERE tenant_id=? AND user_id=?`, tenantID, userID)
	return err
}

// CreateUserAndMembership 原子创建新用户及其首个租户成员关系，避免成员关系落库
// 失败后遗留无法登录的孤儿用户。
func (s *Store) CreateUserAndMembership(u *User, m *Membership) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := insertUserTx(tx, u); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := insertMembershipTx(tx, m); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// CreateTenantOwner 原子创建租户、owner 成员关系，以及可选的新 owner 用户。
// existing owner 复用全局身份时 newUser 传 nil。
func (s *Store) CreateTenantOwner(t *Tenant, newUser *User, m *Membership) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if newUser != nil {
		if err := insertUserTx(tx, newUser); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := insertTenantTx(tx, t); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := insertMembershipTx(tx, m); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ── 登录会话 ───────────────────────────────────────────────────────

func (s *Store) CreateAuthSession(as *AuthSession) error {
	_, err := s.db.Exec(`INSERT INTO auth_sessions
		(id, token_hash, user_id, active_tenant_id, csrf_hash, expires_at, last_seen_at, revoked_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		as.ID, as.TokenHash, as.UserID, as.ActiveTenantID, as.CSRFHash, as.ExpiresAt, as.LastSeenAt, as.RevokedAt)
	return err
}

func (s *Store) GetAuthSessionByTokenHash(hash string) (*AuthSession, error) {
	row := s.db.QueryRow(`SELECT id, token_hash, user_id, active_tenant_id, csrf_hash, expires_at, last_seen_at, revoked_at
		FROM auth_sessions WHERE token_hash=?`, hash)
	var as AuthSession
	err := row.Scan(&as.ID, &as.TokenHash, &as.UserID, &as.ActiveTenantID, &as.CSRFHash,
		&as.ExpiresAt, &as.LastSeenAt, &as.RevokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &as, err
}

func (s *Store) RevokeSession(id string) error {
	_, err := s.db.Exec(`UPDATE auth_sessions SET revoked_at=? WHERE id=? AND revoked_at IS NULL`, time.Now().UTC(), id)
	return err
}

// RevokeUserSessions 废止某用户全部会话（登录轮换/改密/停用时用）。
func (s *Store) RevokeUserSessions(userID, exceptID string) error {
	now := time.Now().UTC()
	if exceptID == "" {
		_, err := s.db.Exec(`UPDATE auth_sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL`, now, userID)
		return err
	}
	_, err := s.db.Exec(`UPDATE auth_sessions SET revoked_at=? WHERE user_id=? AND id<>? AND revoked_at IS NULL`, now, userID, exceptID)
	return err
}

// RevokeTenantSessions 废止当前激活在某租户上的全部会话。停用/删除租户时用，
// 不影响这些用户在其他租户重新登录或切换后的会话。
func (s *Store) RevokeTenantSessions(tenantID string) error {
	_, err := s.db.Exec(`UPDATE auth_sessions SET revoked_at=?
		WHERE active_tenant_id=? AND revoked_at IS NULL`, time.Now().UTC(), tenantID)
	return err
}

// RevokeTenantUserSessions 废止某用户当前激活在指定租户的会话。移除/停用成员时用。
func (s *Store) RevokeTenantUserSessions(tenantID, userID string) error {
	_, err := s.db.Exec(`UPDATE auth_sessions SET revoked_at=?
		WHERE active_tenant_id=? AND user_id=? AND revoked_at IS NULL`,
		time.Now().UTC(), tenantID, userID)
	return err
}

// UpdateSessionTenant 原子切换登录会话的活动租户并轮换 CSRF Secret。
func (s *Store) UpdateSessionTenant(sessionID, tenantID, csrfSecret string) error {
	res, err := s.db.Exec(`UPDATE auth_sessions SET active_tenant_id=?, csrf_hash=?, last_seen_at=?
		WHERE id=? AND revoked_at IS NULL`, tenantID, csrfSecret, time.Now().UTC(), sessionID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

// UpdateSessionCSRF 轮换会话的 CSRF Secret。
func (s *Store) UpdateSessionCSRF(id, csrfSecret string) error {
	_, err := s.db.Exec(`UPDATE auth_sessions SET csrf_hash=? WHERE id=?`, csrfSecret, id)
	return err
}

// TouchSession 节流更新 last_seen_at（由 Service 决定是否写）。
func (s *Store) TouchSession(id string, at time.Time) error {
	_, err := s.db.Exec(`UPDATE auth_sessions SET last_seen_at=? WHERE id=?`, at, id)
	return err
}

// ── 安全审计 ───────────────────────────────────────────────────────

// InsertAuditLog 写入一条安全审计记录（不保存密码/Token/Key/客户原文）。
func (s *Store) InsertAuditLog(l *AuditLog) error {
	_, err := s.db.Exec(`INSERT INTO security_audit_logs
		(tenant_id, actor_user_id, action, object_type, object_id, result, ip, user_agent, created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		l.TenantID, l.ActorUserID, l.Action, l.ObjectType, l.ObjectID, l.Result, l.IP, l.UserAgent, l.CreatedAt)
	return err
}

// AuditLog 安全审计记录。
type AuditLog struct {
	TenantID    string
	ActorUserID string
	Action      string
	ObjectType  string
	ObjectID    string
	Result      string
	IP          string
	UserAgent   string
	CreatedAt   time.Time
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.EmailNorm, &u.DisplayName, &u.PasswordHash, &u.Status,
		&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.PlatformAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

// ── 事务辅助（注册流程 user+tenant+membership 原子落库）──────────────

func insertUserTx(tx *sql.Tx, u *User) error {
	_, err := tx.Exec(`INSERT INTO users
		(id, email_norm, display_name, password_hash, status, last_login_at, created_at, updated_at, platform_admin)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		u.ID, u.EmailNorm, u.DisplayName, u.PasswordHash, u.Status, u.LastLoginAt, u.CreatedAt, u.UpdatedAt, u.PlatformAdmin)
	return err
}

func insertTenantTx(tx *sql.Tx, t *Tenant) error {
	_, err := tx.Exec(`INSERT INTO tenants (id, name, slug, status, created_by, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?)`,
		t.ID, t.Name, t.Slug, t.Status, t.CreatedBy, t.CreatedAt, t.UpdatedAt)
	return err
}

func insertMembershipTx(tx *sql.Tx, m *Membership) error {
	_, err := tx.Exec(`INSERT INTO memberships (tenant_id, user_id, role, status, created_at) VALUES (?,?,?,?,?)`,
		m.TenantID, m.UserID, m.Role, m.Status, m.CreatedAt)
	return err
}
