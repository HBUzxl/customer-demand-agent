// Package auth 实现密码 Hash、登录会话 Token、Cookie、CSRF 和认证中间件。
//
// 安全基线（设计文档 §7）：
//   - 密码用 Argon2id + 每用户随机 Salt，PHC 格式自描述，便于未来升级参数。
//   - 登录 Token ≥256 bit 随机，数据库只存 SHA-256 Hash。
//   - Cookie 只存随机会话 Token，不存用户资料/角色/租户 ID。
//   - 所有修改类请求校验 CSRF，并校验 Origin/Referer；SameSite 只是纵深防护。
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id 参数基线（部署机实测约 200~500ms 为目标，见 design doc §7.4）。
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MiB
	argonThreads = 2
	argonKeyLen  = 32
)

// 校验时 cost 上限（防 DoS：攻击者注入超大 m/t/p 的 PHC 拖垮校验）。
const (
	argonMemoryMax  = 1 << 20 // ≤ 1 GiB
	argonTimeMax    = 32
	argonThreadsMax = 16
)

// HashPassword 用 Argon2id + 随机 Salt 生成 PHC 格式密码 Hash。
// 格式：$argon2id$v=19$m=65536,t=3,p=2$<b64 salt>$<b64 key>
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("生成 salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword 常量时间校验密码与 PHC Hash 是否匹配。
func VerifyPassword(phc, password string) (bool, error) {
	params, salt, want, err := parsePHC(phc)
	if err != nil {
		return false, err
	}
	// cost 上限（防 DoS）：超限直接拒绝，不执行昂贵的哈希。
	if params.m > argonMemoryMax || params.t > argonTimeMax || params.p > argonThreadsMax {
		return false, fmt.Errorf("PHC 参数超限")
	}
	got := argon2.IDKey([]byte(password), salt, params.t, params.m, uint8(params.p), uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// NeedsRehash 判断当前 Hash 参数是否低于基线（登录成功后应重新 Hash）。
func NeedsRehash(phc string) bool {
	params, _, _, err := parsePHC(phc)
	if err != nil {
		return true
	}
	return params.m != argonMemory || params.t != argonTime || params.p != argonThreads
}

type argonParams struct {
	m, t, p uint32
}

// parsePHC 解析 $argon2id$v=19$m=65536,t=3,p=2$salt$hash。
func parsePHC(phc string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(phc, "$")
	// ["", "argon2id", "v=19", "m=...,t=...,p=...", "salt", "key"]
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return argonParams{}, nil, nil, fmt.Errorf("无效的 PHC 格式")
	}
	var p argonParams
	for _, kv := range strings.Split(parts[3], ",") {
		kv = strings.TrimSpace(kv)
		var v uint32
		if _, err := fmt.Sscanf(kv, "m=%d", &v); err == nil {
			p.m = v
		} else if _, err := fmt.Sscanf(kv, "t=%d", &v); err == nil {
			p.t = v
		} else if _, err := fmt.Sscanf(kv, "p=%d", &v); err == nil {
			p.p = v
		}
	}
	if p.m == 0 || p.t == 0 || p.p == 0 {
		return argonParams{}, nil, nil, fmt.Errorf("PHC 参数缺失")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, fmt.Errorf("解码 salt: %w", err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argonParams{}, nil, nil, fmt.Errorf("解码 hash: %w", err)
	}
	return p, salt, key, nil
}

// NewSessionToken 生成 ≥256 bit 随机登录 Token，返回明文（只进 Cookie）与
// SHA-256 Hash（只入库）。
func NewSessionToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("生成 token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256sum(raw)
	return raw, hex.EncodeToString(sum), nil
}

// TokenHash 计算 Token 的 SHA-256（查库用）。
func TokenHash(raw string) string {
	return hex.EncodeToString(sha256sum(raw))
}

// NewRandomHex 生成任意长度的随机 hex 串（CSRF raw、trace_id 等）。
func NewRandomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func sha256sum(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
