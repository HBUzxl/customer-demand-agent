// Command agent 启动客户需求分析智能体的 HTTP 服务（多租户，设计文档 multi-tenant-design.md）。
//
// 配置完全来自 config.json（不读环境变量）。--config 可指定路径，默认 ./config.json。
// 文件不存在时自动写入模板。模型注册表与路由从配置文件加载；/api/config 的改动回写文件。
//
// 多租户装配（§6-§9）：认证控制面（中央身份库）+ 租户独立数据面（每租户
// tenants/<uuid>/history.db + wiki）。业务请求经登录态解析 TenantScope 后懒加载
// 对应租户运行时；平台级依赖（模型管理/系统知识/商机数据源）只建一次共享。
// 首次启动且存在旧单租户数据（config 指定的 history_db）时自动迁入 Legacy 租户。
// --setup 交互式创建平台管理员 + Legacy 租户 owner membership（创建后退出）。
package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"customer-demand-agent/internal/audit"
	"customer-demand-agent/internal/auth"
	httpapi "customer-demand-agent/internal/channel/http"
	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/identity"
	"customer-demand-agent/internal/leads"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/model"
	"customer-demand-agent/internal/taskbg"
	"customer-demand-agent/internal/tenancy"
)

// legacyTenantSlug 是存量数据自动迁入的固定租户 slug（启动引导与 -setup 共用）。
const legacyTenantSlug = "legacy"

