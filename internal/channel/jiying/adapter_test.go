package jiying

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/api"
)

// ── 测试替身 ────────────────────────────────────────────────

// fakeConn 是 conn 的阻塞式假实现：writeText 自动回 ok:true 响应，readMessage
// 阻塞直到有消息或 close，模拟真实 socket 读语义。
type fakeConn struct {
	mu       sync.Mutex
	cond     *sync.Cond
	incoming [][]byte
	written  [][]byte
	closed   bool
	respFor  map[string]json.RawMessage // method → 自动响应 payload（可选）
}

func newFakeConn() *fakeConn {
	f := &fakeConn{}
	f.cond = sync.NewCond(&f.mu)
	return f
}

func (f *fakeConn) enqueue(data []byte) {
	f.mu.Lock()
	f.incoming = append(f.incoming, data)
	f.mu.Unlock()
	f.cond.Broadcast()
}

func (f *fakeConn) readMessage() ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for len(f.incoming) == 0 && !f.closed {
		f.cond.Wait()
	}
	if len(f.incoming) == 0 {
		return nil, io.EOF
	}
	m := f.incoming[0]
	f.incoming = f.incoming[1:]
	return m, nil
}

func (f *fakeConn) writeText(b []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.written = append(f.written, append([]byte(nil), b...))
	// 自动回 ok 响应；respFor 可按 method 定制 payload。
	var env envelope
	if json.Unmarshal(b, &env) == nil && env.Kind == kindRequest {
		payload := json.RawMessage(`{}`)
		if f.respFor != nil {
			if p, ok := f.respFor[env.Method]; ok {
				payload = p
			}
		}
		ok := true
		resp := envelope{V: protocolVersion, Kind: kindResponse, ReplyTo: env.ID, OK: &ok, Payload: payload}
		if data, err := json.Marshal(resp); err == nil {
			f.incoming = append(f.incoming, data)
		}
	}
	f.cond.Broadcast()
	return nil
}

func (f *fakeConn) setReadDeadline(time.Time) error { return nil }

func (f *fakeConn) close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	f.cond.Broadcast()
	return nil
}

func (f *fakeConn) requests() []envelope {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []envelope
	for _, b := range f.written {
		var env envelope
		if json.Unmarshal(b, &env) == nil {
			out = append(out, env)
		}
	}
	return out
}

// fakeProcessor 记录调用并把预设结果回传。
type fakeProcessor struct {
	out   *api.OutboundMessage
	err   error
	calls []*api.InboundMessage
}

func (f *fakeProcessor) Process(_ context.Context, in *api.InboundMessage) (*api.OutboundMessage, error) {
	f.calls = append(f.calls, in)
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

// ── 测试辅助 ────────────────────────────────────────────────

func eventEnvelope(cursor int64, name string, payload any) []byte {
	env := envelope{V: protocolVersion, Kind: kindEvent, Cursor: cursor, Event: name}
	env.Payload, _ = json.Marshal(payload)
	data, _ := json.Marshal(env)
	return data
}

func messageCreatedEvent(cursor int64, convType, senderType, text string) []byte {
	return eventEnvelope(cursor, eventMessageCreated, messageCreatedPayload{
		Conversation: conversation{ID: "conv-1", Type: convType},
		Sender:       user{Type: senderType},
		Message:      message{ID: "msg-1", Body: messageBody{Type: "text", Content: text}},
	})
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("等待超时")
}

// runServe 起 serve 循环，等 written 达到 n 条后关闭连接并等待退出。
func runServe(t *testing.T, a *Adapter, fc *fakeConn, expectWrites int) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- a.serve(context.Background(), fc) }()
	waitFor(t, 2*time.Second, func() bool {
		fc.mu.Lock()
		defer fc.mu.Unlock()
		return len(fc.written) >= expectWrites
	})
	fc.close()
	if err := <-done; !errors.Is(err, io.EOF) {
		t.Fatalf("serve 退出期望 io.EOF，得到 %v", err)
	}
}

// ── 事件处理 ────────────────────────────────────────────────

