package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"customer-demand-agent/internal/domain"
)

// ToolAnalysisSubmit 是业务工具：Agent 判断输入包含真实客户需求时，
// 通过它提交结构化分析结果（ADR-013）。区别于 memory_* 工具（读写记忆），
// 它的执行副作用是把结果挂到本轮 Trace/Event 流，返回简单确认。
const ToolAnalysisSubmit = "analysis_submit"

// ToolMissingAnswer 是业务工具：此前分析 missing_info 里的追问在后续对话中
// 得到回答时，Agent 调用它记录答案（追问闭环，避免重复追问）。
const ToolMissingAnswer = "missing_answer"

// AnalysisSubmission 是 analysis_submit 的参数：结构对齐 domain.AnalysisResult，
// 额外带 is_reanalysis 标记（Agent 自主判断是首次分析还是需求实质变化）。
type AnalysisSubmission struct {
	domain.AnalysisResult
	IsReanalysis bool `json:"is_reanalysis"` // 可选：判定为需求实质变化（换文档/需求转向）时 true
}

// analysisSubmitDef 返回 analysis_submit 的 function-calling 定义（schema 即契约）。
func analysisSubmitDef() domain.Tool {
	return domain.Tool{
		Type: "function",
		Function: domain.ToolFunction{
			Name: ToolAnalysisSubmit,
			Description: "提交一份完整的客户需求分析结果。当且仅当你判断用户输入包含真实的客户安全需求" +
				"（而非寒暄、闲聊或简单提问），且你已完成必要的记忆检索与思考时调用。" +
				"覆盖语义：同一轮内多次调用，最后一次生效（覆盖前面的）；跨轮次若需求实质变化" +
				"（换客户文档/需求转向），再次调用并置 is_reanalysis=true。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"demand_analysis": map[string]any{
						"type":        "string",
						"description": "需求理解（中文）：把客户业务语言翻译成具体安全需求",
					},
					"matched_products": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name":       map[string]any{"type": "string"},
								"confidence": map[string]any{"type": "number", "description": "置信度 0-1"},
								"reason":     map[string]any{"type": "string"},
								"suggestion": map[string]any{"type": "string", "description": "销售可直接使用的推荐话术"},
							},
							"required": []string{"name", "confidence", "reason"},
						},
					},
					"feasibility": map[string]any{
						"type": "string",
						"enum": []string{"direct", "custom", "partner", "reject"},
					},
					"feasibility_detail": map[string]any{"type": "string"},
					"missing_info": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"is_reanalysis": map[string]any{
						"type":        "boolean",
						"description": "需求实质变化（换客户文档/需求转向）时 true；默认 false",
					},
				},
				"required": []string{"demand_analysis", "feasibility"},
			},
		},
	}
}

// parseSubmission 解析并校验 analysis_submit 的参数。
func parseSubmission(raw json.RawMessage) (*AnalysisSubmission, error) {
	var s AnalysisSubmission
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("参数解析: %w", err)
	}
	if s.DemandAnalysis == "" {
		return nil, fmt.Errorf("demand_analysis 不能为空")
	}
	switch s.Feasibility {
	case domain.FeasibilityDirect, domain.FeasibilityCustom,
		domain.FeasibilityPartner, domain.FeasibilityReject:
	default:
		return nil, fmt.Errorf("feasibility 必须是 direct/custom/partner/reject 之一，got %q", s.Feasibility)
	}
	return &s, nil
}

