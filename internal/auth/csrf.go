package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// CSRF 提供传统 Synchronizer Token 的生成/校验原语；Web 登录会话在此基础上
// 使用 DeriveCSRF，让同一会话的多标签页共享稳定 Token。
type CSRF struct{}

// NewCSRF 创建 CSRF 校验器。
func NewCSRF() *CSRF { return &CSRF{} }

// Generate 生成 (raw, hash)。raw 返回给客户端，hash 入库。
func (c *CSRF) Generate() (raw, hash string) {
	raw = NewRandomHex(32)
	return raw, TokenHash(raw)
}

// Verify 常量时间校验客户端 raw 与库存 hash 是否匹配。
func (c *CSRF) Verify(hash, raw string) bool {
	if hash == "" || raw == "" {
		return false
	}
	got := []byte(TokenHash(raw))
	want := []byte(hash)
	return len(got) == len(want) && subtle.ConstantTimeCompare(got, want) == 1
}

// DeriveCSRF 从 HttpOnly 会话 Token 和服务端会话 Secret 派生稳定的 CSRF
// Token。同一会话重复读取 /auth/me 得到相同值，避免多标签页相互挤掉；工作空间
// 切换时只需轮换 Secret 即可让旧值立即失效。数据库单独泄露 Secret 也无法在没有
// 会话 Cookie 的情况下伪造有效请求。
func DeriveCSRF(sessionToken, secret string) string {
	if sessionToken == "" || secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(sessionToken))
	_, _ = mac.Write([]byte("customer-demand-agent/csrf/v1\x00"))
	_, _ = mac.Write([]byte(secret))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyDerivedCSRF 常量时间校验派生 CSRF Token。
func VerifyDerivedCSRF(sessionToken, secret, raw string) bool {
	want := DeriveCSRF(sessionToken, secret)
	return want != "" && len(want) == len(raw) && subtle.ConstantTimeCompare([]byte(want), []byte(raw)) == 1
}
