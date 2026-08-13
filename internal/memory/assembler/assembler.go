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
}

// New 创建拼装器。
func New(knowledge *longterm.WikiStore) *Assembler {
	return &Assembler{knowledge: knowledge}
}

// Assemble 根据 op 拼装完整的 messages 数组。
func (a *Assembler) Assemble(op domain.CheckpointOp, ctx *domain.SessionContext, userInput string) []domain.Message {
	sys := domain.Message{Role: domain.RoleSystem, Content: a.systemPrompt(op)}

	var msgs []domain.Message
	msgs = append(msgs, sys)

	// 注入当前 checkpoint（追问/重分析时）
	if ctx != nil && ctx.Current != nil && op != domain.OpInitial {
		msgs = append(msgs, domain.Message{
			Role:    domain.RoleSystem,
			Content: "【上一次分析结果】\n" + checkpointJSON(ctx.Current),
		})
	}
	if ctx != nil && ctx.Previous != nil {
		msgs = append(msgs, domain.Message{
			Role:    domain.RoleSystem,
			Content: "【更早的分析上下文】\n" + checkpointJSON(ctx.Previous),
		})
	}

	// 用户输入
	msgs = append(msgs, domain.Message{Role: domain.RoleUser, Content: userInput})
	return msgs
}

// systemPrompt 构建系统提示词：角色 + 目标 + 约束 + 可用产品知识。
func (a *Assembler) systemPrompt(op domain.CheckpointOp) string {
	var b strings.Builder
	b.WriteString(systemRole)
	b.WriteString("\n\n## 你的目标\n")
	b.WriteString(systemGoals)
	// 输出格式：仅分析场景要求 JSON；追问场景要自然语言回答
	if op == domain.OpFollowup {
		b.WriteString("\n\n## 输出格式\n")
		b.WriteString("用自然语言直接回答销售的追问，不要输出 JSON，不要重复整个分析。可以引用上一次分析结果中的产品/结论。")
	} else {
		b.WriteString("\n\n## 输出格式\n")
		b.WriteString(systemOutputFormat)
	}
	b.WriteString("\n\n## 约束\n")
	b.WriteString(systemConstraints)

	// 注入全量产品知识（一期产品数量有限，全量注入确定性最高）
	if kb := a.knowledge.AllKnowledge(); kb != nil && (len(kb.Products) > 0 || len(kb.Threats) > 0) {
		b.WriteString("\n\n## 可用产品知识库\n")
		b.WriteString(renderKnowledge(kb))
	}

	b.WriteString("\n\n## 当前任务\n")
	switch op {
	case domain.OpInitial:
		b.WriteString("这是一次全新的客户需求分析。请完整理解需求、匹配产品、判断可行性。")
	case domain.OpFollowup:
		b.WriteString("销售正在追问。基于上一次分析结果回答，必要时补充产品匹配。")
	case domain.OpReanalysis:
		b.WriteString("销售换了新客户/新文档重新分析。请重新完整分析。")
	}
	b.WriteString("\n\n发现新客户特征或新威胁模式时，主动调用 memory_ensure / memory_observe 记录。")

	return b.String()
}

const systemRole = `你是长亭科技（Chaitin）的售前需求分析助手。你的工作是把销售带来的客户沟通文本，翻译成具体的安全需求，并匹配长亭的产品能力，给出可行性判断。`

const systemGoals = `1. 需求理解：把客户的业务语言翻译成具体安全需求（如"网站老被扫"→Web扫描攻击/CC攻击/爬虫）
2. 产品匹配：对照长亭产品库，标出能覆盖该需求的产品，给出置信度（0-1）和理由
3. 可行性判断：直接覆盖 / 基础上定制 / 需整合外部 / 不建议接
4. 追问识别：标出客户没说清、需要回头确认的信息`

const systemOutputFormat = `最终请输出一个 JSON（只输出 JSON，不要多余文字）：
{
  "demand_analysis": "需求理解（中文）",
  "matched_products": [
    {"name": "雷池", "confidence": 0.95, "reason": "匹配理由", "suggestion": "推荐话术"}
  ],
  "feasibility": "direct|custom|partner|reject",
  "feasibility_detail": "可行性详细说明",
  "missing_info": ["待追问信息1", "待追问信息2"]
}`

const systemConstraints = `- 只推荐知识库中存在的长亭产品，禁止编造产品名或能力
- 不确定时标注"需进一步确认"，不要瞎猜
- 置信度要诚实：核心能力给 0.9+，边缘能力给 0.5-0.7，不确定给 0.3 以下
- 每次分析必须给出 feasibility 判断
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

// renderKnowledge 把知识库渲染成 prompt 友好的文本。
func renderKnowledge(kb *longterm.KnowledgeBundle) string {
	var b strings.Builder
	if len(kb.Products) > 0 {
		b.WriteString("### 产品\n")
		for _, p := range kb.Products {
			b.WriteString(fmt.Sprintf("- **%s**（%s）", p.Name, p.Category))
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
				b.WriteString(fmt.Sprintf("  - %s%s：%s\n", c.Name, conf, truncate(c.Description, 80)))
			}
			if len(p.Limitations) > 0 {
				b.WriteString("  边界：" + strings.Join(p.Limitations, "；") + "\n")
			}
		}
	}
	if len(kb.Threats) > 0 {
		b.WriteString("\n### 已知威胁类型\n")
		for _, t := range kb.Threats {
			b.WriteString(fmt.Sprintf("- **%s**", t.Name))
			if len(t.RelatedProducts) > 0 {
				b.WriteString(" → 关联产品：" + strings.Join(t.RelatedProducts, "、"))
			}
			b.WriteString("\n")
		}
	}
	if len(kb.Compliances) > 0 {
		b.WriteString("\n### 合规要求\n")
		for _, c := range kb.Compliances {
			b.WriteString(fmt.Sprintf("- **%s**", c.Name))
			if len(c.RelatedProducts) > 0 {
				b.WriteString(" → 关联产品：" + strings.Join(c.RelatedProducts, "、"))
			}
			b.WriteString("\n")
		}
	}
	if len(kb.Industries) > 0 {
		b.WriteString("\n### 行业场景\n")
		for _, i := range kb.Industries {
			b.WriteString(fmt.Sprintf("- **%s**：痛点 %s\n", i.Name, strings.Join(i.TypicalPains, "、")))
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
