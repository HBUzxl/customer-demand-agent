package longterm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"customer-demand-agent/internal/domain"
)

// WikiStore 是长期记忆的内存实现：启动时加载全部 Wiki 页面，构建索引，
// 运行时提供确定性关键词检索。所有读写受 mutex 保护。
type WikiStore struct {
	dir string

	mu      sync.RWMutex
	entries map[domain.MemoryType]map[string]*Entry // type → title → entry（排除 archived）

	// 待审修订（P0-07）：type → 父条目 title → pending revision。AI 对已验证
	// 受控知识（threat/compliance/industry）的实质修改不直接覆盖活跃条目，而是
	// 生成 pending revision 挂在这里——原 verified 版本继续对 Agent 生效，
	// 人工批准后原子替换、拒绝则丢弃。文件形态 <title>.revision-<ts>.md。
	revisions map[domain.MemoryType]map[string]*Entry

	// 类型化索引（启动时构建，便于 AllKnowledge / 结构化检索）
	products    map[string]*Product
	threats     map[string]*ThreatType
	compliances map[string]*ComplianceRequirement
	industries  map[string]*IndustryScenario
	customers   map[string]*CustomerProfile
	users       map[string]*UserProfile

	// 关键词倒排索引：token(小写) → 命中的 (type, title) → 字段权重（取最大）
	index map[string]map[indexHit]float64
}

type indexHit struct {
	Type  domain.MemoryType
	Title string
}

// NewWikiStore 创建一个指向 dir 的 Wiki 存储（未加载）。
func NewWikiStore(dir string) *WikiStore {
	return &WikiStore{
		dir:         dir,
		entries:     make(map[domain.MemoryType]map[string]*Entry),
		revisions:   make(map[domain.MemoryType]map[string]*Entry),
		products:    make(map[string]*Product),
		threats:     make(map[string]*ThreatType),
		compliances: make(map[string]*ComplianceRequirement),
		industries:  make(map[string]*IndustryScenario),
		customers:   make(map[string]*CustomerProfile),
		users:       make(map[string]*UserProfile),
		index:       make(map[string]map[indexHit]float64),
	}
}

// Load 扫描 Wiki 目录并加载全部页面。目录不存在时创建空结构（不报错）。
func (w *WikiStore) Load() error {
	for _, t := range domain.AllMemoryTypes() {
		w.entries[t] = make(map[string]*Entry)
	}

	info, err := os.Stat(w.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// 目录不存在不算错误：运行时可通过 Upsert 写入
			return nil
		}
		return fmt.Errorf("访问 wiki 目录 %s: %w", w.dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("wiki 路径 %s 不是目录", w.dir)
	}

	var loaded, skipped int
	err = filepath.WalkDir(w.dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		// P6 时效归档（.superseded-*.md）不进活跃索引——历史版本仅磁盘留痕
		if strings.Contains(filepath.Base(path), ".superseded-") {
			return nil
		}
		// P0-07 待审修订（.revision-*.md）：不进活跃索引（原 verified 版本继续生效），
		// 挂在 revisions 表供审核队列/审批。
		if strings.Contains(filepath.Base(path), ".revision-") {
			return w.loadRevisionFile(path)
		}
		raw, rErr := os.ReadFile(path)
		if rErr != nil {
			fmt.Fprintf(os.Stderr, "[wiki] 跳过 %s: %v\n", path, rErr)
			skipped++
			return nil
		}
		entry, pErr := parsePage(string(raw))
		if pErr != nil || entry.Title == "" {
			fmt.Fprintf(os.Stderr, "[wiki] 跳过 %s: %v\n", path, pErr)
			skipped++
			return nil // 单篇损坏不中断整体加载
		}
		entry.FilePath = path
		w.addEntry(entry)
		loaded++
		return nil
	})
	if err != nil {
		return err
	}
	w.rebuildIndex()
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "[wiki] 加载完成：%d 篇，跳过 %d 篇损坏文档\n", loaded, skipped)
	}
	return nil
}

