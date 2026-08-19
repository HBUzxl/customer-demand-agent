// Package assembler builds the messages array sent to the LLM.
//
// 拼装原则（见 agent-autonomy.md / memory-system.md）：不重放对话原文，
// 只注入结构化状态。System prompt 描述角色+目标+约束+工具，按 op 类型
// 注入不同的 checkpoint 上下文。
package assembler

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/memory/longterm"
)

// Assembler 把长期知识 + 短期 checkpoint + 用户输入拼成 messages。
// knowledge 是 longterm.Store 接口（Composite）——多租户下每个租户独立的
// 复合记忆注入（产品目录取系统基线，客户/使用者取本租户覆盖层）。
type Assembler struct {
	knowledge    longterm.Store
	userName     string // legacy 无登录作用域渠道的使用者名回退
	template     string // 外置模板（C3：prompts/system.md；缺失回退内置拼接）
	leadsEnabled bool   // 商机平台已接入（注入数据源身份行；未接入不注入，模型无感知）
}

// New 创建拼装器。Web 多租户按登录 scope.UserID 解析使用者；userName 仅供
// 无登录作用域的 legacy 渠道回退（影响输出风格，见 memory-system.md 分级输出）。
func New(knowledge longterm.Store, userName string) *Assembler {
	return &Assembler{knowledge: knowledge, userName: userName, template: loadTemplate()}
}

// SetLeadsEnabled 标记商机平台数据源是否已接入（Agent.SetLeads 联动调用）。
func (a *Assembler) SetLeadsEnabled(on bool) { a.leadsEnabled = on }

// loadTemplate 加载外置模板（C3）。文件按「## 段名」组织，四个已知段
// （角色/自主性指引/目标/约束）的正文**直接采用外置文件内容**——编辑
// prompts/system.md 重启即生效；已知段名映射为内置标题（如「自主性指引」
// →「## 你是自主的」）保证 systemPrompt 消费方的段落锚点稳定。文件缺失
// 或为空时回退内置常量拼接。未识别段（自定义扩展）原样追加在尾部。
func loadTemplate() string {
	data, err := os.ReadFile("prompts/system.md")
	if err != nil || len(data) == 0 {
		return systemRole + "\n\n## 你是自主的\n" + systemAutonomy + "\n\n## 你的目标\n" + systemGoals + "\n\n## 约束\n" + systemConstraints
	}
	heading := map[string]string{
		"角色":    "",
		"自主性指引": "## 你是自主的",
		"目标":    "## 你的目标",
		"约束":    "## 约束",
	}
	var b strings.Builder
	var extra []string
	inExtra := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if h, ok := strings.CutPrefix(trimmed, "## "); ok {
			if title, knownSeg := heading[h]; knownSeg {
				inExtra = false
				if title != "" {
					b.WriteString("\n" + title + "\n")
				} else {
					b.WriteString(h + "\n") // 角色段：文件首段无标准标题头，保留段名作锚
				}
				continue
			}
			extra = append(extra, "## "+h)
			inExtra = true
			continue
		}
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue // 顶部注释/空行不进 prompt
		}
		if inExtra {
			extra = append(extra, line)
		} else {
			b.WriteString(line + "\n")
		}
	}
	if len(extra) > 0 {
		b.WriteString("\n" + strings.Join(extra, "\n") + "\n")
	}
	return b.String()
}

// TemplateRaw 返回外置模板原文（console C3 只读展示用）。
func (a *Assembler) TemplateRaw() string {
	data, err := os.ReadFile("prompts/system.md")
	if err != nil {
		return ""
	}
	return string(data)
}

