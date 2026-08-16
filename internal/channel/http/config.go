package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"customer-demand-agent/internal/history"
	"customer-demand-agent/internal/taskbg"
	"strings"
	"syscall"
	"time"

	"customer-demand-agent/internal/config"
	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/llm"
	"customer-demand-agent/internal/model"
)

// configResponse is the GET /api/config payload.
type configResponse struct {
	Models      []model.ModelConfig `json:"models"`
	Router      model.RouterConfig  `json:"router"`
	LeadManager *LeadManagerView    `json:"lead_manager,omitempty"` // 商机平台（key 掩码语义与 LLM key 一致）
}

// LeadManagerView 是 lead_manager 的 API 视图（api_key 永不回传明文：
// 已设置显示占位 "********"，未设置为空；PUT 原样发回 = 不改）。
type LeadManagerView struct {
	Enabled    bool   `json:"enabled"`
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key"`
	TimeoutSec int    `json:"timeout_sec"`
	RatePerMin int    `json:"rate_per_min"`
	Burst      int    `json:"burst"`
}

// maskedLeadManager 从配置构造掩码视图。
func maskedLeadManager(lm config.LeadManagerCfg) *LeadManagerView {
	v := &LeadManagerView{
		Enabled: lm.Enabled, BaseURL: lm.BaseURL,
		TimeoutSec: lm.TimeoutSec, RatePerMin: lm.RatePerMin, Burst: lm.Burst,
	}
	if lm.APIKey != "" {
		v.APIKey = "********"
	}
	return v
}

// handleConfigGet: GET /api/config（永不回传真实 api_key；已设置的显示占位 "********"，未设置为空）
func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	models := s.registry.All()
	for i := range models {
		if models[i].APIKey != "" {
			models[i].APIKey = "********" // 占位：表示已设置（真实 key 不出库）；PUT 原样发回=不改
		}
	}
	lm := maskedLeadManager(s.store.Get().LeadManager)
	writeJSON(w, http.StatusOK, configResponse{
		Models:      models,
		Router:      s.modelMgr.Router(),
		LeadManager: lm,
	})
}

// configPutReq is the PUT /api/config body (full replace of models + router;
// behavior 字段可选——指针不传不动，console-config 三态化)。
type configPutReq struct {
	Models             []model.ModelConfig `json:"models"`
	Router             *model.RouterConfig `json:"router"`
	DefaultUser        *string             `json:"default_user,omitempty"`
	LLMTimeoutSec      *int                `json:"llm_timeout_sec,omitempty"`
	AgentMaxIterations *int                `json:"agent_max_iterations,omitempty"`
	LeadManager        *LeadManagerView    `json:"lead_manager,omitempty"` // 可选：商机平台配置（重启生效）
}