func main() {
	configPath := flag.String("config", config.DefaultPath, "配置文件路径（config.json）")
	setupMode := flag.Bool("setup", false, "交互式创建平台管理员 + Legacy 租户 owner（创建后退出）")
	flag.Parse()

	store, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	cfg := store.Get()

	// ── 认证控制面（中央身份库）：打开早于 legacy 引导（引导要写 tenants 记录）──
	identityStore, err := identity.Open(cfg.ControlDBPath())
	if err != nil {
		log.Fatalf("打开身份库失败: %v", err)
	}
	defer identityStore.Close()
	idSvc := identity.NewService(identityStore)
	auditSvc := audit.NewService(identityStore)
	idSvc.SetAuditLogger(auditSvc.Control)
	cookieMgr := auth.NewCookieManager(cfg.IsSecureCookies())

	// ── 平台只读系统知识：播种 + 加载（跨租户共享，只建一次）────────────
	// 首次启动从项目种子 ./wiki 复制产品/行业基线（跳过用户记忆——租户私密）；
	// 存在旧单租户数据时，其产品/行业记忆叠加为系统基线（客户/使用者留待租户迁移）。
	seedSystemWiki(cfg.SystemWikiDirPath(), cfg.WikiDir)
	systemWiki := longterm.NewWikiStore(cfg.SystemWikiDirPath())
	systemStore := longterm.NewSystemStore(systemWiki)
	if err := systemStore.Load(); err != nil {
		log.Printf("[warn] 加载系统知识库失败（继续启动，知识库为空）: %v", err)
	} else {
		log.Printf("[ok] 系统知识库已加载：%s", cfg.SystemWikiDirPath())
	}

	// ── 模型管理：从配置文件加载注册表 + 路由（平台级共享）─────────────
	registry := model.NewRegistry(time.Duration(cfg.LLMTimeoutSec) * time.Second)
	registerModels(registry, cfg)
	modelMgr := model.NewManager(llm.NewClient(), registry, &cfg.Router)

	// ── 商机平台（Lead Manager）：多租户首版禁用（设计文档 §12.1）────────
	// 系统级 Token 无法证明租户归属，外部商机数据可能跨租户泄漏。leadsSvc=nil：
	// Dashboard 返回 enabled=false 引导态，Agent 不注册 leads_* 工具。等外部
	// ACL 或租户专属凭据验证完成后再恢复。
	var leadsSvc *leads.Service // nil：MT 首版禁用
	log.Println("[ok] 商机平台：多租户首版禁用（设计文档 §12.1，待租户凭据映射）")

	// ── 租户数据面：懒加载注册表（provisioner 注入注册流程回调）────────
	runtimeRegistry := tenancy.NewRegistry(modelMgr, systemWiki, cfg.TenantsDirPath(), store, leadsSvc)
	idSvc.SetProvisioner(runtimeRegistry.Provision)

	// ── 存量数据自动迁入 Legacy 租户（用户确认；幂等）──────────────────
	// 旧单租户库存在即迁：租户记录（约定 slug）与数据目录（tenants/<id>/）双重判存。
	if fileExists(cfg.HistoryDB) {
		ctx := context.Background()
		if _, err := ensureLegacyTenant(ctx, identityStore, runtimeRegistry, cfg.HistoryDB, cfg.WikiDir); err != nil {
			log.Printf("[warn] 存量数据迁入 Legacy 租户失败（可稍后处理）: %v", err)
		}
	}

	// ── -setup：交互式创建平台管理员 + Legacy owner membership，然后退出 ──
	if *setupMode {
		if err := runSetup(identityStore, runtimeRegistry, cfg); err != nil {
			log.Fatalf("-setup 失败: %v", err)
		}
		log.Println("[ok] 管理员已就绪，可启动服务并登录")
		return
	}

	// ── 登录限流 + 认证中间件 ─────────────────────────────────────────
	loginLimiter := auth.NewRateLimiter(10, time.Minute)      // §7.2：IP+账号 1 分钟 10 次
	loginIPLimiter := auth.NewRateLimiter(60, time.Minute)    // 防随机邮箱绕过组合 Key
	registerLimiter := auth.NewRateLimiter(5, 10*time.Minute) // 注册含 Argon2/建库，单 IP 更严格
	authMW := identity.NewMiddleware(idSvc, cookieMgr)

	// ── 后台任务域（TaskBackground）：exec 按 Task.TenantID 解析租户运行时 ──
	runner := buildTaskRunner(runtimeRegistry, modelMgr)

	// C3 版本快照：外置模板按内容 hash 归档（data_dir/prompts-snapshots/）
	snapshotTemplate(cfg.DataDir)

	// C2 观测台：日志环形缓冲（tee：stderr 保留 + ring 供 tail）
	logRing := httpapi.NewLogRing(500)
	log.SetOutput(io.MultiWriter(os.Stderr, logRing))

	server := httpapi.New(httpapi.Config{
		Store:    store,
		Runtimes: runtimeRegistry,
		ModelMgr: modelMgr,
		Registry: registry,
		Tasks:    runner,
		LogRing:  logRing,
		Leads:    leadsSvc, // 商机面板只读代理（未接入时为 nil → 面板引导态）
	})
	server.SetAuth(&httpapi.AuthDeps{
		Identity:            idSvc,
		Cookies:             cookieMgr,
		Middleware:          authMW,
		LoginLimiter:        loginLimiter,
		LoginIPLimiter:      loginIPLimiter,
		RegistrationLimiter: registerLimiter,
		RegistrationEnabled: cfg.IsRegistrationEnabled,
	})

	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 可选：托管前端静态文件
	if cfg.Server.FrontendDist != "" {
		srv.Handler = withFrontend(srv.Handler, cfg.Server.FrontendDist)
	}

	// 优雅关闭（含租户运行时释放）
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("收到退出信号，正在关闭…")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		runtimeRegistry.Shutdown()
	}()

	log.Printf("客户需求分析智能体启动，监听 %s（配置：%s）", cfg.Server.Addr, *configPath)
	if cfg.Server.FrontendDist != "" {
		log.Printf("[ok] 托管前端静态文件：%s", cfg.Server.FrontendDist)
	}
	if !hasAPIKey(registry) {
		log.Printf("[warn] 未配置 api_key，分析接口将返回 502（在 config.json 填 models[].api_key 后重启）")
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务退出: %v", err)
	}
	log.Println("已退出")
}

// registerModels 从配置注册模型（跳过停用），并告警回退链引用停用模型。
func registerModels(registry *model.Registry, cfg *config.Config) {
	for _, m := range cfg.Models {
		if !m.IsEnabled() {
			log.Printf("[ok] 模型 %s 已停用（enabled=false，跳过注册）", m.Name)
			continue
		}
		if err := registry.Register(m); err != nil {
			log.Printf("[warn] 注册模型 %s 失败: %v", m.Name, err)
		}
	}
	// fallback 链引用停用模型 → 启动 warn
	for _, name := range cfg.Router.Fallback.Chain {
		for _, m := range cfg.Models {
			if m.Name == name && !m.IsEnabled() {
				log.Printf("[warn] 回退链包含停用模型 %s（该跳将失败，考虑移除）", name)
			}
		}
	}
}

