// Package longterm implements the long-term memory layer as a Wiki adapter.
//
// Wiki 层承载符号化知识（产品/行业/用户记忆），检索是确定性的关键词匹配，
// 不依赖向量相似度（不做 RAG，见 ADR-002）。页面以 Markdown + YAML
// frontmatter 存储，启动时全量加载进内存并构建索引。
//
// 三层记忆分类（ADR-004）：
//   - 产品记忆（product）：能力/场景/边界/竞品，AI 只读
//   - 行业记忆（industry）：威胁类型/合规条款/行业场景，AI 可写但打待审核标记
//   - 用户记忆（user）：客户画像（AI 全权）+ 使用者画像（管理员维护）
package longterm

import "customer-demand-agent/internal/domain"

// Product 是一款长亭产品的结构化知识。
type Product struct {
	Name         string       `json:"name"`         // 雷池
	FullName     string       `json:"full_name"`    // 长亭雷池下一代 Web 应用防火墙
	Aliases      []string     `json:"aliases"`      // ["WAF", "SafeLine"]
	Category     string       `json:"category"`     // 边界安全 / 漏洞扫描 / ...
	Capabilities []Capability `json:"capabilities"` // 结构化能力
	Scenarios    []string     `json:"scenarios"`    // 适用场景
	Limitations  []string     `json:"limitations"`  // 能力边界
	Competitors  []Competitor `json:"competitors"`  // 竞品对比
	Tags         []string     `json:"tags"`         // 检索标签
	Description  string       `json:"description"`  // 简介
}

// Capability 是产品的一项能力。
type Capability struct {
	Name        string   `json:"name"`        // CC 攻击防护
	Description string   `json:"description"` // 详细说明
	Confidence  float64  `json:"confidence"`  // 1.0=核心能力, 0.5=边缘能力
	Keywords    []string `json:"keywords"`    // 检索关键词
}

// Competitor 是一个竞品对比项。
type Competitor struct {
	Name    string `json:"name"`
	Compare string `json:"compare"` // 对比说明（优势/差异）
}

// ThreatType 是一类安全威胁。
type ThreatType struct {
	Name            string   `json:"name"`             // CC 攻击
	Aliases         []string `json:"aliases"`          // ["CC", "HTTP Flood"]
	Description     string   `json:"description"`      // 攻击原理
	TypicalSigns    []string `json:"typical_signs"`    // 典型表现
	RelatedProducts []string `json:"related_products"` // 关联产品名
	Tags            []string `json:"tags"`
}

// ComplianceRequirement 是一条合规要求。
type ComplianceRequirement struct {
	Name            string   `json:"name"` // 等保三级
	Aliases         []string `json:"aliases"`
	Requirements    []string `json:"requirements"` // 具体要求
	RelatedProducts []string `json:"related_products"`
	Tags            []string `json:"tags"`
}

// IndustryScenario 是一个行业场景。
type IndustryScenario struct {
	Name         string   `json:"name"`          // 电商
	TypicalPains []string `json:"typical_pains"` // 典型痛点
	CommonNeeds  []string `json:"common_needs"`  // 常见需求
	Keywords     []string `json:"keywords"`
	Tags         []string `json:"tags"`
}

// CustomerProfile 是一个客户画像（AI 可自主维护，ADR-005）。
type CustomerProfile struct {
	Name             string   `json:"name"`              // 客户名称（脱敏）
	Industry         string   `json:"industry"`          // 行业
	Scale            string   `json:"scale"`             // 规模
	TechStack        []string `json:"tech_stack"`        // 技术栈
	ExistingSecurity []string `json:"existing_security"` // 已有安全能力
	PainPoints       []string `json:"pain_points"`       // 已知痛点
	ProcurementPref  string   `json:"procurement_pref"`  // 采购偏好
	Notes            string   `json:"notes"`             // 备注
	Tags             []string `json:"tags"`
	UpdatedAt        string   `json:"updated_at"`
}

// UserProfile 是一个使用者（销售）画像。
type UserProfile struct {
	Name      string   `json:"name"`      // 销售姓名
	Level     string   `json:"level"`     // 初级/中级/高级
	Expertise []string `json:"expertise"` // 擅长领域
	Accuracy  float64  `json:"accuracy"`  // 历史判断准确率
	Tags      []string `json:"tags"`
}

// Entry 是一条通用记忆条目（记忆工具的统一读写单位）。
// 不同记忆类型共享这个扁平结构，便于 memory_search/memory_list 跨类型检索。
type Entry struct {
	Type      domain.MemoryType `json:"type"`
	Title     string            `json:"title"` // 唯一标识
	Aliases   []string          `json:"aliases"`
	Tags      []string          `json:"tags"`
	Summary   string            `json:"summary"`             // 摘要
	Content   string            `json:"content"`             // Markdown 正文
	Category  string            `json:"category,omitempty"`  // 分类（产品的安全域）
	Product   string            `json:"product,omitempty"`   // 所属产品（子文档关联到产品）
	Status    EntryStatus       `json:"status"`              // verified / pending_review / archived
	Relevance float64           `json:"relevance,omitempty"` // 检索时填充
	FilePath  string            `json:"-"`                   // 对应的磁盘文件（写回用）
	typed     *frontmatter      `json:"-"`                   // frontmatter 结构化字段（类型化解析用）
}

// EntryStatus 是记忆条目的审核状态（待审核机制，ADR-005）。
type EntryStatus string

const (
	// StatusVerified 已通过审核，可直接用于分析
	StatusVerified EntryStatus = "verified"
	// StatusPendingReview AI 写入的待审核条目（行业记忆），可用于分析但降权
	StatusPendingReview EntryStatus = "pending_review"
	// StatusArchived 归档（软删除）
	StatusArchived EntryStatus = "archived"
)

// KnowledgeBundle 是全量知识的打包（用于注入 system prompt）。
type KnowledgeBundle struct {
	Products    []Product               `json:"products"`
	Threats     []ThreatType            `json:"threats"`
	Compliances []ComplianceRequirement `json:"compliances"`
	Industries  []IndustryScenario      `json:"industries"`
}
