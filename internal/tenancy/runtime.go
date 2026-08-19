// Package tenancy implements the tenant data plane for multi-tenancy (设计文档 §6.4/§6.5).
//
// 认证控制面 + 租户独立数据面：每个租户拥有独立的 data 目录
// （tenants/<uuid>/history.db + wiki/ + tenant.json），服务端按登录会话解析
// TenantScope 后懒加载对应 Runtime。同一 session_id 天然属于其所在租户的
// history.db——跨租户访问在数据面打开前就被作用域隔离（§9.3）。
//
// Runtime 是某租户在进程内的完整数据面（每租户一整套 wiki/agent/sessions），
// 平台级依赖（modelMgr/systemWiki/leads）只建一次、跨租户共享。
package tenancy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/history"
	"customer-demand-agent/internal/leads"
	"customer-demand-agent/internal/memory/assembler"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/memory/shortterm"
	"customer-demand-agent/internal/memory/tools"
	"customer-demand-agent/internal/model"
	"customer-demand-agent/internal/review"
)

// ErrNotProvisioned 租户数据面尚未初始化（注册失败或目录被删）。
var ErrNotProvisioned = errors.New("租户数据面未初始化")

// Runtime 是某租户的完整数据面。History/Wiki 是该租户私有的存储实例——
// 本身即租户边界，session_id/记忆标题跨租户不存在。
type Runtime struct {
	TenantID  string
	History   *history.Store
	Wiki      longterm.Store // Composite：system 只读基线 + 租户覆盖层
	Sessions  *shortterm.SessionManager
	Tools     *tools.Registry
	Assembler *assembler.Assembler
	Review    *review.Service
	Agent     *agent.Agent
}

// Close 关闭租户持有的资源（历史库连接）。
func (rt *Runtime) Close() error {
	if rt.History != nil {
		return rt.History.Close()
	}
	return nil
}

// Registry 持有全部租户运行时，按 tenantID 懒加载缓存（§6.4 懒加载）。
type Registry struct {
	mu       sync.Mutex
	runtimes map[string]*Runtime

	// 平台级共享依赖（只建一次，跨租户共享）
	modelMgr   *model.Manager
	systemWiki *longterm.WikiStore
	leads      *leads.Service
	cfg        *config.Store
	tenantsDir string
	maxIters   int
}

// NewRegistry 创建租户运行时注册表。systemWiki 是已加载的平台只读系统知识；
// tenantsDir 是 <data_dir>/tenants（每租户独立子目录）。
func NewRegistry(modelMgr *model.Manager, systemWiki *longterm.WikiStore, tenantsDir string, cfg *config.Store, leads *leads.Service) *Registry {
	maxIters := cfg.Get().AgentMaxIterations
	if maxIters <= 0 {
		maxIters = agent.DefaultMaxIterations
	}
	return &Registry{
		runtimes:   make(map[string]*Runtime),
		modelMgr:   modelMgr,
		systemWiki: systemWiki,
		leads:      leads,
		cfg:        cfg,
		tenantsDir: tenantsDir,
		maxIters:   maxIters,
	}
}

// TenantsDir 返回租户数据面根目录（迁移/测试用）。
func (r *Registry) TenantsDir() string { return r.tenantsDir }