// ensureLegacyTenant 把旧单租户数据自动迁入 Legacy 租户（幂等）：
// ① 按约定 slug 查租户记录，已存在直接返回；② 建租户记录（created_by NULL——
// 系统引导，先于任何用户存在）→ tenancy.MigrateLegacy 初始化数据面 → 激活。
// 迁移中断可安全重试（数据目录判存幂等）。
func ensureLegacyTenant(ctx context.Context, idStore *identity.Store, reg *tenancy.Registry, legacyDB, legacyWiki string) (*identity.Tenant, error) {
	if ten, err := idStore.GetTenantBySlug(legacyTenantSlug); err == nil {
		if ten.Status != identity.TenantStatusActive {
			if merr := reg.MigrateLegacy(ctx, ten.ID, legacyDB, legacyWiki); merr != nil {
				return nil, merr
			}
			_ = idStore.UpdateTenantStatus(ten.ID, identity.TenantStatusActive)
			ten.Status = identity.TenantStatusActive // 同步内存视图（DB 已激活）
		}
		return ten, nil
	} else if !errors.Is(err, identity.ErrNotFound) {
		return nil, err
	}
	now := time.Now().UTC()
	ten := &identity.Tenant{
		ID: auth.NewRandomHex(16), Name: "Legacy 存量工作空间", Slug: legacyTenantSlug,
		Status: identity.TenantStatusProvisioning, CreatedAt: now, UpdatedAt: now,
	}
	if err := idStore.CreateSystemTenant(ten); err != nil {
		return nil, fmt.Errorf("创建 Legacy 租户: %w", err)
	}
	if err := reg.MigrateLegacy(ctx, ten.ID, legacyDB, legacyWiki); err != nil {
		_ = idStore.UpdateTenantStatus(ten.ID, identity.TenantStatusProvisioningFailed)
		return nil, fmt.Errorf("迁移存量数据: %w", err)
	}
	if err := idStore.UpdateTenantStatus(ten.ID, identity.TenantStatusActive); err != nil {
		return nil, err
	}
	ten.Status = identity.TenantStatusActive // 同步内存视图（DB 已激活）
	log.Printf("[ok] 存量数据已迁入 Legacy 租户（%s）", ten.ID)
	return ten, nil
}

