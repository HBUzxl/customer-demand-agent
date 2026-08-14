package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestParseSubmission 验证 analysis_submit 参数解析与校验（ADR-013 工具契约）。
func TestParseSubmission(t *testing.T) {
	ok := `{"demand_analysis":"客户网站被CC攻击，需要抗DDoS能力","matched_products":[{"name":"雷池","confidence":0.9,"reason":"WAF核心场景","suggestion":"推荐雷池标准版"}],"feasibility":"direct","missing_info":["预算范围"],"is_reanalysis":false}`
	sub, err := parseSubmission(json.RawMessage(ok))
	if err != nil {
		t.Fatalf("合法参数应解析成功: %v", err)
	}
	if sub.DemandAnalysis == "" || sub.Feasibility != "direct" {
		t.Errorf("字段解析错: %+v", sub)
	}
	if len(sub.MatchedProducts) != 1 || sub.MatchedProducts[0].Name != "雷池" {
		t.Errorf("matched_products 解析错: %+v", sub.MatchedProducts)
	}
	if sub.IsReanalysis {
		t.Error("is_reanalysis 默认应为 false")
	}

	// is_reanalysis = true
	sub2, err := parseSubmission(json.RawMessage(`{"demand_analysis":"新文档重分析","feasibility":"custom","is_reanalysis":true}`))
	if err != nil || !sub2.IsReanalysis {
		t.Errorf("is_reanalysis=true 应保留: %v %+v", err, sub2)
	}

	// 非法：缺 demand_analysis
	if _, err := parseSubmission(json.RawMessage(`{"feasibility":"direct"}`)); err == nil {
		t.Error("缺 demand_analysis 应报错")
	}
	// 非法：feasibility 枚举外
	if _, err := parseSubmission(json.RawMessage(`{"demand_analysis":"x","feasibility":"maybe"}`)); err == nil {
		t.Error("feasibility 非法值应报错")
	}
	// 非法：坏 JSON
	if _, err := parseSubmission(json.RawMessage(`not json`)); err == nil {
		t.Error("坏 JSON 应报错")
	}
}

// TestAnalysisSubmitDef 验证工具定义可序列化且关键字段齐全（schema 即契约）。
func TestAnalysisSubmitDef(t *testing.T) {
	def := analysisSubmitDef()
	if def.Function.Name != ToolAnalysisSubmit {
		t.Fatalf("工具名: %s", def.Function.Name)
	}
	b, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("定义应可序列化: %v", err)
	}
	s := string(b)
	for _, key := range []string{"demand_analysis", "matched_products", "feasibility", "is_reanalysis", "required"} {
		if !strings.Contains(s, key) {
			t.Errorf("schema 缺少字段 %q", key)
		}
	}
}

// TestParseMissingAnswer 校验 missing_answer 参数解析。
func TestParseMissingAnswer(t *testing.T) {
	// 合法：多条答案
	ans, err := parseMissingAnswer(json.RawMessage(
		`{"answers":[{"item":"预算多少","answer":"50万"},{"item":"部署环境","answer":"私有云"}]}`))
	if err != nil {
		t.Fatalf("合法参数应通过: %v", err)
	}
	if len(ans) != 2 || ans[0].Item != "预算多少" || ans[1].Answer != "私有云" {
		t.Fatalf("解析结果不对: %+v", ans)
	}
	// 空 answers 拒绝
	if _, err := parseMissingAnswer(json.RawMessage(`{"answers":[]}`)); err == nil {
		t.Fatal("空 answers 应拒绝")
	}
	// 缺 answer 拒绝
	if _, err := parseMissingAnswer(json.RawMessage(`{"answers":[{"item":"x"}]}`)); err == nil {
		t.Fatal("缺 answer 应拒绝")
	}
	// 坏 JSON 拒绝
	if _, err := parseMissingAnswer(json.RawMessage(`{`)); err == nil {
		t.Fatal("坏 JSON 应拒绝")
	}
}

// TestMissingAnswerDef 校验工具定义契约。
func TestMissingAnswerDef(t *testing.T) {
	def := missingAnswerDef()
	if def.Function.Name != ToolMissingAnswer {
		t.Fatalf("工具名应为 %s，got %s", ToolMissingAnswer, def.Function.Name)
	}
	data, err := json.Marshal(def)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"answers", "item", "answer"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("schema 缺少字段 %s", want)
		}
	}
}
