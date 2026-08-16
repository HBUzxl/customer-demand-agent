// tools.go 实现 3 个只读商机工具（leads_search / leads_get / leads_stats），
// 与 memory/tools.Registry 同构：Definitions() 给 LLM 工具定义，Execute() 分发执行。
//
// 设计基线（spec：文档未明确=无）：leads_search 仅支持 stage 筛选 + 分页，
// 无排序能力（服务端无、本地也无价值字段可排）；「高价值」口径 = stage=mql；
// stage 以外的筛选参数一律不传；stats 裸调端点；响应包络容错解析后透传。
package leads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"customer-demand-agent/internal/domain"
)

// 工具名常量（verb_object 风格，与 memory_* / history_search 并列）。
const (
	ToolSearch = "leads_search"
	ToolGet    = "leads_get"
	ToolStats  = "leads_stats"
)

// Service 持有客户端，提供工具定义与执行分发。
// Agent 注入语义：未启用（nil）时工具定义根本不注册——模型完全看不到。
type Service struct {
	client *Client
}

// New 按配置创建服务。
func New(opts Options) *Service { return &Service{client: NewClient(opts)} }

// Handles 判断 name 是否为本服务工具（Agent 分发用）。
func (s *Service) Handles(name string) bool {
	switch name {
	case ToolSearch, ToolGet, ToolStats:
		return true
	}
	return false
}

// maxPageSize 是分页上限（防单次拉爆上下文；平台文档未记载上限，客户端自律）。
const maxPageSize = 50

// Definitions 返回全部工具的 LLM function-calling 定义。
func (s *Service) Definitions() []domain.Tool {
	return []domain.Tool{
		{Type: "function", Function: searchDef()},
		{Type: "function", Function: getDef()},
		{Type: "function", Function: statsDef()},
	}
}

// Execute 分发执行一次工具调用，返回结果字符串（JSON，回填给 LLM）。
// 参数校验失败返回 error（Agent 包成 {"error":...}）；平台侧错误映射为
// 带行动指引的 JSON 字符串（不重试轰炸，让 LLM 按指引向用户说明）。
func (s *Service) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	switch name {
	case ToolSearch:
		return s.execSearch(ctx, args)
	case ToolGet:
		return s.execGet(ctx, args)
	case ToolStats:
		return s.execStats(ctx, args)
	default:
		return "", fmt.Errorf("未知工具: %s", name)
	}
}

// ── leads_search：检索线索列表（GET /api/leads） ──────────────

func searchDef() domain.ToolFunction {
	return domain.ToolFunction{
		Name: ToolSearch,
		Description: "检索商机平台（Lead Manager）的线索列表。使用要点：" +
			"①查「高价值线索」→ stage=mql（MQL=市场认可线索的质量分层，这是唯一支持的筛选）；" +
			"②平台无排序能力，结果按平台默认顺序返回——呈现时不要声称「按价值排序」；" +
			"③按客户名找线索 → 拉页后在返回的 items 里自行匹配客户名，未命中时如实说明「仅扫描了前 N 条」，可用 page 翻页继续；" +
			"④结果含商业敏感数据，只向当前用户呈现，不要写入 memory_* 或外发。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"stage": map[string]any{
					"type":        "string",
					"description": "阶段筛选（如 mql）。唯一支持的筛选参数，其余筛选条件平台未提供",
				},
				"page":      map[string]any{"type": "number", "description": "页码，默认 1"},
				"page_size": map[string]any{"type": "number", "description": "每页条数，默认 10，上限 50"},
			},
		},
	}
}