// loadRevisionFile 加载一份 pending revision 文件（P0-07）到 revisions 表。
// 启动期单线程调用，无需加锁。
func (w *WikiStore) loadRevisionFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	entry, pErr := parsePage(string(raw))
	if pErr != nil || entry.Title == "" {
		fmt.Fprintf(os.Stderr, "[wiki] 跳过修订 %s: %v\n", path, pErr)
		return nil
	}
	entry.FilePath = path
	entry.Status = StatusPendingReview
	if w.revisions[entry.Type] == nil {
		w.revisions[entry.Type] = make(map[string]*Entry)
	}
	w.revisions[entry.Type][entry.Title] = entry
	return nil
}

// frontmatter 捕获所有可能字段的超集（按 type 取用）。
type frontmatter struct {
	Type    string   `yaml:"type"`
	Title   string   `yaml:"title"`
	Aliases []string `yaml:"aliases"`
	Tags    []string `yaml:"tags"`
	Status  string   `yaml:"status"`
	Summary string   `yaml:"summary"`

	// 时效性（P6）：valid_at/invalid_at 记录事实有效期；superseded_by
	// 指向替代条目（覆盖时旧版本归档写入）。
	ValidAt      string `yaml:"valid_at"`
	InvalidAt    string `yaml:"invalid_at"`
	SupersededBy string `yaml:"superseded_by"`

	// 修订（P0-07）：revision_of 标记这是某已验证条目的待审修订（批准后原子替换）。
	RevisionOf string `yaml:"revision_of"`

	// product
	FullName     string       `yaml:"full_name"`
	Category     string       `yaml:"category"`
	Product      string       `yaml:"product"` // 所属产品（子文档用）
	Capabilities []Capability `yaml:"capabilities"`
	Scenarios    []string     `yaml:"scenarios"`
	Limitations  []string     `yaml:"limitations"`
	Competitors  []Competitor `yaml:"competitors"`
	Description  string       `yaml:"description"`

	// threat / industry
	TypicalSigns    []string `yaml:"typical_signs"`
	RelatedProducts []string `yaml:"related_products"`
	Keywords        []string `yaml:"keywords"`

	// compliance
	Requirements []string `yaml:"requirements"`

	// industry
	TypicalPains []string `yaml:"typical_pains"`
	CommonNeeds  []string `yaml:"common_needs"`

	// customer / user
	Industry         string   `yaml:"industry"`
	Scale            string   `yaml:"scale"`
	TechStack        []string `yaml:"tech_stack"`
	ExistingSecurity []string `yaml:"existing_security"`
	PainPoints       []string `yaml:"pain_points"`
	ProcurementPref  string   `yaml:"procurement_pref"`
	Notes            string   `yaml:"notes"`
	Level            string   `yaml:"level"`
	Expertise        []string `yaml:"expertise"`
	Accuracy         float64  `yaml:"accuracy"`
	UserID           string   `yaml:"user_id"`
	UpdatedAt        string   `yaml:"updated_at"`
}

