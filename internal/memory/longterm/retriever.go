package longterm

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"customer-demand-agent/internal/domain"
)

// ── 类型化检索（供 assembler / agent 使用）──────────────────────

// AllProducts 返回全部已验证产品（按名排序）。
func (w *WikiStore) AllProducts() []Product {
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]Product, 0, len(w.products))
	for _, p := range w.products {
		if e, ok := w.entries[domain.MemoryProduct][p.Name]; ok && e.Status == StatusArchived {
			continue
		}
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// GetProduct 按名称（或别名）精确查找产品。
func (w *WikiStore) GetProduct(name string) (*Product, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if p, ok := w.products[name]; ok {
		return p, nil
	}
	// 别名匹配
	for _, p := range w.products {
		for _, a := range p.Aliases {
			if strings.EqualFold(a, name) {
				return p, nil
			}
		}
	}
	return nil, fmt.Errorf("产品 %q 不存在", name)
}

// SearchProducts 关键词检索产品，按命中关键词数排序。
func (w *WikiStore) SearchProducts(query string) []Product {
	hits := w.genericSearch(query, domain.MemoryProduct, 10)
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]Product, 0, len(hits))
	for _, h := range hits {
		if p, ok := w.products[h.Title]; ok {
			out = append(out, *p)
		}
	}
	return out
}

// SearchThreats 按关键词检索威胁类型。
func (w *WikiStore) SearchThreats(keywords []string) []ThreatType {
	hits := w.genericSearch(strings.Join(keywords, " "), domain.MemoryThreat, 10)
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]ThreatType, 0, len(hits))
	for _, h := range hits {
		if t, ok := w.threats[h.Title]; ok {
			out = append(out, *t)
		}
	}
	return out
}

// SearchCompliance 按关键词检索合规要求。
func (w *WikiStore) SearchCompliance(keywords []string) []ComplianceRequirement {
	hits := w.genericSearch(strings.Join(keywords, " "), domain.MemoryCompliance, 10)
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]ComplianceRequirement, 0, len(hits))
	for _, h := range hits {
		if c, ok := w.compliances[h.Title]; ok {
			out = append(out, *c)
		}
	}
	return out
}

// SearchIndustryScenario 按关键词检索行业场景。
func (w *WikiStore) SearchIndustryScenario(keywords []string) []IndustryScenario {
	hits := w.genericSearch(strings.Join(keywords, " "), domain.MemoryIndustry, 10)
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]IndustryScenario, 0, len(hits))
	for _, h := range hits {
		if i, ok := w.industries[h.Title]; ok {
			out = append(out, *i)
		}
	}
	return out
}

// GetCustomerProfile 按名称获取客户画像。
func (w *WikiStore) GetCustomerProfile(name string) (*CustomerProfile, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if c, ok := w.customers[name]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("客户画像 %q 不存在", name)
}

// GetUserProfile 按名称获取使用者画像。
func (w *WikiStore) GetUserProfile(name string) (*UserProfile, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if u, ok := w.users[name]; ok {
		return u, nil
	}
	return nil, fmt.Errorf("使用者画像 %q 不存在", name)
}

// AllKnowledge 打包全量已验证知识（注入 system prompt）。
func (w *WikiStore) AllKnowledge() *KnowledgeBundle {
	return &KnowledgeBundle{
		Products:    w.AllProducts(),
		Threats:     w.SearchThreats(nil),
		Compliances: w.SearchCompliance(nil),
		Industries:  w.SearchIndustryScenario(nil),
	}
}

// ── 通用 Entry 操作（供 memory FC 工具使用）─────────────────────

// SearchEntry 跨类型关键词搜索。type 为空则搜全部。返回命中（已排序、排除归档）。
func (w *WikiStore) SearchEntry(query, typeStr string, limit int) []*Entry {
	if limit <= 0 {
		limit = 10
	}
	var mt domain.MemoryType
	if typeStr != "" && typeStr != "all" {
		t, ok := domain.ParseMemoryType(typeStr)
		if !ok {
			return nil
		}
		mt = t
	}
	hits := w.genericSearch(query, mt, limit)
	w.mu.RLock()
	defer w.mu.RUnlock()
	out := make([]*Entry, 0, len(hits))
	for _, h := range hits {
		if e, ok := w.entries[h.Type][h.Title]; ok && e.Status != StatusArchived {
			cp := *e
			cp.Relevance = h.score
			out = append(out, &cp)
		}
	}
	return out
}