func TestServeMessageCreatedRepliesAndAcks(t *testing.T) {
	fc := newFakeConn()
	fc.enqueue(messageCreatedEvent(1, "app", "user", "请分析一下这段需求"))
	proc := &fakeProcessor{out: &api.OutboundMessage{Content: "已收到，正在分析。"}}
	a := New(Config{}, proc)

	runServe(t, a, fc, 2)

	reqs := fc.requests()
	if len(reqs) != 2 {
		t.Fatalf("期望 2 条请求（message.send + events.ack），得到 %d", len(reqs))
	}
	if reqs[0].Method != "message.send" || reqs[1].Method != "events.ack" {
		t.Fatalf("请求顺序不符: %s, %s", reqs[0].Method, reqs[1].Method)
	}
	var send messageSendPayload
	if err := json.Unmarshal(reqs[0].Payload, &send); err != nil {
		t.Fatalf("解析 message.send payload: %v", err)
	}
	if send.Target.ConversationID != "conv-1" || send.Message.Type != "markdown" || send.Message.Content != "已收到，正在分析。" {
		t.Fatalf("message.send 不符: %+v", send)
	}
	var ack ackPayload
	if err := json.Unmarshal(reqs[1].Payload, &ack); err != nil || ack.Cursor != 1 {
		t.Fatalf("events.ack 不符: %v cursor=%d", err, ack.Cursor)
	}
	if len(proc.calls) != 1 {
		t.Fatalf("期望 1 次处理调用，得到 %d", len(proc.calls))
	}
	in := proc.calls[0]
	if in.SessionID != "conv-1" || in.Content != "请分析一下这段需求" || in.Source != "jiying" {
		t.Fatalf("InboundMessage 不符: %+v", in)
	}
}

func TestServeSkipsNonUserSender(t *testing.T) {
	fc := newFakeConn()
	fc.enqueue(messageCreatedEvent(1, "group", "app", "别的应用说的"))
	proc := &fakeProcessor{out: &api.OutboundMessage{Content: "x"}}
	a := New(Config{}, proc)

	runServe(t, a, fc, 1)

	reqs := fc.requests()
	if len(reqs) != 1 || reqs[0].Method != "events.ack" {
		t.Fatalf("应用消息应只 ACK，得到 %d 条: %+v", len(reqs), reqs)
	}
	if len(proc.calls) != 0 {
		t.Fatalf("应用消息不应触发处理，得到 %d 次调用", len(proc.calls))
	}
}

func TestServeTopicClosedAcksOnly(t *testing.T) {
	fc := newFakeConn()
	fc.enqueue(eventEnvelope(2, eventTopicClosed, topicClosedPayload{
		Archived: true, ConversationID: "conv-1",
	}))
	proc := &fakeProcessor{out: &api.OutboundMessage{Content: "x"}}
	a := New(Config{}, proc)

	runServe(t, a, fc, 1)

	reqs := fc.requests()
	if len(reqs) != 1 || reqs[0].Method != "events.ack" {
		t.Fatalf("topic.closed 应只 ACK，得到 %+v", reqs)
	}
	if len(proc.calls) != 0 {
		t.Fatalf("topic.closed 不应触发处理")
	}
}

func TestServeProcessErrorSendsFallbackAndAcks(t *testing.T) {
	fc := newFakeConn()
	fc.enqueue(messageCreatedEvent(3, "app", "user", "正常提问"))
	proc := &fakeProcessor{err: errors.New("LLM 挂了")}
	a := New(Config{}, proc)

	runServe(t, a, fc, 2)

	reqs := fc.requests()
	if len(reqs) != 2 || reqs[0].Method != "message.send" || reqs[1].Method != "events.ack" {
		t.Fatalf("处理失败应回退回复并 ACK，得到 %+v", reqs)
	}
	var send messageSendPayload
	_ = json.Unmarshal(reqs[0].Payload, &send)
	if !strings.Contains(send.Message.Content, "出错") {
		t.Fatalf("回退文案应含「出错」，得到 %q", send.Message.Content)
	}
}

func TestServeDedupByCursor(t *testing.T) {
	fc := newFakeConn()
	fc.enqueue(messageCreatedEvent(5, "group", "user", "重复投递"))
	proc := &fakeProcessor{out: &api.OutboundMessage{Content: "x"}}
	a := New(Config{}, proc)
	a.mu.Lock()
	a.acked = 10 // 已确认到 10，cursor 5 是重复
	a.mu.Unlock()

	runServe(t, a, fc, 1)

	reqs := fc.requests()
	if len(reqs) != 1 || reqs[0].Method != "events.ack" {
		t.Fatalf("重复 cursor 应只 ACK，得到 %+v", reqs)
	}
	if len(proc.calls) != 0 {
		t.Fatalf("重复 cursor 不应触发处理")
	}
}

func TestName(t *testing.T) {
	if got := New(Config{}, nil).Name(); got != "jiying" {
		t.Fatalf("Name() = %q", got)
	}
}

