package longterm

import (
	"os"
	"path/filepath"
	"testing"

	"customer-demand-agent/internal/domain"
)

// writeTestPage 写一个临时 Wiki 页面用于测试。
func writeTestPage(t *testing.T, dir, sub, name, content string) {
	t.Helper()
	p := filepath.Join(dir, sub, name+".md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAndSearch(t *testing.T) {
	dir := t.TempDir()
	writeTestPage(t, dir, "产品记忆", "雷池", `---
type: product
title: 雷池
aliases: ["WAF", "SafeLine"]
tags: ["WAF", "CC防护"]
capabilities:
  - name: CC 攻击防护
    confidence: 1.0
    keywords: ["CC", "DDoS"]
description: 下一代 WAF
---
雷池是 WAF。`)

	store := NewWikiStore(dir)
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// 类型化检索
	prods := store.AllProducts()
	if len(prods) != 1 || prods[0].Name != "雷池" {
		t.Fatalf("expected 1 product 雷池, got %+v", prods)
	}
	if len(prods[0].Capabilities) != 1 || prods[0].Capabilities[0].Name != "CC 攻击防护" {
		t.Fatalf("capability not parsed: %+v", prods[0].Capabilities)
	}

	// 关键词检索：CC 应命中雷池
	hits := store.SearchProducts("CC")
	if len(hits) != 1 || hits[0].Name != "雷池" {
		t.Fatalf("search CC should hit 雷池, got %+v", hits)
	}

	// 别名检索
	if _, err := store.GetProduct("WAF"); err != nil {
		t.Fatalf("GetProduct by alias WAF failed: %v", err)
	}
	if _, err := store.GetProduct("不存在"); err == nil {
		t.Fatal("GetProduct 不存在 should error")
	}
}

func TestUpsertAndReview(t *testing.T) {
	dir := t.TempDir()
	store := NewWikiStore(dir)
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}

	// AI 写入一条 threat（待审核）
	e := &Entry{
		Type:    domain.MemoryThreat,
		Title:   "新型API滥用",
		Content: "测试威胁",
		Status:  StatusPendingReview,
		Tags:    []string{"API"},
	}
	if err := store.UpsertEntry(e); err != nil {
		t.Fatalf("UpsertEntry: %v", err)
	}

	// 应出现在待审核队列
	pending := store.PendingReviews()
	if len(pending) != 1 || pending[0].Title != "新型API滥用" {
		t.Fatalf("pending should have 1 item, got %+v", pending)
	}

	// 批准后转为 verified
	if err := store.ApproveEntry("threat", "新型API滥用"); err != nil {
		t.Fatalf("ApproveEntry: %v", err)
	}
	pending = store.PendingReviews()
	if len(pending) != 0 {
		t.Fatalf("after approve pending should be empty, got %+v", pending)
	}

	// 文件确实写回了磁盘
	if _, err := os.Stat(e.FilePath); err != nil {
		t.Fatalf("file not written back: %v", err)
	}

	// 归档（软删除）
	if err := store.DeleteEntry("threat", "新型API滥用", true); err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}
	hits := store.SearchEntry("测试威胁", "threat", 10)
	if len(hits) != 0 {
		t.Fatalf("archived entry should not appear in search, got %+v", hits)
	}
}

func TestGenericSearchAllTypes(t *testing.T) {
	dir := t.TempDir()
	writeTestPage(t, dir, "产品记忆", "雷池", "---\ntype: product\ntitle: 雷池\ntags: [\"WAF\"]\n---\nx")
	writeTestPage(t, dir, "行业记忆/威胁类型", "CC攻击", "---\ntype: threat\ntitle: CC攻击\nrelated_products: [\"雷池\"]\n---\nx")
	store := NewWikiStore(dir)
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}

	// 跨类型搜索 "雷池"：应同时命中 product 和 threat（related_products）
	hits := store.SearchEntry("雷池", "", 10)
	titles := map[string]bool{}
	for _, h := range hits {
		titles[string(h.Type)+"/"+h.Title] = true
	}
	if !titles["product/雷池"] {
		t.Errorf("expected product/雷池 in hits, got %v", titles)
	}
	if !titles["threat/CC攻击"] {
		t.Errorf("expected threat/CC攻击 (related_products) in hits, got %v", titles)
	}
}

// TestListChildrenAndCatalogExcludesSubdocs 子文档（product 字段指向主页）
// 不进产品目录（AllProducts），但能被 ListChildren 导航到。
func TestListChildrenAndCatalogExcludesSubdocs(t *testing.T) {
	dir := t.TempDir()
	writeTestPage(t, dir, "产品记忆", "雷池", `---
type: product
title: 雷池
capabilities:
  - name: CC 攻击防护
    confidence: 1.0
description: 下一代 WAF
---
雷池是 WAF。`)
	writeTestPage(t, dir, "产品记忆", "雷池-FAQ", `---
type: product
title: 雷池-FAQ
product: 雷池
summary: 高频问题
---
Q: 误报怎么办？`)

	store := NewWikiStore(dir)
	if err := store.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// 目录只列主页
	prods := store.AllProducts()
	if len(prods) != 1 || prods[0].Name != "雷池" {
		t.Fatalf("目录应只含产品主页，got %+v", prods)
	}
	// 子文档导航
	children := store.ListChildren("雷池")
	if len(children) != 1 || children[0].Title != "雷池-FAQ" || children[0].Summary != "高频问题" {
		t.Fatalf("ListChildren 应返回雷池-FAQ，got %+v", children)
	}
	// 无子文档的产品返回空
	if len(store.ListChildren("不存在的产品")) != 0 {
		t.Fatal("无子文档应返回空")
	}
}