// validateSubmission 是 analysis_submit 的证据门禁（P0-02）。返回拒绝原因（空=通过）：
//  1. 每个 matched product 必须真实存在于产品库（不许编造产品名）；
//  2. confidence 必须在 [0,1]（越界即拒绝）；
//  3. 必须有匹配理由（能力依据，不许空推荐）；
//  4. 当轮必须 memory_get 过该产品主页（读详情核验能力后才许推荐，防凭目录臆断）。
//
// 拒绝时结果回填给模型修正（received:false），模型补证据后重新调用。
func (a *Agent) validateSubmission(s *AnalysisSubmission, st *turnState) string {
	// P0-02b 一致性门禁：用户把未证实的交付/商业/集成条件作为核心诉求时，
	// 模型不能一边在 feasibility_detail/missing_info 写“需确认”，一边把整体
	// 判为 direct。此类用例必须降为 custom/partner，形成可执行的确认闭环。
	if s.Feasibility == domain.FeasibilityDirect && st != nil && criticalDirectConflict(st.userText, s) {
		return "analysis_submit 可行性一致性门禁拒绝：核心诉求仍含未证实/待确认条件，feasibility 不能为 direct；请改为 custom 或 partner，并明确已证实能力与待确认边界"
	}
	if len(s.MatchedProducts) == 0 {
		return "" // 无产品匹配：reject / 纯信息缺失场景不触发产品门禁
	}
	var errs []string
	for _, mp := range s.MatchedProducts {
		name := strings.TrimSpace(mp.Name)
		if name == "" {
			errs = append(errs, "matched_products 存在空产品名")
			continue
		}
		if a.productExists != nil && !a.productExists(name) {
			errs = append(errs, fmt.Sprintf("产品 %q 不存在于产品库——先 memory_search 确认产品再推荐，不得编造", name))
			continue
		}
		if mp.Confidence < 0 || mp.Confidence > 1 {
			errs = append(errs, fmt.Sprintf("产品 %q 的 confidence 必须在 0..1，got %v", name, mp.Confidence))
		}
		if strings.TrimSpace(mp.Reason) == "" {
			errs = append(errs, fmt.Sprintf("产品 %q 缺少匹配理由（能力依据）", name))
		}
		if st == nil || !st.readProduct(name) {
			errs = append(errs, fmt.Sprintf("产品 %q 本轮未读取详情——推荐前先用 memory_get 或 memory_get_many 核验该产品主页能力与边界", name))
		}
	}
	if len(errs) > 0 {
		return "analysis_submit 证据门禁拒绝：" + strings.Join(errs, "；")
	}
	return ""
}

func criticalDirectConflict(userText string, s *AnalysisSubmission) bool {
	q := strings.ToLower(userText)
	critical := []string{"saas", "免费试用", "微信群", "不限ip", "不限制ip", "全套", "一站式", "多租户", "友商", "相比"}
	hasCritical := false
	for _, term := range critical {
		if strings.Contains(q, term) {
			hasCritical = true
			break
		}
	}
	if !hasCritical {
		return false
	}
	detail := strings.ToLower(s.FeasibilityDetail + " " + strings.Join(s.MissingInfo, " "))
	for _, term := range []string{"未证实", "需确认", "待确认", "尚未", "不明确", "缺口", "未覆盖", "产品线确认", "外部整合"} {
		if strings.Contains(detail, term) {
			return true
		}
	}
	return false
}

// readProduct 判断某产品名本轮是否已 memory_get 读详情。容忍轻微名称差异：
// 归一化（小写/去空白）后相同，或两者互相包含（如「雷池 WAF」vs
// 「雷池（SafeLine）WAF」）都算已读——避免证据门禁因产品名拼写偏差，
// 把模型卡在「读详情→重提被拒→再读→再被拒」的死循环里（2026-08-19）。
func (st *turnState) readProduct(name string) bool {
	name = normProductName(name)
	for t := range st.readProducts {
		t = normProductName(t)
		if t == name {
			return true
		}
		if t != "" && name != "" && (strings.Contains(t, name) || strings.Contains(name, t)) {
			return true
		}
	}
	return false
}

// normProductName 产品名归一化：小写、剥离括号内容（如「雷池（SafeLine）WAF」
// →「雷池WAF」）、移除全部空白——「雷池 WAF」「雷池WAF」「雷池（SafeLine）WAF」
// 归一化后均视为「雷池waf」，避免产品名拼写偏差把模型卡在门禁死循环里。
func normProductName(s string) string {
	s = strings.ToLower(stripParens(s))
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r', '　':
			return -1
		}
		return r
	}, s)
}

// stripParens 剥离全角/半角括号及其内容（rune 安全）。无配对闭合时截断到开括号前。
func stripParens(s string) string {
	runes := []rune(s)
	for {
		oi, hi := -1, -1
		for i, r := range runes {
			if r == '（' && oi < 0 {
				oi = i
			}
			if r == '(' && hi < 0 {
				hi = i
			}
			if oi >= 0 && hi >= 0 {
				break
			}
		}
		start := oi
		if hi >= 0 && (oi < 0 || hi < oi) {
			start = hi
		}
		if start < 0 {
			return string(runes)
		}
		close := -1
		for i := start + 1; i < len(runes); i++ {
			if runes[i] == '）' || runes[i] == ')' {
				close = i
				break
			}
		}
		if close < 0 {
			return string(runes[:start])
		}
		runes = append(runes[:start], runes[close+1:]...)
	}
}