// ── 握手 ────────────────────────────────────────────────────

func TestDialWSHandshake(t *testing.T) {
	type captured struct{ appID, auth, key string }
	captures := make(chan captured, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captures <- captured{
			appID: r.Header.Get("X-MagicChat-App-ID"),
			auth:  r.Header.Get("Authorization"),
			key:   r.Header.Get("Sec-WebSocket-Key"),
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("不支持 Hijack")
			return
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			t.Errorf("Hijack: %v", err)
			return
		}
		defer conn.Close()
		fmt.Fprintf(buf, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",
			computeAccept(r.Header.Get("Sec-WebSocket-Key")))
		_ = buf.Flush()
	}))
	defer srv.Close()

	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://") + "/api/app/ws"
	h := http.Header{}
	h.Set("X-MagicChat-App-ID", "app-1")
	h.Set("Authorization", "Bearer secret")
	c, err := dialWS(wsURL, "", h)
	if err != nil {
		t.Fatalf("dialWS: %v", err)
	}
	defer c.close()

	got := <-captures
	if got.appID != "app-1" || got.auth != "Bearer secret" || got.key == "" {
		t.Fatalf("握手头不符: appID=%q auth=%q key=%q", got.appID, got.auth, got.key)
	}
}

func TestDialWSHandshakeFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, "unauthorized")
	}))
	defer srv.Close()

	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://") + "/"
	_, err := dialWS(wsURL, "", http.Header{})
	var he *handshakeError
	if !errors.As(err, &he) || he.status != http.StatusUnauthorized {
		t.Fatalf("期望 401 handshakeError，得到 %v", err)
	}
}

// ── 受理回执（慢处理反馈） ─────────────────────────────────

// gatedProcessor 阻塞到 gate 关闭才返回，模拟慢处理。
type gatedProcessor struct {
	fakeProcessor
	gate chan struct{}
}

func (g *gatedProcessor) Process(ctx context.Context, in *api.InboundMessage) (*api.OutboundMessage, error) {
	<-g.gate
	return g.fakeProcessor.Process(ctx, in)
}

// TestServeSlowProcessSendsAckFirst 处理超过阈值时：回执 → 结果 → ACK，顺序固定。
func TestServeSlowProcessSendsAckFirst(t *testing.T) {
	fc := newFakeConn()
	fc.enqueue(messageCreatedEvent(7, "app", "user", "帮我分析这段需求"))
	gate := make(chan struct{})
	proc := &gatedProcessor{fakeProcessor{out: &api.OutboundMessage{Content: "分析结果"}}, gate}
	a := New(Config{AckAfter: 20 * time.Millisecond}, proc)

	done := make(chan error, 1)
	go func() { done <- a.serve(context.Background(), fc) }()

	// 阶段一：等受理回执到达（此时处理仍在进行）。
	waitFor(t, 2*time.Second, func() bool {
		reqs := fc.requests()
		return len(reqs) >= 1 && reqs[0].Method == "message.send"
	})
	var ackMsg messageSendPayload
	if err := json.Unmarshal(fc.requests()[0].Payload, &ackMsg); err != nil {
		t.Fatalf("解析受理回执: %v", err)
	}
	if ackMsg.Message.Content != ackText {
		t.Fatalf("受理回执文案不符: %q", ackMsg.Message.Content)
	}

	close(gate) // 放行处理

	// 阶段二：最终结果 + ACK。
	waitFor(t, 2*time.Second, func() bool {
		return len(fc.requests()) >= 3
	})
	reqs := fc.requests()
	if reqs[1].Method != "message.send" || reqs[2].Method != "events.ack" {
		t.Fatalf("请求顺序不符: %s, %s", reqs[1].Method, reqs[2].Method)
	}
	var final messageSendPayload
	_ = json.Unmarshal(reqs[1].Payload, &final)
	if final.Message.Content != "分析结果" {
		t.Fatalf("最终回复不符: %q", final.Message.Content)
	}
	var ack ackPayload
	if err := json.Unmarshal(reqs[2].Payload, &ack); err != nil || ack.Cursor != 7 {
		t.Fatalf("events.ack 不符: %v cursor=%d", err, ack.Cursor)
	}

	fc.close()
	if err := <-done; !errors.Is(err, io.EOF) {
		t.Fatalf("serve 退出期望 io.EOF，得到 %v", err)
	}
}