type searchArgs struct {
	Stage    string `json:"stage"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

func (s *Service) execSearch(ctx context.Context, args json.RawMessage) (string, error) {
	var a searchArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析: %w", err)
	}
	if a.Page <= 0 {
		a.Page = 1
	}
	if a.PageSize <= 0 {
		a.PageSize = 10
	}
	if a.PageSize > maxPageSize {
		a.PageSize = maxPageSize
	}
	q := url.Values{}
	if a.Stage != "" {
		q.Set("stage", a.Stage)
	}
	q.Set("page", strconv.Itoa(a.Page))
	q.Set("page_size", strconv.Itoa(a.PageSize))

	raw, err := s.client.Get(ctx, "/api/leads?"+q.Encode())
	if err != nil {
		return errResult(err), nil
	}
	return shapeList(raw, a.Page), nil
}

// itemsKeys 是常见的列表字段名（平台响应包络未记载，容错识别后透传）。
var itemsKeys = []string{"items", "data", "list", "records", "results", "leads"}

// totalKeys 是常见的总数总数字段名。
var totalKeys = []string{"total", "total_count", "count"}

// shapeList 把平台列表响应整形为 {"count":N,"total":T,"page":P,"items":[...]}。
// 容错解析：顶层数组、或对象中任一常见 items 字段名 → 统一形状；识别不出
// 则整体透传（LLM 直接读原始响应，联调冒烟后按实况收紧）。
func shapeList(raw []byte, page int) string {
	// 形状 1：顶层即数组
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		return marshalList(len(arr), -1, page, arr)
	}
	// 形状 2：对象，找常见 items 字段
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err == nil {
		for _, k := range itemsKeys {
			if v, ok := obj[k]; ok {
				if err := json.Unmarshal(v, &arr); err == nil {
					total := -1
					for _, tk := range totalKeys {
						if tv, ok := obj[tk]; ok {
							if err := json.Unmarshal(tv, &total); err == nil {
								break
							}
							total = -1
						}
					}
					return marshalList(len(arr), total, page, arr)
				}
			}
		}
	}
	// 形状 3：识别不出 → 原样透传（附分页提示）
	return fmt.Sprintf(`{"page":%d,"raw":%s}`, page, rawJSONOrString(raw))
}

func marshalList(count, total, page int, items []json.RawMessage) string {
	if items == nil {
		items = []json.RawMessage{}
	}
	totalStr := "null"
	if total >= 0 {
		totalStr = strconv.Itoa(total)
	}
	return fmt.Sprintf(`{"count":%d,"total":%s,"page":%d,"items":[%s]}`,
		count, totalStr, page, joinRaw(items))
}

// joinRaw 把 RawMessage 片段拼成逗号分隔（保持原始空白规整）。
func joinRaw(items []json.RawMessage) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = string(json.RawMessage(it))
	}
	return strings.Join(parts, ",")
}

// rawJSONOrString：合法 JSON 原样嵌入，否则按 JSON 字符串嵌入。
func rawJSONOrString(raw []byte) string {
	if json.Valid(raw) {
		return string(raw)
	}
	b, _ := json.Marshal(string(raw))
	return string(b)
}

// ── leads_get：线索详情（GET /api/leads/:id[，/activities]） ──

func getDef() domain.ToolFunction {
	return domain.ToolFunction{
		Name: ToolGet,
		Description: "读取单条线索的详情（leads_search 结果里拿到线索 ID 后用它精读）。" +
			"需要跟进轨迹（沟通/状态变更记录）时置 with_activities=true（会串行再调一次活动接口）。" +
			"结果含商业敏感数据，只向当前用户呈现，不要写入 memory_* 或外发。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{"type": "string", "description": "线索 ID（必填）"},
				"with_activities": map[string]any{
					"type":        "boolean",
					"description": "true 时同时拉取该线索的活动记录（两次 API 调用），默认 false",
				},
			},
			"required": []string{"id"},
		},
	}
}

type getArgs struct {
	ID             string `json:"id"`
	WithActivities bool   `json:"with_activities"`
}

func (s *Service) execGet(ctx context.Context, args json.RawMessage) (string, error) {
	var a getArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析: %w", err)
	}
	a.ID = strings.TrimSpace(a.ID)
	if a.ID == "" {
		return "", errors.New("id 不能为空")
	}
	if strings.ContainsAny(a.ID, "/?#") {
		return "", errors.New("id 非法（不能包含 / ? #）")
	}

	lead, err := s.client.Get(ctx, "/api/leads/"+url.PathEscape(a.ID))
	if err != nil {
		return errResult(err), nil
	}
	if !a.WithActivities {
		return fmt.Sprintf(`{"id":%s,"lead":%s}`, quoteJSON(a.ID), rawJSONOrString(lead)), nil
	}
	acts, err := s.client.Get(ctx, "/api/leads/"+url.PathEscape(a.ID)+"/activities")
	if err != nil {
		// 详情已到手：活动失败不整体报错，降级标注（LLM 据此向用户说明）。
		return fmt.Sprintf(`{"id":%s,"lead":%s,"activities":%s}`,
			quoteJSON(a.ID), rawJSONOrString(lead), errResult(err)), nil
	}
	return fmt.Sprintf(`{"id":%s,"lead":%s,"activities":%s}`,
		quoteJSON(a.ID), rawJSONOrString(lead), rawJSONOrString(acts)), nil
}

// ── leads_stats：统计（4 个 stats 端点统一入口） ─────────────

// statsEndpoints 把 metric 枚举映射到平台 stats 路径（裸调，不传时间参数——
// 文档未记载 from/to 等参数，联调确认前一律不传）。
var statsEndpoints = map[string]string{
	"summary":            "/api/leads/stats",
	"conversion":         "/api/stats/conversion",
	"trends":             "/api/stats/conversion/trends",
	"ml_creators":        "/api/stats/ml-creators",
	"product_conversion": "/api/stats/product-conversion",
}

func statsDef() domain.ToolFunction {
	return domain.ToolFunction{
		Name: ToolStats,
		Description: "查询商机平台统计数据：summary（线索统计）/ conversion（转化）/ trends（转化趋势）/" +
			"ml_creators（ML 创建者）/ product_conversion（产品转化）。统计权限随系统 Token 创建人的角色，" +
			"无权限时返回 403 提示——如实向用户说明即可，不要重试或绕过。" +
			"结果含商业敏感数据，只向当前用户呈现，不要写入 memory_* 或外发。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"metric": map[string]any{
					"type":        "string",
					"enum":        []string{"summary", "conversion", "trends", "ml_creators", "product_conversion"},
					"description": "统计维度",
				},
			},
			"required": []string{"metric"},
		},
	}
}

type statsArgs struct {
	Metric string `json:"metric"`
}

func (s *Service) execStats(ctx context.Context, args json.RawMessage) (string, error) {
	var a statsArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析: %w", err)
	}
	ep, ok := statsEndpoints[a.Metric]
	if !ok {
		return "", fmt.Errorf("metric 必须是 summary/conversion/trends/ml_creators/product_conversion 之一，got %q", a.Metric)
	}
	raw, err := s.client.Get(ctx, ep)
	if err != nil {
		// 统计 403 是权限语义（随 Token 创建人角色），给统计专用文案。
		var ae *APIError
		if errors.As(err, &ae) && ae.StatusCode == 403 {
			return `{"error":"当前 Token 无该统计权限（HTTP 403），请勿重试，向用户说明"}`, nil
		}
		return errResult(err), nil
	}
	return fmt.Sprintf(`{"metric":%s,"data":%s}`, quoteJSON(a.Metric), rawJSONOrString(raw)), nil
}

// ── 错误映射（平台现象 → 给 LLM 看的中文行动指引） ───────────

func errResult(err error) string {
	var ae *APIError
	if errors.As(err, &ae) {
		switch {
		case ae.StatusCode == 401:
			return `{"error":"系统 Token 失效或过期（HTTP 401），请告知用户联系管理员重建并更新配置"}`
		case ae.StatusCode == 403:
			return `{"error":"无该接口权限（HTTP 403），不要重试或绕过"}`
		case ae.StatusCode == 404:
			return `{"error":"线索或接口不存在（HTTP 404），检查线索 ID 是否正确"}`
		case ae.StatusCode == 429:
			return `{"error":"请求频率超限（HTTP 429），稍后再试"}`
		case ae.StatusCode >= 500:
			return `{"error":"商机平台服务端暂时不可用（HTTP ` + strconv.Itoa(ae.StatusCode) +
				`，常见为限流依赖抖动），稍后再试，不要连续重试"}`
		}
		return fmt.Sprintf(`{"error":"商机平台返回 HTTP %d：%s"}`, ae.StatusCode, truncateQuote(ae.Body))
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return `{"error":"请求被取消或超时"}`
	}
	var te *TransportError
	if errors.As(err, &te) {
		// 超时/连接失败/跨 host 重定向——统一按「连不上」口径（最常见原因是未连 VPN）。
		return `{"error":"连不上商机平台（可能未连公司 VPN），请告知用户"}`
	}
	return fmt.Sprintf(`{"error":%s}`, quoteJSON(err.Error()))
}

func truncateQuote(s string) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > 120 {
		r = append(r[:120], []rune("…")...)
	}
	return quoteJSON(string(r))
}

func quoteJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
