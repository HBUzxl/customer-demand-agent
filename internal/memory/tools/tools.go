// Package tools implements the 7 resident function-calling tools the Agent
// uses to autonomously maintain and consult long-term memory. Naming follows
// the LLM Wiki verb_object style: memory_search / memory_get / memory_ensure /
// memory_observe / memory_delete / memory_recall / memory_list.
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
	Get     ToolName = "memory_get"
	Ensure  ToolName = "memory_ensure"
	Observe ToolName = "memory_observe"
	Delete  ToolName = "memory_delete"
	Recall  ToolName = "memory_recall"
	List    ToolName = "memory_list"
)

// Registry 持有全部记忆工具，提供 LLM 工具定义与执行分发。
// store 是 longterm.Store 接口（Composite：system 只读基线 + 租户覆盖层）——
// 工具读写经路由层，天然落在正确层（产品读系统、客户写租户、威胁/合规/行业合并）。
type Registry struct {
	store longterm.Store
}

// NewRegistry 创建工具注册表。
func NewRegistry(store longterm.Store) *Registry {
	return &Registry{store: store}
}

// Definitions 返回全部工具的 LLM function-calling 定义（注入 system prompt / tools）。
func (r *Registry) Definitions() []domain.Tool {
	return []domain.Tool{
		{Type: "function", Function: searchDef()},
		{Type: "function", Function: getDef()},
		{Type: "function", Function: ensureDef()},
		{Type: "function", Function: observeDef()},
		{Type: "function", Function: deleteDef()},
		{Type: "function", Function: recallDef()},
		{Type: "function", Function: listDef()},
	}
}

// DefinitionsFor 返回当前 Agent 轮次可见的最小权限工具集。读取工具对所有
// 已认证租户成员开放；记忆写入与 HTTP 管理路由保持一致，仅 owner/admin 可用；
// memory_delete 永不交给模型，删除必须由显式的人类管理操作完成。
// nil scope 保留 legacy 单租户调用兼容，但同样不向模型暴露删除工具。
func (r *Registry) DefinitionsFor(scope *domain.TenantScope) []domain.Tool {
	defs := []domain.Tool{
		{Type: "function", Function: searchDef()},
		{Type: "function", Function: getDef()},
		{Type: "function", Function: recallDef()},
		{Type: "function", Function: listDef()},
	}
	if scope == nil || scope.IsTenantAdmin() {
		defs = append(defs,
			domain.Tool{Type: "function", Function: ensureDef()},
			domain.Tool{Type: "function", Function: observeDef()},
		)
	}
	return defs
}