// TestServeAckDisabled 负数 AckAfter 禁用回执：慢处理也只发最终结果。
func TestServeAckDisabled(t *testing.T) {
	fc := newFakeConn()
	fc.enqueue(messageCreatedEvent(8, "app", "user", "慢但不需要回执"))
	gate := make(chan struct{})
	proc := &gatedProcessor{fakeProcessor{out: &api.OutboundMessage{Content: "结果"}}, gate}
	a := New(Config{AckAfter: -1}, proc)

	done := make(chan error, 1)
	go func() { done <- a.serve(context.Background(), fc) }()

	time.Sleep(50 * time.Millisecond) // 若回执逻辑误启用，这里足够暴露
	close(gate)

	waitFor(t, 2*time.Second, func() bool { return len(fc.requests()) >= 2 })
	reqs := fc.requests()
	if len(reqs) != 2 || reqs[0].Method != "message.send" || reqs[1].Method != "events.ack" {
		t.Fatalf("禁用回执时应只有回复+ACK，得到 %+v", reqs)
	}
	fc.close()
	if err := <-done; !errors.Is(err, io.EOF) {
		t.Fatalf("serve 退出期望 io.EOF，得到 %v", err)
	}
}

// ── ask_user ↔ choice 桥接 ─────────────────────────────────

// askUserEvent 构造 agent 的 ask_user 事件。
func askUserEvent() agent.Event {
	return agent.Event{
		Type:     agent.EventAskUser,
		Question: "**这是哪家客户？**",
		Options: []agent.AskOption{
			{Label: "长亭科技", Value: "chaitin"},
			{Label: "某银行", Value: "bank"},
		},
	}
}

// TestHandleAgentEventSendsChoice ask_user 事件转成 choice 消息发出并标记
// sentChoice；需起 serve 提供 RPC 响应投递回路。
func TestHandleAgentEventSendsChoice(t *testing.T) {
	fc := newFakeConn()
	proc := &fakeProcessor{out: &api.OutboundMessage{Content: ""}}
	a := New(Config{}, proc)
	go func() { _ = a.serve(context.Background(), fc) }()
	t.Cleanup(func() { _ = fc.close() })

	turn := &activeTurn{conn: fc, conv: conversation{ID: "conv-9"}}
	a.setTurn(turn)

	a.HandleAgentEvent(askUserEvent())

	waitFor(t, 2*time.Second, func() bool { return len(fc.requests()) >= 1 })
	reqs := fc.requests()
	if len(reqs) != 1 || reqs[0].Method != "message.send" {
		t.Fatalf("期望 1 条 message.send，得到 %+v", reqs)
	}
	var send messageSendPayload
	if err := json.Unmarshal(reqs[0].Payload, &send); err != nil {
		t.Fatalf("解析 payload: %v", err)
	}
	if send.Message.Type != "choice" || send.Message.Content != "**这是哪家客户？**" || send.Message.Selection != "single" {
		t.Fatalf("choice 消息不符: %+v", send.Message)
	}
	if len(send.Message.Options) != 2 ||
		send.Message.Options[0].ID != "chaitin" || send.Message.Options[0].Label != "长亭科技" ||
		send.Message.Options[1].ID != "bank" {
		t.Fatalf("选项不符: %+v", send.Message.Options)
	}
	if !turn.sentChoice {
		t.Fatal("应标记 sentChoice（finishTurn 跳过文本回复）")
	}

	// 非 ask_user 事件与无轮次时均不发消息。
	fc2 := newFakeConn()
	a2 := New(Config{}, proc)
	go func() { _ = a2.serve(context.Background(), fc2) }()
	t.Cleanup(func() { _ = fc2.close() })
	a2.HandleAgentEvent(askUserEvent()) // 无轮次
	a2.setTurn(&activeTurn{conn: fc2})
	a2.HandleAgentEvent(agent.Event{Type: agent.EventDone}) // 非 ask_user
	time.Sleep(30 * time.Millisecond)
	if n := len(fc2.requests()); n != 0 {
		t.Fatalf("非 ask_user / 无轮次不应发消息，得到 %d 条", n)
	}
}

// TestSanitizeOptionID 中文 value 等非法字符替换为 _，空/冲突回退 opt-N。
func TestSanitizeOptionID(t *testing.T) {
	used := map[string]bool{}
	if got := sanitizeOptionID("金融行业", 0, used); got != "opt-0" {
		t.Fatalf("全非法字符应回退 opt-0，得到 %q", got)
	}
	if got := sanitizeOptionID("bank-2", 1, used); got != "bank-2" {
		t.Fatalf("合法 value 应原样，得到 %q", got)
	}
	used["dup"] = true
	if got := sanitizeOptionID("dup", 2, used); got != "opt-2" {
		t.Fatalf("冲突应回退 opt-2，得到 %q", got)
	}
}

