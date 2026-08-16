package jiying

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// ── 测试用服务端帧实现（收帧解掩码、发帧不掩码）────────────

func serverReadFrame(br *bufio.Reader) (fin bool, opcode byte, payload []byte, err error) {
	var hdr [2]byte
	if _, err = io.ReadFull(br, hdr[:]); err != nil {
		return
	}
	fin = hdr[0]&0x80 != 0
	opcode = hdr[0] & 0x0f
	masked := hdr[1]&0x80 != 0
	length := uint64(hdr[1] & 0x7f)
	switch length {
	case 126:
		var ext [2]byte
		if _, err = io.ReadFull(br, ext[:]); err != nil {
			return
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err = io.ReadFull(br, ext[:]); err != nil {
			return
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	var mask [4]byte
	if masked {
		if _, err = io.ReadFull(br, mask[:]); err != nil {
			return
		}
	}
	payload = make([]byte, length)
	if length > 0 {
		if _, err = io.ReadFull(br, payload); err != nil {
			return
		}
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return
}

func serverWriteFrame(w io.Writer, fin bool, opcode byte, payload []byte) error {
	n := len(payload)
	var hdr []byte
	switch {
	case n < 126:
		hdr = make([]byte, 2)
		hdr[1] = byte(n)
	case n < 65536:
		hdr = make([]byte, 4)
		hdr[1] = 126
		binary.BigEndian.PutUint16(hdr[2:4], uint16(n))
	default:
		hdr = make([]byte, 10)
		hdr[1] = 127
		binary.BigEndian.PutUint64(hdr[2:10], uint64(n))
	}
	if fin {
		hdr[0] = 0x80
	}
	hdr[0] |= opcode
	if _, err := w.Write(hdr); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	_, err := w.Write(payload)
	return err
}

func newPipePair(t *testing.T) (*wsConn, net.Conn, *bufio.Reader) {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { clientConn.Close(); serverConn.Close() })
	c := &wsConn{conn: clientConn, br: bufio.NewReader(clientConn)}
	return c, serverConn, bufio.NewReader(serverConn)
}

// TestWSConnRoundTrip 客户端发文本（必掩码）→ 服务端解掩码；服务端回文本（不掩码）→ 客户端读回。
func TestWSConnRoundTrip(t *testing.T) {
	c, serverConn, sbr := newPipePair(t)

	sendDone := make(chan error, 1)
	go func() { sendDone <- c.writeText([]byte("hello 世界")) }()

	fin, op, payload, err := serverReadFrame(sbr)
	if err != nil {
		t.Fatalf("服务端读帧: %v", err)
	}
	if !fin || op != opText || string(payload) != "hello 世界" {
		t.Fatalf("帧不符: fin=%v op=%d payload=%q", fin, op, payload)
	}
	if err := <-sendDone; err != nil {
		t.Fatalf("客户端写: %v", err)
	}

	recvDone := make(chan struct{})
	var got []byte
	var rerr error
	go func() { got, rerr = c.readMessage(); close(recvDone) }()
	if err := serverWriteFrame(serverConn, true, opText, []byte("hi back")); err != nil {
		t.Fatalf("服务端写: %v", err)
	}
	<-recvDone
	if rerr != nil || string(got) != "hi back" {
		t.Fatalf("客户端读: data=%q err=%v", got, rerr)
	}
}

// TestWSConnPingPong 客户端收到 ping 应自动回 pong。net.Pipe 无缓冲：服务端必须
// 先读掉 pong 再写后续帧，否则双方互等（客户端等 pong 被读、服务端等 text 被读）。
func TestWSConnPingPong(t *testing.T) {
	c, serverConn, sbr := newPipePair(t)

	type pongFrame struct {
		fin     bool
		opcode  byte
		payload []byte
	}
	pongCh := make(chan pongFrame, 1)
	go func() {
		_ = serverWriteFrame(serverConn, true, opPing, []byte("probe"))
		fin, op, payload, err := serverReadFrame(sbr)
		if err != nil {
			return
		}
		pongCh <- pongFrame{fin: fin, opcode: op, payload: payload}
		_ = serverWriteFrame(serverConn, true, opText, []byte("after-ping"))
	}()

	got, err := c.readMessage()
	if err != nil {
		t.Fatalf("客户端读: %v", err)
	}
	if string(got) != "after-ping" {
		t.Fatalf("got %q", got)
	}

	select {
	case p := <-pongCh:
		if !p.fin || p.opcode != opPong || string(p.payload) != "probe" {
			t.Fatalf("pong 不符: fin=%v op=%d payload=%q", p.fin, p.opcode, p.payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("等待 pong 超时")
	}
}

// TestWSConnFragmented 分片消息应被重组为完整消息。
func TestWSConnFragmented(t *testing.T) {
	c, serverConn, _ := newPipePair(t)

	go func() {
		_ = serverWriteFrame(serverConn, false, opText, []byte("hel"))
		_ = serverWriteFrame(serverConn, true, opContinuation, []byte("lo"))
	}()

	got, err := c.readMessage()
	if err != nil {
		t.Fatalf("客户端读: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}
}

// TestWSConnExtendedLength 载荷 > 125 字节走 16 位扩展长度。
func TestWSConnExtendedLength(t *testing.T) {
	c, _, sbr := newPipePair(t)
	big := strings.Repeat("a", 300)

	sendDone := make(chan error, 1)
	go func() { sendDone <- c.writeText([]byte(big)) }()

	fin, op, payload, err := serverReadFrame(sbr)
	if err != nil {
		t.Fatalf("服务端读帧: %v", err)
	}
	if !fin || op != opText || string(payload) != big {
		t.Fatalf("帧不符: fin=%v op=%d len=%d", fin, op, len(payload))
	}
	if err := <-sendDone; err != nil {
		t.Fatalf("客户端写: %v", err)
	}
}

// TestWSConnClose 收到 close 帧应回 close 并返回 io.EOF。客户端回包需要服务端
// 并发去读（net.Pipe 无缓冲），否则客户端阻塞在 readMessage 内部的写上。
func TestWSConnClose(t *testing.T) {
	c, serverConn, sbr := newPipePair(t)

	closeReply := make(chan bool, 1)
	go func() {
		_ = serverWriteFrame(serverConn, true, opClose, []byte{0x03, 0xE8})
		fin, op, _, err := serverReadFrame(sbr)
		closeReply <- err == nil && fin && op == opClose
	}()

	if _, err := c.readMessage(); err != io.EOF {
		t.Fatalf("期望 io.EOF，得到 %v", err)
	}

	select {
	case ok := <-closeReply:
		if !ok {
			t.Fatal("close 回包不符：期望 fin=1 opcode=close")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("等待 close 回包超时")
	}
}

// TestResolveAddr URL 未带端口时按 scheme 补默认端口（wss→443、ws→80），
// 带端口与 IPv6 字面量保持原样。
func TestResolveAddr(t *testing.T) {
	cases := []struct{ raw, addr, name string }{
		{"wss://chat.chaitin.net/api/app/ws", "chat.chaitin.net:443", "chat.chaitin.net"},
		{"ws://example.com", "example.com:80", "example.com"},
		{"ws://example.com:8080/x", "example.com:8080", "example.com"},
		{"ws://[::1]:9000/", "[::1]:9000", "::1"},
	}
	for _, c := range cases {
		u, err := url.Parse(c.raw)
		if err != nil {
			t.Fatalf("解析 %q: %v", c.raw, err)
		}
		addr, name := resolveAddr(u)
		if addr != c.addr || name != c.name {
			t.Errorf("resolveAddr(%q) = (%q, %q)，期望 (%q, %q)", c.raw, addr, name, c.addr, c.name)
		}
	}
}

// TestComputeAccept 用 RFC 示例值校验 Sec-WebSocket-Accept 计算。
func TestComputeAccept(t *testing.T) {
	// RFC 6455 示例值。
	got := computeAccept("dGhlIHNhbXBsZSBub25jZQ==")
	want := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	if got != want {
		t.Fatalf("accept: got %q want %q", got, want)
	}
}

// fakeProxy 起一个最小 HTTP CONNECT 代理：校验 CONNECT 目标与认证头，回 200
// 后把同一条连接当作「目标服务器」继续用（直通模式：响应 WS 升级请求）。
type fakeProxy struct {
	ln        net.Listener
	connectTo chan string // 捕获的 CONNECT 目标
	auth      chan string // 捕获的 Proxy-Authorization
	requireOK chan struct{}
}

func newFakeProxy(t *testing.T) *fakeProxy {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听: %v", err)
	}
	fp := &fakeProxy{
		ln:        ln,
		connectTo: make(chan string, 1),
		auth:      make(chan string, 1),
		requireOK: make(chan struct{}),
	}
	go fp.serve()
	t.Cleanup(func() { ln.Close() })
	return fp
}

func (f *fakeProxy) addr() string { return f.ln.Addr().String() }

func (f *fakeProxy) serve() {
	c, err := f.ln.Accept()
	if err != nil {
		return
	}
	defer c.Close()
	br := bufio.NewReader(c)

	// 1) 读 CONNECT 请求
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	f.connectTo <- req.Host
	f.auth <- req.Header.Get("Proxy-Authorization")
	fmt.Fprint(c, "HTTP/1.1 200 Connection established\r\n\r\n")

	// 2) 直通：同连接上响应 WebSocket 升级
	upReq, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	accept := computeAccept(upReq.Header.Get("Sec-WebSocket-Key"))
	fmt.Fprintf(c, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", accept)

	// 3) 收一帧文本回一帧文本，验证隧道承载 WS 帧
	_, _, payload, err := serverReadFrame(br)
	if err != nil {
		return
	}
	_ = serverWriteFrame(c, true, opText, []byte("echo:"+string(payload)))
	close(f.requireOK)
}

// TestDialWSViaProxy 经 HTTP CONNECT 隧道完成 WS 握手并收发帧。
func TestDialWSViaProxy(t *testing.T) {
	fp := newFakeProxy(t)

	wsURL := "ws://target.example.com/api/app/ws"
	h := http.Header{}
	h.Set("X-MagicChat-App-ID", "app-1")
	c, err := dialWS(wsURL, "http://user:pass@"+fp.addr(), h)
	if err != nil {
		t.Fatalf("dialWS 经代理: %v", err)
	}
	defer c.close()

	if got := <-fp.connectTo; got != "target.example.com:80" {
		t.Fatalf("CONNECT 目标 = %q，期望 target.example.com:80", got)
	}
	auth := <-fp.auth
	if !strings.HasPrefix(auth, "Basic ") {
		t.Fatalf("代理认证头缺失或非 Basic: %q", auth)
	}

	if err := c.writeText([]byte("ping")); err != nil {
		t.Fatalf("写帧: %v", err)
	}
	got, err := c.readMessage()
	if err != nil {
		t.Fatalf("读帧: %v", err)
	}
	if string(got) != "echo:ping" {
		t.Fatalf("隧道回帧不符: %q", got)
	}
	select {
	case <-fp.requireOK:
	default:
		t.Fatal("代理侧未完成帧交换")
	}
}