// handleConfigPut: PUT /api/config
// 全量替换模型 + 路由，并回写 config.json（API key 与配置集中存文件）。
func (s *Server) handleConfigPut(w http.ResponseWriter, r *http.Request) {
	var req configPutReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	// lead_manager 先行校验（错误早返回，不动任何现有状态）。
	// api_key 为空或占位 "********" 时保留已存 key（与 LLM key 合并语义一致）。
	var lmNew *config.LeadManagerCfg
	if req.LeadManager != nil {
		lm := s.store.Get().LeadManager // 快照现值（key 合并用）
		v := *req.LeadManager
		if v.BaseURL != "" {
			u, perr := url.Parse(v.BaseURL)
			if perr != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				writeError(w, http.StatusBadRequest, "lead_manager.base_url 非法（须为 http/https URL）")
				return
			}
		}
		if v.APIKey == "" || v.APIKey == "********" {
			v.APIKey = lm.APIKey
		}
		lmNew = &config.LeadManagerCfg{
			Enabled: v.Enabled, BaseURL: v.BaseURL, APIKey: v.APIKey,
			TimeoutSec: v.TimeoutSec, RatePerMin: v.RatePerMin, Burst: v.Burst,
		}
	}
	var merged []model.ModelConfig
	if req.Models != nil {
		// 先快照现有真实密钥 + endpoint（reset 前的真相源）
		existingKey := map[string]string{}
		existingEndpoint := map[string]string{}
		for _, name := range s.registry.Names() {
			if c, err := s.registry.Get(name); err == nil {
				existingKey[name] = c.APIKey
				existingEndpoint[name] = c.Endpoint
			}
		}
		// 合并：传入 api_key 为空或占位 "********" → 仅当 endpoint 未变才保留旧 key；
		// 改了 endpoint 还留空/占位 → 不代填（防把已存 key 指向被篡改的 endpoint）
		merged = make([]model.ModelConfig, len(req.Models))
		for i, m := range req.Models {
			if m.APIKey == "" || m.APIKey == "********" {
				if ep, ok := existingEndpoint[m.Name]; ok && sameEndpoint(m.Endpoint, ep) {
					m.APIKey = existingKey[m.Name]
				}
			}
			merged[i] = m
		}
		_ = s.resetRegistry()
		for _, m := range merged {
			if err := s.registry.Register(m); err != nil {
				writeError(w, http.StatusBadRequest, "注册模型 %s: %v", m.Name, err)
				return
			}
		}
	}
	if req.Router != nil {
		s.modelMgr.SetRouter(req.Router)
	}
	// 回写配置文件（合并后的含密钥模型 + 路由），保留 server/wiki/history 设置
	if err := s.store.Update(func(cfg *config.Config) {
		if req.Models != nil {
			cfg.Models = merged
		}
		if req.Router != nil {
			cfg.Router = *req.Router
		}
		if lmNew != nil {
			cfg.LeadManager = *lmNew // 重启生效（不做热切换，同 jiying 语义）
		}
		// console-config：行为参数可选合并（热生效——Agent/Registry 即时重配）
		if req.DefaultUser != nil {
			cfg.DefaultUser = *req.DefaultUser
		}
		if req.LLMTimeoutSec != nil && *req.LLMTimeoutSec > 0 {
			cfg.LLMTimeoutSec = *req.LLMTimeoutSec
			s.registry.SetTimeout(time.Duration(*req.LLMTimeoutSec) * time.Second)
		}
		if req.AgentMaxIterations != nil && *req.AgentMaxIterations > 0 {
			cfg.AgentMaxIterations = *req.AgentMaxIterations
			s.agent.SetMaxIterations(*req.AgentMaxIterations)
		}
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "回写配置文件: %v", err)
		return
	}
	// 响应同样不回传真实密钥（已设置显示占位）
	for i := range merged {
		if merged[i].APIKey != "" {
			merged[i].APIKey = "********"
		}
	}
	writeJSON(w, http.StatusOK, configResponse{
		Models:      merged,
		Router:      s.modelMgr.Router(),
		LeadManager: maskedLeadManager(s.store.Get().LeadManager),
	})
}

// resetRegistry clears all registered models (for full-replace config).
func (s *Server) resetRegistry() error {
	for _, name := range s.registry.Names() {
		_ = s.registry.Delete(name)
	}
	return nil
}

// handleSessionList: GET /api/sessions?limit=&offset=
// handleSessionsSearch: GET /api/sessions/search?q=&limit= —— 会话搜索
// （标题命中优先 + 消息内容命中带 snippet）。
func (s *Server) handleSessionsSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	hits, err := s.history.SearchSessions(q, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "搜索会话: %v", err)
		return
	}
	if hits == nil {
		hits = []history.SessionSearchHit{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(hits), "items": hits})
}

func (s *Server) handleSessionList(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := s.history.ListSessions(limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "%v", err)
		return
	}
	// F0：附带运行状态（侧栏运行指示）
	type itemWithRun struct {
		history.SessionListItem
		Running bool `json:"running"`
	}
	out := make([]itemWithRun, len(items))
	for i, it := range items {
		out[i] = itemWithRun{SessionListItem: it, Running: s.runs.Running(it.SessionID)}
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(out), "items": out})
}

// handleSessionGet: GET /api/sessions/{id}?branch=main|b1-xxx
// branch 空=全分支汇总（admin 用），否则只返回该分支的消息+工具+checkpoint。
func (s *Server) handleSessionGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	branch := r.URL.Query().Get("branch")
	det, err := s.history.GetSession(id, branch)
	if err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, det)
}

