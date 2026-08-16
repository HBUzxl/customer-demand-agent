package jiying

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"

	"customer-demand-agent/internal/agent"
	"customer-demand-agent/internal/api"
	"customer-demand-agent/internal/channel"
)

// 默认参数（Config 零值时的兜底）。
const (
	defaultReconnectBase  = time.Second
	defaultReconnectMax   = 30 * time.Second
	defaultReadTimeout    = 90 * time.Second
	defaultRequestTimeout = 15 * time.Second
	defaultAckDelay       = 2 * time.Second // 慢处理受理回执的触发阈值
)

// ackText 是慢处理的受理回执文案（平台无消息编辑接口，做不了真流式）。
const ackText = "已收到，正在分析，请稍候…"

// Config 是即应应用接入配置（AppID/Secret/WSURL 来自平台「应用接入信息」页）。
type Config struct {
	AppID  string // 应用 ID
	Secret string // 连接密钥，只放服务端
	WSURL  string // WebSocket 地址
	Proxy  string // HTTP 代理；空则直连

	// 可选调参，零值取默认。
	ReconnectBase  time.Duration
	ReconnectMax   time.Duration
	ReadTimeout    time.Duration
	RequestTimeout time.Duration
	AckAfter       time.Duration // 受理回执阈值；0=默认，负数=禁用
}

// conn 抽象 WebSocket 连接，便于测试注入；wsConn 是唯一生产实现。
type conn interface {
	readMessage() ([]byte, error)
	writeText([]byte) error
	setReadDeadline(time.Time) error
	close() error
}

// Adapter 是即应渠道适配器：维护 WebSocket 连接，串行处理可靠事件，调 RPC 并 ACK。
// 实现 channel.Channel，Process 委托给 channel.Processor。
type Adapter struct {
	cfg  Config
	proc channel.Processor

	mu      sync.Mutex
	pending map[string]chan envelope // 请求 id → 响应（read 协程投递）
	acked   int64                    // 已确认的最大 cursor（去重：<= 该值的事件跳过）

	turnMu  sync.Mutex
	curTurn *activeTurn // 当前处理中的轮次（serve 串行，至多一个）

	choiceMu sync.Mutex
	choices  map[string]map[string]string // 已发 choice 消息 id → 选项 id → 回传 value

	onTopicClosed func(sessionID string) // 话题关闭回调（驱逐短期记忆等），由 main 注入，可空
}

// activeTurn 记录处理中的一轮，供 agent 流式事件定位目标会话。
type activeTurn struct {
	conn       conn
	conv       conversation
	sentChoice bool // 本轮已发 choice，收尾时跳过文本回复
}

// New 创建即应渠道适配器。
func New(cfg Config, proc channel.Processor) *Adapter {
	return &Adapter{cfg: cfg, proc: proc, pending: make(map[string]chan envelope), choices: make(map[string]map[string]string)}
}

// Name 返回渠道标识。
func (a *Adapter) Name() string { return "jiying" }

// Process 同步处理一条消息；异步事件循环见 Run。
func (a *Adapter) Process(ctx context.Context, in *api.InboundMessage) (*api.OutboundMessage, error) {
	return a.proc.Process(ctx, in)
}

// Run 连接平台并处理事件，断线自动重连；ctx 取消时返回。
func (a *Adapter) Run(ctx context.Context) error {
	attempt := 0
	for {
		if ctx.Err() != nil {
			return nil
		}
		if attempt > 0 {
			d := a.backoff(attempt)
			log.Printf("[warn] 即应渠道 %d 秒后重连（第 %d 次）", int(d.Seconds()), attempt)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(d):
			}
		}
		c, err := a.connect()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			attempt++
			log.Printf("[warn] 即应渠道连接失败: %v", err)
			continue
		}
		attempt = 0
		log.Printf("[ok] 即应渠道已连接 %s", a.cfg.WSURL)
		err = a.serve(ctx, c)
		if ctx.Err() != nil {
			return nil
		}
		attempt++
		log.Printf("[warn] 即应渠道连接断开，将重连（未确认事件由平台重放）: %v", err)
	}
}

