// Package assembler builds the messages array sent to the LLM.
//
// 拼装原则（见 agent-autonomy.md / memory-system.md）：不重放对话原文，
// 只注入结构化状态。System prompt 描述角色+目标+约束+工具，按 op 类型
// 注入不同的 checkpoint 上下文。
package assembler

import (
	"encoding/json"
	"fmt"
	"strings"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/memory/longterm"
)

// Assembler 把长期知识 + 短期 checkpoint + 用户输入拼成 messages。
type Assembler struct {
	knowledge *longterm.WikiStore
	userName  string // 当前使用者（销售），用于分级输出
}

// New 创建拼装器。userName 是当前使用者名（影响输出风格，见 memory-system.md 分级输出）。
func New(knowledge *longterm.WikiStore, userName string) *Assembler {
	return &Assembler{knowledge: knowledge, userName: userName}
}

// Assemble 根据 op 拼装完整的 messages 数组。
func (a *Assembler) Assemble(op domain.CheckpointOp, ctx *domain.SessionContext, userInput string) []domain.Message {
	sys := domain.Message{Role: domain.RoleSystem, Content: a.systemPrompt(op)}

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
	// 跨会话客户上下文：会话关联客户时注入其画像（同一客户多次沟通统一视图）。
	if ctx != nil && ctx.Customer != "" {
		if cp, err := a.knowledge.GetCustomerProfile(ctx.Customer); err == nil && cp != nil {
			msgs = append(msgs, domain.Message{
				Role:    domain.RoleSystem,
				Content: "【当前客户画像：" + cp.Name + "】\n" + customerJSON(cp),
			})
		}
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

	// 用户输入
	msgs = append(msgs, domain.Message{Role: domain.RoleUser, Content: userInput})
	return msgs
}

// systemPrompt 构建系统提示词：角色 + 自主性 + 目标 + 约束 + 可用产品知识。
// 自主化（ADR-013）：不规定输出格式——寒暄直接回答、需求走 analysis_submit 工具，
// 决策权在 LLM，schema 即契约。
func (a *Assembler) systemPrompt(op domain.CheckpointOp) string {
	var b strings.Builder
	b.WriteString(systemRole)
	b.WriteString("\n\n## 你是自主的\n")
	b.WriteString(systemAutonomy)
	b.WriteString("\n\n## 你的目标\n")
	b.WriteString(systemGoals)
	b.WriteString("\n\n## 约束\n")
	b.WriteString(systemConstraints)

	// 分级输出：根据使用者（销售）等级调整输出风格（memory-system.md）
	if a.userName != "" {
		if u, err := a.knowledge.GetUserProfile(a.userName); err == nil {
			b.WriteString("\n\n## 输出风格（当前销售：" + u.Name + "，" + levelLabel(u.Level) + "）\n")
			b.WriteString(styleForLevel(u.Level))
		}
	}

	// 注入全量产品知识（一期产品数量有限，全量注入确定性最高）
	if kb := a.knowledge.AllKnowledge(); kb != nil && (len(kb.Products) > 0 || len(kb.Threats) > 0) {
		b.WriteString("\n\n## 可用产品知识库\n")
		b.WriteString(renderKnowledge(kb))
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

const systemRole = `你是长亭科技（Chaitin）的售前需求分析助手，服务对象是长亭的销售/售前团队。`

const systemAutonomy = `每次用户发言，你自己判断怎么回应，没有固定流程。这是和 Agent 的对话，不是普通 chat——你的记忆工具全程在线，任何轮次都该自然使用：
- 聊天中涉及产品/威胁/合规事实 → 先 memory_search 查证再回答（不凭记忆瞎说）
- 聊天中出现新客户信息/线索 → 主动 memory_observe / memory_ensure 记录
- 此前分析的追问（missing_info）在对话中得到回答 → 调用 missing_answer 记录答案（追问闭环，避免重复追问）
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
- 结构化分析结果只通过 analysis_submit 工具提交，不要把 JSON 贴在回复文本里
- 只推荐知识库中存在的长亭产品，禁止编造产品名或能力
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

// customerJSON 把客户画像渲染为 prompt 友好的文本（跨会话客户上下文）。
func customerJSON(cp *longterm.CustomerProfile) string {
	var b strings.Builder
	if cp.Industry != "" {
		fmt.Fprintf(&b, "- 行业：%s\n", cp.Industry)
	}
	if cp.Scale != "" {
		fmt.Fprintf(&b, "- 规模：%s\n", cp.Scale)
	}
	if len(cp.TechStack) > 0 {
		fmt.Fprintf(&b, "- 技术栈：%s\n", strings.Join(cp.TechStack, "、"))
	}
	if len(cp.ExistingSecurity) > 0 {
		fmt.Fprintf(&b, "- 已有安全能力：%s\n", strings.Join(cp.ExistingSecurity, "、"))
	}
	if len(cp.PainPoints) > 0 {
		fmt.Fprintf(&b, "- 已知痛点：%s\n", strings.Join(cp.PainPoints, "、"))
	}
	if cp.ProcurementPref != "" {
		fmt.Fprintf(&b, "- 采购偏好：%s\n", cp.ProcurementPref)
	}
	if cp.Notes != "" {
		fmt.Fprintf(&b, "- 备注：%s\n", truncate(cp.Notes, 200))
	}
	if len(cp.Tags) > 0 {
		fmt.Fprintf(&b, "- 标签：%s\n", strings.Join(cp.Tags, "、"))
	}
	return b.String()
}

// renderKnowledge 把知识库渲染成 prompt 友好的文本。
func renderKnowledge(kb *longterm.KnowledgeBundle) string {
	var b strings.Builder
	if len(kb.Products) > 0 {
		b.WriteString("### 产品\n")
		for _, p := range kb.Products {
			fmt.Fprintf(&b, "- **%s**（%s）", p.Name, p.Category)
			if len(p.Aliases) > 0 {
				b.WriteString(" 别名：" + strings.Join(p.Aliases, "/"))
			}
			b.WriteString("\n")
			if p.Description != "" {
				b.WriteString("  " + truncate(p.Description, 120) + "\n")
			}
			for _, c := range p.Capabilities {
				conf := ""
				if c.Confidence > 0 {
					conf = fmt.Sprintf(" [%.1f]", c.Confidence)
				}
				fmt.Fprintf(&b, "  - %s%s：%s\n", c.Name, conf, truncate(c.Description, 80))
			}
			if len(p.Limitations) > 0 {
				b.WriteString("  边界：" + strings.Join(p.Limitations, "；") + "\n")
			}
		}
	}
	if len(kb.Threats) > 0 {
		b.WriteString("\n### 已知威胁类型\n")
		for _, t := range kb.Threats {
			fmt.Fprintf(&b, "- **%s**", t.Name)
			if len(t.RelatedProducts) > 0 {
				b.WriteString(" → 关联产品：" + strings.Join(t.RelatedProducts, "、"))
			}
			b.WriteString("\n")
		}
	}
	if len(kb.Compliances) > 0 {
		b.WriteString("\n### 合规要求\n")
		for _, c := range kb.Compliances {
			fmt.Fprintf(&b, "- **%s**", c.Name)
			if len(c.RelatedProducts) > 0 {
				b.WriteString(" → 关联产品：" + strings.Join(c.RelatedProducts, "、"))
			}
			b.WriteString("\n")
		}
	}
	if len(kb.Industries) > 0 {
		b.WriteString("\n### 行业场景\n")
		for _, i := range kb.Industries {
			fmt.Fprintf(&b, "- **%s**：痛点 %s\n", i.Name, strings.Join(i.TypicalPains, "、"))
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
