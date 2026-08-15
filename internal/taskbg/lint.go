package taskbg

import (
	"fmt"
	"strings"
)

// LintFinding 是一条 Lint 发现（审核建议——不直接改，人审后处理）。
type LintFinding struct {
	Kind    string `json:"kind"`    // orphan / incomplete / duplicate-alias
	Type    string `json:"type"`    // 记忆类型
	Title   string `json:"title"`   // 条目
	Message string `json:"message"` // 说明+建议
}

// LintInput Lint 扫描的输入（main 侧收集，避免本包依赖 wiki 结构）。
type LintInput struct {
	Entries []LintEntry `json:"entries"`
}

// LintEntry 是参与 Lint 的条目快照。
type LintEntry struct {
	Type     string   `json:"type"`
	Title    string   `json:"title"`
	Aliases  []string `json:"aliases"`
	Tags     []string `json:"tags"`
	Keywords []string `json:"keywords"`
	Summary  string   `json:"summary"`
	Content  string   `json:"content"`
}

// RunLint 确定性检测（不调 LLM——快且稳）：
// ①孤儿：无 tags/keywords/aliases 的条目（检索几乎到不了）
// ②残缺：summary 或 content 为空
// ③别名冲突：同一类型内两个条目共用了别名
func RunLint(in LintInput) []LintFinding {
	var out []LintFinding
	aliasOwner := map[string]string{} // "type/alias" → title
	for _, e := range in.Entries {
		if e.Tags == nil && e.Keywords == nil && e.Aliases == nil && e.Type != "product" {
			out = append(out, LintFinding{
				Kind: "orphan", Type: e.Type, Title: e.Title,
				Message: "无 tags/keywords/别名——关键词检索几乎不可达。建议补 tags（3-6 个）。",
			})
		}
		if strings.TrimSpace(e.Summary) == "" || strings.TrimSpace(e.Content) == "" {
			out = append(out, LintFinding{
				Kind: "incomplete", Type: e.Type, Title: e.Title,
				Message: "summary 或正文为空。建议补全（空条目检索命中后无内容可用）。",
			})
		}
		for _, a := range e.Aliases {
			key := e.Type + "/" + a
			if prev, dup := aliasOwner[key]; dup {
				out = append(out, LintFinding{
					Kind: "duplicate-alias", Type: e.Type, Title: e.Title,
					Message: fmt.Sprintf("别名 %q 与条目 %q 冲突——检索会歧义。建议二选一保留。", a, prev),
				})
			} else {
				aliasOwner[key] = e.Title
			}
		}
	}
	return out
}

// BuildTitlePrompt 构造标题生成提示词（G4）。
func BuildTitlePrompt(firstUserText string) string {
	return "给下面这段客户对话起一个简短的会话标题（8-14 字，概括客户/场景/诉求，不带引号）：\n" +
		firstUserText + "\n\n只输出标题本身。"
}