// connect 建立带应用身份握手的 WebSocket 连接。
func (a *Adapter) connect() (conn, error) {
	h := http.Header{}
	h.Set("X-MagicChat-App-ID", a.cfg.AppID)
	h.Set("Authorization", "Bearer "+a.cfg.Secret)
	return dialWS(a.cfg.WSURL, a.cfg.Proxy, h)
}

// backoff 全抖动指数退避：base * 2^(n-1)，封顶 ReconnectMax，[0,d) 抖动。
func (a *Adapter) backoff(attempt int) time.Duration {
	base := a.cfg.ReconnectBase
	if base <= 0 {
		base = defaultReconnectBase
	}
	max := a.cfg.ReconnectMax
	if max <= 0 {
		max = defaultReconnectMax
	}
	d := base
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= max {
			return jitter(max)
		}
	}
	if d > max {
		d = max
	}
	return jitter(d)
}

func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(d)))
}

func (a *Adapter) readTimeout() time.Duration {
	if a.cfg.ReadTimeout > 0 {
		return a.cfg.ReadTimeout
	}
	return defaultReadTimeout
}

func (a *Adapter) requestTimeout() time.Duration {
	if a.cfg.RequestTimeout > 0 {
		return a.cfg.RequestTimeout
	}
	return defaultRequestTimeout
}

// serve 维护单条连接：读协程投递 RPC 响应、事件进队列；主循环按 cursor 串行
// 处理事件，成功才 ACK。
func (a *Adapter) serve(ctx context.Context, c conn) error {
	serveCtx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
		c.close() // 解除 reader 阻塞读
	}()

	events := make(chan envelope, 16)
	readErr := make(chan error, 1)

	go func() {
		for {
			if err := c.setReadDeadline(time.Now().Add(a.readTimeout())); err != nil {
				readErr <- err
				return
			}
			data, err := c.readMessage()
			if err != nil {
				readErr <- err
				return
			}
			var env envelope
			if err := json.Unmarshal(data, &env); err != nil {
				log.Printf("[warn] 即应渠道忽略无法解析的消息: %v", err)
				continue
			}
			switch env.Kind {
			case kindResponse:
				a.deliver(env)
			case kindEvent:
				select {
				case events <- env:
				case <-serveCtx.Done():
					return
				}
			default:
				log.Printf("[warn] 即应渠道忽略未知消息 kind: %s", env.Kind)
			}
		}
	}()

	for {
		select {
		case env := <-events:
			if err := a.handleEvent(ctx, c, env); err != nil {
				// 处理失败不 ACK，断开重连让平台按 cursor 重放（可靠投递语义）。
				return err
			}
		case err := <-readErr:
			return err
		case <-serveCtx.Done():
			return serveCtx.Err()
		}
	}
}

// deliver 把响应按 reply_to 关联到挂起的请求。
func (a *Adapter) deliver(env envelope) {
	a.mu.Lock()
	ch, ok := a.pending[env.ReplyTo]
	a.mu.Unlock()
	if !ok {
		return
	}
	select {
	case ch <- env:
	default:
	}
}

// request 发一次 RPC 并等待响应。
func (a *Adapter) request(c conn, method string, payload any) error {
	_, err := a.requestRaw(c, method, payload)
	return err
}

// requestRaw 同 request，但返回响应 payload。
func (a *Adapter) requestRaw(c conn, method string, payload any) (json.RawMessage, error) {
	id := newRequestID()
	ch := make(chan envelope, 1)
	a.mu.Lock()
	a.pending[id] = ch
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.pending, id)
		a.mu.Unlock()
	}()

	env := envelope{V: protocolVersion, Kind: kindRequest, ID: id, Method: method}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化 %s 参数: %w", method, err)
	}
	env.Payload = raw
	data, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("序列化 %s 请求: %w", method, err)
	}
	if err := c.writeText(data); err != nil {
		return nil, fmt.Errorf("发送 %s: %w", method, err)
	}

	select {
	case resp := <-ch:
		if resp.OK != nil && !*resp.OK {
			return nil, &rpcError{Method: method, Code: resp.errCode(), Message: resp.errMessage()}
		}
		return resp.Payload, nil
	case <-time.After(a.requestTimeout()):
		return nil, fmt.Errorf("%s 超时（%s 内未收到响应）", method, a.requestTimeout())
	}
}