// runSetup 交互式创建平台管理员账号（-setup）：
// 读邮箱 + 两次密码（golang.org/x/term 不回显），创建用户（幂等：已存在则复用），
// 确保 Legacy 租户存在并补 owner membership。任何写入不落日志。
func runSetup(idStore *identity.Store, reg *tenancy.Registry, cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("管理员邮箱: ")
	email, _ := reader.ReadString('\n')
	email = identity.NormalizeEmail(strings.TrimSpace(email))
	if !identity.ValidateEmail(email) {
		return errors.New("邮箱格式无效")
	}
	fmt.Print("管理员密码（输入不回显）: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("读取密码: %w", err)
	}
	password := string(pw)
	if err := identity.ValidatePassword(password); err != nil {
		return err
	}

	now := time.Now().UTC()
	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("密码哈希: %w", err)
	}
	// 幂等：邮箱已存在则复用（不覆盖密码——避免重复运行改坏账号）。
	u, err := idStore.GetUserByEmailNorm(email)
	if errors.Is(err, identity.ErrNotFound) {
		u = &identity.User{
			ID: auth.NewRandomHex(16), EmailNorm: email, DisplayName: "平台管理员",
			PasswordHash: hash, Status: identity.UserStatusActive, CreatedAt: now, UpdatedAt: now,
		}
		if err := idStore.CreateUser(u); err != nil {
			return fmt.Errorf("创建管理员: %w", err)
		}
		log.Printf("[ok] 已创建管理员账号：%s", email)
	} else if err != nil {
		return fmt.Errorf("查询管理员: %w", err)
	} else {
		log.Printf("[ok] 复用已有管理员账号：%s", email)
	}
	// 平台管理员标记：-setup 创建的是平台管理员（管理全局模型/系统知识/平台健康，
	// §4.2——非"超级租户"，业务数据仍按成员租户隔离）。幂等。
	if err := idStore.SetUserPlatformAdmin(u.ID, true); err != nil {
		return fmt.Errorf("授予平台管理员: %w", err)
	}

	// Legacy 租户：按约定 slug 幂等查找；无存量数据时建空工作空间（Provision 幂等）。
	ten, err := idStore.GetTenantBySlug(legacyTenantSlug)
	if errors.Is(err, identity.ErrNotFound) {
		ten = &identity.Tenant{
			ID: auth.NewRandomHex(16), Name: "Legacy 存量工作空间", Slug: legacyTenantSlug,
			Status: identity.TenantStatusProvisioning, CreatedAt: now, UpdatedAt: now,
		}
		if err := idStore.CreateSystemTenant(ten); err != nil {
			return fmt.Errorf("创建 Legacy 租户: %w", err)
		}
		if err := reg.Provision(context.Background(), ten.ID); err != nil {
			_ = idStore.UpdateTenantStatus(ten.ID, identity.TenantStatusProvisioningFailed)
			return fmt.Errorf("初始化 Legacy 工作空间: %w", err)
		}
		if err := idStore.UpdateTenantStatus(ten.ID, identity.TenantStatusActive); err != nil {
			return err
		}
		log.Printf("[ok] 已创建空 Legacy 租户（%s，无存量数据）", ten.ID)
	} else if err != nil {
		return fmt.Errorf("查询 Legacy 租户: %w", err)
	}

	if _, err := idStore.GetMembership(ten.ID, u.ID); errors.Is(err, identity.ErrNotFound) {
		if err := idStore.CreateMembership(&identity.Membership{
			TenantID: ten.ID, UserID: u.ID, Role: domain.RoleOwner,
			Status: identity.MembershipStatusActive, CreatedAt: now,
		}); err != nil {
			return fmt.Errorf("创建成员关系: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("查询成员关系: %w", err)
	}
	log.Printf("[ok] 管理员已加入 Legacy 租户（%s，owner）", ten.ID)
	return nil
}

// hasAPIKey 检查注册表中是否有任一模型配置了 api_key。
func hasAPIKey(reg *model.Registry) bool {
	for _, name := range reg.Names() {
		if c, err := reg.Get(name); err == nil && c.APIKey != "" {
			return true
		}
	}
	return false
}

// fileExists 判断路径是否为存在的文件（非目录）。
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// withFrontend wraps the API handler to also serve the built frontend (SPA fallback).
// index.html 每次请求实时从磁盘读取（前端重新构建后不重启也能生效），避免缓存旧
// 哈希资源导致白屏。
func withFrontend(apiHandler http.Handler, dist string) http.Handler {
	absDist, err := filepath.Abs(dist)
	if err != nil {
		absDist = dist
	}
	fs := http.FileServer(http.Dir(absDist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			apiHandler.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			fs.ServeHTTP(w, r)
			return
		}
		full := filepath.Join(absDist, filepath.Clean(r.URL.Path))
		if info, statErr := os.Stat(full); statErr == nil && !info.IsDir() {
			http.ServeFile(w, r, full)
			return
		}
		indexBytes, readErr := os.ReadFile(filepath.Join(absDist, "index.html"))
		if readErr != nil {
			http.Error(w, "frontend not built: "+readErr.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexBytes)
	})
}

// seedSystemWiki 播种平台只读系统知识（产品记忆 + 行业记忆）。项目 ./wiki 是
// 随版本发布的权威基线，每次启动都增量覆盖到 system wiki，避免升级后运行目录
// 永久停留在旧产品口径；旧单租户产品/行业知识只在首次初始化时叠加，客户/
// 使用者记忆仍留给 Legacy 租户迁移。
func seedSystemWiki(systemDir, legacyWikiDir string) {
	entries, readErr := os.ReadDir(systemDir)
	firstInit := readErr != nil || len(entries) == 0
	if err := os.MkdirAll(systemDir, 0o700); err != nil {
		log.Printf("[warn] 创建系统知识目录失败: %v", err)
		return
	}
	// 首次初始化先迁入旧单租户产品/行业知识；随后由当前版本种子覆盖同路径，
	// 保留旧库独有条目但不允许旧副本反向覆盖新官方页面。
	if firstInit && legacyWikiDir != "" && legacyWikiDir != "./wiki" && legacyWikiDir != systemDir {
		if _, err := os.Stat(legacyWikiDir); err == nil {
			if err := copySystemTypes(legacyWikiDir, systemDir); err != nil {
				log.Printf("[warn] 叠加旧库系统知识失败: %v", err)
			}
		}
	}
	// 当前版本项目种子始终是系统产品/行业知识的权威基线。
	if _, err := os.Stat("./wiki"); err == nil {
		if err := copySystemTypes("./wiki", systemDir); err != nil {
			log.Printf("[warn] 同步系统知识失败: %v", err)
		} else if firstInit {
			log.Printf("[ok] 已播种系统知识：./wiki → %s（首次启动）", systemDir)
		}
	}
}

// copySystemTypes 复制 src 中的系统知识类型子目录（产品记忆/行业记忆）到 dst。
// 跳过用户记忆（客户/使用者——租户私密记忆，不共享）。
func copySystemTypes(src, dst string) error {
	for _, sub := range []string{"产品记忆", "行业记忆"} {
		s := filepath.Join(src, sub)
		if _, err := os.Stat(s); err != nil {
			continue // 种子可能缺该类型
		}
		if err := copyDirRec(s, filepath.Join(dst, sub)); err != nil {
			return err
		}
	}
	return nil
}

// copyDirRec 递归复制目录（保留结构；目录 0700/文件 0600——知识含产品/行业画像）。
func copyDirRec(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		return os.WriteFile(target, data, 0o600)
	})
}

