// Package jiying 实现即应（MagicChat）第三方应用渠道：应用以独立身份经
// WebSocket 接入，接收可靠事件、调用 RPC、成功后 ACK；断线重连后平台按
// cursor 重放未确认事件。
//
// 依赖方向：channel/jiying → agent（经 channel.Processor 缝合），agent 核心
// 不感知渠道。
//
// ws.go 是纯标准库实现的最小 WebSocket 客户端（ADR-008：不引第三方依赖）：
// 握手、掩码帧、分片重组、ping/pong。
package jiying

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// wsGUID 是握手魔数（Sec-WebSocket-Accept 计算用）。
const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// maxMessageSize 单条消息上限 1 MiB（平台约束）。
const maxMessageSize = 1 << 20

// 帧 opcode。
const (
	opContinuation = 0x0
	opText         = 0x1
	opBinary       = 0x2
	opClose        = 0x8
	opPing         = 0x9
	opPong         = 0xA
)

// handshakeError 记录握手失败（非 101）的 HTTP 状态与响应体。
type handshakeError struct {
	status int
	body   string
}

func (e *handshakeError) Error() string {
	return fmt.Sprintf("WebSocket 握手失败（HTTP %d）: %s", e.status, strings.TrimSpace(e.body))
}

// wsConn 是已握手的 WebSocket 连接（客户端发帧必掩码、收帧不掩码）。
type wsConn struct {
	conn net.Conn
	br   *bufio.Reader
	wmu  sync.Mutex // 串行化写（回 pong 与业务 RPC 并发）
}

// dialWS 拨号（可选经 HTTP 代理隧道）并完成 WebSocket 握手；header 携带
// 应用身份。
func dialWS(rawURL string, proxyURL string, header http.Header) (*wsConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("解析 WebSocket 地址 %q: %w", rawURL, err)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("WebSocket 地址缺少 host: %q", rawURL)
	}
	addr, serverName := resolveAddr(u)

	var conn net.Conn
	switch {
	case proxyURL != "":
		conn, err = dialViaProxy(proxyURL, addr, 10*time.Second)
	case u.Scheme == "ws":
		conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
	case u.Scheme == "wss":
		conn, err = net.DialTimeout("tcp", addr, 10*time.Second)
	default:
		return nil, fmt.Errorf("WebSocket 地址必须是 ws:// 或 wss://，收到 %q", u.Scheme)
	}
	if err != nil {
		return nil, fmt.Errorf("连接 %s: %w", addr, err)
	}

	if u.Scheme == "wss" {
		// 直连与隧道统一：先拿裸连接，再包 TLS。
		tlsConn := tls.Client(conn, &tls.Config{ServerName: serverName})
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("TLS 握手 %s: %w", serverName, err)
		}
		_ = conn.SetDeadline(time.Time{})
		conn = tlsConn
	}

	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		conn.Close()
		return nil, fmt.Errorf("生成 Sec-WebSocket-Key: %w", err)
	}
	keyB64 := base64.StdEncoding.EncodeToString(key)

	path := u.RequestURI()
	if path == "" {
		path = "/"
	}

	var req strings.Builder
	fmt.Fprintf(&req, "GET %s HTTP/1.1\r\n", path)
	fmt.Fprintf(&req, "Host: %s\r\n", u.Host)
	req.WriteString("Upgrade: websocket\r\n")
	req.WriteString("Connection: Upgrade\r\n")
	fmt.Fprintf(&req, "Sec-WebSocket-Key: %s\r\n", keyB64)
	req.WriteString("Sec-WebSocket-Version: 13\r\n")
	for k, vs := range header {
		for _, v := range vs {
			fmt.Fprintf(&req, "%s: %s\r\n", k, v)
		}
	}
	req.WriteString("\r\n")

	if _, err := conn.Write([]byte(req.String())); err != nil {
		conn.Close()
		return nil, fmt.Errorf("发送握手请求: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodGet})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("读取握手响应: %w", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		conn.Close()
		return nil, &handshakeError{status: resp.StatusCode, body: string(body)}
	}
	if got := resp.Header.Get("Sec-WebSocket-Accept"); got != computeAccept(keyB64) {
		conn.Close()
		return nil, errors.New("Sec-WebSocket-Accept 校验失败")
	}
	// 101 后剩余字节仍在 br 缓冲中；resp.Body 不关（避免误关连接）。
	return &wsConn{conn: conn, br: br}, nil
}

// dialViaProxy 建立到目标地址的 HTTP CONNECT 隧道；支持 http/https 代理
// 与 Basic 代理认证。
func dialViaProxy(proxyRaw, targetAddr string, timeout time.Duration) (net.Conn, error) {
	pu, err := url.Parse(proxyRaw)
	if err != nil {
		return nil, fmt.Errorf("解析代理地址 %q: %w", proxyRaw, err)
	}
	if pu.Scheme != "http" && pu.Scheme != "https" {
		return nil, fmt.Errorf("代理地址必须是 http:// 或 https://，收到 %q", pu.Scheme)
	}
	port := pu.Port()
	if port == "" {
		if pu.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	pAddr := net.JoinHostPort(pu.Hostname(), port)

	conn, err := net.DialTimeout("tcp", pAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("连接代理 %s: %w", pAddr, err)
	}
	if pu.Scheme == "https" {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: pu.Hostname()})
		_ = conn.SetDeadline(time.Now().Add(timeout))
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("与代理 %s 的 TLS 握手: %w", pAddr, err)
		}
		_ = conn.SetDeadline(time.Time{})
		conn = tlsConn
	}

	var req strings.Builder
	fmt.Fprintf(&req, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n", targetAddr, targetAddr)
	if pu.User != nil {
		pass, _ := pu.User.Password()
		auth := base64.StdEncoding.EncodeToString([]byte(pu.User.Username() + ":" + pass))
		fmt.Fprintf(&req, "Proxy-Authorization: Basic %s\r\n", auth)
	}
	req.WriteString("\r\n")
	if _, err := conn.Write([]byte(req.String())); err != nil {
		conn.Close()
		return nil, fmt.Errorf("发送 CONNECT: %w", err)
	}

	// ReadResponse 可能预读到隧道数据，包一层 tunnelConn 防丢字节。
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("读取代理响应: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("代理拒绝 CONNECT（HTTP %d）: %s", resp.StatusCode, resp.Status)
	}
	return &tunnelConn{Conn: conn, br: br}, nil
}