func (e envelope) errCode() string {
	if e.Error != nil {
		return e.Error.Code
	}
	return ""
}

func (e envelope) errMessage() string {
	if e.Error != nil {
		return e.Error.Message
	}
	return "未知错误"
}

// handleEvent 串行处理一条可靠事件；返回 error 表示不 ACK，由 serve 断开重连重放。
func (a *Adapter) handleEvent(ctx context.Context, c conn, env envelope) error {
	// 去重：不大于已确认 cursor 的重复投递直接确认。
	a.mu.Lock()
	dup := a.acked > 0 && env.Cursor <= a.acked
	a.mu.Unlock()
	if dup {
		return a.ack(c, env.Cursor)
	}

	switch env.Event {
	case eventMessageCreated:
		return a.onMessageCreated(ctx, c, env)
	case eventChoiceResponseCreated:
		return a.onChoiceResponse(ctx, c, env)
	case eventTopicClosed:
		// 话题关闭：驱逐该会话短期记忆，然后 ACK。
		var tc topicClosedPayload
		if err := json.Unmarshal(env.Payload, &tc); err == nil && tc.ConversationID != "" && a.onTopicClosed != nil {
			a.onTopicClosed(tc.ConversationID)
		}
		return a.ack(c, env.Cursor)
	default:
		// 未知事件只 ACK。
		return a.ack(c, env.Cursor)
	}
}

// onMessageCreated 处理用户消息：群聊先从来源消息建话题（话题即独立会话，
// 多用户互不串上下文），单聊/话题消息直接在当前会话处理。
func (a *Adapter) onMessageCreated(ctx context.Context, c conn, env envelope) error {
	var mc messageCreatedPayload
	if err := json.Unmarshal(env.Payload, &mc); err != nil {
		return fmt.Errorf("解析 message.created: %w", err)
	}
	// 只回应「用户」消息（不回应其他应用）；空文本直接 ACK。
	if mc.Sender.Type != "user" || strings.TrimSpace(mc.text()) == "" {
		return a.ack(c, env.Cursor)
	}

	// 群聊：建话题后处理；失败退回群内直接回复，不阻断响应。
	if mc.Conversation.Type == "group" {
		topic, err := a.ensureTopic(c, mc)
		if err != nil {
			log.Printf("[warn] 即应渠道建话题失败，退回群内回复: %v", err)
		} else {
			in := &api.InboundMessage{SessionID: topic.ID, Content: mc.text(), Source: "jiying"}
			return a.processInbound(ctx, c, topic, mc.Message.ID, env.Cursor, in)
		}
	}

	// 单聊/话题：直接处理。
	in := &api.InboundMessage{SessionID: mc.Conversation.ID, Content: mc.text(), Source: "jiying"}
	return a.processInbound(ctx, c, mc.Conversation, mc.Message.ID, env.Cursor, in)
}

