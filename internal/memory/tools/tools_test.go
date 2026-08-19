package tools

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"customer-demand-agent/internal/domain"
	"customer-demand-agent/internal/memory/longterm"
)

func newTestRegistry(t *testing.T) (*Registry, *longterm.WikiStore) {
	t.Helper()
	store := longterm.NewWikiStore(t.TempDir())
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}
	return NewRegistry(store), store
}

func TestPermissionMatrix(t *testing.T) {
	r, _ := newTestRegistry(t)

	// product 不可写
	_, err := r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "product", "title": "雷池", "content": "x",
	}))
	if err == nil {
		t.Fatal("writing product should be rejected")
	}

	// threat 可写但打待审核
	_, err = r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "threat", "title": "新型威胁", "content": "测试",
	}))
	if err != nil {
		t.Fatalf("writing threat should succeed: %v", err)
	}
	e, _ := r.store.GetEntry("threat", "新型威胁")
	if e.Status != longterm.StatusPendingReview {
		t.Fatalf("threat should be pending_review, got %s", e.Status)
	}

	// customer 可写且直接 verified
	_, err = r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "customer", "title": "客户A", "content": "x",
	}))
	if err != nil {
		t.Fatalf("writing customer should succeed: %v", err)
	}
	e, _ = r.store.GetEntry("customer", "客户A")
	if e.Status != longterm.StatusVerified {
		t.Fatalf("customer should be verified, got %s", e.Status)
	}

	// user 不可写
	_, err = r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "user", "title": "张三", "content": "x",
	}))
	if err == nil {
		t.Fatal("writing user should be rejected")
	}
}

func TestDeletePermission(t *testing.T) {
	r, _ := newTestRegistry(t)
	// 先写一个 customer
	_, _ = r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "customer", "title": "客户X", "content": "x",
	}))
	// customer 可删
	_, err := r.Execute("memory_delete", rawJSON(map[string]any{
		"type": "customer", "title": "客户X",
	}))
	if err != nil {
		t.Fatalf("deleting customer should succeed: %v", err)
	}
	// threat 不可删
	_, _ = r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "threat", "title": "威胁Y", "content": "x",
	}))
	_, err = r.Execute("memory_delete", rawJSON(map[string]any{
		"type": "threat", "title": "威胁Y",
	}))
	if err == nil {
		t.Fatal("deleting threat should be rejected")
	}
}

func TestUnknownTool(t *testing.T) {
	r, _ := newTestRegistry(t)
	_, err := r.Execute("memory_nonexistent", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("unknown tool should error")
	}
}

func TestSearchAndList(t *testing.T) {
	r, _ := newTestRegistry(t)
	// 写几条
	_, _ = r.Execute("memory_ensure", rawJSON(map[string]any{"type": "customer", "title": "电商客户", "content": "做电商", "tags": []string{"电商"}}))
	_, _ = r.Execute("memory_ensure", rawJSON(map[string]any{"type": "customer", "title": "金融客户", "content": "做金融"}))

	// list
	out, err := r.Execute("memory_list", rawJSON(map[string]any{"type": "customer"}))
	if err != nil {
		t.Fatal(err)
	}
	var lr struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal([]byte(out), &lr)
	if lr.Count != 2 {
		t.Fatalf("expected 2 customers, got %d", lr.Count)
	}

	// search
	out, err = r.Execute("memory_search", rawJSON(map[string]any{"query": "电商", "type": "customer"}))
	if err != nil {
		t.Fatal(err)
	}
	var sr struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal([]byte(out), &sr)
	if sr.Count != 1 {
		t.Fatalf("search 电商 expected 1, got %d", sr.Count)
	}
}

func TestDefinitions(t *testing.T) {
	r, _ := newTestRegistry(t)
	defs := r.Definitions()
	if len(defs) != 7 {
		t.Fatalf("expected 7 tool definitions, got %d", len(defs))
	}
	names := map[string]bool{}
	for _, d := range defs {
		names[d.Function.Name] = true
	}
	for _, want := range []string{"memory_search", "memory_get", "memory_ensure", "memory_observe", "memory_delete", "memory_recall", "memory_list"} {
		if !names[want] {
			t.Errorf("missing tool definition: %s", want)
		}
	}
	_ = os.Stdout
}

