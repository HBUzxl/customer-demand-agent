package history

import (
	"testing"

	"customer-demand-agent/internal/domain"
)

// TestToolCallMessageAssociation 持久化回放回归（审计要求）：
// tool_calls 按 message_id 归属 assistant 消息——多轮会话各自的分析可正确还原，
// 不依赖 messages/tool_calls 两张表各自独立的 seq 空间。
func TestToolCallMessageAssociation(t *testing.T) {
	s := openTestStore(t)
	if err := s.EnsureSession("s1", "多轮回放", ""); err != nil {
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

	det, err := s.GetSession("s1")
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
	if err := s.EnsureSession("s-old", "老会话", ""); err != nil {
		t.Fatal(err)
	}
	// 直接 SQL 造一行「迁移前的老数据」：message_id 为 NULL
	_, err := s.db.Exec(`INSERT INTO tool_calls(session_id, message_id, tool_name, params_json, result_json, seq, created_at)
		VALUES('s-old', NULL, 'memory_search', '{}', '{}', 1, '2026-08-13 00:00:00')`)
	if err != nil {
		t.Fatal(err)
	}
	det, err := s.GetSession("s-old")
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

// TestSearchMessages P1 history_search 后端：会话范围 + LIKE 命中（de-tenancy 后无租户维度）。
func TestSearchMessages(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("sa", "A", "")
	_ = s.EnsureSession("sb", "B", "")
	_ = s.EnsureSession("sc", "C", "")
	_, _ = s.AppendMessage("sa", "user", "预算50万怎么花", "")
	_, _ = s.AppendMessage("sb", "user", "预算紧张", "")
	_, _ = s.AppendMessage("sc", "user", "预算保密", "")

	// 空 scope 跨全部会话
	hits, err := s.SearchMessages("", "预算", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 3 {
		t.Fatalf("应命中全部 3 条，got %d", len(hits))
	}
	// 限定会话
	hits, err = s.SearchMessages("sa", "预算", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].SessionID != "sa" {
		t.Fatalf("限定 sa 应只命中 1 条 sa，got %+v", hits)
	}
	// 空 scope = 跨全部会话（de-tenancy 后无租户维度）
	hits, err = s.SearchMessages("", "预算", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) < 2 {
		t.Fatalf("空 scope 应命中全部会话的匹配，got %+v", hits)
	}
	// 无命中
	hits, _ = s.SearchMessages("", "不存在的话题", 10)
	if len(hits) != 0 {
		t.Fatalf("无关键词应 0 命中，got %d", len(hits))
	}
}

// TestTruncateAfterThreeTables F1 截断三表联删（store 级直查断言）。
func TestTruncateAfterThreeTables(t *testing.T) {
	s := openTestStore(t)
	_ = s.EnsureSession("st", "标题", "")
	_, _ = s.AppendMessage("st", "user", "u1", "")
	aid2, _ := s.AppendMessage("st", "assistant", "a1", "")
	_, _ = s.AppendToolCall("st", aid2, "memory_search", "{}", "{}")
	_ = s.AppendCheckpoint("st", &domain.Checkpoint{ID: "cp1", Type: domain.CheckpointInitial})
	// 截断 user（seq=1）及其后
	n, err := s.TruncateAfter("st", 1)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("应删 2 条消息，got %d", n)
	}
	// 三表全空
	for tbl, want := range map[string]int{"messages": 0, "tool_calls": 0, "checkpoints": 0} {
		rows, err := s.db.Query("SELECT COUNT(*) FROM " + tbl + " WHERE session_id='st'")
		if err != nil {
			t.Fatal(err)
		}
		var got int
		rows.Next()
		_ = rows.Scan(&got)
		rows.Close()
		if got != want {
			t.Errorf("%s 应 %d，got %d", tbl, want, got)
		}
	}
}