// ensureTopic 基于来源消息创建或复用话题（同来源消息幂等）。
func (a *Adapter) ensureTopic(c conn, mc messageCreatedPayload) (conversation, error) {
	payload, err := a.requestRaw(c, "conversation.topic.create", topicCreatePayload{
		ConversationID:  mc.Conversation.ID,
		SourceMessageID: mc.Message.ID,
	})
	if err != nil {
		return conversation{}, err
	}
	var resp struct {
		Conversation conversation `json:"conversation"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil || resp.Conversation.ID == "" {
		return conversation{}, fmt.Errorf("解析 conversation.topic.create 响应: %w", err)
	}
	if resp.Conversation.Type == "" {
		resp.Conversation.Type = "topic"
	}
	return resp.Conversation, nil
}

// processInbound 是用户输入的公共处理管线（普通消息与 choice 回答共用）：
// 异步跑 Processor，超过阈值未完成先发受理回执，完成后回结果并 ACK。
func (a *Adapter) processInbound(ctx context.Context, c conn, conv conversation, replyTo string, cursor int64, in *api.InboundMessage) error {
	// 注册本轮上下文，供 agent 流式事件定位目标会话。
	turn := &activeTurn{conn: c, conv: conv}
	a.setTurn(turn)
	defer a.setTurn(nil)

	done := make(chan procResult, 1)
	go func() {
		out, err := a.proc.Process(ctx, in)
		done <- procResult{out: out, err: err}
	}()

	// 回执定时器只触发一次（置 nil 后阻塞）；快问答先完成则不发回执。
	var ackCh <-chan time.Time
	if d := a.ackDelay(); d > 0 {
		t := time.NewTimer(d)
		defer t.Stop()
		ackCh = t.C
	}
	for {
		select {
		case <-ackCh:
			// done 与定时器同时就绪时 select 随机选择，先检查 done 避免发冗余回执。
			select {
			case res := <-done:
				return a.finishTurn(c, turn, conv, replyTo, cursor, res)
			default:
			}
			ackCh = nil
			if err := a.reply(c, conv, replyTo, ackText); err != nil {
				return fmt.Errorf("发送受理回执: %w", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		case res := <-done:
			return a.finishTurn(c, turn, conv, replyTo, cursor, res)
		}
	}
}

// SetOnTopicClosed 注册话题关闭回调。
func (a *Adapter) SetOnTopicClosed(fn func(sessionID string)) { a.onTopicClosed = fn }

// setTurn 更新当前轮次上下文。
func (a *Adapter) setTurn(t *activeTurn) {
	a.turnMu.Lock()
	a.curTurn = t
	a.turnMu.Unlock()
}

// HandleAgentEvent 把 agent 的 ask_user 事件转成平台 choice 消息，用户点选后
// 经 choice.response_created 闭环；其余事件忽略。
func (a *Adapter) HandleAgentEvent(ev agent.Event) {
	if ev.Type != agent.EventAskUser || len(ev.Options) < 2 {
		return
	}
	a.turnMu.Lock()
	t := a.curTurn
	a.turnMu.Unlock()
	if t == nil {
		return
	}
	if err := a.sendChoice(t, ev); err != nil {
		log.Printf("[warn] 即应渠道发送 choice 失败（会话 %s）: %v", t.conv.ID, err)
		return
	}
	t.sentChoice = true
}

// sendChoice 发送 choice 消息并登记「消息 id → 选项 id → value」映射。
func (a *Adapter) sendChoice(t *activeTurn, ev agent.Event) error {
	body := messageBody{
		Type:        "choice",
		ContentType: "markdown",
		Content:     ev.Question,
		Selection:   "single",
	}
	idToValue := make(map[string]string, len(ev.Options))
	used := make(map[string]bool, len(ev.Options))
	for i, opt := range ev.Options {
		value := opt.Value
		if value == "" {
			value = opt.Label
		}
		id := sanitizeOptionID(value, i, used)
		used[id] = true
		idToValue[id] = value
		body.Options = append(body.Options, choiceOption{ID: id, Label: opt.Label})
	}

	payload, err := a.requestRaw(t.conn, "message.send", messageSendPayload{
		Target:  messageTarget{Type: "conversation", ConversationID: t.conv.ID},
		Message: body,
	})
	if err != nil {
		return err
	}
	var resp struct {
		Message struct {
			ID string `json:"id"`
		} `json:"message"`
	}
	if json.Unmarshal(payload, &resp) != nil || resp.Message.ID == "" {
		return nil // 拿不到消息 id 时，回答回退 label
	}
	a.choiceMu.Lock()
	a.choices[resp.Message.ID] = idToValue
	a.choiceMu.Unlock()
	return nil
}

// sanitizeOptionID 把 value 规整为平台合法选项 id（字母数字-_，≤64 字符）；
// 冲突/空时用 opt-N 兑底。
func sanitizeOptionID(value string, idx int, used map[string]bool) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
		if b.Len() >= 64 {
			break
		}
	}
	id := strings.Trim(b.String(), "_")
	if id == "" || used[id] {
		return fmt.Sprintf("opt-%d", idx)
	}
	return id
}

// procResult 是一轮异步处理的结果。
type procResult struct {
	out *api.OutboundMessage
	err error
}

// finishTurn 收尾一轮：失败回退文案，成功回最终内容（已发 choice 则跳过）；
// 随后 ACK。返回 error 表示不应 ACK，由 serve 重连重放。
func (a *Adapter) finishTurn(c conn, turn *activeTurn, conv conversation, replyTo string, cursor int64, res procResult) error {
	if res.err != nil {
		log.Printf("[warn] 即应渠道处理会话 %s 失败: %v", conv.ID, res.err)
		if serr := a.reply(c, conv, replyTo, "抱歉，处理你的请求时出错了，请稍后再试。"); serr != nil {
			return fmt.Errorf("发送错误回复: %w", serr)
		}
		return a.ack(c, cursor)
	}
	if turn.sentChoice || strings.TrimSpace(res.out.Content) == "" {
		return a.ack(c, cursor)
	}
	if err := a.reply(c, conv, replyTo, res.out.Content); err != nil {
		return fmt.Errorf("message.send 失败: %w", err)
	}
	return a.ack(c, cursor)
}

// ackDelay 返回回执阈值，负数禁用。
func (a *Adapter) ackDelay() time.Duration {
	if a.cfg.AckAfter != 0 {
		return a.cfg.AckAfter
	}
	return defaultAckDelay
}

// onChoiceResponse 处理选择消息的点选：把选项映射回 value，作为用户输入
// 继续会话（与 Web 前端点选同语义）。
func (a *Adapter) onChoiceResponse(ctx context.Context, c conn, env envelope) error {
	var cc choiceResponsePayload
	if err := json.Unmarshal(env.Payload, &cc); err != nil {
		return fmt.Errorf("解析 choice.response_created: %w", err)
	}
	if cc.Conversation.ID == "" || len(cc.Response.OptionIDs) == 0 {
		return a.ack(c, env.Cursor)
	}
	// choice 消息总在当前会话发出，回答直接进同一会话链。
	in := &api.InboundMessage{
		SessionID: cc.Conversation.ID,
		Content:   strings.Join(a.mapOptionValues(cc), "\n"),
		Source:    "jiying",
	}
	return a.processInbound(ctx, c, cc.Conversation, cc.ChoiceMessage.ID, env.Cursor, in)
}

// mapOptionValues 把选项 id 映射回 value，映射缺失时回退 label；用后即删。
func (a *Adapter) mapOptionValues(cc choiceResponsePayload) []string {
	a.choiceMu.Lock()
	idToValue := a.choices[cc.ChoiceMessage.ID]
	delete(a.choices, cc.ChoiceMessage.ID)
	a.choiceMu.Unlock()

	labels := make(map[string]string, len(cc.ChoiceMessage.Body.Options))
	for _, opt := range cc.ChoiceMessage.Body.Options {
		labels[opt.ID] = opt.Label
	}
	values := make([]string, 0, len(cc.Response.OptionIDs))
	for _, id := range cc.Response.OptionIDs {
		if v, ok := idToValue[id]; ok && v != "" {
			values = append(values, v)
		} else if l := labels[id]; l != "" {
			values = append(values, l)
		}
	}
	return values
}

// reply 向会话发送 markdown 回复（统一 target.type=conversation，
// 对单聊/群聊/话题通用）。
func (a *Adapter) reply(c conn, conv conversation, replyTo, text string) error {
	return a.request(c, "message.send", messageSendPayload{
		Target:           messageTarget{Type: "conversation", ConversationID: conv.ID},
		ReplyToMessageID: replyTo,
		Message:          messageBody{Type: "markdown", Content: text},
	})
}

// ack 确认 cursor；会同时确认更早的事件，必须在业务成功后调用。
func (a *Adapter) ack(c conn, cursor int64) error {
	if err := a.request(c, "events.ack", ackPayload{Cursor: cursor}); err != nil {
		return fmt.Errorf("events.ack(cursor %d) 失败: %w", cursor, err)
	}
	a.mu.Lock()
	if cursor > a.acked {
		a.acked = cursor
	}
	a.mu.Unlock()
	return nil
}

// newRequestID 生成唯一请求 id（幂等键）。
func newRequestID() string {
	b := make([]byte, 8)
	_, _ = crand.Read(b)
	return "req-" + hex.EncodeToString(b)
}
