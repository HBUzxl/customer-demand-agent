// Package domain holds shared domain types with zero internal dependencies.
//
// This is a leaf package: AnalysisResult, Checkpoint, Message etc. live here
// (not in agent/ or memory/shortterm/) so that agent → assembler → shortterm
// and shortterm → AnalysisResult never form an import cycle.
package domain

// Feasibility 是可行性判断的枚举。
type Feasibility string

const (
	// FeasibilityDirect 现有产品直接覆盖
	FeasibilityDirect Feasibility = "direct"
	// FeasibilityCustom 现有基础上需定制
	FeasibilityCustom Feasibility = "custom"
	// FeasibilityPartner 需整合外部资源
	FeasibilityPartner Feasibility = "partner"
	// FeasibilityReject 做不了，不建议接
	FeasibilityReject Feasibility = "reject"
)

// AnalysisResult 是 Agent 分析后的结构化输出（写在 system prompt 里让 LLM 产出）。
type AnalysisResult struct {
	DemandAnalysis    string           `json:"demand_analysis"`    // 需求理解：客户业务语言翻译成的安全需求
	MatchedProducts   []MatchedProduct `json:"matched_products"`   // 匹配的长亭产品 + 置信度
	Feasibility       Feasibility      `json:"feasibility"`        // 可行性判断
	FeasibilityDetail string           `json:"feasibility_detail"` // 可行性详细说明
	MissingInfo       []string         `json:"missing_info"`       // 待追问信息：客户没说清的
}

// MatchedProduct 是一条产品匹配结果。
type MatchedProduct struct {
	Name       string  `json:"name"`       // 产品名（如"雷池"）
	Confidence float64 `json:"confidence"` // 置信度 0-1
	Reason     string  `json:"reason"`     // 匹配理由
	Suggestion string  `json:"suggestion"` // 推荐话术
}

// MemoryType 是长期记忆的类型枚举。
type MemoryType string

const (
	MemoryProduct    MemoryType = "product"
	MemoryThreat     MemoryType = "threat"
	MemoryCompliance MemoryType = "compliance"
	MemoryIndustry   MemoryType = "industry"
	MemoryCustomer   MemoryType = "customer"
	MemoryUser       MemoryType = "user"
)

// AllMemoryTypes 返回全部记忆类型（用于遍历/校验）。
func AllMemoryTypes() []MemoryType {
	return []MemoryType{
		MemoryProduct, MemoryThreat, MemoryCompliance,
		MemoryIndustry, MemoryCustomer, MemoryUser,
	}
}

// ParseMemoryType 将字符串安全转为 MemoryType，非法值返回空串 + false。
func ParseMemoryType(s string) (MemoryType, bool) {
	for _, t := range AllMemoryTypes() {
		if string(t) == s {
			return t, true
		}
	}
	return "", false
}