func TestAgentDefinitionsAndExecutionRespectTenantRole(t *testing.T) {
	r, store := newTestRegistry(t)
	analyst := &domain.TenantScope{TenantID: "t1", UserID: "u1", Roles: []string{domain.RoleAnalyst}}
	manager := &domain.TenantScope{TenantID: "t1", UserID: "u2", Roles: []string{domain.RoleAdmin}}

	names := func(defs []domain.Tool) map[string]bool {
		out := make(map[string]bool, len(defs))
		for _, d := range defs {
			out[d.Function.Name] = true
		}
		return out
	}
	analystDefs := names(r.DefinitionsFor(analyst))
	if analystDefs["memory_ensure"] || analystDefs["memory_observe"] || analystDefs["memory_delete"] {
		t.Fatalf("analyst 只能看到读工具，got %+v", analystDefs)
	}
	managerDefs := names(r.DefinitionsFor(manager))
	if !managerDefs["memory_ensure"] || !managerDefs["memory_observe"] || managerDefs["memory_delete"] {
		t.Fatalf("manager 应可写但模型永不获得 delete，got %+v", managerDefs)
	}

	args := rawJSON(map[string]any{"type": "customer", "title": "越权客户", "content": "x"})
	if _, err := r.ExecuteFor(analyst, "memory_ensure", args); err == nil {
		t.Fatal("即使伪造工具调用，analyst 写入也应被执行层拒绝")
	}
	if _, err := store.GetEntry("customer", "越权客户"); err == nil {
		t.Fatal("被拒的越权写入不应产生数据")
	}
	if _, err := r.ExecuteFor(manager, "memory_ensure", args); err != nil {
		t.Fatalf("manager 写入应成功: %v", err)
	}
	if _, err := r.ExecuteFor(manager, "memory_delete", rawJSON(map[string]any{"type": "customer", "title": "越权客户"})); err == nil {
		t.Fatal("模型侧 delete 对 manager 也必须拒绝")
	}
}

func rawJSON(m map[string]any) json.RawMessage {
	b, _ := json.Marshal(m)
	return b
}

var _ domain.MemoryType // keep import

// TestEnsureOverwriteVerifiedDemotes P0-07：AI 覆盖已 verified 受控知识时，
// 生成 pending revision——活跃条目保持 verified 继续生效（Agent 检索不到半成品），
// 审核队列出现修订；批准后原子替换、拒绝则保持原样。
func TestEnsureOverwriteVerifiedDemotes(t *testing.T) {
	r, store := newTestRegistry(t)
	// 人工先建一条 verified 威胁（模拟已审核通过）
	if err := store.UpsertEntry(&longterm.Entry{
		Type: domain.MemoryThreat, Title: "已审威胁", Status: longterm.StatusVerified,
		Content: "原始内容",
	}); err != nil {
		t.Fatal(err)
	}
	// AI ensure 覆盖
	if _, err := r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "threat", "title": "已审威胁", "content": "AI 改写的内容",
	})); err != nil {
		t.Fatal(err)
	}
	// 活跃条目保持 verified、内容不变（原版本继续生效）
	e, err := store.GetEntry("threat", "已审威胁")
	if err != nil {
		t.Fatal(err)
	}
	if e.Status != longterm.StatusVerified {
		t.Fatalf("活跃条目应保持 verified，got %s", e.Status)
	}
	if !strings.Contains(e.Content, "原始内容") {
		t.Fatalf("活跃条目内容不应被未审修订改动，got %q", e.Content)
	}
	// 审核队列出现修订
	pending := store.PendingReviews()
	if len(pending) != 1 || pending[0].Title != "已审威胁" || pending[0].RevisionOf == "" {
		t.Fatalf("应出现一条 pending revision，got %+v", pending)
	}
	// 批准 → 原子替换为修订内容，保持 verified
	if err := store.ApproveEntry("threat", "已审威胁"); err != nil {
		t.Fatal(err)
	}
	e, _ = store.GetEntry("threat", "已审威胁")
	if !strings.Contains(e.Content, "AI 改写的内容") {
		t.Fatalf("批准后应为修订内容，got %q", e.Content)
	}
	if e.Status != longterm.StatusVerified {
		t.Fatalf("批准后应保持 verified，got %s", e.Status)
	}
	if len(store.PendingReviews()) != 0 {
		t.Fatal("批准后审核队列应清空")
	}
}

