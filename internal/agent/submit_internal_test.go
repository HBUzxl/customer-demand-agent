package agent

import (
	"testing"

	"customer-demand-agent/internal/domain"
)

// TestValidateSubmissionEvidenceGate P0-02 验收：analysis_submit 证据门禁——
// 未 memory_get、产品不存在、confidence 越界、缺理由均拒绝；读详情后通过；
// 无产品匹配（reject/信息缺失）不触发门禁。
func TestValidateSubmissionEvidenceGate(t *testing.T) {
	a := &Agent{productExists: func(name string) bool { return name == "雷池" }}
	sub := func(mp []domain.MatchedProduct) *AnalysisSubmission {
		return &AnalysisSubmission{AnalysisResult: domain.AnalysisResult{
			DemandAnalysis: "客户遭遇CC攻击", Feasibility: domain.FeasibilityDirect, MatchedProducts: mp,
		}}
	}
	good := []domain.MatchedProduct{{Name: "雷池", Confidence: 0.9, Reason: "CC 防护核心场景", Suggestion: "推荐雷池"}}
	read := func(products ...string) *turnState {
		m := map[string]bool{}
		for _, p := range products {
			m[p] = true
		}
		return &turnState{readProducts: m}
	}

	// 未 memory_get → 拒绝
	if reason := a.validateSubmission(sub(good), read()); reason == "" {
		t.Fatal("未 memory_get 读详情应被拒绝（证据门禁）")
	}
	// 已读详情 → 通过
	if reason := a.validateSubmission(sub(good), read("雷池")); reason != "" {
		t.Fatalf("已读详情应通过，got %s", reason)
	}
	// confidence 越界（>1）→ 拒绝
	over := []domain.MatchedProduct{{Name: "雷池", Confidence: 1.5, Reason: "x"}}
	if reason := a.validateSubmission(sub(over), read("雷池")); reason == "" {
		t.Fatal("confidence>1 应被拒绝")
	}
	// confidence 越界（<0）→ 拒绝
	under := []domain.MatchedProduct{{Name: "雷池", Confidence: -0.1, Reason: "x"}}
	if reason := a.validateSubmission(sub(under), read("雷池")); reason == "" {
		t.Fatal("confidence<0 应被拒绝")
	}
	// 产品不存在于产品库（productExists=false）→ 拒绝
	unknown := []domain.MatchedProduct{{Name: "虚构产品", Confidence: 0.8, Reason: "x"}}
	if reason := a.validateSubmission(sub(unknown), read("虚构产品")); reason == "" {
		t.Fatal("编造的产品名应被拒绝")
	}
	// 缺匹配理由 → 拒绝
	noreason := []domain.MatchedProduct{{Name: "雷池", Confidence: 0.8}}
	if reason := a.validateSubmission(sub(noreason), read("雷池")); reason == "" {
		t.Fatal("缺匹配理由应被拒绝")
	}
	// 无 matched_products → 通过（reject / 纯信息缺失场景）
	if reason := a.validateSubmission(sub(nil), read()); reason != "" {
		t.Fatalf("无产品匹配应通过，got %s", reason)
	}
	// 多产品：一个未读 → 整体拒绝（不许个别产品漏证据）
	mixed := []domain.MatchedProduct{
		{Name: "雷池", Confidence: 0.9, Reason: "x"},
		{Name: "雷池X", Confidence: 0.7, Reason: "y"},
	}
	if reason := a.validateSubmission(sub(mixed), read("雷池")); reason == "" {
		t.Fatal("存在未读产品应整体拒绝")
	}
}

func TestValidateSubmissionRejectsDirectWhenCriticalRequirementIsUnverified(t *testing.T) {
	a := &Agent{}
	s := &AnalysisSubmission{AnalysisResult: domain.AnalysisResult{
		DemandAnalysis:    "AI研判并发送微信群通知",
		Feasibility:       domain.FeasibilityDirect,
		FeasibilityDetail: "AI研判已覆盖，但微信群通知当前知识库未证实，需产品线确认",
	}}
	st := &turnState{userText: "客户要 AI 研判日志并发微信群通知", readProducts: map[string]bool{}}
	if reason := a.validateSubmission(s, st); reason == "" {
		t.Fatal("核心渠道未证实时不得判定 direct")
	}
	s.Feasibility = domain.FeasibilityCustom
	if reason := a.validateSubmission(s, st); reason != "" {
		t.Fatalf("降为 custom 并说明边界后应通过，got %s", reason)
	}
}