// tunnelConn 把 bufio 预读的字节接回读路径，防丢帧。
type tunnelConn struct {
	net.Conn
	br *bufio.Reader
}

func (t *tunnelConn) Read(p []byte) (int, error) { return t.br.Read(p) }

// resolveAddr 补默认端口（wss→443、ws→80），返回拨号地址与 TLS SNI。
func resolveAddr(u *url.URL) (addr, serverName string) {
	port := u.Port()
	if port == "" {
		if u.Scheme == "wss" {
			port = "443"
		} else {
			port = "80"
		}
	}
	return net.JoinHostPort(u.Hostname(), port), u.Hostname()
}

// computeAccept 计算 Sec-WebSocket-Accept。
func computeAccept(key string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, key+wsGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// writeFrame 写一帧（客户端必掩码）。调用方负责写锁。
func (c *wsConn) writeFrame(opcode byte, payload []byte) error {
	n := len(payload)
	var headerLen int
	switch {
	case n < 126:
		headerLen = 2
	case n < 65536:
		headerLen = 4
	default:
		headerLen = 10
	}
	hdr := make([]byte, headerLen+4) // +4 掩码 key
	hdr[0] = 0x80 | opcode           // FIN=1，不分片
	switch {
	case n < 126:
		hdr[1] = 0x80 | byte(n)
	case n < 65536:
		hdr[1] = 0x80 | 126
		binary.BigEndian.PutUint16(hdr[2:4], uint16(n))
	default:
		hdr[1] = 0x80 | 127
		binary.BigEndian.PutUint64(hdr[2:10], uint64(n))
	}
	mask := hdr[headerLen : headerLen+4]
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	if _, err := c.conn.Write(hdr); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	masked := make([]byte, n)
	for i := range masked {
		masked[i] = payload[i] ^ mask[i%4]
	}
	_, err := c.conn.Write(masked)
	return err
}

// writeText 写一条文本消息（单帧）。
func (c *wsConn) writeText(data []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	return c.writeFrame(opText, data)
}

// writeControl 写控制帧（ping/pong/close；载荷 ≤125）。
func (c *wsConn) writeControl(opcode byte, payload []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	return c.writeFrame(opcode, payload)
}

// readFrame 读一帧：返回 FIN、opcode 与（已解掩码的）载荷。
func (c *wsConn) readFrame() (bool, byte, []byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(c.br, hdr[:]); err != nil {
		return false, 0, nil, err
	}
	fin := hdr[0]&0x80 != 0
	opcode := hdr[0] & 0x0f
	masked := hdr[1]&0x80 != 0
	length := uint64(hdr[1] & 0x7f)
	switch length {
	case 126:
		var ext [2]byte
		if _, err := io.ReadFull(c.br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err := io.ReadFull(c.br, ext[:]); err != nil {
			return false, 0, nil, err
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	if length > maxMessageSize {
		return false, 0, nil, fmt.Errorf("帧过大（%d > %d）", length, maxMessageSize)
	}

	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(c.br, mask[:]); err != nil {
			return false, 0, nil, err
		}
	}
	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(c.br, payload); err != nil {
			return false, 0, nil, err
		}
	}
	// 服务端帧不应掩码，防御性解掩码。
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return fin, opcode, payload, nil
}

// readMessage 读一条完整消息：处理控制帧并重组分片。
func (c *wsConn) readMessage() ([]byte, error) {
	var msgOpcode byte
	var buf []byte
	for {
		fin, op, payload, err := c.readFrame()
		if err != nil {
			return nil, err
		}
		switch op {
		case opPing:
			if err := c.writeControl(opPong, payload); err != nil {
				return nil, err
			}
			continue
		case opPong:
			continue
		case opClose:
			// 尽力回 close，随后视为连接关闭。
			_ = c.writeControl(opClose, payload)
			return nil, io.EOF
		case opText, opBinary:
			if msgOpcode != 0 {
				return nil, errors.New("协议错误：连续两个首帧")
			}
			msgOpcode = op
			buf = append(buf, payload...)
		case opContinuation:
			if msgOpcode == 0 {
				return nil, errors.New("协议错误：无首帧的 continuation")
			}
			buf = append(buf, payload...)
		default:
			return nil, fmt.Errorf("未知 opcode: %d", op)
		}
		if len(buf) > maxMessageSize {
			return nil, fmt.Errorf("消息超过 %d 字节", maxMessageSize)
		}
		if fin {
			return buf, nil
		}
	}
}

// setReadDeadline 设置读超时（每次读前刷新）。
func (c *wsConn) setReadDeadline(t time.Time) error { return c.conn.SetReadDeadline(t) }

// close 尽力完成关闭握手并关闭底层连接。
func (c *wsConn) close() error {
	_ = c.writeControl(opClose, []byte{0x03, 0xE8}) // 1000 normal closure
	return c.conn.Close()
}