// TestObserveVerifiedKnowledgeCreatesPendingRevision P0-07 验收：对已验证
// threat/compliance/industry 追加观察必须生成 pending revision，不得保持 verified
// 直接生效；拒绝修订时原版本不受影响。customer 观察不受审（AI 全权）。
func TestObserveVerifiedKnowledgeCreatesPendingRevision(t *testing.T) {
	r, store := newTestRegistry(t)
	// 已验证威胁（模拟审核通过）
	if err := store.UpsertEntry(&longterm.Entry{
		Type: domain.MemoryThreat, Title: "已审威胁", Status: longterm.StatusVerified,
		Content: "原始内容",
	}); err != nil {
		t.Fatal(err)
	}
	// observe 追加
	out, err := r.Execute("memory_observe", rawJSON(map[string]any{
		"type": "threat", "title": "已审威胁", "content": "客户观察到新现象", "relevance": "high",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"needs_review":true`) {
		t.Fatalf("observe 已验证受控知识应 needs_review:true，got %s", out)
	}
	// 活跃条目保持 verified + 原内容
	e, _ := store.GetEntry("threat", "已审威胁")
	if e.Status != longterm.StatusVerified || !strings.Contains(e.Content, "原始内容") {
		t.Fatalf("活跃条目应保持 verified 原内容，got status=%s content=%q", e.Status, e.Content)
	}
	// 审核队列含修订
	pending := store.PendingReviews()
	if len(pending) != 1 || pending[0].RevisionOf == "" {
		t.Fatalf("应出现 pending revision，got %+v", pending)
	}
	// 拒绝修订 → 原版本不受影响，队列清空
	if err := store.RejectEntry("threat", "已审威胁"); err != nil {
		t.Fatal(err)
	}
	e, _ = store.GetEntry("threat", "已审威胁")
	if e.Status != longterm.StatusVerified || !strings.Contains(e.Content, "原始内容") {
		t.Fatalf("拒绝后原版本应保持，got %q", e.Content)
	}
	if len(store.PendingReviews()) != 0 {
		t.Fatal("拒绝后审核队列应清空")
	}
	// customer observe：无审核，直接生效
	if _, err := r.Execute("memory_observe", rawJSON(map[string]any{
		"type": "customer", "title": "某客户", "content": "客户观察", "relevance": "medium",
	})); err != nil {
		t.Fatal(err)
	}
	if len(store.PendingReviews()) != 0 {
		t.Fatalf("customer observe 不应进审核队列，got %d", len(store.PendingReviews()))
	}
}

// TestEnsureObserveNeedsReviewFlag F4：result JSON 带 needs_review
// （threat 新建 true；customer false；observe 新建 threat true）。
func TestEnsureObserveNeedsReviewFlag(t *testing.T) {
	r, _ := newTestRegistry(t)
	// threat ensure → true
	out, err := r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "threat", "title": "F4威胁", "content": "x",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"needs_review":true`) {
		t.Fatalf("threat ensure 应 needs_review:true，got %s", out)
	}
	// customer ensure → false
	out, _ = r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "customer", "title": "F4客户", "content": "x",
	}))
	if !strings.Contains(string(out), `"needs_review":false`) {
		t.Fatalf("customer ensure 应 needs_review:false，got %s", out)
	}
	// observe 新建 threat → true
	out, _ = r.Execute("memory_observe", rawJSON(map[string]any{
		"type": "threat", "title": "F4观察", "content": "洞察", "relevance": "medium",
	}))
	if !strings.Contains(string(out), `"needs_review":true`) {
		t.Fatalf("observe 新建 threat 应 needs_review:true，got %s", out)
	}
}