// missingAnswerDef 返回 missing_answer 的 function-calling 定义（追问闭环）。
func missingAnswerDef() domain.Tool {
	return domain.Tool{
		Type: "function",
		Function: domain.ToolFunction{
			Name: ToolMissingAnswer,
			Description: "记录此前需求分析 missing_info 中某条追问已得到答案。当用户在后续对话中" +
				"给出了此前缺失的信息时调用，让追问闭环（后续分析不再重复问已回答的问题）。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"answers": map[string]any{
						"type":        "array",
						"description": "本轮得到答案的追问列表（可一次记录多条）",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"item":   map[string]any{"type": "string", "description": "原追问（与 missing_info 条目对应）"},
								"answer": map[string]any{"type": "string", "description": "得到的答案"},
							},
							"required": []string{"item", "answer"},
						},
					},
				},
				"required": []string{"answers"},
			},
		},
	}
}

// parseMissingAnswer 解析并校验 missing_answer 的参数。
func parseMissingAnswer(raw json.RawMessage) ([]domain.AnsweredInfo, error) {
	var req struct {
		Answers []domain.AnsweredInfo `json:"answers"`
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("参数解析: %w", err)
	}
	if len(req.Answers) == 0 {
		return nil, fmt.Errorf("answers 不能为空")
	}
	for i, a := range req.Answers {
		if a.Item == "" || a.Answer == "" {
			return nil, fmt.Errorf("answers[%d] 的 item/answer 不能为空", i)
		}
	}
	return req.Answers, nil
}

// ToolHistorySearch 是业务工具：检索历史对话原文（P1）。
// checkpoint 只保留压缩状态，细节（客户原话、当时怎么答的）要回捞时用它。
const ToolHistorySearch = "history_search"

// historySearchDef 返回 history_search 的 function-calling 定义。
func historySearchDef() domain.Tool {
	return domain.Tool{
		Type: "function",
		Function: domain.ToolFunction{
			Name: ToolHistorySearch,
			Description: "按关键词检索历史对话原文（跨会话）。当前会话的摘要上下文不够用、需要回溯" +
				"某个客户说过什么/之前怎么回答的细节时调用。返回最近的匹配消息。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "关键词（如客户名、需求关键词）"},
					"all_sessions": map[string]any{
						"type": "boolean", "description": "true 跨全部会话检索（默认 false 只搜当前会话）",
					},
					"limit": map[string]any{"type": "number", "description": "返回条数，默认 10，上限 50"},
				},
				"required": []string{"query"},
			},
		},
	}
}

// execHistorySearch 执行 history_search（agent 层业务工具：走 histSearch 回调）。
func (a *Agent) execHistorySearch(argsRaw string, st *turnState) string {
	var args struct {
		Query       string `json:"query"`
		AllSessions bool   `json:"all_sessions"`
		Limit       int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(argsRaw), &args); err != nil {
		return fmt.Sprintf(`{"error":"参数解析: %s"}`, jsonEscape(err.Error()))
	}
	if strings.TrimSpace(args.Query) == "" {
		return `{"error":"query 不能为空"}`
	}
	if a.histSearch == nil {
		return `{"error":"历史检索未配置"}`
	}
	searchID := st.session
	if args.AllSessions {
		searchID = ""
	}
	// P0-03：携带登录作用域执行检索——后端按 scope 强制用户级可见性
	// （非 admin 只搜自己 owner 的会话），history_search 不再能跨用户翻历史。
	hits, err := a.histSearch(st.scope, searchID, args.Query, args.Limit)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, jsonEscape(err.Error()))
	}
	data, _ := json.Marshal(map[string]any{"count": len(hits), "items": hits})
	return string(data)
}

// ToolAskUser 是业务工具：向用户提问并给出选项（F3，轮次终止式）。
// Agent 调用后本轮自然结束，SSE 推 ask_user 事件（question+options），
// 前端渲染按钮组；用户点选 = value 作为普通消息发回，missing_answer 闭环。
const ToolAskUser = "ask_user"