// Execute 分发执行一次工具调用，返回结果字符串（回填给 LLM）。
func (r *Registry) Execute(name string, args json.RawMessage) (string, error) {
	switch ToolName(name) {
	case Search:
		return r.execSearch(args)
	case Get:
		return r.execGet(args)
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

// ExecuteFor 是 Agent 专用的带作用域执行入口。即使模型伪造一个未注册的
// 工具调用，执行层也会再次授权，不能只依赖 DefinitionsFor 的展示过滤。
func (r *Registry) ExecuteFor(scope *domain.TenantScope, name string, args json.RawMessage) (string, error) {
	switch ToolName(name) {
	case Delete:
		return "", fmt.Errorf("memory_delete 仅允许通过人工管理界面执行")
	case Ensure, Observe:
		if scope != nil && !scope.IsTenantAdmin() {
			return "", fmt.Errorf("当前角色无权修改租户记忆")
		}
	}
	return r.Execute(name, args)
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

// ── memory_get（读正文：search 定位 → get 精读）──────────────

type getArgs struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	Section    string `json:"section"`
	BodyOffset int    `json:"body_offset"`
}

// execGet 返回一条记忆的完整内容（结构化字段 + 子文档导航 + 正文，分级分页）。
// 三级漏斗的第 3 级：search 给定位，get 给细节。
func (r *Registry) execGet(args json.RawMessage) (string, error) {
	var a getArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("参数解析: %w", err)
	}
	if _, ok := domain.ParseMemoryType(a.Type); !ok {
		return "", fmt.Errorf("未知 type: %q", a.Type)
	}
	e, err := r.store.GetEntry(a.Type, a.Title)
	if err != nil || e.Status == longterm.StatusPendingReview || e.Status == longterm.StatusArchived {
		// 不存在或对 Agent 不可见（审核门禁，与 SearchEntry 同语义）。
		// 不作为 error 返回——带相近候选，让 LLM 自纠错后重调。
		return r.getNotFound(a.Type, a.Title), nil
	}
	return r.renderFull(e, a), nil
}

// getNotFound 构造「不存在」响应，附相近候选（自纠错提示）。
func (r *Registry) getNotFound(typeStr, title string) string {
	sugg := make([]string, 0, 3)
	for _, h := range r.store.SearchEntry(title, typeStr, 3) {
		sugg = append(sugg, h.Title)
	}
	b, _ := json.Marshal(map[string]any{
		"error":       fmt.Sprintf("%s/%s 不存在或不可见", typeStr, title),
		"suggestions": sugg,
	})
	return string(b)
}

// renderFull 渲染一条记忆的完整内容（markdown，给 LLM 读）。
func (r *Registry) renderFull(e *longterm.Entry, a getArgs) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s（%s｜%s）\n", e.Title, e.Type, e.Status)
	if e.Summary != "" {
		b.WriteString("> " + e.Summary + "\n")
	}
	if len(e.Aliases) > 0 {
		b.WriteString("别名：" + strings.Join(e.Aliases, " / ") + "\n")
	}
	if len(e.Tags) > 0 {
		b.WriteString("标签：" + strings.Join(e.Tags, " / ") + "\n")
	}
	switch e.Type {
	case domain.MemoryProduct:
		r.renderProduct(&b, e)
	case domain.MemoryThreat:
		if t, ok := r.store.GetThreat(e.Title); ok {
			renderList(&b, "典型表现（客户语言对照）", t.TypicalSigns)
			renderList(&b, "关联产品", t.RelatedProducts)
		}
	case domain.MemoryCompliance:
		if c, ok := r.store.GetCompliance(e.Title); ok {
			renderList(&b, "具体要求", c.Requirements)
			renderList(&b, "关联产品", c.RelatedProducts)
		}
	case domain.MemoryIndustry:
		if i, ok := r.store.GetIndustry(e.Title); ok {
			renderList(&b, "典型痛点", i.TypicalPains)
			renderList(&b, "常见需求", i.CommonNeeds)
		}
	case domain.MemoryCustomer:
		if c, err := r.store.GetCustomerProfile(e.Title); err == nil {
			b.WriteString("\n## 客户画像\n")
			if c.Industry != "" {
				fmt.Fprintf(&b, "- 行业：%s\n", c.Industry)
			}
			if c.Scale != "" {
				fmt.Fprintf(&b, "- 规模：%s\n", c.Scale)
			}
			if len(c.TechStack) > 0 {
				b.WriteString("- 技术栈：" + strings.Join(c.TechStack, "、") + "\n")
			}
			if len(c.ExistingSecurity) > 0 {
				b.WriteString("- 已有安全能力：" + strings.Join(c.ExistingSecurity, "、") + "\n")
			}
			if len(c.PainPoints) > 0 {
				b.WriteString("- 已知痛点：" + strings.Join(c.PainPoints, "、") + "\n")
			}
			if c.ProcurementPref != "" {
				fmt.Fprintf(&b, "- 采购偏好：%s\n", c.ProcurementPref)
			}
		}
	case domain.MemoryUser:
		if u, err := r.store.GetUserProfile(e.Title); err == nil {
			b.WriteString("\n## 使用者画像\n")
			fmt.Fprintf(&b, "- 级别：%s（历史判断准确率 %.2f）\n", u.Level, u.Accuracy)
			if len(u.Expertise) > 0 {
				b.WriteString("- 擅长领域：" + strings.Join(u.Expertise, "、") + "\n")
			}
		}
	}
	b.WriteString(renderBody(e.Content, a.Section, a.BodyOffset))
	return b.String()
}

// renderProduct 渲染产品条目：主页给结构化精华 + 子文档导航；子文档指回主页。
func (r *Registry) renderProduct(b *strings.Builder, e *longterm.Entry) {
	if e.Product != "" {
		fmt.Fprintf(b, "\n所属产品：%s（主页 memory_get 可读）\n", e.Product)
		return
	}
	p, err := r.store.GetProduct(e.Title)
	if err != nil {
		return
	}
	if p.FullName != "" {
		fmt.Fprintf(b, "\n%s", p.FullName)
		if p.Category != "" {
			fmt.Fprintf(b, "｜%s", p.Category)
		}
		b.WriteString("\n")
	}
	if len(p.Capabilities) > 0 {
		b.WriteString("\n## 能力（置信度）\n")
		for _, c := range p.Capabilities {
			line := fmt.Sprintf("- **%s** [%.1f]", c.Name, c.Confidence)
			if c.Description != "" {
				line += "：" + c.Description
			}
			if len(c.Keywords) > 0 {
				line += "（关键词：" + strings.Join(c.Keywords, "/") + "）"
			}
			b.WriteString(line + "\n")
		}
	}
	renderList(b, "适用场景", p.Scenarios)
	renderList(b, "能力边界（推荐前必读）", p.Limitations)
	if len(p.Competitors) > 0 {
		b.WriteString("\n## 竞品对比\n")
		for _, c := range p.Competitors {
			fmt.Fprintf(b, "- vs %s：%s\n", c.Name, c.Compare)
		}
	}
	if children := r.store.ListChildren(e.Title); len(children) > 0 {
		b.WriteString("\n## 相关文档（memory_get 可读）\n")
		for _, c := range children {
			line := "- " + c.Title
			if c.Summary != "" {
				line += "：" + c.Summary
			}
			b.WriteString(line + "\n")
		}
	}
}

