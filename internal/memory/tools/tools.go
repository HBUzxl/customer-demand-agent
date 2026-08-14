// Package tools implements the 6 resident function-calling tools the Agent
// uses to autonomously maintain long-term memory. Naming follows the LLM Wiki
// verb_object style: memory_search / memory_ensure / memory_observe /
// memory_delete / memory_recall / memory_list.
//
// 权限矩阵（ADR-005）：
//   - product：AI 只读
//   - threat/compliance/industry：AI 可写，打"待审核"标记
//   - customer：AI 全权（读写删）
//   - user：AI 只读（管理员维护）
package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/memory/longterm"
)

// ToolName 是工具名常量。
type ToolName string

const (
	Search  ToolName = "memory_search"
	Ensure  ToolName = "memory_ensure"
	Observe ToolName = "memory_observe"
	Delete  ToolName = "memory_delete"
	Recall  ToolName = "memory_recall"
	List    ToolName = "memory_list"
)

// Registry 持有全部记忆工具，提供 LLM 工具定义与执行分发。
type Registry struct {
	store *longterm.WikiStore
}

// NewRegistry 创建工具注册表。
func NewRegistry(store *longterm.WikiStore) *Registry {
	return &Registry{store: store}
}

// Definitions 返回全部工具的 LLM function-calling 定义（注入 system prompt / tools）。
func (r *Registry) Definitions() []domain.Tool {
	return []domain.Tool{
		{Type: "function", Function: searchDef()},
		{Type: "function", Function: ensureDef()},
		{Type: "function", Function: observeDef()},
		{Type: "function", Function: deleteDef()},
		{Type: "function", Function: recallDef()},
		{Type: "function", Function: listDef()},
	}
}

// Execute 分发执行一次工具调用，返回结果字符串（回填给 LLM）。
func (r *Registry) Execute(name string, args json.RawMessage) (string, error) {
	switch ToolName(name) {
	case Search:
		return r.execSearch(args)
	case Ensure:
		return r.execEnsure(args)
	case Observe:
		return r.execObserve(args)
	case Delete:
		return r.execDelete(args)
	case Recall:
		return r.execRecall(args)
	case List:
		return r.execList(args)
	default:
		return "", fmt.Errorf("未知工具: %s", name)
	}
}

// ── memory_search ──────────────────────────────────────────────

type searchArgs struct {
	Query string `json:"query"`
	Type  string `json:"type"`
	Limit int    `json:"limit"`
}

func (r *Registry) execSearch(args json.RawMessage) (string, error) {
	var a searchArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析: %w", err)
	}
	hits := r.store.SearchEntry(a.Query, a.Type, a.Limit)
	return marshalHits(hits), nil
}

// ── memory_ensure ─────────────────────────────────────────────

type ensureArgs struct {
	Type    string   `json:"type"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Aliases []string `json:"aliases"`
}

func (r *Registry) execEnsure(args json.RawMessage) (string, error) {
	var a ensureArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析: %w", err)
	}
	mt, ok := domain.ParseMemoryType(a.Type)
	if !ok {
		return "", fmt.Errorf("未知 type: %q", a.Type)
	}
	// 权限校验：product 不可写，user 不可写
	if err := assertWritable(mt); err != nil {
		return "", err
	}
	status := longterm.StatusVerified
	if needsReview(mt) {
		status = longterm.StatusPendingReview
	}
	e := &longterm.Entry{
		Type:    mt,
		Title:   a.Title,
		Content: a.Content,
		Tags:    a.Tags,
		Aliases: a.Aliases,
		Summary: firstLine(a.Content),
		Status:  status,
	}
	if err := r.store.UpsertEntry(e); err != nil {
		return "", err
	}
	note := "已创建/更新"
	if status == longterm.StatusPendingReview {
		note = "已记录（待审核，降权使用）"
	}
	return fmt.Sprintf(`{"status":"ok","message":"%s","type":"%s","title":"%s"}`, note, a.Type, a.Title), nil
}

// ── memory_observe ────────────────────────────────────────────

type observeArgs struct {
	Type      string   `json:"type"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Relevance string   `json:"relevance"`
	Tags      []string `json:"tags"`
}

func (r *Registry) execObserve(args json.RawMessage) (string, error) {
	var a observeArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析: %w", err)
	}
	if a.Relevance == "" {
		a.Relevance = "medium"
	}
	// observe 作为轻量洞察，追加到对应类型条目的 content（若条目存在）
	// 若不存在则创建一条待审核条目（threat/compliance/industry）。
	rel := validateRelevance(a.Relevance)
	mt, ok := domain.ParseMemoryType(a.Type)
	if !ok {
		return "", fmt.Errorf("未知 type: %q", a.Type)
	}
	if err := assertWritable(mt); err != nil {
		return "", err
	}
	obsText := fmt.Sprintf("### 观察 [%s]\n%s", rel, strings.TrimSpace(a.Content))
	if existing, err := r.store.GetEntry(a.Type, a.Title); err == nil {
		existing.Content = existing.Content + "\n\n" + obsText // 保留原 frontmatter 结构化字段
		if err := r.store.UpsertEntry(existing); err != nil {
			return "", err
		}
	} else {
		e := &longterm.Entry{
			Type:    mt,
			Title:   a.Title,
			Content: obsText,
			Tags:    a.Tags,
			Summary: firstLine(a.Content),
			Status:  reviewOrVerified(mt),
		}
		if err := r.store.UpsertEntry(e); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf(`{"status":"ok","message":"已记录观察","relevance":"%s"}`, rel), nil
}