// Assemble 根据 op 拼装完整的 messages 数组。
// scope 提供当前登录作用域（多租户）：按 scope.UserID 解析本租户使用者画像，
// 注入输出风格段；nil 回退 legacy userName（dingtalk/旧单租户）。
func (a *Assembler) Assemble(scope *domain.TenantScope, op domain.CheckpointOp, ctx *domain.SessionContext, userInput string) []domain.Message {
	sys := domain.Message{Role: domain.RoleSystem, Content: a.systemPrompt(op, scope)}

	var msgs []domain.Message
	msgs = append(msgs, sys)

	// 注入最近的分析上下文：用链上最近带 Analysis 的 checkpoint（LastAnalysis），
	// 而非 current/previous——连续纯聊天 followup 会把它们挤掉，但分析上下文
	// 必须跨任意多轮聊天保持（审计修复：防分析上下文被聊天轮挤出 prompt）。
	if ctx != nil && ctx.LastAnalysis != nil {
		msgs = append(msgs, domain.Message{
			Role:    domain.RoleSystem,
			Content: "【最近的分析结果】\n" + checkpointJSON(ctx.LastAnalysis),
		})
	}
	// 对话近况：最近一条 checkpoint（可能是纯聊天 followup，让 Agent 知道刚才聊了什么）
	if ctx != nil && ctx.Current != nil && ctx.Current != ctx.LastAnalysis {
		msgs = append(msgs, domain.Message{
			Role:    domain.RoleSystem,
			Content: "【最近的对话状态】\n" + checkpointJSON(ctx.Current),
		})
	}
	// 跨会话客户上下文（ADR-016 L2）：只注入"身份"一行——画像是知识，
	// 走 memory_search type=customer 按名取，不注入内容。
	if ctx != nil && ctx.Customer != "" {
		msgs = append(msgs, domain.Message{
			Role:    domain.RoleSystem,
			Content: "【当前会话客户】" + ctx.Customer + "（需要画像细节时用 memory_search 检索该客户）",
		})
	}
	// 追问闭环：链上已回答的追问（missing_answer 记录），避免重复追问。
	if ctx != nil && len(ctx.Answered) > 0 {
		var b strings.Builder
		b.WriteString("【已回答的追问（不要再重复问）】\n")
		for _, ai := range ctx.Answered {
			fmt.Fprintf(&b, "- %s → %s\n", ai.Item, ai.Answer)
		}
		msgs = append(msgs, domain.Message{Role: domain.RoleSystem, Content: b.String()})
	}

	// 商机数据源身份行（lead-manager 接入，同客户身份行模式）：启用时注入
	// 一行让模型知道工具可用；未启用不注入（工具也未注册，模型无感知）。
	if a.leadsEnabled {
		msgs = append(msgs, domain.Message{
			Role: domain.RoleSystem,
			Content: "【外部数据源】商机平台（Lead Manager）已接入：leads_search（线索列表）/ leads_get（线索详情）/ leads_stats（商机统计）。" +
				"涉及商机/线索/MQL/转化数据时用这些工具查询；工具返回的错误提示已含行动指引，按指引向用户说明。",
		})
	}

	// 用户输入
	msgs = append(msgs, domain.Message{Role: domain.RoleUser, Content: userInput})
	return msgs
}

// systemPrompt 构建系统提示词：角色 + 自主性 + 目标 + 约束 + 可用产品知识。
// 自主化（ADR-013）：不规定输出格式——寒暄直接回答、需求走 analysis_submit 工具，
// 决策权在 LLM，schema 即契约。
func (a *Assembler) systemPrompt(op domain.CheckpointOp, scope *domain.TenantScope) string {
	var b strings.Builder
	b.WriteString(a.template) // C3：外置模板（含角色/自主性/目标/约束四段）

	// 分级输出：按登录用户（scope.UserID）解析本租户使用者画像（多租户，替代
	// 全局 default_user）；legacy/dingtalk 无 scope 时回退按 userName 名解析。
	if name, lvl, style := a.userStyle(scope); name != "" {
		b.WriteString("\n\n## 输出风格（当前使用者：" + name + "，" + lvl + "）\n")
		b.WriteString(style)
	}

	// 产品目录索引（ADR-016 L3a）：只注入"有什么"（产品线分组 + 名字+一句话+别名），
	// 不注入内容——能力/场景/竞品等细节走 memory_search 定位、memory_get 阅读。
	// 产品线是官方分类（主页 tags 首位），注入它避免模型自行语义分类。
	if products := a.knowledge.AllProducts(); len(products) > 0 {
		b.WriteString("\n\n## 长亭产品目录（按产品线分组；详情经 memory_search 定位、memory_get 阅读）\n")
		groups := groupByLine(products)
		for _, g := range groups {
			fmt.Fprintf(&b, "\n### %s\n", g.line)
			for _, l := range g.items {
				b.WriteString(l + "\n")
			}
		}
	}

	// 会话状态提示：只描述事实，不规定行为（决策权在 LLM）。
	b.WriteString("\n\n## 会话状态\n")
	switch op {
	case domain.OpInitial:
		b.WriteString("这是一个新会话的第一条消息，此前没有分析记录。")
	case domain.OpReanalysis:
		b.WriteString("销售似乎带来了新的客户文档/新的需求方向（与此前分析不同）。")
	default:
		b.WriteString("此前已有对话与分析上下文（见上方注入的会话状态）。")
	}
	b.WriteString("\n\n发现新客户特征或新威胁模式时，主动调用 memory_ensure / memory_observe 记录。")

	return b.String()
}

