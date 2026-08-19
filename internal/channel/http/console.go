package http

import (
	"net/http"
	"strings"
)

// handleConsoleConfig: GET /api/platform/config —— 配置中心完整只读视图（C1）。
// 平台管理路由（platform_admin 专用）：展示 data_dir/wiki_dir/history_db 等平台
// 敏感路径与行为配置；api_key 绝不出现（只返回 has_key 布尔）。租户用户只见
// /api/config 安全视图（model 别名 + 掩码 key，§10.1）。
func (s *Server) handleConsoleConfig(w http.ResponseWriter, r *http.Request) {
	cfg := s.store.Get()

	// 模型视图（脱敏）
	models := []map[string]any{}
	for _, m := range cfg.Models {
		models = append(models, map[string]any{
			"name": m.Name, "endpoint": m.Endpoint, "model": m.Model,
			"temperature": m.Temperature, "max_tokens": m.MaxTokens,
			"protocol": m.Protocol, "has_key": m.APIKey != "",
		})
	}
	// fallback 链可视化
	fallback := map[string]any{
		"max_retries":     cfg.Router.Fallback.MaxRetries,
		"backoff_base_ms": cfg.Router.Fallback.BackoffMs,
		"chain":           cfg.Router.Fallback.Chain,
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"models": models,
		"router": map[string]any{"default": cfg.Router.Default, "routes": cfg.Router.Routes, "fallback": fallback},
		"data":   map[string]any{"data_dir": cfg.DataDir, "wiki_dir": cfg.WikiDir, "history_db": cfg.HistoryDB},
		"behavior": map[string]any{
			"agent_max_iterations": s.store.Get().AgentMaxIterations,
			"llm_timeout_sec":      cfg.LLMTimeoutSec,
		},
		// 商机平台（lead-manager）：只读视图，api_key 只出 has_key（同 LLM key 纪律）
		"lead_manager": map[string]any{
			"enabled":  cfg.LeadManager.Enabled,
			"base_url": cfg.LeadManager.BaseURL,
			"has_key":  cfg.LeadManager.APIKey != "",
			"active":   cfg.LeadManager.Enabled && cfg.LeadManager.APIKey != "",
		},
	})
}

// handleConsolePrompts: GET /api/console/prompts —— Prompt 分层视图（C1/ADR-016 可视化）。
// C3：L0 模板读外置 prompts/system.md 原文（前端展示真实生效内容+快照版本）。
// 多租户：模板由当前租户运行时装配器持有（system 基线 + 租户覆盖层）。
func (s *Server) handleConsolePrompts(w http.ResponseWriter, r *http.Request) {
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	tmplRaw := rt.Agent.TemplateRaw()
	writeJSON(w, http.StatusOK, map[string]any{
		"template_raw": tmplRaw,
		"layers": []map[string]any{
			{"id": "L0", "name": "静态模板（外置 prompts/system.md）", "dynamic": false,
				"raw": tmplRaw},
			{"id": "L1", "name": "使用者画像（风格）", "dynamic": true,
				"note": "按当前登录用户 user_id 自动加载本租户使用者画像；无需单独配置当前销售"},
			{"id": "L2", "name": "会话状态", "dynamic": true,
				"note": "每轮动态注入：最近的分析结果 / 对话状态 / 已答追问 / 客户身份行（会话关联客户时）"},
			{"id": "L3a", "name": "产品目录索引", "dynamic": true,
				"note": "产品名+一句话+别名（索引注入；详情走 memory_search 工具检索）"},
		},
		"tool_prompts": []map[string]string{
			{"tool": "analysis_submit", "desc": toolDescSummary("提交结构化需求分析（schema=AnalysisResult+is_reanalysis）")},
			{"tool": "missing_answer", "desc": toolDescSummary("记录追问答案（闭环）")},
			{"tool": "ask_user", "desc": toolDescSummary("向用户提问并给选项")},
			{"tool": "history_search", "desc": toolDescSummary("检索历史对话原文")},
			{"tool": "leads_search", "desc": toolDescSummary("检索商机平台线索列表（外部数据源，启用才注册）")},
			{"tool": "leads_get", "desc": toolDescSummary("读取线索详情/活动记录")},
			{"tool": "leads_stats", "desc": toolDescSummary("商机统计（权限随 Token 创建人角色）")},
		},
	})
}

// handleConsoleMemoryStats: GET /api/console/memory —— 记忆统计（当前租户视角）。
func (s *Server) handleConsoleMemoryStats(w http.ResponseWriter, r *http.Request) {
	rt, err := s.rt(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "%v", err)
		return
	}
	stats := map[string]map[string]int{}
	for _, typ := range []string{"product", "threat", "compliance", "industry", "customer", "user"} {
		entries := rt.Wiki.ListEntry(typ, 0, 500)
		m := map[string]int{"total": len(entries), "verified": 0, "pending": 0}
		for _, e := range entries {
			switch {
			case strings.Contains(string(e.Status), "verified"):
				m["verified"]++
			case strings.Contains(string(e.Status), "pending"):
				m["pending"]++
			}
		}
		stats[typ] = m
	}
	writeJSON(w, http.StatusOK, map[string]any{"stats": stats})
}

// handleConsoleHealth: GET /api/console/health —— 系统健康（C2 的一部分）。
func (s *Server) handleConsoleHealth(w http.ResponseWriter, r *http.Request) {
	tasks := []map[string]any{}
	if s.tasks != nil {
		if sc, err := s.sc(r); err == nil {
			for _, t := range s.tasks.List(&sc, 5) {
				tasks = append(tasks, map[string]any{"id": t.ID, "type": t.Type, "status": t.Status})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"tasks":  tasks,
	})
}

func toolDescSummary(s string) string { return s }

// handleConsoleLLMAudit: GET /api/platform/llm-audit —— 最近 LLM 调用审计。
// 平台管理路由（platform_admin 专用）：LLM 审计跨全部租户，不对单个租户开放（§7.5）。
func (s *Server) handleConsoleLLMAudit(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.modelMgr.AuditRecent(50)})
}
