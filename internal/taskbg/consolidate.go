package taskbg

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ConsolidateInput 固化任务的输入（观察注记原文+所属条目）。
type ConsolidateInput struct {
	Type     string   `json:"type"`
	Title    string   `json:"title"`
	Observes []string `json:"observes"` // 该条目下累积的观察注记
}

// consolidator 由 main 注入：具体固化执行（LLM 抽结构→UpsertEntry pending）。
// 独立签名避免本包依赖 llm/wiki（依赖倒置）。
type Consolidator func(ctx context.Context, in ConsolidateInput) (summary string, err error)

// BuildConsolidatePrompt 构造固化提示词（一次性调用，无记忆依赖——ADR-015 后台域）。
func BuildConsolidatePrompt(in ConsolidateInput) string {
	var b strings.Builder
	b.WriteString("你是知识库整理助手。以下是关于「" + in.Title + "」（类型 " + in.Type + "）的多条观察注记。\n")
	b.WriteString("请把它们固化为一条结构化记忆条目：\n")
	b.WriteString("1. summary：一段话概括（80 字内）\n2. tags：3-6 个检索关键词\n3. content：整合全部观察的 Markdown 正文（保留具体细节，不要丢信息）\n\n")
	b.WriteString("观察注记：\n")
	for i, o := range in.Observes {
		fmt.Fprintf(&b, "%d. %s\n", i+1, o)
	}
	b.WriteString("\n只输出 JSON：{\"summary\":\"...\",\"tags\":[...],\"content\":\"...\"}")
	return b.String()
}

// ParseConsolidateOutput 解析 LLM 输出。
func ParseConsolidateOutput(raw string) (summary string, tags []string, content string, err error) {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "{"); i > 0 {
		raw = raw[i:]
	}
	if j := strings.LastIndex(raw, "}"); j >= 0 {
		raw = raw[:j+1]
	}
	var out struct {
		Summary string   `json:"summary"`
		Tags    []string `json:"tags"`
		Content string   `json:"content"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return "", nil, "", fmt.Errorf("解析固化输出: %w", err)
	}
	if out.Summary == "" || out.Content == "" {
		return "", nil, "", fmt.Errorf("固化输出缺 summary/content")
	}
	return out.Summary, out.Tags, out.Content, nil
}