// userStyle 按当前作用域解析本租户使用者画像，返回输出风格信息
// （name/label/style；空 name=无画像，不注入）。
// 多租户：scope.UserID → 租户覆盖层使用者画像（替代全局 default_user）；
// 无 scope（legacy/dingtalk）回退按 userName 名解析。
func (a *Assembler) userStyle(scope *domain.TenantScope) (name, label, style string) {
	if scope != nil && scope.UserID != "" {
		if u, err := a.knowledge.GetUserProfileByUserID(scope.UserID); err == nil {
			return u.Name, levelLabel(u.Level), styleForLevel(u.Level)
		}
		// 自动画像写盘失败时仍以认证控制面的显示名提供标准输出风格；身份字段
		// 来自服务端会话，不接受请求体伪造。
		if strings.TrimSpace(scope.DisplayName) != "" {
			return strings.TrimSpace(scope.DisplayName), levelLabel(""), styleForLevel("")
		}
		return "", "", ""
	}
	if a.userName != "" {
		if u, err := a.knowledge.GetUserProfile(a.userName); err == nil {
			return u.Name, levelLabel(u.Level), styleForLevel(u.Level)
		}
	}
	return "", "", ""
}

const systemRole = `你是长亭科技（Chaitin）的售前需求分析助手，服务对象是长亭的销售/售前团队。`

// lineOrder 是官方产品线的展示顺序（五大产品线）；未知分类追加在后。
var lineOrder = []string{"流量安全", "端点安全", "安全平台", "安全开发", "漏洞扫描"}

type lineGroup struct {
	line  string
	items []string
}

// groupByLine 按产品线（主页 tags 首位）分组渲染目录项；无 tags 归入「其他」。
func groupByLine(products []longterm.Product) []lineGroup {
	idx := map[string]*lineGroup{}
	var order []string
	group := func(line string) *lineGroup {
		if g, ok := idx[line]; ok {
			return g
		}
		g := &lineGroup{line: line}
		idx[line] = g
		order = append(order, line)
		return g
	}
	for _, p := range products {
		line := "其他"
		if len(p.Tags) > 0 && p.Tags[0] != "" {
			line = p.Tags[0]
		}
		item := "- " + p.Name
		if p.Description != "" {
			item += "：" + p.Description
		}
		if len(p.Aliases) > 0 {
			item += "（别名：" + strings.Join(p.Aliases, "/") + "）"
		}
		group(line).items = append(group(line).items, item)
	}
	// 排序：官方顺序优先，未知分类按字典序追加
	known := map[string]int{}
	for i, l := range lineOrder {
		known[l] = i
	}
	sort.Slice(order, func(i, j int) bool {
		ki, oki := known[order[i]]
		kj, okj := known[order[j]]
		if oki && okj {
			return ki < kj
		}
		if oki != okj {
			return oki
		}
		return order[i] < order[j]
	})
	out := make([]lineGroup, 0, len(order))
	for _, l := range order {
		out = append(out, *idx[l])
	}
	return out
}

const systemAutonomy = `每次用户发言，你自己判断怎么回应，没有固定流程。这是和 Agent 的对话，不是普通 chat——你的记忆工具全程在线，任何轮次都该自然使用：
- 聊天中涉及产品/威胁/合规事实 → 先 memory_search 查证再回答（不凭记忆瞎说）；单个候选用 memory_get，多个候选优先用 memory_get_many 批量读取完整页。推荐产品与引用细节前必须读取详情证实——置信度参考能力点标注，能力边界（limitations）用于避免过度承诺
- 需要回溯之前对话的细节（客户原话、之前怎么答的、别的会话聊过什么）→ history_search 检索历史原文
- 涉及商机/线索/MQL/转化数据（当前有哪些线索、某客户线索进展、转化趋势）→ 用 leads_search / leads_get / leads_stats 检索商机平台；若工具不可用则如实说明商机平台未接入，让用户在设置里配置；结果与产品能力结合回答时，产品细节仍以 memory_get 为准（商机平台的产品口径 ≠ 产品能力事实）
- 聊天中出现新客户信息/线索 → 主动 memory_observe / memory_ensure 记录
- 新会话客户归属已由前端发送前完成；系统没有另行注入客户身份时表示用户选择了仅产品咨询，不再追问客户是谁，不尝试改绑
- 定位阶段最多 3 次 search/recall/list，详情读取最多 5 次；候选明确后立即 get/get_many → analysis_submit。预算耗尽或仍无法证明商业/交付结论时，停止搜索并标注需产品线确认
- 此前分析的追问（missing_info）在对话中得到回答 → 调用 missing_answer 记录答案（追问闭环，避免重复追问）
- 关键信息缺失且能枚举选项（部署环境/预算区间/行业等）→ 调用 ask_user 给选项让用户点选，比开放追问省事；用户的选择回来后用 missing_answer 记录
- 寒暄、闲聊、关于你自己的问题 → 轻松自然语言回答（无需查库的就直接答，不要调用 analysis_submit）
- 真实的客户需求描述（客户沟通原文、场景、痛点）→ 完整分析：
  1. 检索记忆（产品 / 客户画像 / 威胁 / 合规）
  2. 思考需求理解、产品匹配与可行性
  3. 调用 analysis_submit 提交结构化分析结果（唯一"分析专用"工具）
  4. 用自然语言给销售一个可读的总结（结论 + 推荐话术）
- 销售追问 → 基于已有上下文回答；若判断需求实质变化（换客户文档/需求转向），重新调用 analysis_submit 并置 is_reanalysis=true`

