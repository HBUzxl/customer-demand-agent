package longterm

import (
	"errors"
	"fmt"
	"sort"

	"customer-demand-agent/internal/domain"
)

// ErrSystemReadOnly 系统知识只读（产品/平台基线），禁止租户写入（§6.5）。
var ErrSystemReadOnly = errors.New("系统知识只读，禁止修改")

// Store 是长期记忆的统一接口（多租户 §6.5）。
// WikiStore 为磁盘实体实现；SystemStore 只读系统基线；TenantStore 读写租户
// 覆盖层；CompositeStore 按记忆类型路由（product→system 只读；
// customer/user→tenant 只读；threat/compliance/industry→system+tenant 合并）。
type Store interface {
	Load() error
	AllProducts() []Product
	GetProduct(name string) (*Product, error)
	SearchProducts(query string) []Product
	SearchThreats(keywords []string) []ThreatType
	SearchCompliance(keywords []string) []ComplianceRequirement
	SearchIndustryScenario(keywords []string) []IndustryScenario
	GetThreat(name string) (*ThreatType, bool)
	GetCompliance(name string) (*ComplianceRequirement, bool)
	GetIndustry(name string) (*IndustryScenario, bool)
	ListChildren(product string) []EntryBrief
	GetCustomerProfile(name string) (*CustomerProfile, error)
	GetUserProfile(name string) (*UserProfile, error)
	GetUserProfileByUserID(userID string) (*UserProfile, error)
	AllKnowledge() *KnowledgeBundle
	SearchEntry(query, typeStr string, limit int) []*Entry
	GetEntryHistory(typeStr, title string) []*Entry
	ListEntry(typeStr string, offset, limit int) []*Entry
	GetEntry(typeStr, title string) (*Entry, error)
	UpsertEntry(e *Entry) error
	SubmitRevision(e *Entry) error
	DeleteEntry(typeStr, title string, archive bool) error
	PendingReviews() []*Entry
	ApproveEntry(typeStr, title string) error
	RejectEntry(typeStr, title string) error
}

// SystemStore 是只读系统知识基线（产品/基础威胁/合规/行业）。Load 后剔除
// customer/user 类型（系统层不携带租户客户与使用者示例），写操作一律拒绝。
type SystemStore struct {
	*WikiStore
}

// NewSystemStore 创建系统知识只读包装。
func NewSystemStore(w *WikiStore) *SystemStore { return &SystemStore{WikiStore: w} }