func renderList(b *strings.Builder, header string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "\n## %s\n", header)
	for _, it := range items {
		b.WriteString("- " + it + "\n")
	}
}

// ── 正文分级分页：短文全文 / 长文章节目录 / section 精读 / offset 兜底 ──

const (
	bodyFullLimit = 4000 // rune：正文不超此值直接全文返回
	readWindow    = 4000 // rune：offset 续读单次窗口
	sectionCap    = 8000 // rune：单次章节输出上限
	tocCap        = 60   // 章节目录最多列出的标题数
)

// renderBody 按正文长度与参数分级返回：
//   - 带 section → 章节精读（无论长短）
//   - 带 body_offset > 0 → 偏移续读（无章节结构时的兜底）
//   - 短文（≤ bodyFullLimit）→ 全文
//   - 长文无参数 → 章节目录（LLM 选章后再来）
func renderBody(body, section string, offset int) string {
	if body == "" {
		return ""
	}
	total := len([]rune(body))
	secs := parseSections(body)
	switch {
	case section != "":
		if content, ru, ok := findSection(secs, body, section); ok {
			return fmt.Sprintf("\n## 正文·%s\n\n%s", section, capRunes(content, ru, sectionCap))
		}
		var b strings.Builder
		fmt.Fprintf(&b, "\n## 正文\n\n（未找到章节「%s」。可用章节：）\n", section)
		writeTOC(&b, secs)
		return b.String()
	case offset > 0:
		r := []rune(body)
		if offset >= len(r) {
			return fmt.Sprintf("\n（body_offset=%d 超出正文长度 %d）\n", offset, len(r))
		}
		end := min(offset+readWindow, len(r))
		b := fmt.Sprintf("\n## 正文 [%d-%d/%d]\n\n%s\n", offset, end, len(r), string(r[offset:end]))
		if end < len(r) {
			b += fmt.Sprintf("（已截断，body_offset=%d 续读）\n", end)
		}
		return b
	case total <= bodyFullLimit:
		return "\n## 正文\n\n" + body + "\n"
	default:
		var b strings.Builder
		fmt.Fprintf(&b, "\n## 正文目录（全文 %d 字超长，用 section 参数读指定章节；无章节结构用 body_offset）\n", total)
		writeTOC(&b, secs)
		if len(secs) == 0 {
			// 无标题结构的长文：给开头一段 + 引导 offset 续读
			fmt.Fprintf(&b, "\n（无章节结构，先给前 %d 字）\n\n%s\n（可用 body_offset 续读）\n", bodyFullLimit, firstNRunes(body, bodyFullLimit))
		}
		return b.String()
	}
}

// docSection 是正文里的一个章节（markdown 标题 #/##/###）。
type docSection struct {
	level int
	title string
	start int // 字节偏移：标题行行首
	end   int // 字节偏移：下一标题行行首（或文末）
}

// parseSections 扫描 markdown 标题，记录每章字节区间（章节即分页单元）。
func parseSections(body string) []docSection {
	var secs []docSection
	off := 0
	for _, line := range strings.Split(body, "\n") {
		if lvl, t, ok := headingOf(line); ok {
			if len(secs) > 0 {
				secs[len(secs)-1].end = off
			}
			secs = append(secs, docSection{level: lvl, title: t, start: off, end: len(body)})
		}
		off += len(line) + 1 // +1 补偿分掉的 '\n'
	}
	return secs
}

// headingOf 解析一行是否为 1-3 级 markdown 标题。
func headingOf(line string) (level int, title string, ok bool) {
	s := strings.TrimRight(line, " \t")
	n := 0
	for n < len(s) && s[n] == '#' {
		n++
	}
	if n < 1 || n > 3 {
		return 0, "", false
	}
	rest := strings.TrimSpace(s[n:])
	if rest == "" {
		return 0, "", false
	}
	return n, rest, true
}

// findSection 按名找章（先精确后包含），返回章节内容与其 rune 起始偏移（截断提示用）。
func findSection(secs []docSection, body, name string) (content string, runeStart int, ok bool) {
	for _, pass := range []bool{true, false} {
		for _, s := range secs {
			match := s.title == name
			if !pass {
				match = strings.Contains(s.title, name) || strings.Contains(name, s.title)
			}
			if match {
				return body[s.start:s.end], len([]rune(body[:s.start])), true
			}
		}
	}
	return "", 0, false
}