const systemGoals = `1. 需求理解：把客户的业务语言翻译成具体安全需求（如"网站老被扫"→Web扫描攻击/CC攻击/爬虫）
2. 产品匹配：对照长亭产品库，标出能覆盖该需求的产品，给出置信度（0-1）和理由
3. 可行性判断：直接覆盖 / 基础上定制 / 需整合外部 / 不建议接
4. 追问识别：标出客户没说清、需要回头确认的信息`

const systemConstraints = `- 你的回复始终是给销售看的自然语言（可用 markdown）
- 全程使用中文表达（包括思考过程、推理、中间说明与最终回答）；专有名词（产品名、公司名、人名、技术术语等）保留原文
- 结构化分析结果只通过 analysis_submit 工具提交，不要把 JSON 贴在回复文本里
- 只推荐产品目录中存在的长亭产品，禁止编造产品名或能力
- 推荐/断言任何产品能力前，必须先 memory_search 检索证实——未检索就推荐视为违规（目录只有一句话定位，细节你并不知道）
- 同名或相互矛盾的资料不能拼接成新能力；优先当前产品线目录下的详细官方主页，蜜罐/欺骗防御与 NDR/全流量检测不得混为同一产品
- 免费试用、价格、授权次数、不限 IP、标准 SaaS、微信群通知、竞品强弱等结论只有知识原文明确支持时才可肯定，否则标注未证实/需确认
- 友商事故与漏报只能按“客户反馈的本次个案”表述，不推断友商普遍技术架构；具体攻击样本在未复测时只建议 POC，不承诺天然有效或一定拦截
- “全套/一站式”存在自有产品未覆盖项、第三方测评/咨询或交付口径未证实时，feasibility 不得填 direct，必须明确伙伴整合或待确认边界
- “已有某产品”只证明资产存在，不证明已完成日志接入、联动配置或授权开通；未知集成状态必须标为待核实，不能改写成已经汇聚/联动
- 不确定时标注"需进一步确认"，不要瞎猜
- 置信度要诚实：核心能力给 0.9+，边缘能力给 0.5-0.7，不确定给 0.3 以下
- 真实需求的分析必须给出 feasibility 判断
- matched_products 可以为空（如果确实没有匹配产品），但要说明原因`

// checkpointJSON 把 checkpoint 序列化为可读的上下文。
func checkpointJSON(cp *domain.Checkpoint) string {
	var b strings.Builder
	if cp.Document != "" {
		b.WriteString("原始文档摘要：")
		b.WriteString(truncate(cp.Document, 300))
		b.WriteString("\n\n")
	}
	if cp.Analysis != nil {
		data, _ := json.MarshalIndent(cp.Analysis, "", "  ")
		b.Write(data)
	} else if cp.Question != "" {
		b.WriteString("追问：")
		b.WriteString(cp.Question)
		if cp.Answer != "" {
			b.WriteString("\n回答：")
			b.WriteString(cp.Answer)
		}
	}
	return b.String()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// levelLabel 把等级转中文标签（初级/高级原样，其余归中级）。
func levelLabel(level string) string {
	if level != "初级" && level != "高级" {
		return "中级"
	}
	return level
}

// styleForLevel 根据销售等级返回输出风格约束（分级输出）。
func styleForLevel(level string) string {
	switch level {
	case "初级":
		return `- 给直接结论，少用专业术语，多用通俗语言解释
- 每个产品匹配都要给出清晰的推荐话术（销售可直接对客户说）
- 需求理解部分简化，不展开过多技术细节`
	case "高级":
		return `- 给完整分析过程：需求翻译的逻辑、产品匹配的依据、可行性判断的推导
- 补充竞品对比与差异化话术（客户可能提到哪些竞品、怎么应对）
- 可以展开技术细节，销售能看懂`
	default:
		return `- 结论 + 关键理由，适度展开
- 重点匹配产品可适当给推荐话术
- 技术细节点到为止`
	}
}