// handleSessionDelete: DELETE /api/sessions/{id}
func (s *Server) handleSessionDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.history.DeleteSession(id); err != nil {
		writeError(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ── 模型探测：测试连接 + 自动获取模型列表 ───────────────────────

// testModelReq 是测试连接 / 拉取模型列表的请求体（传当前表单值，未保存也能探）。
type probeReq struct {
	Name     string `json:"name"` // 可选：已保存模型名（用于补全 api_key）
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"api_key"`
	Protocol string `json:"protocol"`
	Model    string `json:"model"`
}

// resolveKey 解析探测请求的 api key。
// 当未显式传 key 时，仅当请求 endpoint 与已保存模型 endpoint 同 host 才补全 key，
// 防止把已存密钥发往被篡改的 endpoint（凭据外泄）。
func (s *Server) resolveKey(req *probeReq) (string, error) {
	if req.APIKey != "" {
		return req.APIKey, nil
	}
	if req.Name == "" {
		return "", nil
	}
	m, err := s.registry.GetModel(req.Name)
	if err != nil {
		return "", nil // 模型不存在，不补全
	}
	if !sameEndpoint(req.Endpoint, m.Endpoint) {
		return "", fmt.Errorf("请求 endpoint 与已保存模型 %s 的 endpoint 不同源，拒绝代填 api_key（请显式填写）", req.Name)
	}
	return m.APIKey, nil
}

// handleConfigTest: POST /api/config/test —— 发一个 "hi" 测试模型连通性。
func (s *Server) handleConfigTest(w http.ResponseWriter, r *http.Request) {
	var req probeReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	if err := validateProbeEndpoint(req.Endpoint); err != nil {
		writeError(w, http.StatusBadRequest, "endpoint 不允许: %v", err)
		return
	}
	key, err := s.resolveKey(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	if key == "" {
		writeError(w, http.StatusBadRequest, "缺少 api_key")
		return
	}
	client := llm.NewClientWithHTTP(probeChatClient) // C1：探测 chat 也走硬化 transport（IP 校验+重定向再校验）
	cfg := llm.Config{
		Endpoint: req.Endpoint, APIKey: key, Protocol: req.Protocol,
		Model: req.Model, MaxTokens: 64,
	}
	start := time.Now()
	resp, err := client.Chat(r.Context(), cfg, &llm.ChatRequest{
		Messages:  []domain.Message{{Role: domain.RoleUser, Content: "hi"}},
		MaxTokens: 64,
	})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "latency_ms": latency, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "latency_ms": latency, "reply": resp.Message.Content})
}

// handleListModels: POST /api/models —— 代理网关的 /models 拉可用模型列表。
func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	var req probeReq
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	if err := validateProbeEndpoint(req.Endpoint); err != nil {
		writeError(w, http.StatusBadRequest, "endpoint 不允许: %v", err)
		return
	}
	key, err := s.resolveKey(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}
	if key == "" {
		writeError(w, http.StatusBadRequest, "缺少 api_key")
		return
	}

	// 按协议构造 /models 请求
	proto := req.Protocol
	if proto == "" {
		proto = llm.ProtocolOpenAIChat
	}
	base := trimSlash(req.Endpoint)
	var url, authHeader, authValue string
	switch proto {
	case llm.ProtocolAnthropic:
		url = base + "/v1/models"
		authHeader, authValue = "x-api-key", key
	default:
		url = base + "/models"
		authHeader, authValue = "Authorization", "Bearer "+key
	}

	hreq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, "构造请求失败（endpoint 非法？）: %v", err)
		return
	}
	hreq.Header.Set(authHeader, authValue)
	if proto == llm.ProtocolAnthropic {
		hreq.Header.Set("anthropic-version", "2023-06-01")
	}
	resp, err := probeClient.Do(hreq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "请求网关失败: %v", err)
		return
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 上限 10MB，防内存耗尽
	if err != nil {
		writeError(w, http.StatusBadGateway, "读取响应失败: %v", err)
		return
	}
	if resp.StatusCode >= 400 {
		writeError(w, http.StatusBadGateway, "网关返回 %d: %s", resp.StatusCode, truncate(string(raw), 200))
		return
	}
	// 解析 {data:[{id:...}]}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		writeError(w, http.StatusBadGateway, "解析模型列表失败: %v", err)
		return
	}
	ids := make([]string, 0, len(body.Data))
	for _, d := range body.Data {
		ids = append(ids, d.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": ids})
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ── 探测端点安全：SSRF 防护 ──────────────────────────────────────

// probeTransport 对探测出站连接做 IP 级二次校验（防 DNS rebinding：URL 校验通过后，
// 连接时再校验解析到的真实 IP）。
//
// 安全取舍：私网段（10/8、172.16/12、192.168/16）放行——本项目支持「公司内部网关」
// 测试（见 README），若禁私网会破坏该场景。被禁的是高价值 SSRF 目标：云元数据
// （169.254.169.254，可窃云凭据）、回环（本机服务 pivot）、unspecified、multicast。
// 引入鉴权后可收紧为 allowlist。
var probeTransport = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout: 5 * time.Second,
		Control: func(network, addr string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return err
			}
			if ip := net.ParseIP(host); ip != nil && isForbiddenIP(ip) {
				return fmt.Errorf("拒绝访问受保护地址 %s", ip)
			}
			return nil
		},
	}).DialContext,
}

// probeClient 是探测端点专用 HTTP client：短超时 + IP 校验 transport + 同源重定向策略。
var probeClient = &http.Client{
	Timeout:       15 * time.Second,
	Transport:     probeTransport,
	CheckRedirect: probeRedirectPolicy,
}

// probeChatClient 用于 handleConfigTest：复用 probeTransport（连接时 IP 校验，防 DNS rebinding）
// + 同源重定向策略，但超时放宽到 120s（聊天探测比 /models 慢）。C1 SSRF 真正闭环的关键。
var probeChatClient = &http.Client{
	Timeout:       120 * time.Second,
	Transport:     probeTransport,
	CheckRedirect: probeRedirectPolicy,
}

// probeRedirectPolicy 探测端点重定向策略：
// ① 拒绝跨源重定向——凭据（尤其 anthropic 的 x-api-key：Go 不像 Authorization 那样在跨源
//
//	重定向时自动剥）会随重定向泄漏到他源；只允许同源（scheme+host+port）重定向。
//
// ② 对重定向目标再做 SSRF 校验（防指向受保护 IP）。
// via[0] 恒为原始请求，故每跳都对照原始 origin（链 A→B→C 中 C 须与 A 同源）。
func probeRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 3 {
		return fmt.Errorf("过多重定向")
	}
	if len(via) == 0 {
		return nil
	}
	if !sameEndpoint(req.URL.String(), via[0].URL.String()) {
		return fmt.Errorf("拒绝跨源重定向（防凭据泄漏）：%s → %s", via[0].URL.Host, req.URL.Host)
	}
	return validateProbeEndpoint(req.URL.String())
}

// isForbiddenIP 判定 IP 是否落在探测端点禁止段：
// 回环（127/::1）、link-local（含云元数据 169.254.169.254）、unspecified、multicast。
// 私网段不在禁止之列（支持内部网关测试，见 probeTransport 注释）。
func isForbiddenIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()
}

// validateProbeEndpoint 校验探测端点：scheme 必须 http/https；字面 IP 不得落在受保护段。
// 域名类 host 的最终 IP 由 probeTransport.Dialer.Control 在连接时二次校验。
func validateProbeEndpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("非法 endpoint: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("仅支持 http/https，got %q", u.Scheme)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("endpoint 缺少 host")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && isForbiddenIP(ip) {
		return fmt.Errorf("endpoint 指向受保护地址 %s", ip)
	}
	return nil
}

// sameEndpoint 判定 a、b 是否同源（scheme + host + port，端口按 scheme 默认归一化）。
// 用于凭据保护：仅当请求 endpoint 与已保存模型 endpoint 同源，才允许代填已存 key。
// 比 host 比较更严：堵住同主机不同端口、以及 HTTPS→HTTP 降级的外泄路径。
func sameEndpoint(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	return normalizeOrigin(ua) == normalizeOrigin(ub)
}

// normalizeOrigin 归一化 URL 的 scheme+host+port（端口为空时按 scheme 默认补全）。
func normalizeOrigin(u *url.URL) string {
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" {
		switch scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		}
	}
	return scheme + "://" + host + ":" + port
}

// handleTasksList: GET /api/tasks —— 后台任务列表（观测台）。
func (s *Server) handleTasksList(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.tasks.List(50)})
}

// handleTaskConsolidate: POST /api/tasks/consolidate {type,title} —— 手动触发固化。
func (s *Server) handleTaskConsolidate(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeError(w, http.StatusServiceUnavailable, "后台任务未启用")
		return
	}
	var req struct {
		Type  string `json:"type"`
		Title string `json:"title"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "解析请求体: %v", err)
		return
	}
	if req.Type == "" || req.Title == "" {
		writeError(w, http.StatusBadRequest, "type/title 不能为空")
		return
	}
	t := s.tasks.Submit(fmt.Sprintf("cons-%d", time.Now().UnixNano()), taskbg.TaskConsolidate, req.Type+"/"+req.Title)
	writeJSON(w, http.StatusAccepted, t)
}

// handleTaskLint: POST /api/tasks/lint —— 手动触发 Wiki Lint。
func (s *Server) handleTaskLint(w http.ResponseWriter, r *http.Request) {
	if s.tasks == nil {
		writeError(w, http.StatusServiceUnavailable, "后台任务未启用")
		return
	}
	t := s.tasks.Submit(fmt.Sprintf("lint-%d", time.Now().UnixNano()), taskbg.TaskLint, "全库扫描")
	writeJSON(w, http.StatusAccepted, t)
}
