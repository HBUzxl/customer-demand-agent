package longterm

import (
	"errors"
	"strings"
	"testing"

	"customer-demand-agent/internal/domain"
)

// buildComposite 构造 system（只读基线）+ tenant（覆盖层）复合记忆（§6.5 路由）：
// system 含产品雷池、威胁 SQL注入、行业 银行、客户 系统示例客户（应被 SystemStore 剥离）；
// tenant 含客户 某集团、使用者 张三、威胁覆盖 SQL注入（tenant 优先）、行业 银行（tenant 优先）。
func buildComposite(t *testing.T) *CompositeStore {
	t.Helper()
	sysDir := t.TempDir()
	writeTestPage(t, sysDir, "产品记忆", "雷池", "---\ntype: product\ntitle: 雷池\ntags: [\"WAF\"]\n---\nx")
	writeTestPage(t, sysDir, "行业记忆/威胁类型", "SQL注入", "---\ntype: threat\ntitle: SQL注入\n---\nx")
	writeTestPage(t, sysDir, "行业记忆/行业场景", "银行", "---\ntype: industry\ntitle: 银行\n---\nx")
	writeTestPage(t, sysDir, "用户记忆/客户", "系统示例客户", "---\ntype: customer\ntitle: 系统示例客户\n---\nx")
	sys := NewWikiStore(sysDir)

	tenDir := t.TempDir()
	writeTestPage(t, tenDir, "用户记忆/客户", "某集团", "---\ntype: customer\ntitle: 某集团\n---\n客户画像")
	writeTestPage(t, tenDir, "用户记忆/使用者", "张三", "---\ntype: user\ntitle: 张三\n---\n销售")
	writeTestPage(t, tenDir, "行业记忆/威胁类型", "SQL注入", "---\ntype: threat\ntitle: SQL注入\n---\n租户补充")
	writeTestPage(t, tenDir, "行业记忆/行业场景", "银行", "---\ntype: industry\ntitle: 银行\n---\n租户行业记忆")
	ten := NewWikiStore(tenDir)

	c := NewCompositeStore(NewSystemStore(sys), NewTenantStore(ten))
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	return c
}

// TestCompositeRoutingProductSystem product → 系统只读（绝不回退租户）。
func TestCompositeRoutingProductSystem(t *testing.T) {
	c := buildComposite(t)
	if len(c.AllProducts()) != 1 || c.AllProducts()[0].Name != "雷池" {
		t.Fatalf("产品目录应只含系统雷池: %+v", c.AllProducts())
	}
	if _, err := c.GetProduct("雷池"); err != nil {
		t.Fatalf("GetProduct 雷池: %v", err)
	}
	if _, err := c.GetEntry("product", "雷池"); err != nil {
		t.Fatalf("GetEntry product/雷池: %v", err)
	}
	// 租户层试图覆盖产品 → 拒绝（写操作闸门）
	err := c.UpsertEntry(&Entry{Type: domain.MemoryProduct, Title: "雷池", Content: "租户篡改"})
	if !errors.Is(err, ErrSystemReadOnly) {
		t.Fatalf("租户写产品应 ErrSystemReadOnly，got %v", err)
	}
}

// TestCompositeRoutingCustomerTenant customer/user → 租户只读（绝不回退系统示例）。
func TestCompositeRoutingCustomerTenant(t *testing.T) {
	c := buildComposite(t)
	cp, err := c.GetCustomerProfile("某集团")
	if err != nil || cp == nil {
		t.Fatalf("客户画像应来自租户: %v", err)
	}
	if _, err := c.GetEntry("customer", "某集团"); err != nil {
		t.Fatalf("GetEntry customer/某集团: %v", err)
	}
	// 系统示例客户不可见（不参与租户检索）
	if _, err := c.GetEntry("customer", "系统示例客户"); err == nil {
		t.Fatal("系统示例客户不应在租户检索中可见")
	}
	// 使用者画像（张三）只在租户层
	up, err := c.GetUserProfile("张三")
	if err != nil || up == nil {
		t.Fatalf("使用者画像应来自租户: %v", err)
	}
}

// TestCompositeRoutingMergeTenantPriority threat/compliance/industry → 合并且 tenant 优先。
func TestCompositeRoutingMergeTenantPriority(t *testing.T) {
	c := buildComposite(t)
	th, ok := c.GetThreat("SQL注入")
	if !ok {
		t.Fatal("威胁应存在（合并）")
	}
	if th.Description != "租户补充" {
		t.Fatalf("同名校验应 tenant 优先，got %q", th.Description)
	}
	ind, ok := c.GetIndustry("银行")
	if !ok || ind == nil {
		t.Fatalf("行业应合并返回: %+v", ind)
	}
	// 合并去重：ListEntry threat 只 1 条
	entries := c.ListEntry("threat", 0, 10)
	if len(entries) != 1 {
		t.Fatalf("威胁合并后应 1 条（去重），got %d", len(entries))
	}
}

