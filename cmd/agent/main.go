// Command agent 启动客户需求分析智能体的 HTTP 服务。
//
// 配置完全来自 config.json（不读环境变量）。--config 可指定路径，默认 ./config.json。
// 文件不存在时自动写入模板。模型注册表与路由从配置文件加载；/api/config 的改动回写文件。
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/channel"
	httpapi "customer-demand-agent/internal/channel/http"
	"customer-demand-agent/internal/channel/jiying"
	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/history"
	"customer-demand-agent/internal/leads"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/memory/assembler"
	"customer-demand-agent/internal/memory/longterm"
	"customer-demand-agent/internal/memory/shortterm"
	"customer-demand-agent/internal/memory/tools"
	"customer-demand-agent/internal/model"
	"customer-demand-agent/internal/review"
	"customer-demand-agent/internal/taskbg"
)

func main() {
	configPath := flag.String("config", config.DefaultPath, "配置文件路径（config.json）")
	flag.Parse()

	store, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	cfg := store.Get()

	// ── 长期记忆：Wiki 适配器（启动时全量加载）──────────────
	// P9 数据根：首次启动播种（数据根 wiki 为空时从项目种子 ./wiki 复制——
	// Docker 挂空卷即得种子知识；种子更新后由人工重放）+ 旧 ./data 迁移提示。
	seedWiki(cfg.DataDir, cfg.WikiDir, cfg.WikiDir == "./data/wiki" && cfg.HistoryDB == "./data/history.db")
	wikiStore := longterm.NewWikiStore(cfg.WikiDir)
	if err := wikiStore.Load(); err != nil {
		log.Printf("[warn] 加载 Wiki 知识库失败（继续启动，知识库为空）: %v", err)
	} else {
		log.Printf("[ok] Wiki 知识库已加载：%s", cfg.WikiDir)
	}

	// ── 历史记录：SQLite ────────────────────────────────────
	hist, err := history.Open(cfg.HistoryDB)
	if err != nil {
		log.Fatalf("打开历史数据库失败: %v", err)
	}
	defer hist.Close()

	// ── 短期记忆：Checkpoint 链 ──────────────────────────────
	sessions := shortterm.NewSessionManager()

	// ── 记忆工具：6 个 Function Calling ─────────────────────
	toolRegistry := tools.NewRegistry(wikiStore)

	// ── Prompt 拼装器 ──────────────────────────────────────
	asm := assembler.New(wikiStore, cfg.DefaultUser)

	// ── 模型管理：从配置文件加载注册表 + 路由 ───────────────
	registry := model.NewRegistry(time.Duration(cfg.LLMTimeoutSec) * time.Second)
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
	router := cfg.Router
	client := llm.NewClient()
	modelMgr := model.NewManager(client, registry, &router)

	// ── Agent 核心：自主循环 ────────────────────────────────
	ag := agent.New(modelMgr, toolRegistry, asm, sessions)
	ag.SetMaxIterations(cfg.AgentMaxIterations) // console-config：行为参数可配置
	// 断点续传：checkpoint 持久化到 SQLite，重启后恢复
	ag.SetCheckpointSink(func(sessionID string, cp *domain.Checkpoint) {
		_ = hist.AppendCheckpoint(sessionID, cp)
	})
	ag.SetCheckpointSource(hist.ListCheckpoints)
	// 跨会话客户上下文：从会话记录解析关联客户（非空时注入客户画像）
	ag.SetCustomerResolver(func(sessionID string) string {
		if det, err := hist.GetSession(sessionID, ""); err == nil {
			return det.Session.Customer
		}
		return ""
	})
	// P1 history_search：Agent 可回溯历史对话原文（跨会话）
	ag.SetHistorySearcher(hist.SearchMessages)

	// 客户绑定 Agent 自主化（2026-08-15）：session_bind_customer 工具落库
	ag.SetCustomerBinder(func(sessionID, customer string) error {
		return hist.EnsureSession(sessionID, "", customer)
	})

	// ── 商机平台（Lead Manager，只读外部数据源）────────────
	// 启用条件：enabled && api_key 齐备，缺一不接入（工具不注册，模型无感知）。
	// Token 是系统级的：管理员配一次、全员共用；重启生效（无热切换）。
	// 实例只建一份：Agent 工具与 HTTP 面板代理共享（进程级限流合并）。
	var leadsSvc *leads.Service
	if cfg.LeadManager.Enabled && cfg.LeadManager.APIKey != "" {
		leadsSvc = leads.New(leads.Options{
			BaseURL:    cfg.LeadManager.BaseURL,
			APIKey:     cfg.LeadManager.APIKey,
			Timeout:    time.Duration(cfg.LeadManager.TimeoutSec) * time.Second,
			RatePerMin: cfg.LeadManager.RatePerMin,
			Burst:      cfg.LeadManager.Burst,
		})
		ag.SetLeads(leadsSvc)
		log.Printf("[ok] 商机平台数据源已接入（Agent 工具 + 商机面板）：%s（限流 %d/分钟，突发 %d）",
			cfg.LeadManager.BaseURL, cfg.LeadManager.RatePerMin, cfg.LeadManager.Burst)
	} else {
		log.Printf("[ok] 商机平台未接入（lead_manager.enabled=%t，api_key %s）",
			cfg.LeadManager.Enabled, hasKeyLabel(cfg.LeadManager.APIKey))
	}

	// ── 审核系统 ────────────────────────────────────────────
	reviewSvc := review.New(wikiStore)

	// ── 即应渠道（与 HTTP 共用 AgentProcessor 处理管线）──
	if cfg.Jiying.Enabled {
		opts := jiying.Config{
			AppID:  cfg.Jiying.AppID,
			Secret: cfg.Jiying.Secret,
			WSURL:  cfg.Jiying.WSURL,
			Proxy:  cfg.Jiying.Proxy,
		}
		proc := channel.NewAgentProcessor(ag, hist)
		jiyingAdapter := jiying.New(opts, proc)
		// ask_user → 平台 choice 消息（点选闭环）；话题关闭 → 驱逐短期记忆。
		proc.SetOnEvent(jiyingAdapter.HandleAgentEvent)
		jiyingAdapter.SetOnTopicClosed(func(sessionID string) {
			sessions.Delete(sessionID)
			log.Printf("[ok] 即应话题已关闭，驱逐会话短期记忆：%s", sessionID)
		})
		if cfg.Jiying.AppID == "" || cfg.Jiying.Secret == "" || cfg.Jiying.WSURL == "" {
			log.Printf("[warn] 即应渠道已启用但接入信息不完整（app_id/secret/ws_url），连接将失败")
		}
		go func() {
			if err := jiyingAdapter.Run(context.Background()); err != nil {
				log.Printf("[warn] 即应渠道退出: %v", err)
			}
		}()
		log.Printf("[ok] 即应渠道已启动（AppID：%s，代理：%s）", cfg.Jiying.AppID, orDirect(cfg.Jiying.Proxy))
	}

	// ── HTTP Channel（配置回写由 store 持久化）─────────────
	// ── 后台任务域（TaskBackground，ADR-015/P11）：无记忆依赖的一次性调用 ──
	runner := buildTaskRunner(wikiStore, modelMgr, hist)

	// C3 版本快照：外置模板按内容 hash 归档（data_dir/prompts-snapshots/）
	snapshotTemplate(cfg.DataDir)

	// C2 观测台：日志环形缓冲（tee：stderr 保留 + ring 供 tail）
	logRing := httpapi.NewLogRing(500)
	log.SetOutput(io.MultiWriter(os.Stderr, logRing))

	server := httpapi.New(store, ag, reviewSvc, wikiStore, hist, modelMgr, registry, runner, logRing)
	server.SetLeads(leadsSvc) // 商机面板只读代理（未接入时为 nil → 面板引导态）

	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 可选：托管前端静态文件
	if cfg.Server.FrontendDist != "" {
		srv.Handler = withFrontend(srv.Handler, cfg.Server.FrontendDist)
	}

	// 优雅关闭
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("收到退出信号，正在关闭…")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
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

// orDirect 空代理显示为「直连」。
func orDirect(proxy string) string {
	if proxy == "" {
		return "直连"
	}
	return proxy
}

// hasKeyLabel 密钥存在性文案（日志用，绝不输出明文 key）。
func hasKeyLabel(key string) string {
	if key != "" {
		return "已配置"
	}
	return "未配置"
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

// withFrontend wraps the API handler to also serve the built frontend (SPA fallback).
func withFrontend(apiHandler http.Handler, dist string) http.Handler {
	absDist, err := filepath.Abs(dist)
	if err != nil {
		absDist = dist
	}
	indexBytes, _ := os.ReadFile(filepath.Join(absDist, "index.html"))
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
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexBytes)
	})
}

