package history

import (
	"testing"
)

// TestToolCallMessageAssociation 持久化回放回归（审计要求）：
// tool_calls 按 message_id 归属 assistant 消息——多轮会话各自的分析可正确还原，
// 不依赖 messages/tool_calls 两张表各自独立的 seq 空间。
func TestToolCallMessageAssociation(t *testing.T) {
	s := openTestStore(t)
	if err := s.EnsureSession("t1", "s1", "多轮回放", ""); err != nil {
		t.Fatal(err)
	}

	// 轮 1：user + assistant(A1) + 2 个工具调用（search + submit）
	if _, err := s.AppendMessage("s1", "user", "我们网站被CC攻击", ""); err != nil {
		t.Fatal(err)
	}
	a1, err := s.AppendMessage("s1", "assistant", "结论：推荐雷池。", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AppendToolCall("s1", a1, "memory_search", `{"query":"CC"}`, `{"count":1}`); err != nil {
		t.Fatal(err)
	}
	submit1 := `{"demand_analysis":"CC攻击防护","feasibility":"direct"}`
	if _, err := s.AppendToolCall("s1", a1, "analysis_submit", submit1, `{"received":true}`); err != nil {
		t.Fatal(err)
	}

	// 轮 2：user + assistant(A2) + 无工具（纯聊天）
	if _, err := s.AppendMessage("s1", "user", "谢谢", ""); err != nil {
		t.Fatal(err)
	}
	a2, err := s.AppendMessage("s1", "assistant", "不客气！", "")
	if err != nil {
		t.Fatal(err)
	}

	det, err := s.GetSession("t1", "s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(det.Messages) != 4 {
		t.Fatalf("应 4 条消息，got %d", len(det.Messages))
	}
	if len(det.ToolCalls) != 2 {
		t.Fatalf("应 2 条工具调用，got %d", len(det.ToolCalls))
	}
	// 全部工具调用归属 A1，不串到 A2（这是回放 ResultCard 的边界）。
	for _, tc := range det.ToolCalls {
		if tc.MessageID != a1 {
			t.Errorf("tool %s 的 message_id=%d，应归属 assistant#%d", tc.ToolName, tc.MessageID, a1)
		}
		if tc.MessageID == a2 {
			t.Errorf("纯聊天轮 assistant#%d 不应挂任何工具调用", a2)
		}
	}
	// submit 的 params 保留完整（前端 parseSubmitParams 还原 ResultCard）。
	var submitParams string
	for _, tc := range det.ToolCalls {
		if tc.ToolName == "analysis_submit" {
			submitParams = tc.Params
		}
	}
	if submitParams == "" || !containsStr(submitParams, "demand_analysis") {
		t.Errorf("analysis_submit params 不完整: %q", submitParams)
	}
}

// TestToolCallNullMessageIDReadable 老库迁移回归：ALTER TABLE 加的 message_id 列
// 对旧行为 NULL——GetSession 必须能读（COALESCE 成 0），不能 Scan 报错。
func TestToolCallNullMessageIDReadable(t *testing.T) {
	s := openTestStore(t)
	if err := s.EnsureSession("t1", "s-old", "老会话", ""); err != nil {
		t.Fatal(err)
	}
	// 直接 SQL 造一行「迁移前的老数据」：message_id 为 NULL
	_, err := s.db.Exec(`INSERT INTO tool_calls(session_id, message_id, tool_name, params_json, result_json, seq, created_at)
		VALUES('s-old', NULL, 'memory_search', '{}', '{}', 1, '2026-08-13 00:00:00')`)
	if err != nil {
		t.Fatal(err)
	}
	det, err := s.GetSession("t1", "s-old")
	if err != nil {
		t.Fatalf("老数据（NULL message_id）应可读: %v", err)
	}
	if len(det.ToolCalls) != 1 || det.ToolCalls[0].MessageID != 0 {
		t.Fatalf("NULL message_id 应 COALESCE 成 0，got %+v", det.ToolCalls)
	}
}

// TestToolCallMigrationIdempotent 迁移幂等：同一库重复 Open（列已存在）不报错。
func TestToolCallMigrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	p := dir + "/t.db"
	s, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	// 重新打开同一文件——ALTER TABLE duplicate column 应被忽略
	s2, err := Open(p)
	if err != nil {
		t.Fatalf("重复 Open 应幂等: %v", err)
	}
	s2.Close()
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