// extractObserves 从条目正文提取观察注记（### 观察 [...] 段）。
func extractObserves(content string) []string {
	var out []string
	paras := strings.Split(content, "\n\n")
	for i, para := range paras {
		if !strings.HasPrefix(strings.TrimSpace(para), "### 观察") {
			continue
		}
		// 标题段本身 + 后续紧邻的列表/正文段（到下一个标题或结尾）
		var b strings.Builder
		b.WriteString(strings.TrimSpace(para))
		for j := i + 1; j < len(paras); j++ {
			p := strings.TrimSpace(paras[j])
			if p == "" || strings.HasPrefix(p, "#") {
				break
			}
			b.WriteString("\n\n")
			b.WriteString(p)
		}
		out = append(out, b.String())
	}
	return out
}

// collectLintEntries 收集全库条目快照（Lint 输入）。
func collectLintEntries(w longterm.Store) []taskbg.LintEntry {
	var out []taskbg.LintEntry
	for _, typ := range []string{"threat", "compliance", "industry", "customer", "product"} {
		for _, e := range w.ListEntry(typ, 0, 200) {
			out = append(out, taskbg.LintEntry{
				Type: typ, Title: e.Title, Aliases: e.Aliases, Tags: e.Tags,
				Summary: e.Summary, Content: e.Content,
			})
		}
	}
	return out
}