// TestSanitizeOptionIDNoCollision 回退 opt-N 不得与已用 id 相撞（否则两个选项
// 共用 id，点选会映射到另一个选项的 value——「点了 A 收到 B」）。
func TestSanitizeOptionIDNoCollision(t *testing.T) {
	values := []string{"opt-1", "银行", "opt-2"}
	used := map[string]bool{}
	ids := make([]string, len(values))
	for i, v := range values {
		ids[i] = sanitizeOptionID(v, i, used)
		if used[ids[i]] {
			t.Fatalf("value %q 得到已占用 id %q", v, ids[i])
		}
		used[ids[i]] = true
	}
}

// TestServeChoiceResponseRoundTrip 点选答案映射回 value 进入会话，回复并 ACK。
func TestServeChoiceResponseRoundTrip(t *testing.T) {
	fc := newFakeConn()
	proc := &fakeProcessor{out: &api.OutboundMessage{Content: "已确认：金融行业。"}}
	a := New(Config{}, proc)
	a.choiceMu.Lock()
	a.choices["cm-1"] = map[string]string{"opt-0": "finance"}
	a.choiceMu.Unlock()

	fc.enqueue(eventEnvelope(9, eventChoiceResponseCreated, choiceResponsePayload{
		Conversation:  conversation{ID: "conv-9"},
		ChoiceMessage: message{ID: "cm-1", Body: messageBody{Options: []choiceOption{{ID: "opt-0", Label: "金融"}}}},
		Response:      choiceResponse{OptionIDs: []string{"opt-0"}},
	}))

	runServe(t, a, fc, 2)

	reqs := fc.requests()
	if len(reqs) != 2 || reqs[0].Method != "message.send" || reqs[1].Method != "events.ack" {
		t.Fatalf("请求顺序不符: %+v", reqs)
	}
	var send messageSendPayload
	_ = json.Unmarshal(reqs[0].Payload, &send)
	if send.Target.ConversationID != "conv-9" || send.ReplyToMessageID != "cm-1" {
		t.Fatalf("回复目标不符: %+v", send)
	}
	if len(proc.calls) != 1 || proc.calls[0].Content != "finance" {
		t.Fatalf("应映射回 value 作为输入，得到 %+v", proc.calls)
	}
	a.choiceMu.Lock()
	_, remain := a.choices["cm-1"]
	a.choiceMu.Unlock()
	if remain {
		t.Fatal("choice 映射应已删除")
	}
}

// TestServeChoiceResponseFallbackLabel 映射缺失时回退 label。
func TestServeChoiceResponseFallbackLabel(t *testing.T) {
	fc := newFakeConn()
	proc := &fakeProcessor{out: &api.OutboundMessage{Content: "ok"}}
	a := New(Config{}, proc)

	fc.enqueue(eventEnvelope(10, eventChoiceResponseCreated, choiceResponsePayload{
		Conversation:  conversation{ID: "conv-9"},
		ChoiceMessage: message{ID: "cm-x", Body: messageBody{Options: []choiceOption{{ID: "o1", Label: "互联网"}}}},
		Response:      choiceResponse{OptionIDs: []string{"o1"}},
	}))

	runServe(t, a, fc, 2)

	if len(proc.calls) != 1 || proc.calls[0].Content != "互联网" {
		t.Fatalf("应回退 label 作为输入，得到 %+v", proc.calls)
	}
}

// TestServeAskUserSkipsTextReply 已发 choice 的轮次跳过文本回复，只 ACK。
func TestServeAskUserSkipsTextReply(t *testing.T) {
	fc := newFakeConn()
	gate := make(chan struct{})
	proc := &gatedProcessor{fakeProcessor{out: &api.OutboundMessage{Content: "请选择上面的选项"}}, gate}
	a := New(Config{AckAfter: -1}, proc)

	done := make(chan error, 1)
	go func() { done <- a.serve(context.Background(), fc) }()
	fc.enqueue(messageCreatedEvent(11, "app", "user", "帮我分析"))

	time.Sleep(20 * time.Millisecond)
	a.HandleAgentEvent(askUserEvent())
	close(gate)

	waitFor(t, 2*time.Second, func() bool { return len(fc.requests()) >= 2 })
	reqs := fc.requests()
	var sp messageSendPayload
	_ = json.Unmarshal(reqs[0].Payload, &sp)
	if reqs[0].Method != "message.send" || sp.Message.Type != "choice" {
		t.Fatalf("第一条应是 choice 消息，得到 %+v", reqs[0])
	}
	if len(reqs) != 2 || reqs[1].Method != "events.ack" {
		t.Fatalf("choice 后只应 ACK（跳过文本回复），得到 %+v", reqs)
	}
	fc.close()
	if err := <-done; !errors.Is(err, io.EOF) {
		t.Fatalf("serve 退出期望 io.EOF，得到 %v", err)
	}
}

