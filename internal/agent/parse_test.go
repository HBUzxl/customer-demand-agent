package agent

import (
	"testing"
)

func TestParseAnalysis(t *testing.T) {
	cases := []struct {
		name  string
		input string
		ok    bool
		da    string
	}{
		{"plain json", `{"demand_analysis":"x","matched_products":[],"feasibility":"direct","missing_info":[]}`, true, "x"},
		{"code fenced", "```json\n{\"demand_analysis\":\"fenced\",\"feasibility\":\"custom\",\"matched_products\":[],\"missing_info\":[]}\n```", true, "fenced"},
		{"json with prose", `好的，分析如下：{"demand_analysis":"embedded","feasibility":"reject","matched_products":[],"missing_info":[]} 希望有帮助`, true, "embedded"},
		{"not json", `这只是一段普通文字`, false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := parseAnalysis(c.input)
			if c.ok && err != nil {
				t.Fatalf("expected ok, got err: %v", err)
			}
			if !c.ok && err == nil {
				t.Fatal("expected error for non-json")
			}
			if c.ok && r.DemandAnalysis != c.da {
				t.Fatalf("demand_analysis = %q, want %q", r.DemandAnalysis, c.da)
			}
		})
	}
}

func TestStripCodeFence(t *testing.T) {
	out := stripCodeFence("```json\n{}\n```")
	if out != "{}" {
		t.Fatalf("stripCodeFence = %q", out)
	}
	out = stripCodeFence("plain")
	if out != "plain" {
		t.Fatalf("stripCodeFence plain = %q", out)
	}
}

func TestExtractJSON(t *testing.T) {
	if got := extractJSON(`前缀 {"a":{"b":1}} 后缀`); got != `{"a":{"b":1}}` {
		t.Fatalf("extractJSON nested = %q", got)
	}
	if got := extractJSON(`no braces`); got != "" {
		t.Fatalf("extractJSON none = %q", got)
	}
	// 带字符串里的花括号
	if got := extractJSON(`{"x":"a}b"}`); got != `{"x":"a}b"}` {
		t.Fatalf("extractJSON with brace in string = %q", got)
	}
}
