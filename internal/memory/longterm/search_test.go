package longterm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"customer-demand-agent/internal/domain"
)

// TestTokenizeCJKBigram 中文 bigram 切分：查询"被扫描"能命中索引词"扫描防护"。
func TestTokenizeCJKBigram(t *testing.T) {
	toks := tokenize("网站被扫描")
	for _, want := range []string{"网站", "被扫", "扫描"} {
		found := false
		for _, tk := range toks {
			if tk == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tokenize(网站被扫描) 缺少 %s，got %v", want, toks)
		}
	}
	// ASCII 整词 + 混合
	mixed := tokenize("WAF 防护, CC攻击")
	if len(mixed) < 3 {
		t.Errorf("混合查询 token 过少: %v", mixed)
	}
}

// TestSearchBigramMatchesKeyword 端到端：条目关键词"扫描防护"，
// 自然语言查询"网站经常被扫描攻击"应能命中（修复前整段匹配永远不中）。
func TestSearchBigramMatchesKeyword(t *testing.T) {
	w := NewWikiStore(t.TempDir())
	if err := w.Load(); err != nil {
		t.Fatal(err)
	}
	// 通过 frontmatter 挂能力关键词"扫描防护"（keywords 字段）
	content := "---\ntype: product\nstatus: verified\nsummary: Web 应用防火墙\nkeywords: [扫描防护, Web防护]\n---\n雷池 WAF"
	if err := w.UpsertEntry(&Entry{
		Type: domain.MemoryProduct, Title: "雷池", Status: StatusVerified,
		Summary: "Web 应用防火墙", Content: content,
	}); err != nil {
		t.Fatal(err)
	}

	hits := w.SearchEntry("网站经常被扫描攻击，用户数据被爬", "product", 5)
	if len(hits) == 0 {
		t.Fatal("bigram 后自然语言查询应命中雷池（修复前整段匹配永不中）")
	}
	if hits[0].Title != "雷池" {
		t.Fatalf("最佳命中应为雷池，got %s", hits[0].Title)
	}
}

// TestSearchWeightTitleBeatsSummary 字段权重：标题命中 > 摘要命中。
func TestSearchWeightTitleBeatsSummary(t *testing.T) {
	w := NewWikiStore(t.TempDir())
	if err := w.Load(); err != nil {
		t.Fatal(err)
	}
	// A：标题含"等保"；B：只有摘要含"等保"
	for _, mk := range []struct {
		title, summary string
	}{
		{"等保三级", "适用于重要业务系统的等级保护要求"},
		{"行业规范X", "内容提到等保三级的相关合规背景说明"},
	} {
		if err := w.UpsertEntry(&Entry{
			Type: domain.MemoryCompliance, Title: mk.title, Summary: mk.summary, Status: StatusVerified,
		}); err != nil {
			t.Fatal(err)
		}
	}
	hits := w.SearchEntry("等保", "compliance", 5)
	if len(hits) < 2 {
		t.Fatalf("应至少命中 2 条，got %d", len(hits))
	}
	if hits[0].Title != "等保三级" {
		t.Fatalf("标题命中应排最前，got %s", hits[0].Title)
	}
}

// TestSupersedeArchive P6 时效性：同 title 实质变更覆盖 → 旧版本归档
// （.superseded-*.md 带 archived/invalid_at/superseded_by），重载后活跃
// 索引只有新版本。
func TestSupersedeArchive(t *testing.T) {
	dir := t.TempDir()
	w := NewWikiStore(dir)
	if err := w.Load(); err != nil {
		t.Fatal(err)
	}
	// 初版
	if err := w.UpsertEntry(&Entry{
		Type: domain.MemoryThreat, Title: "客户偏好", Status: StatusVerified,
		Content: "---\ntype: threat\nstatus: verified\nsummary: 初版偏好\n---\n喜欢私有云",
	}); err != nil {
		t.Fatal(err)
	}
	// 实质变更覆盖
	if err := w.UpsertEntry(&Entry{
		Type: domain.MemoryThreat, Title: "客户偏好", Status: StatusVerified,
		Content: "---\ntype: threat\nstatus: verified\nsummary: 新偏好\n---\n转向公有云",
	}); err != nil {
		t.Fatal(err)
	}
	// 归档文件存在
	matches, _ := filepath.Glob(filepath.Join(dir, "行业记忆", "威胁类型", "客户偏好.superseded-*.md"))
	if len(matches) != 1 {
		t.Fatalf("应有一个归档版本文件，got %v", matches)
	}
	archived, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	body := string(archived)
	for _, want := range []string{"status: archived", "superseded_by: 客户偏好", "invalid_at: "} {
		if !strings.Contains(body, want) {
			t.Errorf("归档文件缺 %q:\n%s", want, body[:200])
		}
	}
	// 重载：活跃索引只有新版本，归档不进
	w2 := NewWikiStore(dir)
	if err := w2.Load(); err != nil {
		t.Fatal(err)
	}
	hits := w2.SearchEntry("偏好", "threat", 10)
	if len(hits) != 1 || !strings.Contains(hits[0].Content, "公有云") {
		t.Fatalf("重载后应只命中新版本，got %+v", hits)
	}
}
