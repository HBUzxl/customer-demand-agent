// Command agent 启动客户需求分析智能体的 HTTP 服务。
//
// 配置完全来自 config.json（不读环境变量）。--config 可指定路径，默认 ./config.json。
// 文件不存在时自动写入模板。模型注册表与路由从配置文件加载；/api/config 的改动回写文件。
package main

import (
	"context"
	"crypto/sha256"
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
	httpapi "customer-demand-agent/internal/channel/http"
	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/history"
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
	seedWiki(cfg.DataDir, cfg.WikiDir)
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
		if err := registry.Register(m); err != nil {
			log.Printf("[warn] 注册模型 %s 失败: %v", m.Name, err)
		}
	}
	router := cfg.Router
	client := llm.NewClient()
	modelMgr := model.NewManager(client, registry, &router)

	// ── Agent 核心：自主循环 ────────────────────────────────
	ag := agent.New(modelMgr, toolRegistry, asm, sessions)
	// 断点续传：checkpoint 持久化到 SQLite，重启后恢复
	ag.SetCheckpointSink(func(sessionID string, cp *domain.Checkpoint) {
		_ = hist.AppendCheckpoint(sessionID, cp)
	})
	ag.SetCheckpointSource(hist.ListCheckpoints)
	// 跨会话客户上下文：从会话记录解析关联客户（非空时注入客户画像）
	ag.SetCustomerResolver(func(sessionID string) string {
		if det, err := hist.GetSession(sessionID); err == nil {
			return det.Session.Customer
		}
		return ""
	})
	// P1 history_search：Agent 可回溯历史对话原文（跨会话）
	ag.SetHistorySearcher(hist.SearchMessages)

	// ── 审核系统 ────────────────────────────────────────────
	reviewSvc := review.New(wikiStore)

	// ── HTTP Channel（配置回写由 store 持久化）─────────────
	// ── 后台任务域（TaskBackground，ADR-015/P11）：无记忆依赖的一次性调用 ──
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
			findings := taskbg.RunLint(taskbg.LintInput{Entries: entries})
			if len(findings) == 0 {
				t.Result = "全库健康，无发现"
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

	// C3 版本快照：外置模板按内容 hash 归档（data_dir/prompts-snapshots/）
	snapshotTemplate(cfg.DataDir)

	// C2 观测台：日志环形缓冲（tee：stderr 保留 + ring 供 tail）
	logRing := httpapi.NewLogRing(500)
	log.SetOutput(io.MultiWriter(os.Stderr, logRing))

	server := httpapi.New(store, ag, reviewSvc, wikiStore, hist, modelMgr, registry, runner, logRing)

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
// 旧项目内 ./data（历史遗留位置）存在而数据根为空时打印迁移指引（不自动搬，
// 避免误吞用户数据——人工 mv 一次即可）。
func seedWiki(dataDir, wikiDir string) {
	if entries, err := os.ReadDir(wikiDir); err == nil && len(entries) > 0 {
		return // 已有运行副本
	}
	if _, err := os.Stat("./wiki"); err == nil {
		if err := copyDir("./wiki", wikiDir); err != nil {
			log.Printf("[warn] 播种知识库失败: %v", err)
		} else {
			log.Printf("[ok] 已播种知识库：./wiki → %s（首次启动）", wikiDir)
		}
	}
	if dataDir != "./data" {
		if _, err := os.Stat("./data/history.db"); err == nil {
			log.Printf("[warn] 检测到项目内旧数据 ./data —— 迁移：cp -r ./data/* %s/（或设 CDA_DATA_DIR=./data 继续用旧位置）", dataDir)
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
	for _, para := range strings.Split(content, "\n\n") {
		if strings.HasPrefix(strings.TrimSpace(para), "### 观察") {
			out = append(out, strings.TrimSpace(para))
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