// capRunes 超限截断，尾部给出续读提示（body_offset 从全文算）。
func capRunes(s string, runeStart, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s + "\n"
	}
	return string(r[:limit]) + fmt.Sprintf("\n（本章超长已截断，body_offset=%d 从全文续读）\n", runeStart+limit)
}

// writeTOC 输出章节目录（带层级缩进，超出上限截断提示）。
func writeTOC(b *strings.Builder, secs []docSection) {
	n := min(len(secs), tocCap)
	for _, s := range secs[:n] {
		b.WriteString(strings.Repeat("  ", max(s.level-1, 0)) + "- " + s.title + "\n")
	}
	if len(secs) > tocCap {
		fmt.Fprintf(b, "- …（共 %d 章，仅列前 %d）\n", len(secs), tocCap)
	}
}

// firstNRunes 取前 n 个 rune（超出加省略号）。
func firstNRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
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
	// F2 防重复建条目（wiki-hygiene）：customer 类型 ensure 时，若客户名与
	// 已有条目互相包含（如「某跨境电商」vs「某跨境电商平台」），提示续写
	// 而非静默新建第 N 个变体。仍执行写入（Agent 可改名续写），但 result 带警告。
	var existingHint string
	if mt == domain.MemoryCustomer && r.store != nil {
		isNew := true
		for _, ex := range r.store.ListEntry("customer", 0, 200) {
			if ex.Title == a.Title {
				isNew = false // 同名=更新语义，正常
				break
			}
		}
		// 新建且客户名与已有条目互相包含 → 续写提示（防 N 变体）
		if isNew {
			for _, ex := range r.store.ListEntry("customer", 0, 200) {
				if strings.Contains(ex.Title, a.Title) || strings.Contains(a.Title, ex.Title) {
					existingHint = fmt.Sprintf(`,"existing_hint":"客户记忆已有「%s」，若为同一客户请续写该条目（title 用它）而非新建变体"`, ex.Title)
					break
				}
			}
		}
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
	// P0-07：对已验证受控知识（threat/compliance/industry）的整体覆盖同样生成
	// pending revision（原 verified 版本继续生效，批准后原子替换），而不是把活跃
	// 条目替换成待审版导致知识对 Agent 消失。
	if mt != domain.MemoryCustomer {
		if ex, err := r.store.GetEntry(a.Type, a.Title); err == nil && ex.Status == longterm.StatusVerified && needsReview(mt) {
			if err := r.store.SubmitRevision(e); err != nil {
				return "", err
			}
			return fmt.Sprintf(`{"status":"ok","message":"%s","type":"%s","title":"%s","needs_review":true%s,"note":"已暂存为待审修订——批准后替换原版本，拒绝则保持原样"}`,
				"已记录（修订待审核，原版本继续生效）", a.Type, a.Title, existingHint), nil
		}
	}
	if err := r.store.UpsertEntry(e); err != nil {
		return "", err
	}
	note := "已创建/更新"
	needsReview := status == longterm.StatusPendingReview
	if needsReview {
		note = "已记录（待审核，降权使用）"
	}
	// F4：needs_review+条目标识给前端渲染对话内审批卡片
	if needsReview {
		return fmt.Sprintf(`{"status":"ok","message":"%s","type":"%s","title":"%s","needs_review":true%s,"note":"已暂存待审——审批前你检索不到它，用户批准后生效"}`,
			note, a.Type, a.Title, existingHint), nil
	}
	return fmt.Sprintf(`{"status":"ok","message":"%s","type":"%s","title":"%s","needs_review":false%s}`,
		note, a.Type, a.Title, existingHint), nil
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
	// F4：先判断是否新建（写入前查——写入后再查永远查得到）
	existing, getErr := r.store.GetEntry(a.Type, a.Title)
	isNew := getErr != nil
	var isReview bool
	if !isNew {
		existing.Content = existing.Content + "\n\n" + obsText // 保留原 frontmatter 结构化字段
		// P0-07：对已验证受控知识（threat/compliance/industry）追加观察属实质修改，
		// 不得保持 verified 直接生效——提交 pending revision，原 verified 版本继续
		// 对 Agent 生效，人工批准后原子替换、拒绝则丢弃。customer 不受审（AI 全权）。
		if existing.Status == longterm.StatusVerified && needsReview(mt) {
			if err := r.store.SubmitRevision(existing); err != nil {
				return "", err
			}
			isReview = true
		} else if err := r.store.UpsertEntry(existing); err != nil {
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
		isReview = needsReview(mt)
	}
	return fmt.Sprintf(`{"status":"ok","message":"已记录观察","relevance":"%s","type":"%s","title":"%s","needs_review":%t}`,
		rel, a.Type, a.Title, isReview), nil
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
