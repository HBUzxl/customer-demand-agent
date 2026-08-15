package taskbg

import (
	"fmt"
	"strings"
)

// LintFinding 是一条 Lint 发现（审核建议——不直接改，人审后处理）。
type LintFinding struct {
	Kind    string `json:"kind"`    // orphan / incomplete / duplicate-alias / overlap
	Type    string `json:"type"`    // 记忆类型
	Title   string `json:"title"`   // 条目
	Message string `json:"message"` // 说明+建议
}

// LintInput Lint 扫描的输入（main 侧收集，避免本包依赖 wiki 结构）。
type LintInput struct {
	Entries []LintEntry `json:"entries"`
	// RecentMissQueries 近期 memory_search 零命中查询原文（main 从 history
	// tool_calls 收集）——「该建未建」检测输入。
	RecentMissQueries []string `json:"recent_miss_queries"`
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
// ④重叠/矛盾候选：同类型内两个条目正文 bigram Jaccard > 0.6（描述高度
//
//	重叠——疑似重复定义或事实矛盾，人工裁决合并/区分）
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
	// ④重叠/矛盾候选（同类型正文 Jaccard > 0.6）
	out = append(out, findOverlap(in.Entries)...)
	// ⑤该建未建（检索零命中查询）
	out = append(out, findMissingEntries(in)...)
	return out
}

// findMissingEntries 检索零命中的查询 → 该建未建建议（去重，同查询词只报一次）。
func findMissingEntries(in LintInput) []LintFinding {
	seen := map[string]bool{}
	var out []LintFinding
	for _, q := range in.RecentMissQueries {
		q = strings.TrimSpace(q)
		if len([]rune(q)) < 3 || seen[q] {
			continue
		}
		seen[q] = true
		out = append(out, LintFinding{
			Kind: "missing-entry", Type: "all", Title: q,
			Message: "近期检索零命中：知识库可能缺这个主题的条目，建议人工评估是否新建",
		})
	}
	return out
}

// BuildTitlePrompt 构造标题生成提示词（G4）。
func BuildTitlePrompt(firstUserText string) string {
	return "给下面这段客户对话起一个简短的会话标题（8-14 字，概括客户/场景/诉求，不带引号）：\n" +
		firstUserText + "\n\n只输出标题本身。"
}

// cjkBigrams 提取 CJK bigram 集合（重叠检测的确定性签名）。
func cjkBigrams(s string) map[string]struct{} {
	rs := []rune(s)
	out := map[string]struct{}{}
	for i := 0; i+1 < len(rs); i++ {
		if isCJK(rs[i]) && isCJK(rs[i+1]) {
			out[string(rs[i:i+2])] = struct{}{}
		}
	}
	return out
}

func isCJK(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || (r >= 0x3400 && r <= 0x4DBF)
}

// jaccard 集合相似度。
func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// findOverlap 同类型条目两两 bigram Jaccard > 0.6 报重叠（矛盾候选）。
func findOverlap(entries []LintEntry) []LintFinding {
	var out []LintFinding
	byType := map[string][]LintEntry{}
	for _, e := range entries {
		if len(e.Content) >= 20 { // 太短的不参与（噪声）
			byType[e.Type] = append(byType[e.Type], e)
		}
	}
	for _, es := range byType {
		sigs := make([]map[string]struct{}, len(es))
		for i, e := range es {
			sigs[i] = cjkBigrams(e.Content)
		}
		for i := 0; i < len(es); i++ {
			for j := i + 1; j < len(es); j++ {
				if sim := jaccard(sigs[i], sigs[j]); sim > 0.6 {
					out = append(out, LintFinding{
						Kind: "overlap", Type: es[i].Type,
						Title:   es[i].Title,
						Message: fmt.Sprintf("与「%s」正文重叠度 %.0f%%——疑似重复定义或事实矛盾，建议人工裁决合并或区分", es[j].Title, sim*100),
					})
				}
			}
		}
	}
	return out
}