// askUserDef 返回 ask_user 的 function-calling 定义。
func askUserDef() domain.Tool {
	return domain.Tool{
		Type: "function",
		Function: domain.ToolFunction{
			Name: ToolAskUser,
			Description: "向用户（销售）提问并给出候选项。当关键信息缺失且能枚举选项时用（如部署环境/预算区间/行业），" +
				"用户点选即回传——比开放追问省事。一次调用结束本轮，等用户选择。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"question": map[string]any{"type": "string", "description": "要问的问题（简洁，一句）"},
					"options": map[string]any{
						"type":        "array",
						"description": "选项（2-5 个）",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"label":       map[string]any{"type": "string", "description": "按钮文案"},
								"value":       map[string]any{"type": "string", "description": "回传值（空则用 label）"},
								"description": map[string]any{"type": "string", "description": "补充说明（可选）"},
								"input":       map[string]any{"type": "string", "description": "选此项后需要用户补充的信息提示（如新建客户的名称）——前端会先弹输入框让用户填写，再以「选项文案：用户输入」回传（可选，不填则点击即回传）"},
							},
							"required": []string{"label"},
						},
					},
				},
				"required": []string{"question", "options"},
			},
		},
	}
}

// execAskUser 拦截处理：记录待答问题（随轮次结束经 done 发出），本轮即收尾。
func (a *Agent) execAskUser(argsRaw string, st *turnState) string {
	var args struct {
		Question string      `json:"question"`
		Options  []AskOption `json:"options"`
	}
	if err := json.Unmarshal([]byte(argsRaw), &args); err != nil {
		return fmt.Sprintf(`{"error":"参数解析: %s"}`, jsonEscape(err.Error()))
	}
	if strings.TrimSpace(args.Question) == "" || len(args.Options) < 2 {
		return `{"error":"question 不能为空且 options 至少 2 个"}`
	}
	for i := range args.Options {
		if args.Options[i].Value == "" {
			args.Options[i].Value = args.Options[i].Label
		}
	}
	st.pendingQuestion = &Event{Type: EventAskUser, Question: args.Question, Options: args.Options}
	return `{"received":true,"note":"本轮结束，等待用户选择"}`
}

// ToolBindCustomer 是业务工具：把当前会话绑定到某客户（客户身份 Agent 自主
// 绑定，2026-08-15 用户裁决——绑定后每轮 prompt 自动注入该客户画像）。
const ToolBindCustomer = "session_bind_customer"

// bindCustomerDef 返回 session_bind_customer 的 function-calling 定义。
func bindCustomerDef() domain.Tool {
	return domain.Tool{
		Type: "function",
		Function: domain.ToolFunction{
			Name: ToolBindCustomer,
			Description: "把当前会话绑定到某客户。调用时机：对话开始涉及具体客户的需求/场景，且本会话尚未绑定客户时" +
				"（每会话只绑一次）。绑定后每轮自动携带该客户的画像上下文。通常先 memory_list(customer) 看已有客户，" +
				"再 ask_user 问用户「这是谁家客户」，得到答案后调用本工具。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"customer": map[string]any{"type": "string", "description": "客户名（与客户记忆条目 title 一致最佳）"},
				},
				"required": []string{"customer"},
			},
		},
	}
}

// execBindCustomer 拦截处理：写会话客户绑定（经回调落库 sessions.customer）。
func (a *Agent) execBindCustomer(argsRaw string, sessionID string) string {
	var args struct {
		Customer string `json:"customer"`
	}
	if err := json.Unmarshal([]byte(argsRaw), &args); err != nil {
		return fmt.Sprintf(`{"error":"参数解析: %s"}`, jsonEscape(err.Error()))
	}
	if strings.TrimSpace(args.Customer) == "" {
		return `{"error":"customer 不能为空"}`
	}
	if a.bindCustomer == nil {
		return `{"error":"未配置绑定回调"}`
	}
	if err := a.bindCustomer(sessionID, strings.TrimSpace(args.Customer)); err != nil {
		return fmt.Sprintf(`{"error":"绑定失败: %s"}`, jsonEscape(err.Error()))
	}
	custJSON, _ := json.Marshal(strings.TrimSpace(args.Customer))
	return fmt.Sprintf(`{"status":"ok","customer":%s,"note":"后续轮次自动携带该客户画像"}`, string(custJSON))
}