// ── memory_delete ─────────────────────────────────────────────

type deleteArgs struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Archive *bool  `json:"archive"`
}

func (r *Registry) execDelete(args json.RawMessage) (string, error) {
	var a deleteArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析: %w", err)
	}
	mt, ok := domain.ParseMemoryType(a.Type)
	if !ok {
		return "", fmt.Errorf("未知 type: %q", a.Type)
	}
	if err := assertDeletable(mt); err != nil {
		return "", err
	}
	archive := true
	if a.Archive != nil {
		archive = *a.Archive
	}
	if err := r.store.DeleteEntry(a.Type, a.Title, archive); err != nil {
		return "", err
	}
	action := "已归档"
	if !archive {
		action = "已删除"
	}
	return fmt.Sprintf(`{"status":"ok","message":"%s %s/%s"}`, action, a.Type, a.Title), nil
}

// ── memory_recall（一期回退为关键词模糊匹配）──────────────────

type recallArgs struct {
	Query      string `json:"query"`
	Type       string `json:"type"`
	MaxResults int    `json:"max_results"`
}

func (r *Registry) execRecall(args json.RawMessage) (string, error) {
	var a recallArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析: %w", err)
	}
	if a.MaxResults <= 0 {
		a.MaxResults = 5
	}
	// 一期不做向量检索（ADR-002），recall 退化为更宽松的关键词 + 别名/摘要匹配。
	hits := r.store.SearchEntry(a.Query, a.Type, a.MaxResults)
	return marshalHits(hits), nil
}

// ── memory_list ───────────────────────────────────────────────

type listArgs struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

func (r *Registry) execList(args json.RawMessage) (string, error) {
	var a listArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析: %w", err)
	}
	entries := r.store.ListEntry(a.Type, a.Offset, a.Limit)
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		out = append(out, map[string]any{
			"title":   e.Title,
			"tags":    e.Tags,
			"aliases": e.Aliases,
			"summary": e.Summary,
			"status":  e.Status,
		})
	}
	b, _ := json.Marshal(map[string]any{"count": len(out), "items": out})
	return string(b), nil
}

// ── 权限矩阵 ──────────────────────────────────────────────────

// assertWritable 校验某类型是否允许 AI 写入。
func assertWritable(mt domain.MemoryType) error {
	switch mt {
	case domain.MemoryProduct:
		return fmt.Errorf("产品记忆由人工维护，AI 不可写入（ADR-005）")
	case domain.MemoryUser:
		return fmt.Errorf("使用者画像由管理员维护，AI 不可写入（ADR-005）")
	case domain.MemoryThreat, domain.MemoryCompliance, domain.MemoryIndustry, domain.MemoryCustomer:
		return nil
	}
	return fmt.Errorf("不支持的记忆类型: %s", mt)
}

// assertDeletable 校验某类型是否允许 AI 删除。
func assertDeletable(mt domain.MemoryType) error {
	switch mt {
	case domain.MemoryCustomer:
		return nil
	default:
		return fmt.Errorf("仅客户画像允许 AI 删除（ADR-005），%s 不可删", mt)
	}
}

// needsReview 判断 AI 写入是否需要打待审核标记。
func needsReview(mt domain.MemoryType) bool {
	switch mt {
	case domain.MemoryThreat, domain.MemoryCompliance, domain.MemoryIndustry:
		return true
	}
	return false
}

func reviewOrVerified(mt domain.MemoryType) longterm.EntryStatus {
	if needsReview(mt) {
		return longterm.StatusPendingReview
	}
	return longterm.StatusVerified
}

func validateRelevance(rel string) string {
	switch strings.ToLower(rel) {
	case "low", "medium", "high", "critical":
		return strings.ToLower(rel)
	}
	return "medium"
}

// ── 辅助 ──────────────────────────────────────────────────────

func marshalHits(hits []*longterm.Entry) string {
	out := make([]map[string]any, 0, len(hits))
	for _, e := range hits {
		out = append(out, map[string]any{
			"type":    e.Type,
			"title":   e.Title,
			"summary": e.Summary,
			"tags":    e.Tags,
			"status":  e.Status,
		})
	}
	b, _ := json.Marshal(map[string]any{"count": len(out), "items": out})
	return string(b)
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if r := []rune(s); len(r) > 80 {
		return string(r[:80]) + "…"
	}
	return s
}
