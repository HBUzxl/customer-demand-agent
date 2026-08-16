// leads.go 商机面板（Dashboard）HTTP 只读代理：
// 面板发现高价值线索 → 前端「AI 分析」一键交接对话（闭环的「看数据」一侧）。
// 与 Agent 工具（leads_search 等）共享同一 leads.Service 实例——进程级
// 限流（令牌桶）天然合并，面板刷屏不会挤占 Agent 的工具调用配额。
package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"customer-demand-agent/internal/leads"
)

// SetLeads 注入商机平台服务（nil = 未启用；main.go 与 agent 共享同一实例）。
// 重启生效（与工具注册同语义，无热切换）。
func (s *Server) SetLeads(ls *leads.Service) { s.leads = ls }

// handleLeadsDashboard: GET /api/leads/dashboard?stage=mql&page=1&page_size=20
// 「高价值」默认口径 stage=mql（与 leads_search 的一致口径）。
// 未接入返回 200 {"enabled":false}——前端渲染接入引导而非报错；
// 平台错误映射为面板用户可行动的中文提示（502 不区分上游细节）。
func (s *Server) handleLeadsDashboard(w http.ResponseWriter, r *http.Request) {
	if s.leads == nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	q := r.URL.Query()
	stage := q.Get("stage")
	if stage == "" {
		stage = "mql"
	}
	page := atoiOr(q.Get("page"), 1)
	pageSize := atoiOr(q.Get("page_size"), 20)
	if pageSize > leads.MaxPageSize {
		pageSize = leads.MaxPageSize // 与工具层同一封顶（参数纪律一致）
	}
	lst, err := s.leads.List(r.Context(), stage, page, pageSize)
	if err != nil {
		writeError(w, http.StatusBadGateway, "%s", leadsErrText(err))
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Enabled  bool              `json:"enabled"`
		Stage    string            `json:"stage"`
		Count    int               `json:"count"`
		Total    *int              `json:"total"`
		Page     int               `json:"page"`
		PageSize int               `json:"page_size"`
		Items    []json.RawMessage `json:"items"`
	}{
		Enabled: true, Stage: stage,
		Count: lst.Count, Total: lst.Total, Page: lst.Page,
		PageSize: pageSize, Items: lst.Items,
	})
}

// handleLeadsStats: GET /api/leads/stats?metric=summary
// KPI 统计（best-effort 语义）：统计权限随 Token 创建人角色，403/错误不
// 5xx——列表照常渲染，统计区显示原因；只有未接入才返回 enabled=false。
func (s *Server) handleLeadsStats(w http.ResponseWriter, r *http.Request) {
	if s.leads == nil {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false})
		return
	}
	metric := r.URL.Query().Get("metric")
	if metric == "" {
		metric = "summary"
	}
	data, err := s.leads.Stats(r.Context(), metric)
	if err != nil {
		var ae *leads.APIError
		var te *leads.TransportError
		if !errors.As(err, &ae) && !errors.As(err, &te) {
			// 非平台错误 = metric 参数问题（调用方问题 → 400）
			writeError(w, http.StatusBadRequest, "%s", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"enabled": true, "error": leadsErrText(err)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": true, "metric": metric, "data": json.RawMessage(data),
	})
}

// atoiOr 解析十进制整数，失败/非法回退 def。
func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// leadsErrText 平台错误 → 面板用户能行动的中文提示（不给 LLM 的行动指引，
// 也不透出上游响应体）。
func leadsErrText(err error) string {
	var ae *leads.APIError
	if errors.As(err, &ae) {
		switch {
		case ae.StatusCode == http.StatusUnauthorized:
			return "商机平台 Token 已失效，请联系管理员在设置中更新"
		case ae.StatusCode == http.StatusForbidden:
			return "无该统计权限（HTTP 403，权限随 Token 创建人角色）"
		case ae.StatusCode == http.StatusTooManyRequests:
			return "请求频率超限（HTTP 429），请稍后刷新"
		case ae.StatusCode >= 500:
			return "商机平台暂时不可用（HTTP " + strconv.Itoa(ae.StatusCode) + "），请稍后刷新"
		}
		return "商机平台返回 HTTP " + strconv.Itoa(ae.StatusCode)
	}
	var te *leads.TransportError
	if errors.As(err, &te) {
		// 超时/连接失败/跨 host 重定向——统一按「连不上」口径（最常见原因：未连 VPN）。
		return "连不上商机平台（可能未连公司 VPN），请检查网络后重试"
	}
	return err.Error()
}
