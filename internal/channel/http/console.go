package http

import (
	"net/http"
	"strings"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/taskbg"
)

// handleConsoleConfig: GET /api/console/config —— 配置中心只读视图（C1）。
// 展示后端行为的关键配置；api_key 绝不出现（只返回 has_key 布尔）。
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
			"agent_max_iterations": agent.MaxIterations,
			"default_user":         cfg.DefaultUser,
			"llm_timeout_sec":      cfg.LLMTimeoutSec,
		},
	})
}

// handleConsolePrompts: GET /api/console/prompts —— Prompt 分层视图（C1/ADR-016 可视化）。
// 返回静态模板的各段（前端分块展示并标注层号）；动态注入部分标注占位说明。
func (s *Server) handleConsolePrompts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"layers": []map[string]any{
			{"id": "L0", "name": "静态模板", "dynamic": false,
				"sections": []map[string]string{
					{"title": "角色", "body": "你是长亭科技（Chaitin）的售前需求分析助手，服务对象是长亭的销售/售前团队。"},
					{"title": "自主性指引", "body": "每次用户发言，你自己判断怎么回应……（记忆工具全程在线：查证/记录/ask_user/analysis_submit）"},
					{"title": "目标", "body": "需求理解 / 产品匹配 / 可行性判断 / 追问识别"},
					{"title": "约束", "body": "只推荐目录中产品；推荐前必须 memory_search 检索证实；置信度诚实……"},
				}},
			{"id": "L1", "name": "使用者画像（风格）", "dynamic": true,
				"note": "config.default_user 选定销售，注入输出风格段（当前：" + s.store.Get().DefaultUser + "）"},
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
		},
	})
}

// handleConsoleMemoryStats: GET /api/console/memory —— 记忆统计。
func (s *Server) handleConsoleMemoryStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]map[string]int{}
	for _, typ := range []string{"product", "threat", "compliance", "industry", "customer", "user"} {
		entries := s.storeWiki.ListEntry(typ, 0, 500)
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
		for _, t := range s.tasks.List(5) {
			tasks = append(tasks, map[string]any{"id": t.ID, "type": t.Type, "status": t.Status})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"tasks":  tasks,
	})
}

func toolDescSummary(s string) string { return s }

// 引用 taskbg 防止 unused（若 Runner 未启用时 handlers 仍可用）
var _ = taskbg.TaskConsolidate

// handleConsoleLLMAudit: GET /api/console/llm-audit —— 最近 LLM 调用审计。
func (s *Server) handleConsoleLLMAudit(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.modelMgr.AuditRecent(50)})
}
