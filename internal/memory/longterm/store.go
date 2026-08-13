package longterm

import (
	"crypto/sha1"
	"encoding/hex"
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

	// 类型化索引（启动时构建，便于 AllKnowledge / 结构化检索）
	products    map[string]*Product
	threats     map[string]*ThreatType
	compliances map[string]*ComplianceRequirement
	industries  map[string]*IndustryScenario
	customers   map[string]*CustomerProfile
	users       map[string]*UserProfile

	// 关键词倒排索引：keyword(小写) → 命中的 (type, title) 列表
	index map[string][]indexHit
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
		products:    make(map[string]*Product),
		threats:     make(map[string]*ThreatType),
		compliances: make(map[string]*ComplianceRequirement),
		industries:  make(map[string]*IndustryScenario),
		customers:   make(map[string]*CustomerProfile),
		users:       make(map[string]*UserProfile),
		index:       make(map[string][]indexHit),
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

	var loaded int
	err = filepath.WalkDir(w.dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		raw, rErr := os.ReadFile(path)
		if rErr != nil {
			return fmt.Errorf("读取 %s: %w", path, rErr)
		}
		entry, pErr := parsePage(string(raw))
		if pErr != nil {
			return fmt.Errorf("解析 %s: %w", path, pErr)
		}
		if entry.Title == "" {
			return fmt.Errorf("解析 %s: 缺少 title", path)
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
	_ = loaded
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

	// product
	FullName     string       `yaml:"full_name"`
	Category     string       `yaml:"category"`
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
	UpdatedAt        string   `yaml:"updated_at"`
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
		Aliases: f.Aliases,
		Tags:    f.Tags,
		Summary: f.Summary,
		Content: strings.TrimSpace(body),
		Status:  status,
	}
	// 把结构化字段挂到 entry 的扩展字段上（供 typedIndex 使用）。
	e.typed = &f
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
func (w *WikiStore) addEntry(e *Entry) {
	bucket := w.entries[e.Type]
	bucket[e.Title] = e
	w.buildTyped(e)
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
			Name: e.Title, Level: f.Level, Expertise: f.Expertise,
			Accuracy: f.Accuracy, Tags: f.Tags,
		}
	}
}

// rebuildIndex 重建关键词倒排索引。调用方需持写锁。
func (w *WikiStore) rebuildIndex() {
	w.index = make(map[string][]indexHit)
	for mt, bucket := range w.entries {
		for title, e := range bucket {
			if e.Status == StatusArchived {
				continue
			}
			hits := w.keywordsFor(e)
			for _, kw := range hits {
				kw = strings.ToLower(kw)
				w.index[kw] = append(w.index[kw], indexHit{Type: mt, Title: title})
			}
		}
	}
}

// keywordsFor 汇总一条 entry 的全部可检索关键词。
func (w *WikiStore) keywordsFor(e *Entry) []string {
	out := []string{e.Title}
	out = append(out, e.Aliases...)
	out = append(out, e.Tags...)
	if e.typed != nil {
		f := e.typed
		out = append(out, f.Keywords...)
		out = append(out, f.RelatedProducts...)
		out = append(out, f.Scenarios...)
		out = append(out, f.TypicalSigns...)
		for _, c := range f.Capabilities {
			out = append(out, c.Name)
			out = append(out, c.Keywords...)
		}
	}
	return out
}

// newID 生成简短确定性 id。
func newID(prefix string) string {
	h := sha1.Sum([]byte(fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())))
	return prefix + "_" + hex.EncodeToString(h[:4])
}