// seedWiki 播种：数据根 wiki 目录为空且项目种子 ./wiki 存在时复制之。
// P9 迁移：项目内旧 ./data 存在且数据根为空时自动搬入（history.db + wiki/
// 全量 copy，原位不删——回退只差一次 rm）。幂等：数据根已有数据则跳过。
func seedWiki(dataDir, wikiDir string, explicitLegacy bool) {
	if dataDir != "./data" {
		if _, err := os.Stat("./data/history.db"); err == nil {
			// 数据根尚未初始化（wiki 空 + 目标 history.db 不存在）才搬
			_, histErr := os.Stat(filepath.Join(dataDir, "history.db"))
			wikiEmpty := true
			if entries, err := os.ReadDir(wikiDir); err == nil && len(entries) > 0 {
				wikiEmpty = false
			}
			if os.IsNotExist(histErr) && wikiEmpty {
				if err := copyDir("./data", dataDir); err != nil {
					log.Printf("[warn] 旧数据迁移失败（继续用种子）: %v", err)
				} else {
					log.Printf("[ok] 已迁移旧数据：./data → %s（原位保留可回退）", dataDir)
				}
			}
		}
	}
	if entries, err := os.ReadDir(wikiDir); err == nil && len(entries) > 0 {
		return // 已有运行副本（迁移带入或此前播种）
	}
	if _, err := os.Stat("./wiki"); err == nil {
		if err := copyDir("./wiki", wikiDir); err != nil {
			log.Printf("[warn] 播种知识库失败: %v", err)
		} else {
			log.Printf("[ok] 已播种知识库：./wiki → %s（首次启动）", wikiDir)
		}
	}
}