// ListEntry 列出某类型全部条目（分页，排除归档）。
func (w *WikiStore) ListEntry(typeStr string, offset, limit int) []*Entry {
	if limit <= 0 {
		limit = 50
	}
	mt, ok := domain.ParseMemoryType(typeStr)
	if !ok {
		return nil
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	titles := make([]string, 0, len(w.entries[mt]))
	for t, e := range w.entries[mt] {
		if e.Status != StatusArchived {
			titles = append(titles, t)
		}
	}
	sort.Strings(titles)
	if offset > len(titles) {
		offset = len(titles)
	}
	titles = titles[offset:]
	if limit < len(titles) {
		titles = titles[:limit]
	}
	out := make([]*Entry, 0, len(titles))
	for _, t := range titles {
		e := w.entries[mt][t]
		cp := *e
		out = append(out, &cp)
	}
	return out
}

// GetEntry 按 type+title 精确获取单条（含待审核/归档）。
func (w *WikiStore) GetEntry(typeStr, title string) (*Entry, error) {
	mt, ok := domain.ParseMemoryType(typeStr)
	if !ok {
		return nil, fmt.Errorf("未知 type: %q", typeStr)
	}
	w.mu.RLock()
	defer w.mu.RUnlock()
	if e, ok := w.entries[mt][title]; ok {
		cp := *e
		return &cp, nil
	}
	return nil, fmt.Errorf("%s/%s 不存在", typeStr, title)
}

// UpsertEntry 创建或更新一条记忆，并写回磁盘。status 指定审核状态。
// 注意：调用方（tools 层）负责权限校验（product 不可写等，ADR-005）。
func (w *WikiStore) UpsertEntry(e *Entry) error {
	if e.Title == "" {
		return fmt.Errorf("title 不能为空")
	}
	mt := e.Type
	if _, ok := domain.ParseMemoryType(string(mt)); !ok {
		return fmt.Errorf("未知 type: %q", mt)
	}
	if e.Status == "" {
		e.Status = StatusVerified
	}
	if e.FilePath == "" {
		e.FilePath = w.pathFor(mt, e.Title)
	}
	if err := w.writePage(e); err != nil {
		return fmt.Errorf("写回磁盘: %w", err)
	}

	w.mu.Lock()
	if w.entries[mt] == nil {
		w.entries[mt] = make(map[string]*Entry)
	}
	w.addEntry(e)
	w.rebuildIndex()
	w.mu.Unlock()
	return nil
}

// DeleteEntry 软删除（归档）或物理删除一条记忆。
func (w *WikiStore) DeleteEntry(typeStr, title string, archive bool) error {
	mt, ok := domain.ParseMemoryType(typeStr)
	if !ok {
		return fmt.Errorf("未知 type: %q", typeStr)
	}
	w.mu.Lock()
	e, exists := w.entries[mt][title]
	w.mu.Unlock()
	if !exists {
		return fmt.Errorf("%s/%s 不存在", typeStr, title)
	}
	if archive {
		e.Status = StatusArchived
		if err := w.writePage(e); err != nil {
			return err
		}
		w.mu.Lock()
		w.entries[mt][title] = e
		w.rebuildIndex()
		w.mu.Unlock()
		return nil
	}
	// 物理删除
	if e.FilePath != "" {
		_ = os.Remove(e.FilePath)
	}
	w.mu.Lock()
	delete(w.entries[mt], title)
	w.deleteTyped(mt, title)
	w.rebuildIndex()
	w.mu.Unlock()
	return nil
}

// deleteTyped 从类型化索引移除。调用方持锁。
func (w *WikiStore) deleteTyped(mt domain.MemoryType, title string) {
	switch mt {
	case domain.MemoryProduct:
		delete(w.products, title)
	case domain.MemoryThreat:
		delete(w.threats, title)
	case domain.MemoryCompliance:
		delete(w.compliances, title)
	case domain.MemoryIndustry:
		delete(w.industries, title)
	case domain.MemoryCustomer:
		delete(w.customers, title)
	case domain.MemoryUser:
		delete(w.users, title)
	}
}

// PendingReviews 返回全部待审核条目（供审核系统）。
func (w *WikiStore) PendingReviews() []*Entry {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var out []*Entry
	for _, bucket := range w.entries {
		for _, e := range bucket {
			if e.Status == StatusPendingReview {
				cp := *e
				out = append(out, &cp)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

// ApproveEntry 将待审核条目标记为已验证。
func (w *WikiStore) ApproveEntry(typeStr, title string) error {
	return w.setEntryStatus(typeStr, title, StatusVerified)
}

// RejectEntry 拒绝待审核条目（物理删除）。
func (w *WikiStore) RejectEntry(typeStr, title string) error {
	return w.DeleteEntry(typeStr, title, false)
}

func (w *WikiStore) setEntryStatus(typeStr, title string, status EntryStatus) error {
	mt, ok := domain.ParseMemoryType(typeStr)
	if !ok {
		return fmt.Errorf("未知 type: %q", typeStr)
	}
	w.mu.Lock()
	e, exists := w.entries[mt][title]
	w.mu.Unlock()
	if !exists {
		return fmt.Errorf("%s/%s 不存在", typeStr, title)
	}
	e.Status = status
	if err := w.writePage(e); err != nil {
		return err
	}
	w.mu.Lock()
	w.rebuildIndex()
	w.mu.Unlock()
	return nil
}

// ── 内部检索实现 ───────────────────────────────────────────────

type scoredHit struct {
	indexHit
	score float64
}

// genericSearch 关键词命中检索。mt 为零值时跨全部类型。
func (w *WikiStore) genericSearch(query string, mt domain.MemoryType, limit int) []scoredHit {
	w.mu.RLock()
	defer w.mu.RUnlock()
	terms := tokenize(query)
	scores := map[indexHit]float64{}
	for _, term := range terms {
		for _, hit := range w.index[term] {
			if mt != "" && hit.Type != mt {
				continue
			}
			scores[hit]++
		}
	}
	// 无 query 或无命中：返回该类型全部（用于 AllKnowledge 类场景）
	if len(scores) == 0 {
		return w.allHits(mt, limit)
	}
	out := make([]scoredHit, 0, len(scores))
	for h, s := range scores {
		out = append(out, scoredHit{indexHit: h, score: s})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score > out[j].score })
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out
}

// allHits 返回某类型全部命中（无 query 时）。
func (w *WikiStore) allHits(mt domain.MemoryType, limit int) []scoredHit {
	var out []scoredHit
	for t, bucket := range w.entries {
		if mt != "" && t != mt {
			continue
		}
		for title, e := range bucket {
			if e.Status == StatusArchived {
				continue
			}
			out = append(out, scoredHit{indexHit: indexHit{Type: t, Title: title}, score: 0})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out
}

// tokenize 将 query 切成小写关键词（简易：按非字母数字汉字切分）。
func tokenize(query string) []string {
	if query == "" {
		return nil
	}
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return r == ' ' || r == ',' || r == '，' || r == '/' || r == '|' || r == '\t'
	})
	// 进一步把长串按常见分隔再切，保留中文整段
	return fields
}

// pathFor 计算某条目的磁盘路径。
func (w *WikiStore) pathFor(mt domain.MemoryType, title string) string {
	sub := typedSubdir(mt)
	name := sanitizeFileName(title) + ".md"
	return filepath.Join(w.dir, sub, name)
}

func typedSubdir(mt domain.MemoryType) string {
	switch mt {
	case domain.MemoryProduct:
		return "产品记忆"
	case domain.MemoryThreat:
		return "行业记忆/威胁类型"
	case domain.MemoryCompliance:
		return "行业记忆/合规"
	case domain.MemoryIndustry:
		return "行业记忆/行业场景"
	case domain.MemoryCustomer:
		return "用户记忆/客户"
	case domain.MemoryUser:
		return "用户记忆/使用者"
	}
	return "未分类"
}

func sanitizeFileName(s string) string {
	s = strings.TrimSpace(s)
	for _, c := range `/\:*?"<>|` {
		s = strings.ReplaceAll(s, string(c), "_")
	}
	return s
}

// writePage 将 entry 序列化为 frontmatter + markdown 写回磁盘。
func (w *WikiStore) writePage(e *Entry) error {
	if e.FilePath == "" {
		return fmt.Errorf("缺少 FilePath")
	}
	if err := os.MkdirAll(filepath.Dir(e.FilePath), 0o755); err != nil {
		return err
	}
	f := e.typed
	if f == nil {
		f = &frontmatter{}
	}
	f.Type = string(e.Type)
	f.Title = e.Title
	f.Aliases = e.Aliases
	f.Tags = e.Tags
	f.Status = string(e.Status)
	f.Summary = e.Summary
	if e.Type == domain.MemoryCustomer && f.UpdatedAt == "" {
		f.UpdatedAt = time.Now().Format(time.RFC3339)
	}

	yamlBytes, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("序列化 frontmatter: %w", err)
	}
	// yaml.Marshal 的 aliases 默认为 null 时输出，清掉空别名行保持整洁
	body := e.Content
	if e.Type == domain.MemoryThreat || e.Type == domain.MemoryCustomer {
		body = e.Content // 正文即描述/备注
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.Write(yamlBytes)
	b.WriteString("---\n\n")
	b.WriteString(body)
	b.WriteString("\n")
	return os.WriteFile(e.FilePath, []byte(b.String()), 0o644)
}
