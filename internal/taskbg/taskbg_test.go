package taskbg

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"customer-demand-agent/internal/memory/longterm"
)

func TestRunnerLifecycle(t *testing.T) {
	var gotID string
	r := NewRunner(func(ctx context.Context, task *Task) error {
		gotID = task.ID
		return nil
	})
	tk := r.Submit("t-1", TaskConsolidate, "threat/挂马")
	if r.Status(tk) != "running" {
		t.Fatalf("应 running，got %s", r.Status(tk))
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && r.Status(tk) == "running" {
		time.Sleep(10 * time.Millisecond)
	}
	if r.Status(tk) != "done" {
		t.Fatalf("应 done，got %s", r.Status(tk))
	}
	if gotID != "t-1" {
		t.Fatal("执行函数应收到任务")
	}
	// 列表倒序
	list := r.List(10)
	if len(list) != 1 || list[0].ID != "t-1" {
		t.Fatalf("列表应含任务: %+v", list)
	}
}

func TestRunnerFailure(t *testing.T) {
	r := NewRunner(func(ctx context.Context, task *Task) error {
		return errors.New("LLM 超时")
	})
	tk := r.Submit("t-2", TaskLint, "全库")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && r.Status(tk) == "running" {
		time.Sleep(10 * time.Millisecond)
	}
	if r.Status(tk) != "failed" || r.Result(tk) != "LLM 超时" {
		t.Fatalf("失败态: %s / %s", r.Status(tk), r.Result(tk))
	}
}

func TestConsolidatePromptAndParse(t *testing.T) {
	p := BuildConsolidatePrompt(ConsolidateInput{Type: "threat", Title: "挂马", Observes: []string{"观察1", "观察2"}})
	if p == "" || !contains(p, "挂马") || !contains(p, "观察1") {
		t.Fatal("prompt 应含条目与观察")
	}
	sum, tags, content, err := ParseConsolidateOutput(`{"summary":"概括","tags":["a","b"],"content":"正文"}`)
	if err != nil || sum != "概括" || len(tags) != 2 || content != "正文" {
		t.Fatalf("解析: %v %s %v %s", err, sum, tags, content)
	}
	// 包裹 markdown 的输出
	_, _, _, err = ParseConsolidateOutput("```json\n{\"summary\":\"s\",\"tags\":[],\"content\":\"c\"}\n```")
	if err != nil {
		t.Fatalf("markdown 包裹应可解析: %v", err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestRunLint(t *testing.T) {
	in := LintInput{Entries: []LintEntry{
		{Type: "threat", Title: "正常", Tags: []string{"a"}, Summary: "s", Content: "c"},
		{Type: "threat", Title: "孤儿", Summary: "s", Content: "c"},                       // 无 tags → orphan
		{Type: "customer", Title: "残缺", Tags: []string{"x"}, Summary: "", Content: "c"}, // summary 空
		{Type: "threat", Title: "甲", Aliases: []string{"同别名"}, Summary: "s", Content: "c"},
		{Type: "threat", Title: "乙", Aliases: []string{"同别名"}, Summary: "s", Content: "c"}, // 别名冲突
	}}
	finds := RunLint(in)
	kinds := map[string]int{}
	for _, f := range finds {
		kinds[f.Kind]++
	}
	if kinds["orphan"] != 1 || kinds["incomplete"] != 1 || kinds["duplicate-alias"] != 1 {
		t.Fatalf("检测数不对: %+v", finds)
	}
}

// TestRunLintOverlap P5 第四检查：同类型正文高度重叠（矛盾候选）。
func TestRunLintOverlap(t *testing.T) {
	dup1 := "客户电商网站大促期间遭遇大规模CC攻击，需要Web应用防火墙进行流量清洗和速率限制防护部署"
	dup2 := "客户电商网站大促期间遭遇大规模CC攻击，需要Web应用防火墙进行流量清洗和速率限制防护"
	other := "政务官网被挂马，需要网页防篡改与文件完整性监控，涉及等保三级合规要求"
	in := LintInput{Entries: []LintEntry{
		{Type: "threat", Title: "重叠A", Tags: []string{"x"}, Summary: "s", Content: dup1},
		{Type: "threat", Title: "重叠B", Tags: []string{"x"}, Summary: "s", Content: dup2},
		{Type: "threat", Title: "无关", Tags: []string{"x"}, Summary: "s", Content: other},
	}}
	finds := RunLint(in)
	var overlap int
	for _, f := range finds {
		if f.Kind == "overlap" {
			overlap++
		}
	}
	if overlap == 0 {
		t.Fatalf("重叠检测应命中 重叠A/重叠B: %+v", finds)
	}
}

func TestRunnerPanicMarkedFailed(t *testing.T) {
	r := NewRunner(func(ctx context.Context, task *Task) error {
		panic("boom")
	})
	tk := r.Submit("t-3", TaskTitle, "x/y")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && r.Status(tk) == "running" {
		time.Sleep(10 * time.Millisecond)
	}
	if r.Status(tk) != "failed" || r.Result(tk) != "panic: boom" {
		t.Fatalf("panic 应标 failed: %s / %s", r.Status(tk), r.Result(tk))
	}
}

func TestRunnerSerialExecution(t *testing.T) {
	var mu sync.Mutex
	var order []string
	var active int
	maxActive := 0
	r := NewRunner(func(ctx context.Context, task *Task) error {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()
		time.Sleep(30 * time.Millisecond)
		mu.Lock()
		order = append(order, task.ID)
		active--
		mu.Unlock()
		return nil
	})
	for _, id := range []string{"a", "b", "c"} {
		r.Submit(id, TaskLint, id)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		done := len(order) == 3
		mu.Unlock()
		if done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(order) != 3 {
		t.Fatalf("三个任务应全完成: %v", order)
	}
	if maxActive != 1 {
		t.Fatalf("应串行（同时至多 1 个在跑），峰值并发 %d", maxActive)
	}
	if order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Fatalf("应按提交顺序执行: %v", order)
	}
}

// TestConsolidateEndToEnd 固化管线端到端：条目带观察注记 → LLM 抽结构 →
// UpsertEntry(status=pending) → 审核批准 → verified。证 ADR-015 后台域闭环。
func TestConsolidateEndToEnd(t *testing.T) {
	dir := t.TempDir()
	wiki := longterm.NewWikiStore(dir)
	// 1) 条目 + 观察注记（前台 memory_observe 产生的形态）
	content := "---\ntype: industry\ntitle: 金融AI合规\ntags: [金融, AI]\n---\n\n金融客户关注大模型数据出境。\n\n### 观察\n\n- [high] 银行客户问过大模型训练数据能否出境\n- [medium] 券商关心投研报告用 LLM 生成的合规性\n"
	if err := wiki.UpsertEntry(&longterm.Entry{Type: "industry", Title: "金融AI合规", Content: content}); err != nil {
		t.Fatal(err)
	}
	// 2) 固化 LLM 输出（mock——返回结构化要点）
	mockOut := `{"summary":"金融行业客户普遍关注大模型数据出境与生成内容合规","tags":["金融","AI合规"],"content":"## 固化要点\n\n- 银行关注训练数据出境边界\n- 券商关注投研 LLM 生成内容合规"}`
	// 3) 直接驱动解析+写回（绕过 Runner——管线函数级验证）
	summary, tags, _, err := ParseConsolidateOutput(mockOut)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) == 0 {
		t.Fatal("标签应解析出")
	}
	e, _ := wiki.GetEntry("industry", "金融AI合规")
	e.Content = content + "\n\n### 固化要点（待审核）\n\n- " + summary + "\n"
	e.Status = "pending"
	if err := wiki.UpsertEntry(e); err != nil {
		t.Fatal(err)
	}
	// 4) 人审：pending → verified
	e2, _ := wiki.GetEntry("industry", "金融AI合规")
	e2.Status = "verified"
	if err := wiki.UpsertEntry(e2); err != nil {
		t.Fatal(err)
	}
	e3, _ := wiki.GetEntry("industry", "金融AI合规")
	if e3.Status != "verified" {
		t.Fatalf("人审后应 verified，got %s", e3.Status)
	}
	if !strings.Contains(e3.Content, "固化要点") {
		t.Fatal("固化要点应并入内容")
	}
}

// TestRunLintMissingEntries P5 第五检查：检索零命中查询 → 该建未建建议。
func TestRunLintMissingEntries(t *testing.T) {
	in := LintInput{
		Entries:           []LintEntry{{Type: "threat", Title: "正常", Tags: []string{"a"}, Summary: "s", Content: "c"}},
		RecentMissQueries: []string{"零信任架构", "零信任架构", "  ", "ab"},
	}
	finds := RunLint(in)
	var missing []LintFinding
	for _, f := range finds {
		if f.Kind == "missing-entry" {
			missing = append(missing, f)
		}
	}
	if len(missing) != 1 || missing[0].Title != "零信任架构" {
		t.Fatalf("应去重后报 1 条该建未建（短查询跳过）: %+v", missing)
	}
}
