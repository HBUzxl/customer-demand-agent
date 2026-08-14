package http

import (
	"net"
	"net/http"
	"testing"
)

// TestIsForbiddenIP 验证探测端点的 IP 段判定。
// 禁止段：回环、link-local（含云元数据 169.254.169.254）、unspecified、multicast。
// 放行：公网 + 私网（10/172.16/192.168，支持内部网关测试）。
func TestIsForbiddenIP(t *testing.T) {
	forbidden := []string{
		"169.254.169.254", // 云元数据
		"169.254.0.1",     // link-local
		"127.0.0.1",       // IPv4 回环
		"::1",             // IPv6 回环
		"0.0.0.0",         // unspecified
		"::",              // unspecified v6
		"224.0.0.1",       // multicast
		"fe80::1",         // link-local v6
	}
	for _, s := range forbidden {
		ip := net.ParseIP(s)
		if ip == nil {
			t.Fatalf("无法解析 %s", s)
		}
		if !isForbiddenIP(ip) {
			t.Errorf("isForbiddenIP(%s) = false, want true（应禁止）", s)
		}
	}

	allowed := []string{
		"8.8.8.8",     // 公网
		"1.1.1.1",     // 公网
		"10.0.0.5",    // 私网（内部网关场景，放行）
		"192.168.1.1", // 私网
		"172.16.0.1",  // 私网
		"114.114.114.114",
	}
	for _, s := range allowed {
		ip := net.ParseIP(s)
		if ip == nil {
			t.Fatalf("无法解析 %s", s)
		}
		if isForbiddenIP(ip) {
			t.Errorf("isForbiddenIP(%s) = true, want false（公网/私网应放行）", s)
		}
	}
}

// TestValidateProbeEndpoint 验证探测端点 URL 校验。
func TestValidateProbeEndpoint(t *testing.T) {
	bad := []string{
		"http://169.254.169.254/latest/meta-data/", // 云元数据（字面 IP）
		"http://127.0.0.1:8080/",                   // 回环（字面 IP）
		"http://[::1]:8080/",                       // IPv6 回环
		"ftp://example.com/",                       // 非 http(s) scheme
		"http://",                                  // 无 host
		"://bad",                                   // 解析失败
	}
	for _, raw := range bad {
		if err := validateProbeEndpoint(raw); err == nil {
			t.Errorf("validateProbeEndpoint(%q) 应报错，got nil", raw)
		}
	}

	ok := []string{
		"https://api.deepseek.com/v1",
		"http://10.0.0.5/v1", // 私网放行（内部网关）
		"http://192.168.1.10:8080",
		"http://example.com/models",
	}
	for _, raw := range ok {
		if err := validateProbeEndpoint(raw); err != nil {
			t.Errorf("validateProbeEndpoint(%q) 不应报错: %v", raw, err)
		}
	}
}

// TestProbeRedirectPolicy 验证跨源重定向凭据泄漏防护（审计 blocker）：
// 探测端点的凭据（尤其 anthropic x-api-key，Go 不会像 Authorization 自动剥）不得随重定向到他源。
// 只允许同源重定向；跨源（换主机/端口/降级）或指向受保护 IP 一律拒。
func TestProbeRedirectPolicy(t *testing.T) {
	orig, _ := http.NewRequest("GET", "https://api.deepseek.com/v1/models", nil)

	blocked := []struct {
		name, target string
	}{
		{"cross-origin host", "https://evil.example.com/x"},      // 攻击者公网主机（过 SSRF，但跨源）
		{"cross-origin port", "https://api.deepseek.com:9000/x"}, // 同主机不同端口
		{"https→http downgrade", "http://api.deepseek.com/x"},    // 降级
		{"forbidden IP target", "https://169.254.169.254/x"},     // 受保护 IP
	}
	for _, c := range blocked {
		t.Run("blocked/"+c.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", c.target, nil)
			if err := probeRedirectPolicy(req, []*http.Request{orig}); err == nil {
				t.Errorf("跨源重定向 %s 应被拒（凭据会泄漏），got nil", c.target)
			}
		})
	}

	// 同源重定向（路径不同）应允许
	same, _ := http.NewRequest("GET", "https://api.deepseek.com/v2/models", nil)
	if err := probeRedirectPolicy(same, []*http.Request{orig}); err != nil {
		t.Errorf("同源重定向不应被拒: %v", err)
	}
	// 无 via（首跳前）不应拒绝
	firstHop, _ := http.NewRequest("GET", "https://api.deepseek.com/v1/models", nil)
	if err := probeRedirectPolicy(firstHop, nil); err != nil {
		t.Errorf("无 via 不应拒绝: %v", err)
	}
	// 过多重定向（>=3）应拒
	tooMany := []*http.Request{orig, orig, orig}
	if err := probeRedirectPolicy(same, tooMany); err == nil {
		t.Error(">=3 次重定向应被拒")
	}
}

// TestSameEndpoint 验证凭据保护用的同源比较（scheme+host+port，默认端口归一化）。
// 不同端口、HTTPS→HTTP 降级、不同主机都应判为不同源（拒绝代填 key）。
func TestSameEndpoint(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"http://api.deepseek.com/v1", "http://api.deepseek.com/v2", true},      // 同源，路径不同
		{"http://API.DeepSeek.com", "http://api.deepseek.com", true},            // 大小写无关
		{"https://api.deepseek.com", "https://api.deepseek.com:443", true},      // https 默认 443 归一
		{"http://api.deepseek.com", "http://api.deepseek.com:80", true},         // http 默认 80 归一
		{"http://api.deepseek.com:8080", "http://api.deepseek.com:9000", false}, // 不同端口 → 不同源
		{"https://api.deepseek.com", "http://api.deepseek.com", false},          // HTTPS→HTTP 降级 → 不同源
		{"http://api.deepseek.com", "http://evil.example", false},               // 不同主机
		{"http://10.0.0.5", "http://10.0.0.6", false},                           // 不同 IP
		{"http://api.deepseek.com", "://bad", false},                            // 解析失败
	}
	for _, c := range cases {
		if got := sameEndpoint(c.a, c.b); got != c.want {
			t.Errorf("sameEndpoint(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