// Load 加载后剥离客户/使用者类型（系统层示例客户与「张三」不应进租户检索）。
func (s *SystemStore) Load() error {
	if err := s.WikiStore.Load(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range []domain.MemoryType{domain.MemoryCustomer, domain.MemoryUser} {
		for title := range s.entries[t] {
			delete(s.entries[t], title)
			s.deleteTyped(t, title)
		}
	}
	s.rebuildIndex()
	return nil
}

// UpsertEntry 系统只读。
func (s *SystemStore) UpsertEntry(e *Entry) error { return ErrSystemReadOnly }

// SubmitRevision 系统只读。
func (s *SystemStore) SubmitRevision(e *Entry) error { return ErrSystemReadOnly }

// DeleteEntry 系统只读。
func (s *SystemStore) DeleteEntry(typeStr, title string, archive bool) error {
	return ErrSystemReadOnly
}

// ApproveEntry 系统只读。
func (s *SystemStore) ApproveEntry(typeStr, title string) error { return ErrSystemReadOnly }

// RejectEntry 系统只读。
func (s *SystemStore) RejectEntry(typeStr, title string) error { return ErrSystemReadOnly }

// PendingReviews 系统无待审条目。
func (s *SystemStore) PendingReviews() []*Entry { return nil }

// SearchEntry 对 system 层再做一次类型过滤。这样即便装配方漏调
// SystemStore.Load，通用的无 type 检索也不会把历史示例客户/使用者混入租户。
func (s *SystemStore) SearchEntry(query, typeStr string, limit int) []*Entry {
	return filterSystemEntries(s.WikiStore.SearchEntry(query, typeStr, limit))
}

// ListEntry 与 SearchEntry 使用同一纵深过滤。
func (s *SystemStore) ListEntry(typeStr string, offset, limit int) []*Entry {
	return filterSystemEntries(s.WikiStore.ListEntry(typeStr, offset, limit))
}

// GetEntry 禁止从 system 层按通用接口读取租户私有类型。
func (s *SystemStore) GetEntry(typeStr, title string) (*Entry, error) {
	if mt, ok := domain.ParseMemoryType(typeStr); ok && (mt == domain.MemoryCustomer || mt == domain.MemoryUser) {
		return nil, fmt.Errorf("系统知识不包含租户私有类型: %s", typeStr)
	}
	return s.WikiStore.GetEntry(typeStr, title)
}

func filterSystemEntries(entries []*Entry) []*Entry {
	out := make([]*Entry, 0, len(entries))
	for _, e := range entries {
		if e == nil || e.Type == domain.MemoryCustomer || e.Type == domain.MemoryUser {
			continue
		}
		out = append(out, e)
	}
	return out
}

// TenantStore 是租户知识覆盖层（读写）。产品/系统基线类型禁止租户写入
// （纵深防御：Composite 已路由，这里再加一道闸）。
type TenantStore struct {
	*WikiStore
}

// NewTenantStore 创建租户知识存储包装。
func NewTenantStore(w *WikiStore) *TenantStore { return &TenantStore{WikiStore: w} }

// UpsertEntry 禁止写入产品（系统只读类型）。
func (t *TenantStore) UpsertEntry(e *Entry) error {
	if e.Type == domain.MemoryProduct {
		return ErrSystemReadOnly
	}
	return t.WikiStore.UpsertEntry(e)
}

// SubmitRevision 禁止修订产品。
func (t *TenantStore) SubmitRevision(e *Entry) error {
	if e.Type == domain.MemoryProduct {
		return ErrSystemReadOnly
	}
	return t.WikiStore.SubmitRevision(e)
}

// DeleteEntry 禁止删除产品。
func (t *TenantStore) DeleteEntry(typeStr, title string, archive bool) error {
	if mt, ok := domain.ParseMemoryType(typeStr); ok && mt == domain.MemoryProduct {
		return ErrSystemReadOnly
	}
	return t.WikiStore.DeleteEntry(typeStr, title, archive)
}

// ApproveEntry 禁止批准系统类型。
func (t *TenantStore) ApproveEntry(typeStr, title string) error {
	if mt, ok := domain.ParseMemoryType(typeStr); ok && mt == domain.MemoryProduct {
		return ErrSystemReadOnly
	}
	return t.WikiStore.ApproveEntry(typeStr, title)
}

// RejectEntry 禁止拒绝系统类型。
func (t *TenantStore) RejectEntry(typeStr, title string) error {
	if mt, ok := domain.ParseMemoryType(typeStr); ok && mt == domain.MemoryProduct {
		return ErrSystemReadOnly
	}
	return t.WikiStore.RejectEntry(typeStr, title)
}

// CompositeStore 路由规则（§6.5）：
//   - product → system 只读（绝不回退租户）；
//   - customer/user → tenant 只读（绝不回退系统示例）；
//   - threat/compliance/industry → system 基线 + tenant 覆盖层合并，(type,title)
//     去重、tenant 优先。
//   - 写操作（Upsert/Delete/Approve/Reject/PendingReviews）只落 tenant。
type CompositeStore struct {
	system Store
	tenant Store
}

// NewCompositeStore 构造复合记忆（system=系统只读，tenant=租户覆盖层）。
func NewCompositeStore(system, tenant Store) *CompositeStore {
	return &CompositeStore{system: system, tenant: tenant}
}

// Load 加载两层（系统先、租户后）。
func (c *CompositeStore) Load() error {
	if err := c.system.Load(); err != nil {
		return err
	}
	return c.tenant.Load()
}

// ── product：系统只读 ──────────────────────────────────────────

func (c *CompositeStore) AllProducts() []Product { return c.system.AllProducts() }
func (c *CompositeStore) GetProduct(name string) (*Product, error) {
	return c.system.GetProduct(name)
}
func (c *CompositeStore) SearchProducts(query string) []Product {
	return c.system.SearchProducts(query)
}
func (c *CompositeStore) ListChildren(product string) []EntryBrief {
	return c.system.ListChildren(product)
}

// ── customer/user：租户只读 ───────────────────────────────────

func (c *CompositeStore) GetCustomerProfile(name string) (*CustomerProfile, error) {
	return c.tenant.GetCustomerProfile(name)
}
func (c *CompositeStore) GetUserProfile(name string) (*UserProfile, error) {
	return c.tenant.GetUserProfile(name)
}
func (c *CompositeStore) GetUserProfileByUserID(userID string) (*UserProfile, error) {
	return c.tenant.GetUserProfileByUserID(userID)
}

// ── threat/compliance/industry：合并（tenant 优先）────────────

func (c *CompositeStore) SearchThreats(keywords []string) []ThreatType {
	return mergeByNameThreats(c.system.SearchThreats(keywords), c.tenant.SearchThreats(keywords))
}
func (c *CompositeStore) SearchCompliance(keywords []string) []ComplianceRequirement {
	return mergeByNameCompliance(c.system.SearchCompliance(keywords), c.tenant.SearchCompliance(keywords))
}
func (c *CompositeStore) SearchIndustryScenario(keywords []string) []IndustryScenario {
	return mergeByNameIndustry(c.system.SearchIndustryScenario(keywords), c.tenant.SearchIndustryScenario(keywords))
}
func (c *CompositeStore) GetThreat(name string) (*ThreatType, bool) {
	if t, ok := c.tenant.GetThreat(name); ok {
		return t, true
	}
	return c.system.GetThreat(name)
}
func (c *CompositeStore) GetCompliance(name string) (*ComplianceRequirement, bool) {
	if t, ok := c.tenant.GetCompliance(name); ok {
		return t, true
	}
	return c.system.GetCompliance(name)
}
func (c *CompositeStore) GetIndustry(name string) (*IndustryScenario, bool) {
	if t, ok := c.tenant.GetIndustry(name); ok {
		return t, true
	}
	return c.system.GetIndustry(name)
}

// ── 通用 Entry 操作 ───────────────────────────────────────────

// SearchEntry 跨两层检索（tenant 优先去重）。
func (c *CompositeStore) SearchEntry(query, typeStr string, limit int) []*Entry {
	return mergeEntries(c.tenant.SearchEntry(query, typeStr, limit), c.system.SearchEntry(query, typeStr, limit))
}

// ListEntry 合并两层（tenant 优先），再分页。
func (c *CompositeStore) ListEntry(typeStr string, offset, limit int) []*Entry {
	merged := mergeEntries(c.tenant.ListEntry(typeStr, 0, 1000), c.system.ListEntry(typeStr, 0, 1000))
	if limit <= 0 {
		limit = 50
	}
	if offset > len(merged) {
		offset = len(merged)
	}
	merged = merged[offset:]
	if limit < len(merged) {
		merged = merged[:limit]
	}
	return merged
}

// GetEntry 按类型路由：product→system；customer/user→tenant；其余 tenant 优先。
func (c *CompositeStore) GetEntry(typeStr, title string) (*Entry, error) {
	mt, ok := domain.ParseMemoryType(typeStr)
	if !ok {
		return nil, fmt.Errorf("未知 type: %q", typeStr)
	}
	switch mt {
	case domain.MemoryProduct:
		return c.system.GetEntry(typeStr, title)
	case domain.MemoryCustomer, domain.MemoryUser:
		return c.tenant.GetEntry(typeStr, title)
	default:
		if e, err := c.tenant.GetEntry(typeStr, title); err == nil {
			return e, nil
		}
		return c.system.GetEntry(typeStr, title)
	}
}

// GetEntryHistory 查租户覆盖层版本链（系统基线版本由平台维护）。
func (c *CompositeStore) GetEntryHistory(typeStr, title string) []*Entry {
	return c.tenant.GetEntryHistory(typeStr, title)
}

// ── 写操作：只落租户（系统类型由 TenantStore 闸门拒绝）────────

func (c *CompositeStore) UpsertEntry(e *Entry) error    { return c.tenant.UpsertEntry(e) }
func (c *CompositeStore) SubmitRevision(e *Entry) error { return c.tenant.SubmitRevision(e) }
func (c *CompositeStore) DeleteEntry(typeStr, title string, archive bool) error {
	return c.tenant.DeleteEntry(typeStr, title, archive)
}
func (c *CompositeStore) PendingReviews() []*Entry { return c.tenant.PendingReviews() }
func (c *CompositeStore) ApproveEntry(typeStr, title string) error {
	return c.tenant.ApproveEntry(typeStr, title)
}
func (c *CompositeStore) RejectEntry(typeStr, title string) error {
	return c.tenant.RejectEntry(typeStr, title)
}

func (c *CompositeStore) AllKnowledge() *KnowledgeBundle {
	return &KnowledgeBundle{
		Products:    c.AllProducts(),
		Threats:     c.SearchThreats(nil),
		Compliances: c.SearchCompliance(nil),
		Industries:  c.SearchIndustryScenario(nil),
	}
}

// ── 合并辅助 ───────────────────────────────────────────────────

// mergeByName 合并两层类型化列表（tenant 优先，(name) 去重）。
func mergeByName[T any](sys, ten []T, nameOf func(T) string) []T {
	out := make([]T, 0, len(sys)+len(ten))
	seen := map[string]bool{}
	for _, t := range ten {
		if nameOf(t) == "" {
			continue
		}
		seen[nameOf(t)] = true
		out = append(out, t)
	}
	for _, s := range sys {
		if !seen[nameOf(s)] {
			out = append(out, s)
		}
	}
	return out
}

// mergeEntries 合并两层 Entry 切片（tenant 优先，(type,title) 去重），
// 按 Relevance 降序（SearchEntry 场景）——稳定排序保持同分时 system 在前的
// 稳定顺序。写入顺序：先 system 后 tenant，tenant 后写覆盖 = tenant 优先
// （P0-05：原实现先 tenant 后 system，system 反而覆盖同键 tenant，与注释和
// 产品目标相反——Content/Summary/Status 应取自租户覆盖层）。
func mergeEntries(ten, sys []*Entry) []*Entry {
	byKey := map[string]*Entry{}
	var keys []string
	add := func(l []*Entry) {
		for _, e := range l {
			if e == nil {
				continue
			}
			key := string(e.Type) + "\x00" + e.Title
			if _, exists := byKey[key]; !exists {
				keys = append(keys, key)
			}
			byKey[key] = e // 后写覆盖：tenant 必须在 system 之后
		}
	}
	add(sys)
	add(ten)
	out := make([]*Entry, 0, len(keys))
	for _, k := range keys {
		out = append(out, byKey[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Relevance > out[j].Relevance })
	return out
}

// 显式实现泛型 mergeByName 的 nameOf 参数化（Go 无法推导命名参数，
// 这里为每种类型提供适配器避免调用处重复写闭包）。
func mergeByNameThreats(sys, ten []ThreatType) []ThreatType {
	return mergeByName(sys, ten, func(t ThreatType) string { return t.Name })
}
func mergeByNameCompliance(sys, ten []ComplianceRequirement) []ComplianceRequirement {
	return mergeByName(sys, ten, func(t ComplianceRequirement) string { return t.Name })
}
func mergeByNameIndustry(sys, ten []IndustryScenario) []IndustryScenario {
	return mergeByName(sys, ten, func(t IndustryScenario) string { return t.Name })
}