// copyDir 递归复制目录。
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
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

// collectMissQueries 收集近期 memory_search 零命中查询（该建未建输入）。
func collectMissQueries(h *history.Store) []string {
	var out []string
	rows, err := h.DB().Query(`SELECT params_json, result_json FROM tool_calls
		WHERE tool_name='memory_search' AND result_json LIKE '%"count":0%'
		ORDER BY id DESC LIMIT 200`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var params, result string
		if rows.Scan(&params, &result) != nil {
			continue
		}
		var p struct {
			Query string `json:"query"`
		}
		if json.Unmarshal([]byte(params), &p) == nil && p.Query != "" {
			out = append(out, p.Query)
		}
	}
	return out
}

// collectLintEntries 收集全库条目快照（Lint 输入）。
func collectLintEntries(w *longterm.WikiStore) []taskbg.LintEntry {
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
// 一次性 LLM 调用，无记忆依赖）。独立函数便于 HTTP 端到端测试复用真实装配。
func buildTaskRunner(wikiStore *longterm.WikiStore, modelMgr *model.Manager, hist *history.Store) *taskbg.Runner {
	var runner *taskbg.Runner
	runner = taskbg.NewRunner(func(ctx context.Context, t *taskbg.Task) error {
		switch t.Type {
		case taskbg.TaskConsolidate:
			// 固化：Detail 是 "type/title"——读条目正文中的观察注记 → LLM 抽结构 → pending 覆盖
			parts := strings.SplitN(t.Detail, "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("detail 应为 type/title: %s", t.Detail)
			}
			e, err := wikiStore.GetEntry(parts[0], parts[1])
			if err != nil {
				return fmt.Errorf("条目不存在: %w", err)
			}
			observes := extractObserves(e.Content)
			if len(observes) == 0 {
				return fmt.Errorf("该条目无观察注记可固化")
			}
			prompt := taskbg.BuildConsolidatePrompt(taskbg.ConsolidateInput{Type: parts[0], Title: parts[1], Observes: observes})
			msgs := []domain.Message{{Role: domain.RoleUser, Content: prompt}}
			resp, err := modelMgr.Chat(ctx, model.TaskAnalysis, &llm.ChatRequest{Messages: msgs})
			if err != nil {
				return fmt.Errorf("LLM 固化: %w", err)
			}
			summary, tags, content, err := taskbg.ParseConsolidateOutput(resp.Message.Content)
			if err != nil {
				return err
			}
			fm := "---\ntype: " + parts[0] + "\nstatus: pending_review\nsummary: " + summary + "\ntags: [" + strings.Join(tags, ", ") + "]\n---\n" + content
			if err := wikiStore.UpsertEntry(&longterm.Entry{Type: domain.MemoryType(parts[0]), Title: parts[1], Content: fm, Summary: summary, Tags: tags, Status: longterm.StatusPendingReview}); err != nil {
				return err
			}
			runner.SetResult(t, "已固化 "+parts[1]+"（待审核）")
			return nil
		case taskbg.TaskLint:
			entries := collectLintEntries(wikiStore)
			misses := collectMissQueries(hist)
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
			// Detail = "sessionID/firstLine"——LLM 生成标题后更新会话
			parts := strings.SplitN(t.Detail, "/", 2)
			if len(parts) != 2 {
				return fmt.Errorf("detail 应为 sessionID/firstLine")
			}
			msgs := []domain.Message{{Role: domain.RoleUser, Content: taskbg.BuildTitlePrompt(parts[1])}}
			resp, err := modelMgr.Chat(ctx, model.TaskAnalysis, &llm.ChatRequest{Messages: msgs})
			if err != nil {
				return fmt.Errorf("LLM 标题: %w", err)
			}
			title := strings.TrimSpace(resp.Message.Content)
			title = strings.Trim(title, "\u300c\u300d\"'“”")
			if title == "" || len([]rune(title)) > 40 {
				return fmt.Errorf("标题生成异常: %q", title)
			}
			if err := hist.EnsureSession(parts[0], title, ""); err != nil {
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