// ── 话题隔离（群聊消息 → 话题会话） ─────────────────────────

// TestServeGroupMessageCreatesTopic 群聊消息建话题，处理与回复都落在话题会话。
func TestServeGroupMessageCreatesTopic(t *testing.T) {
	fc := newFakeConn()
	fc.respFor = map[string]json.RawMessage{
		"conversation.topic.create": json.RawMessage(`{"conversation":{"id":"topic-1","type":"topic"}}`),
	}
	fc.enqueue(messageCreatedEvent(13, "group", "user", "分析一下这个需求"))
	proc := &fakeProcessor{out: &api.OutboundMessage{Content: "结果"}}
	a := New(Config{}, proc)

	runServe(t, a, fc, 3)

	reqs := fc.requests()
	if len(reqs) != 3 {
		t.Fatalf("期望 3 条请求（topic.create + message.send + events.ack），得到 %d", len(reqs))
	}
	if reqs[0].Method != "conversation.topic.create" || reqs[1].Method != "message.send" || reqs[2].Method != "events.ack" {
		t.Fatalf("请求顺序不符: %s, %s, %s", reqs[0].Method, reqs[1].Method, reqs[2].Method)
	}
	var tp topicCreatePayload
	if err := json.Unmarshal(reqs[0].Payload, &tp); err != nil || tp.ConversationID != "conv-1" || tp.SourceMessageID != "msg-1" {
		t.Fatalf("topic.create 参数不符: %v %+v", err, tp)
	}
	if len(proc.calls) != 1 || proc.calls[0].SessionID != "topic-1" || proc.calls[0].Content != "分析一下这个需求" {
		t.Fatalf("处理应落在话题会话: %+v", proc.calls)
	}
	var send messageSendPayload
	_ = json.Unmarshal(reqs[1].Payload, &send)
	if send.Target.ConversationID != "topic-1" {
		t.Fatalf("回复应发到话题: %+v", send.Target)
	}
}

// TestServeDirectConversationNoTopic 单聊/话题消息不建话题。
func TestServeDirectConversationNoTopic(t *testing.T) {
	for _, convType := range []string{"app", "topic"} {
		fc := newFakeConn()
		fc.enqueue(messageCreatedEvent(14, convType, "user", "继续刚才的问题"))
		proc := &fakeProcessor{out: &api.OutboundMessage{Content: "ok"}}
		a := New(Config{}, proc)

		runServe(t, a, fc, 2)

		reqs := fc.requests()
		if len(reqs) != 2 || reqs[0].Method != "message.send" || reqs[1].Method != "events.ack" {
			t.Fatalf("%s 应直接回复不建话题，得到 %+v", convType, reqs)
		}
		if proc.calls[0].SessionID != "conv-1" {
			t.Fatalf("session 应为当前会话: %+v", proc.calls[0])
		}
	}
}

// TestServeTopicClosedEvicts 话题关闭：触发清理回调并 ACK。
func TestServeTopicClosedEvicts(t *testing.T) {
	fc := newFakeConn()
	fc.enqueue(eventEnvelope(15, eventTopicClosed, topicClosedPayload{
		Archived: true, ConversationID: "topic-9",
	}))
	proc := &fakeProcessor{out: &api.OutboundMessage{Content: "x"}}
	a := New(Config{}, proc)
	evicted := make(chan string, 1)
	a.SetOnTopicClosed(func(sid string) { evicted <- sid })

	runServe(t, a, fc, 1)

	select {
	case sid := <-evicted:
		if sid != "topic-9" {
			t.Fatalf("应驱逐 topic-9，得到 %s", sid)
		}
	default:
		t.Fatal("话题关闭应触发清理回调")
	}
	reqs := fc.requests()
	if len(reqs) != 1 || reqs[0].Method != "events.ack" {
		t.Fatalf("应 ACK: %+v", reqs)
	}
}