// tenantIDRe 白名单校验租户 ID：UUID（带横线）或 32 位 hex（identity 生成形态）。
// 严格白名单而非黑名单——任何其它形态（../、Unicode、斜杠）直接拒绝，
// 从根上杜绝目录穿越（§6.4 纵深防御）。
var tenantIDRe = regexp.MustCompile(`^[0-9a-f]{32}$|^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ValidTenantID 校验租户 ID 是否为合法白名单形态（供外部防御性校验）。
func ValidTenantID(tenantID string) bool { return tenantIDRe.MatchString(tenantID) }

// tenantDir 返回租户数据目录（已通过白名单校验）。
func (r *Registry) tenantDir(tenantID string) string { return filepath.Join(r.tenantsDir, tenantID) }

// Provision 初始化租户数据面（注册流程回调）：
// staging 临时目录内创建 history.db（迁移+校验）+ wiki/ + tenant.json，
// 全部就绪后原子 rename 为 active——进程中断不留半成品目录（§6.4）。
// 幂等：active 已存在则直接返回成功。
func (r *Registry) Provision(ctx context.Context, tenantID string) error {
	if !ValidTenantID(tenantID) {
		return fmt.Errorf("非法租户 ID: %q", tenantID)
	}
	active := r.tenantDir(tenantID)
	if info, err := os.Stat(active); err == nil && info.IsDir() {
		return nil // 已初始化（幂等）
	}
	staging := active + ".staging"
	_ = os.RemoveAll(staging) // 清残留（上次中断的半成品）
	if err := os.MkdirAll(staging, 0o700); err != nil {
		return fmt.Errorf("创建租户 staging 目录: %w", err)
	}
	fail := func(err error) error {
		_ = os.RemoveAll(staging)
		return err
	}
	// 1. history.db：OpenTenant 会跑 DDL 迁移并写入 tenant_meta（三方核对）。
	h, err := history.OpenTenant(tenantID, filepath.Join(staging, "history.db"))
	if err != nil {
		return fail(fmt.Errorf("初始化租户历史库: %w", err))
	}
	_ = h.Close()
	// 2. wiki 覆盖层目录（空即可，运行时懒加载写入）。
	if err := os.MkdirAll(filepath.Join(staging, "wiki"), 0o700); err != nil {
		return fail(fmt.Errorf("创建租户 wiki 目录: %w", err))
	}
	// 3. tenant.json（进程内核对租户归属）。
	if err := writeTenantMeta(filepath.Join(staging, "tenant.json"), tenantID); err != nil {
		return fail(err)
	}
	// 4. 原子 rename → active。
	if err := os.Rename(staging, active); err != nil {
		// 并发竞态：另一 goroutine 刚完成 rename（active 已存在）
		if _, statErr := os.Stat(active); statErr == nil {
			_ = os.RemoveAll(staging)
			return nil
		}
		return fail(fmt.Errorf("激活租户目录: %w", err))
	}
	return nil
}

// MigrateLegacy 把单租户存量数据迁入指定租户数据面（阶段 5 引导）：
// 旧 history.db 复制到 staging 后经 OpenTenant 跑 DDL 迁移 + tenant_meta
// （会话/消息/工具/checkpoint 原样保留），旧 wiki 的用户记忆（客户/使用者）
// 搬入 tenants/<id>/wiki（系统基线另走 system/wiki 播种，不在此处）。
// 幂等：active 目录已存在则跳过。调用方负责先创建租户记录。
func (r *Registry) MigrateLegacy(ctx context.Context, tenantID, legacyDBPath, legacyWikiDir string) error {
	if !ValidTenantID(tenantID) {
		return fmt.Errorf("非法租户 ID: %q", tenantID)
	}
	active := r.tenantDir(tenantID)
	if info, err := os.Stat(active); err == nil && info.IsDir() {
		return nil // 已迁移（幂等）
	}
	staging := active + ".staging"
	_ = os.RemoveAll(staging) // 清残留（上次中断的半成品）
	if err := os.MkdirAll(staging, 0o700); err != nil {
		return fmt.Errorf("创建租户 staging 目录: %w", err)
	}
	fail := func(err error) error {
		_ = os.RemoveAll(staging)
		return err
	}
	// 1. history.db：复制旧库（WAL 模式下可能残留 -wal/-shm，一并忽略，只取主库），
	//    然后 OpenTenant 就地迁移 DDL + 写入 tenant_meta（三方核对）。
	data, err := os.ReadFile(legacyDBPath)
	if err != nil {
		return fail(fmt.Errorf("读取旧历史库 %s: %w", legacyDBPath, err))
	}
	if err := os.WriteFile(filepath.Join(staging, "history.db"), data, 0o600); err != nil {
		return fail(fmt.Errorf("写入历史库副本: %w", err))
	}
	h, err := history.OpenTenant(tenantID, filepath.Join(staging, "history.db"))
	if err != nil {
		return fail(fmt.Errorf("迁移旧历史库: %w", err))
	}
	_ = h.Close()
	// 2. wiki：只搬 用户记忆（客户/使用者）——租户私密记忆；产品/系统基线另走播种。
	if err := os.MkdirAll(filepath.Join(staging, "wiki"), 0o700); err != nil {
		return fail(fmt.Errorf("创建租户 wiki 目录: %w", err))
	}
	if legacyWikiDir != "" {
		if err := copyLegacyUserWiki(legacyWikiDir, filepath.Join(staging, "wiki")); err != nil {
			return fail(err)
		}
	}
	// 3. tenant.json（进程内核对租户归属）+ 原子 rename → active。
	if err := writeTenantMeta(filepath.Join(staging, "tenant.json"), tenantID); err != nil {
		return fail(err)
	}
	if err := os.Rename(staging, active); err != nil {
		// 并发竞态：另一 goroutine 刚完成 rename（active 已存在）
		if _, statErr := os.Stat(active); statErr == nil {
			_ = os.RemoveAll(staging)
			return nil
		}
		return fail(fmt.Errorf("激活租户目录: %w", err))
	}
	return nil
}

// copyLegacyUserWiki 复制旧 wiki 的用户记忆子树（用户记忆/客户 + 用户记忆/使用者）
// 到租户 wiki 根。无该子树则静默跳过（老库可能只有系统知识）。
func copyLegacyUserWiki(src, dst string) error {
	srcUser := filepath.Join(src, "用户记忆")
	if _, err := os.Stat(srcUser); err != nil {
		return nil
	}
	dstUser := filepath.Join(dst, "用户记忆")
	return filepath.Walk(srcUser, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcUser, path)
		target := filepath.Join(dstUser, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		// 权限 0600：含客户/使用者画像，最小权限（与 longterm 写盘纪律一致）。
		return copyFile(path, target)
	})
}

// copyFile 复制单个文件（权限 0600）。
func copyFile(src, dst string) error {
	if dir := filepath.Dir(dst); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o600)
}

// ForTenant 返回租户运行时（懒加载缓存）。数据面未初始化（注册失败/目录缺失）
// 返回 ErrNotProvisioned。
func (r *Registry) ForTenant(ctx context.Context, tenantID string) (*Runtime, error) {
	if !ValidTenantID(tenantID) {
		return nil, fmt.Errorf("非法租户 ID: %q", tenantID)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if rt, ok := r.runtimes[tenantID]; ok {
		return rt, nil
	}
	rt, err := r.buildRuntime(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	r.runtimes[tenantID] = rt
	return rt, nil
}

// buildRuntime 打开租户 history.db + wiki 覆盖层，装配完整数据面。
// 调用方需持写锁。不重复 Load system 层（平台只建一次）。
func (r *Registry) buildRuntime(ctx context.Context, tenantID string) (*Runtime, error) {
	root := r.tenantDir(tenantID)
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotProvisioned, tenantID)
	}
	hist, err := history.OpenTenant(tenantID, filepath.Join(root, "history.db"))
	if err != nil {
		return nil, fmt.Errorf("打开租户历史库: %w", err)
	}
	tenantWiki := longterm.NewWikiStore(filepath.Join(root, "wiki"))
	if err := tenantWiki.Load(); err != nil {
		_ = hist.Close()
		return nil, fmt.Errorf("加载租户 wiki: %w", err)
	}
	// 复合记忆：system 只读基线 + tenant 覆盖层（product→system；customer/user→tenant；
	// threat/compliance/industry→合并 tenant 优先）。两层均已加载，不再调 Load。
	composite := longterm.NewCompositeStore(
		longterm.NewSystemStore(r.systemWiki),
		longterm.NewTenantStore(tenantWiki),
	)

	sessions := shortterm.NewSessionManager()
	tr := tools.NewRegistry(composite)
	asm := assembler.New(composite, "") // M3：按 scope.UserID 解析当前使用者画像
	rv := review.New(composite)
	ag := agent.New(r.modelMgr, tr, asm, sessions)
	ag.SetMaxIterations(r.maxIters)

	// 全部回调闭包锁定本租户 history/wiki——Agent 无全局存储依赖。
	// P0-04：checkpoint 写回当前分支（空=main），读分支时取共享前缀+分支自身。
	ag.SetCheckpointSink(func(sessionID, branch string, cp *domain.Checkpoint) {
		_ = hist.AppendCheckpoint(sessionID, branch, cp)
	})
	ag.SetCheckpointSource(hist.ListCheckpoints)
	ag.SetCustomerResolver(func(sessionID string) string {
		if det, err := hist.GetSessionInternal(sessionID, ""); err == nil {
			return det.Session.Customer
		}
		return ""
	})
	ag.SetHistorySearcher(func(scope *domain.TenantScope, sessionID, query string, limit int) ([]history.MessageHit, error) {
		// P0-03：history_search 必须遵守用户级可见性——把登录作用域传给
		// SearchMessages，非 admin 只检索自己 owner 的会话，不得跨用户翻历史。
		return hist.SearchMessages(scope, sessionID, query, limit)
	})
	ag.SetProductResolver(func(name string) bool {
		// P0-02：analysis_submit 证据门禁的产品存在性校验（本租户 CompositeStore）
		_, err := composite.GetProduct(name)
		return err == nil
	})
	ag.SetCustomerBinder(func(sessionID, customer string) error {
		return hist.BindCustomerOnce(sessionID, customer)
	})
	if r.leads != nil {
		ag.SetLeads(r.leads)
	}

	return &Runtime{
		TenantID:  tenantID,
		History:   hist,
		Wiki:      composite,
		Sessions:  sessions,
		Tools:     tr,
		Assembler: asm,
		Review:    rv,
		Agent:     ag,
	}, nil
}

// SetAgentMaxIterations 更新全部已缓存运行时的 Agent 上限（console-config 热生效）；
// 后续新建运行时也会使用该值。
func (r *Registry) SetAgentMaxIterations(n int) {
	if n <= 0 {
		return
	}
	r.mu.Lock()
	r.maxIters = n
	for _, rt := range r.runtimes {
		rt.Agent.SetMaxIterations(n)
	}
	r.mu.Unlock()
}

// CloseTenant 关闭并丢弃某租户运行时（管理员删除租户/资源回收）。
func (r *Registry) CloseTenant(tenantID string) error {
	r.mu.Lock()
	rt, ok := r.runtimes[tenantID]
	if ok {
		delete(r.runtimes, tenantID)
	}
	r.mu.Unlock()
	if !ok {
		return nil
	}
	return rt.Close()
}

// Shutdown 关闭全部租户运行时（进程退出）。
func (r *Registry) Shutdown() {
	r.mu.Lock()
	rts := make([]*Runtime, 0, len(r.runtimes))
	for _, rt := range r.runtimes {
		rts = append(rts, rt)
	}
	r.runtimes = make(map[string]*Runtime)
	r.mu.Unlock()
	for _, rt := range rts {
		_ = rt.Close()
	}
}

// tenantMeta 是 tenant.json 的内容（进程内核对租户归属用，不含任何敏感信息）。
type tenantMeta struct {
	TenantID  string `json:"tenant_id"`
	CreatedAt string `json:"created_at"`
}

// writeTenantMeta 原子写 tenant.json。
func writeTenantMeta(path, tenantID string) error {
	meta := tenantMeta{TenantID: tenantID, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