// NewCustomerEntry 从租户成员登记表单构造一条结构化客户画像。专用构造器
// 避免 HTTP 层依赖私有 frontmatter，同时保证行业/规模/现有安全建设等字段可被
// GetCustomerProfile 和 Agent 客户上下文正确读取，而不只是散落在 Markdown 正文。
func NewCustomerEntry(p CustomerProfile) *Entry {
	content := strings.TrimSpace(p.Notes)
	return &Entry{
		Type:    domain.MemoryCustomer,
		Title:   strings.TrimSpace(p.Name),
		Content: content,
		Tags:    append([]string(nil), p.Tags...),
		Summary: strings.TrimSpace(p.Industry),
		Status:  StatusVerified,
		typed: &frontmatter{
			Type:             string(domain.MemoryCustomer),
			Title:            strings.TrimSpace(p.Name),
			Tags:             append([]string(nil), p.Tags...),
			Industry:         strings.TrimSpace(p.Industry),
			Scale:            strings.TrimSpace(p.Scale),
			TechStack:        append([]string(nil), p.TechStack...),
			ExistingSecurity: append([]string(nil), p.ExistingSecurity...),
			PainPoints:       append([]string(nil), p.PainPoints...),
			ProcurementPref:  strings.TrimSpace(p.ProcurementPref),
			Notes:            content,
			UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// NewUserEntry 为租户成员创建身份绑定的使用者画像。user_id 是稳定关联键；
// 标题只是人类可读名称，不再通过全局 default_user 选择当前销售。
func NewUserEntry(p UserProfile, email, role string) *Entry {
	name := strings.TrimSpace(p.Name)
	content := "平台成员自动生成的使用者画像。Agent 会按当前登录用户自动加载，无需配置“当前销售”。"
	return &Entry{
		Type:    domain.MemoryUser,
		Title:   name,
		UserID:  strings.TrimSpace(p.UserID),
		Content: content,
		Tags:    append([]string(nil), p.Tags...),
		Summary: strings.Trim(strings.TrimSpace(role)+" · "+strings.TrimSpace(email), " ·"),
		Status:  StatusVerified,
		typed: &frontmatter{
			Type:      string(domain.MemoryUser),
			Title:     name,
			Tags:      append([]string(nil), p.Tags...),
			UserID:    strings.TrimSpace(p.UserID),
			Level:     p.Level,
			Expertise: append([]string(nil), p.Expertise...),
			Accuracy:  p.Accuracy,
		},
	}
}

// parsePage 解析一份 Markdown 页面为 Entry（+ 类型化字段暂存于 frontmatter）。
func parsePage(raw string) (*Entry, error) {
	fm, body, ok := splitFrontmatter(raw)
	if !ok {
		return nil, errors.New("缺少 YAML frontmatter（--- 分隔符）")
	}
	var f frontmatter
	if err := yaml.Unmarshal([]byte(fm), &f); err != nil {
		return nil, fmt.Errorf("frontmatter YAML 解析: %w", err)
	}
	mt, ok := domain.ParseMemoryType(f.Type)
	if !ok {
		return nil, fmt.Errorf("未知 type: %q", f.Type)
	}
	status := StatusVerified
	if f.Status != "" {
		status = EntryStatus(f.Status)
	}
	e := &Entry{
		Type:    mt,
		Title:   f.Title,
		UserID:  f.UserID,
		Aliases: f.Aliases,
		Tags:    f.Tags,
		Summary: f.Summary,
		Content: strings.TrimSpace(body),
		Status:  status,
	}
	// 把结构化字段挂到 entry 的扩展字段上（供 typedIndex 使用）。
	e.typed = &f
	if f.Category != "" {
		e.Category = f.Category
	}
	if f.Product != "" {
		e.Product = f.Product
	}
	// P6 时效：历史版本链（superseded 归档）的时间字段透出
	e.ValidAt, e.InvalidAt, e.SupersededBy = f.ValidAt, f.InvalidAt, f.SupersededBy
	// P0-07：待审修订标记透出
	e.RevisionOf = f.RevisionOf
	return e, nil
}

// splitFrontmatter 分离 frontmatter 与正文。返回 (frontmatterYAML, body, ok)。
func splitFrontmatter(raw string) (string, string, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "---") {
		return "", "", false
	}
	rest := raw[3:]
	rest = strings.TrimLeft(rest, "\r\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", "", false
	}
	fm := rest[:end]
	body := rest[end+4:] // 跳过 "\n---"
	body = strings.TrimPrefix(body, "\r\n")
	body = strings.TrimPrefix(body, "\n")
	return fm, body, true
}

// addEntry 把 entry 加入内存索引（含类型化解析）。调用方需持锁。
//
// 历史版本曾把产品主页平铺在「产品记忆/<产品>.md」，新版知识库按产品线
// 放在更深的目录。升级时两份文件可能同时存在，且 WalkDir 的字典序会让浅层
// 旧文件最后加载并覆盖新版正文。相同 type+title 冲突时固定选择目录更深的页面，
// 让「产品线/产品.md」成为权威页；同深度仍保持后加载覆盖，兼容正常热更新。
func (w *WikiStore) addEntry(e *Entry) {
	bucket := w.entries[e.Type]
	if current := bucket[e.Title]; current != nil && entryPathDepth(w.dir, current.FilePath) > entryPathDepth(w.dir, e.FilePath) {
		return
	}
	bucket[e.Title] = e
	w.buildTyped(e)
}

func entryPathDepth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return 0
	}
	depth := 0
	for _, part := range strings.Split(filepath.Clean(rel), string(filepath.Separator)) {
		if part != "" && part != "." {
			depth++
		}
	}
	return depth
}