// TestEnsureExistingHint F2 防重复建条目：customer 名互相包含时 result 带
// existing_hint 警告（续写指引）。
func TestEnsureExistingHint(t *testing.T) {
	dir := t.TempDir()
	wiki := longterm.NewWikiStore(dir)
	if err := wiki.Load(); err != nil {
		t.Fatal(err)
	}
	if err := wiki.UpsertEntry(&longterm.Entry{Type: domain.MemoryCustomer, Title: "某跨境电商平台", Content: "已有画像", Status: longterm.StatusVerified}); err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry(wiki)
	// 新建互相包含的变体名 → hint
	out, err := reg.Execute("memory_ensure", json.RawMessage(`{"type":"customer","title":"某跨境电商","content":"新画像"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "existing_hint") || !strings.Contains(out, "某跨境电商平台") {
		t.Fatalf("应带 existing_hint: %s", out)
	}
	// 同名更新 → 无 hint
	out2, err := reg.Execute("memory_ensure", json.RawMessage(`{"type":"customer","title":"某跨境电商平台","content":"更新"}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out2, "existing_hint") {
		t.Fatalf("同名更新不应有 hint: %s", out2)
	}
}

// TestReviewGatingVisibleToAgent 审核门禁（review-gating 2026-08-15）：
// AI 写 threat → pending_review → SearchEntry/ListEntry/getProfile 均 miss
// → approve → 全部 hit/可见。批准即生效。
func TestReviewGatingVisibleToAgent(t *testing.T) {
	dir := t.TempDir()
	wiki := longterm.NewWikiStore(dir)
	if err := wiki.Load(); err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry(wiki)
	// 1) AI 写 threat → pending
	out, err := reg.Execute("memory_ensure", json.RawMessage(`{"type":"threat","title":"零信任沙箱逃逸","content":"新型威胁描述","tags":["零信任"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"needs_review":true`) {
		t.Fatalf("threat 写入应 pending: %s", out)
	}
	// 2) 审批前：检索 miss（门禁核心断言）
	if hits := wiki.SearchEntry("零信任沙箱逃逸", "threat", 5); len(hits) != 0 {
		t.Fatalf("审批前检索应 miss: %+v", hits)
	}
	// 3) approve → verified
	if err := wiki.ApproveEntry("threat", "零信任沙箱逃逸"); err != nil {
		t.Fatal(err)
	}
	// 4) 审批后：hit
	if hits := wiki.SearchEntry("零信任沙箱逃逸", "threat", 5); len(hits) != 1 {
		t.Fatalf("批准后检索应 hit: %+v", hits)
	}
}

// TestReviewGatingCustomerProfilePending 客户画像门禁：pending 客户画像
// GetCustomerProfile 不可见，approve 后可见（assembler L2 注入前提）。
func TestReviewGatingCustomerProfilePending(t *testing.T) {
	dir := t.TempDir()
	wiki := longterm.NewWikiStore(dir)
	if err := wiki.Load(); err != nil {
		t.Fatal(err)
	}
	if err := wiki.UpsertEntry(&longterm.Entry{Type: domain.MemoryCustomer, Title: "待审客户", Content: "画像", Status: longterm.StatusPendingReview}); err != nil {
		t.Fatal(err)
	}
	if _, err := wiki.GetCustomerProfile("待审客户"); err == nil {
		t.Fatal("pending 客户画像应不可见")
	}
	if err := wiki.ApproveEntry("customer", "待审客户"); err != nil {
		t.Fatal(err)
	}
	if _, err := wiki.GetCustomerProfile("待审客户"); err != nil {
		t.Fatalf("批准后应可见: %v", err)
	}
}

// ── memory_get ────────────────────────────────────────────────

// seedProductPages 造一个产品主页（结构化字段）+ 子文档，返回 registry。
func seedProductPages(t *testing.T) *Registry {
	t.Helper()
	r, store := newTestRegistry(t)
	main := `---
type: product
title: 雷池
aliases: ["WAF", "SafeLine"]
full_name: 长亭雷池下一代 WAF
capabilities:
  - {name: CC 攻击防护, confidence: 1.0, keywords: ["CC", "CC攻击"], description: 频率与行为识别并阻断}
  - {name: Web 扫描防护, confidence: 0.9, keywords: ["扫描", "爬虫"], description: 识别拦截扫描器与爬虫探测}
scenarios: ["网站被扫描或攻击"]
limitations: ["不做网络层 DDoS 清洗", "主要防护 7 层 HTTP/HTTPS"]
competitors:
  - {name: 传统正则 WAF, compare: 误报率更低}
---
雷池是语义分析 WAF。`
	if err := store.UpsertEntry(&longterm.Entry{
		Type: domain.MemoryProduct, Title: "雷池",
		Content: main, Summary: "下一代 WAF", Status: longterm.StatusVerified,
	}); err != nil {
		t.Fatal(err)
	}
	child := `---
type: product
title: 雷池-FAQ
product: 雷池
summary: 高频问题：误报调优、性能
---
Q: 误报太多怎么办？可加白名单或调整防护等级。`
	if err := store.UpsertEntry(&longterm.Entry{
		Type: domain.MemoryProduct, Title: "雷池-FAQ", Product: "雷池",
		Content: child, Summary: "高频问题：误报调优、性能", Status: longterm.StatusVerified,
	}); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestGetProductFull(t *testing.T) {
	r := seedProductPages(t)
	out, err := r.Execute("memory_get", rawJSON(map[string]any{"type": "product", "title": "雷池"}))
	if err != nil {
		t.Fatal(err)
	}
	// 结构化字段、置信度、能力边界、竞品、子文档导航、正文都要出现
	for _, want := range []string{
		"## 能力（置信度）",
		"CC 攻击防护",
		"[1.0]",
		"## 能力边界",
		"不做网络层 DDoS 清洗",
		"## 竞品对比",
		"传统正则 WAF",
		"## 相关文档",
		"雷池-FAQ",
		"## 正文",
		"语义分析 WAF",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("get 雷池应含 %q\n--- got ---\n%s", want, out)
		}
	}
	// 主页不被列进自己的相关文档
	if strings.Contains(out, "- 雷池：") {
		t.Errorf("相关文档不应包含主页自身:\n%s", out)
	}
}

func TestGetChildDoc(t *testing.T) {
	r := seedProductPages(t)
	out, err := r.Execute("memory_get", rawJSON(map[string]any{"type": "product", "title": "雷池-FAQ"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "所属产品：雷池") {
		t.Errorf("子文档应指回主页:\n%s", out)
	}
	if !strings.Contains(out, "误报太多怎么办") {
		t.Errorf("子文档正文应可见:\n%s", out)
	}
}

func TestGetNotFoundSuggests(t *testing.T) {
	r := seedProductPages(t)
	out, err := r.Execute("memory_get", rawJSON(map[string]any{"type": "product", "title": "雷池WAF不存在"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "不存在或不可见") || !strings.Contains(out, "雷池") {
		t.Errorf("not found 应带相近候选:\n%s", out)
	}
}

func TestGetPendingInvisible(t *testing.T) {
	r, _ := newTestRegistry(t)
	// threat 经 ensure 写入 → pending，对 Agent 不可见
	if _, err := r.Execute("memory_ensure", rawJSON(map[string]any{
		"type": "threat", "title": "新型攻击", "content": "细节",
	})); err != nil {
		t.Fatal(err)
	}
	out, err := r.Execute("memory_get", rawJSON(map[string]any{"type": "threat", "title": "新型攻击"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "不存在或不可见") {
		t.Errorf("pending 条目 get 应不可见:\n%s", out)
	}
}

func TestGetLongBodySectionNav(t *testing.T) {
	r, store := newTestRegistry(t)
	var sb strings.Builder
	sb.WriteString("这是一份超长手册的引言。\n\n")
	sb.WriteString("## 部署\n\n" + strings.Repeat("部署细节说明。", 500) + "\n\n") // 3500 字
	sb.WriteString("## 配置\n\n" + strings.Repeat("配置项讲解。", 500) + "配置区唯一标记。\n\n")
	sb.WriteString("## 排障\n\n排障章节唯一标记。\n")
	if err := store.UpsertEntry(&longterm.Entry{
		Type: domain.MemoryProduct, Title: "手册", Content: sb.String(), Status: longterm.StatusVerified,
	}); err != nil {
		t.Fatal(err)
	}

	// 1) 首次 get：超长 → 只给章节目录，不给正文内容
	out, err := r.Execute("memory_get", rawJSON(map[string]any{"type": "product", "title": "手册"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "正文目录") || !strings.Contains(out, "- 部署") || !strings.Contains(out, "配置") {
		t.Errorf("超长正文首次应给章节目录:\n%.500s", out)
	}
	if strings.Contains(out, "配置区唯一标记") || strings.Contains(out, "排障章节唯一标记") {
		t.Errorf("首次 get 不应包含正文内容:\n%.500s", out)
	}

	// 2) section 精读：拿到指定章节内容
	out, err = r.Execute("memory_get", rawJSON(map[string]any{"type": "product", "title": "手册", "section": "配置"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "配置区唯一标记") || !strings.Contains(out, "正文·配置") {
		t.Errorf("section 精读应命中配置章:\n%.500s", out)
	}
	if strings.Contains(out, "排障章节唯一标记") {
		t.Errorf("section 精读不应跨章:\n%.500s", out)
	}

	// 3) section 不存在 → 列可用章节
	out, err = r.Execute("memory_get", rawJSON(map[string]any{"type": "product", "title": "手册", "section": "不存在的章"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "未找到章节") || !strings.Contains(out, "部署") {
		t.Errorf("不存在的 section 应回目录:\n%.500s", out)
	}

	// 4) body_offset 续读：从深处窗口读到配置区标记
	out, err = r.Execute("memory_get", rawJSON(map[string]any{"type": "product", "title": "手册", "body_offset": 3800}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[3800-") || !strings.Contains(out, "配置区唯一标记") {
		t.Errorf("offset 续读应给区间窗口与后续内容:\n%.500s", out)
	}

	// 5) offset 超界
	out, err = r.Execute("memory_get", rawJSON(map[string]any{"type": "product", "title": "手册", "body_offset": 999999}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "超出正文长度") {
		t.Errorf("offset 超界应有明确提示:\n%s", out)
	}
}