// snapshotTemplate 把当前外置模板按内容 hash 归档（幂等——同内容不重复存）。
func snapshotTemplate(dataDir string) {
	raw, err := os.ReadFile("prompts/system.md")
	if err != nil {
		return // 无外置模板
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(raw))[:12]
	dir := filepath.Join(dataDir, "prompts-snapshots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	p := filepath.Join(dir, sum+".md")
	if _, err := os.Stat(p); err == nil {
		return // 已有同内容快照
	}
	_ = os.WriteFile(p, raw, 0o644)
	log.Printf("[ok] Prompt 模板快照：%s", p)
}

// buildTaskRunner 装配后台任务执行器（固化/Lint/标题——ADR-015 后台域：
// 一次性 LLM 调用，无记忆依赖）。多租户：exec 按 Task.TenantID 解析租户运行时，
// 操作其私有 wiki/history（跨租户任务在运行时解析前即隔离）。独立函数便于
// HTTP 端到端测试复用真实装配。
func buildTaskRunner(runtimeRegistry *tenancy.Registry, modelMgr *model.Manager) *taskbg.Runner {
	var runner *taskbg.Runner
	runner = taskbg.NewRunner(func(ctx context.Context, t *taskbg.Task) error {
		rt, err := runtimeRegistry.ForTenant(ctx, t.TenantID)
		if err != nil {
			return fmt.Errorf("解析租户运行时: %w", err)
		}
		switch t.Type {
		case taskbg.TaskConsolidate:
			// 固化：Resource.Type/TypeID = 记忆类型/条目标题——读观察注记 → LLM 抽结构
			// → pending_review 覆盖（人审闭环）。
			typ, title := t.Resource.Type, t.Resource.TypeID
			e, err := rt.Wiki.GetEntry(typ, title)
			if err != nil {
				return fmt.Errorf("条目不存在: %w", err)
			}
			observes := extractObserves(e.Content)
			if len(observes) == 0 {
				return fmt.Errorf("该条目无观察注记可固化")
			}
			prompt := taskbg.BuildConsolidatePrompt(taskbg.ConsolidateInput{Type: typ, Title: title, Observes: observes})
			msgs := []domain.Message{{Role: domain.RoleUser, Content: prompt}}
			resp, err := modelMgr.Chat(ctx, model.TaskAnalysis, &llm.ChatRequest{Messages: msgs})
			if err != nil {
				return fmt.Errorf("LLM 固化: %w", err)
			}
			summary, tags, content, err := taskbg.ParseConsolidateOutput(resp.Message.Content)
			if err != nil {
				return err
			}
			fm := "---\ntype: " + typ + "\nstatus: pending_review\nsummary: " + summary + "\ntags: [" + strings.Join(tags, ", ") + "]\n---\n" + content
			if err := rt.Wiki.UpsertEntry(&longterm.Entry{Type: domain.MemoryType(typ), Title: title, Content: fm, Summary: summary, Tags: tags, Status: longterm.StatusPendingReview}); err != nil {
				return err
			}
			runner.SetResult(t, "已固化 "+title+"（待审核）")
			return nil
		case taskbg.TaskLint:
			entries := collectLintEntries(rt.Wiki)
			misses := rt.History.RecentMissQueries(200)
			findings := taskbg.RunLint(taskbg.LintInput{Entries: entries, RecentMissQueries: misses})
			if len(findings) == 0 {
				runner.SetResult(t, "全库健康，无发现")
				return nil
			}
			var sb strings.Builder
			fmt.Fprintf(&sb, "%d 条发现：", len(findings))
			for i, f := range findings {
				if i >= 10 {
					fmt.Fprintf(&sb, "…（共 %d）", len(findings))
					break
				}
				fmt.Fprintf(&sb, "[%s] %s/%s：%s；", f.Kind, f.Type, f.Title, f.Message)
			}
			runner.SetResult(t, sb.String())
			return nil
		case taskbg.TaskTitle:
			// Resource.TypeID=session_id，Resource.Title=累计的用户提问原文——
			// LLM 归纳为简短标题后更新会话（EnsureSession 尊重 title_pinned，不覆盖手动重命名）。
			sessionID, input := t.Resource.TypeID, t.Resource.Title
			if sessionID == "" || input == "" {
				return fmt.Errorf("resource 缺 session_id/用户提问")
			}
			msgs := []domain.Message{{Role: domain.RoleUser, Content: taskbg.BuildTitlePrompt(input)}}
			resp, err := modelMgr.Chat(ctx, model.TaskAnalysis, &llm.ChatRequest{Messages: msgs})
			if err != nil {
				return fmt.Errorf("LLM 标题: %w", err)
			}
			title := strings.TrimSpace(resp.Message.Content)
			title = strings.Trim(title, "\u300c\u300d\"'“”")
			if title == "" || len([]rune(title)) > 40 {
				return fmt.Errorf("标题生成异常: %q", title)
			}
			if err := rt.History.EnsureSession("", sessionID, title, ""); err != nil {
				return err
			}
			runner.SetResult(t, "标题："+title)
			return nil
		default:
			return fmt.Errorf("未知任务类型: %s", t.Type)
		}
	})
	return runner
}