// buildTyped 从 entry 的 frontmatter 字段构建类型化结构体。
func (w *WikiStore) buildTyped(e *Entry) {
	if e.typed == nil {
		return
	}
	f := e.typed
	switch e.Type {
	case domain.MemoryProduct:
		w.products[e.Title] = &Product{
			Name: e.Title, FullName: f.FullName, Aliases: f.Aliases,
			Category: f.Category, Capabilities: f.Capabilities, Scenarios: f.Scenarios,
			Limitations: f.Limitations, Competitors: f.Competitors, Tags: f.Tags,
			Description: f.Description,
		}
	case domain.MemoryThreat:
		w.threats[e.Title] = &ThreatType{
			Name: e.Title, Aliases: f.Aliases, Description: e.Content,
			TypicalSigns: f.TypicalSigns, RelatedProducts: f.RelatedProducts, Tags: f.Tags,
		}
	case domain.MemoryCompliance:
		w.compliances[e.Title] = &ComplianceRequirement{
			Name: e.Title, Aliases: f.Aliases, Requirements: f.Requirements,
			RelatedProducts: f.RelatedProducts, Tags: f.Tags,
		}
	case domain.MemoryIndustry:
		w.industries[e.Title] = &IndustryScenario{
			Name: e.Title, TypicalPains: f.TypicalPains, CommonNeeds: f.CommonNeeds,
			Keywords: f.Keywords, Tags: f.Tags,
		}
	case domain.MemoryCustomer:
		cp := &CustomerProfile{
			Name: e.Title, Industry: f.Industry, Scale: f.Scale, TechStack: f.TechStack,
			ExistingSecurity: f.ExistingSecurity, PainPoints: f.PainPoints,
			ProcurementPref: f.ProcurementPref, Notes: e.Content, Tags: f.Tags,
			UpdatedAt: f.UpdatedAt,
		}
		if cp.UpdatedAt == "" {
			cp.UpdatedAt = time.Now().Format(time.RFC3339)
		}
		w.customers[e.Title] = cp
	case domain.MemoryUser:
		w.users[e.Title] = &UserProfile{
			Name: e.Title, UserID: f.UserID, Level: f.Level, Expertise: f.Expertise,
			Accuracy: f.Accuracy, Tags: f.Tags,
		}
	}
}

// rebuildIndex 重建关键词倒排索引。调用方需持写锁。
// 关键词与查询走同一个 tokenize（中文 bigram）——否则"扫描防护"（索引）对
// "被扫描"（查询）永远对不上。同 token 命中多字段取最大权重。
func (w *WikiStore) rebuildIndex() {
	w.index = make(map[string]map[indexHit]float64)
	for mt, bucket := range w.entries {
		for title, e := range bucket {
			if e.Status == StatusArchived {
				continue
			}
			for _, wk := range w.weightedKeywords(e) {
				for _, tok := range tokenize(wk.kw) {
					h := indexHit{Type: mt, Title: title}
					if w.index[tok] == nil {
						w.index[tok] = make(map[indexHit]float64)
					}
					if old, ok := w.index[tok][h]; !ok || wk.weight > old {
						w.index[tok][h] = wk.weight
					}
				}
			}
		}
	}
}

type weightedKW struct {
	kw     string
	weight float64
}

// weightedKeywords 汇总一条 entry 的可检索关键词（带字段权重）。
// 权重语义：标题 > 别名 > 标签/关键词 > 能力/场景 > 摘要（泛召回兜底）。
func (w *WikiStore) weightedKeywords(e *Entry) []weightedKW {
	add := func(out []weightedKW, kws []string, wt float64) []weightedKW {
		for _, k := range kws {
			if k == "" {
				continue
			}
			out = append(out, weightedKW{kw: k, weight: wt})
		}
		return out
	}
	out := []weightedKW{{kw: e.Title, weight: 3}}
	out = add(out, e.Aliases, 2.5)
	out = add(out, e.Tags, 2)
	if e.typed != nil {
		f := e.typed
		out = add(out, f.Keywords, 2)
		out = add(out, f.RelatedProducts, 1.5)
		out = add(out, f.Scenarios, 1.5)
		out = add(out, f.TypicalSigns, 1.5)
		for _, c := range f.Capabilities {
			out = add(out, []string{c.Name}, 1.5)
			out = add(out, c.Keywords, 1.5)
		}
	}
	// 摘要 bigram 兜底召回（低权重，只在标题/关键词都对不上时贡献排序）
	if e.Summary != "" {
		out = add(out, []string{e.Summary}, 0.5)
	}
	return out
}
