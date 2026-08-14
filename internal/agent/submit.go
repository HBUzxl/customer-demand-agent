package agent

import (
	"encoding/json"
	"fmt"

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
