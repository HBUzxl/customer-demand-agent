package taskbg

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunnerLifecycle(t *testing.T) {
	var got *Task
	r := NewRunner(func(ctx context.Context, task *Task) error {
		got = task
		task.Result = "工作完成"
		return nil
	})
	tk := r.Submit("t-1", TaskConsolidate, "threat/挂马")
	if tk.Status != "running" {
		t.Fatalf("应 running，got %s", tk.Status)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && tk.Status == "running" {
		time.Sleep(10 * time.Millisecond)
	}
	if tk.Status != "done" {
		t.Fatalf("应 done，got %s", tk.Status)
	}
	if got == nil || got.ID != "t-1" {
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
	for time.Now().Before(deadline) && tk.Status == "running" {
		time.Sleep(10 * time.Millisecond)
	}
	if tk.Status != "failed" || tk.Result != "LLM 超时" {
		t.Fatalf("失败态: %+v", tk)
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