// TestSystemStripsCustomerUser 系统层 Load 后剥离客户/使用者（示例不进租户检索）。
func TestSystemStripsCustomerUser(t *testing.T) {
	sysDir := t.TempDir()
	writeTestPage(t, sysDir, "产品记忆", "雷池", "---\ntype: product\ntitle: 雷池\n---\nx")
	writeTestPage(t, sysDir, "用户记忆/客户", "系统示例客户", "---\ntype: customer\ntitle: 系统示例客户\n---\nx")
	writeTestPage(t, sysDir, "用户记忆/使用者", "张三", "---\ntype: user\ntitle: 张三\n---\nx")
	sys := NewSystemStore(NewWikiStore(sysDir))
	if err := sys.Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := sys.GetEntry("customer", "系统示例客户"); err == nil {
		t.Fatal("系统层不应有客户记忆")
	}
	if _, err := sys.GetEntry("user", "张三"); err == nil {
		t.Fatal("系统层不应有使用者记忆")
	}
	// 产品保留
	if _, err := sys.GetEntry("product", "雷池"); err != nil {
		t.Fatalf("系统层产品应保留: %v", err)
	}
}

// TestSystemGenericReadFiltersPrivateTypes 验证装配方即使只加载了底层 WikiStore，
// SystemStore 的通用 Search/List/Get 也不会把客户或使用者数据带进租户复合层。
func TestSystemGenericReadFiltersPrivateTypes(t *testing.T) {
	sysDir := t.TempDir()
	writeTestPage(t, sysDir, "产品记忆", "雷池", "---\ntype: product\ntitle: 雷池\n---\nx")
	writeTestPage(t, sysDir, "用户记忆/客户", "系统示例客户", "---\ntype: customer\ntitle: 系统示例客户\n---\nx")
	writeTestPage(t, sysDir, "用户记忆/使用者", "张三", "---\ntype: user\ntitle: 张三\n---\nx")
	raw := NewWikiStore(sysDir)
	if err := raw.Load(); err != nil {
		t.Fatal(err)
	}
	sys := NewSystemStore(raw) // 故意不调用 sys.Load，验证读路径纵深防御
	for _, e := range sys.SearchEntry("", "", 100) {
		if e.Type == domain.MemoryCustomer || e.Type == domain.MemoryUser {
			t.Fatalf("通用搜索泄露租户私有类型: %+v", e)
		}
	}
	for _, e := range sys.ListEntry("", 0, 100) {
		if e.Type == domain.MemoryCustomer || e.Type == domain.MemoryUser {
			t.Fatalf("通用列表泄露租户私有类型: %+v", e)
		}
	}
	if _, err := sys.GetEntry("customer", "系统示例客户"); err == nil {
		t.Fatal("system 通用 Get 不应读取 customer")
	}
}

// TestCompositeListTenantContentWins P0-05：同 (type,title) 合并时 Content/Summary
// 必须取租户覆盖层（修复前 system 后写覆盖 tenant，与「tenant 优先」注释相反）。
func TestCompositeListTenantContentWins(t *testing.T) {
	c := buildComposite(t)
	// SearchEntry 合并：SQL注入 同键，租户 Content 必须胜出
	entries := c.SearchEntry("SQL注入", "threat", 10)
	found := false
	for _, e := range entries {
		if e.Title == "SQL注入" {
			found = true
			if !strings.Contains(e.Content, "租户补充") {
				t.Fatalf("SearchEntry 同键 Content 应取租户覆盖层，got %q", e.Content)
			}
		}
	}
	if !found {
		t.Fatal("SearchEntry 应返回 SQL注入 合并条目")
	}
	// ListEntry 合并分页同样 tenant 优先
	if listed := c.ListEntry("threat", 0, 10); len(listed) == 1 {
		if !strings.Contains(listed[0].Content, "租户补充") {
			t.Fatalf("ListEntry 同键 Content 应取租户覆盖层，got %q", listed[0].Content)
		}
	}
}

// TestCompositeWriteOnlyTenant 写操作只落租户层：PendingReviews/批准/拒绝不触碰系统。
func TestCompositeWriteOnlyTenant(t *testing.T) {
	c := buildComposite(t)
	// 租户写入威胁（pending）
	if err := c.UpsertEntry(&Entry{Type: domain.MemoryThreat, Title: "新型威胁", Content: "x", Status: StatusPendingReview}); err != nil {
		t.Fatal(err)
	}
	pending := c.PendingReviews()
	if len(pending) != 1 || pending[0].Title != "新型威胁" {
		t.Fatalf("审核队列应只含租户条目，got %+v", pending)
	}
	// 审批生效（复合记忆读回 verified）
	if err := c.ApproveEntry("threat", "新型威胁"); err != nil {
		t.Fatal(err)
	}
	e, _ := c.GetEntry("threat", "新型威胁")
	if e.Status != StatusVerified {
		t.Fatalf("审批后应 verified，got %s", e.Status)
	}
	// 删除租户条目后消失（系统 SQL注入 仍合并可见）
	if err := c.DeleteEntry("threat", "新型威胁", false); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetEntry("threat", "新型威胁"); err == nil {
		t.Fatal("删除后租户条目应消失")
	}
	if _, ok := c.GetThreat("SQL注入"); !ok {
		t.Fatal("删除租户条目不应影响系统基线")
	}
}

// TestCompositeGetEntrySystemBaseline system-only 类型从系统层可见。
func TestCompositeGetEntrySystemBaseline(t *testing.T) {
	c := buildComposite(t)
	e, err := c.GetEntry("industry", "银行")
	if err != nil {
		t.Fatalf("行业应可读（系统基线）: %v", err)
	}
	if !strings.Contains(e.Content, "租户行业记忆") {
		t.Fatalf("行业 tenant 优先应命中租户内容: %q", e.Content)
	}
}
