package auth

import (
	"strings"
	"testing"
)

// TestHashPasswordVerify 密码哈希往返：正确密码通过、错误密码拒绝、PHC 自描述。
func TestHashPasswordVerify(t *testing.T) {
	phc, err := HashPassword("correct horse battery staple 123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(phc, "$argon2id$v=19$") {
		t.Fatalf("PHC 应带 argon2id 版本标记: %s", phc)
	}
	// 正确密码
	if ok, err := VerifyPassword(phc, "correct horse battery staple 123"); err != nil || !ok {
		t.Fatalf("正确密码应通过: ok=%v err=%v", ok, err)
	}
	// 错误密码
	if ok, _ := VerifyPassword(phc, "wrong password 456"); ok {
		t.Fatal("错误密码不应通过")
	}
	// 非法 PHC
	if _, err := VerifyPassword("not-a-phc", "x"); err == nil {
		t.Fatal("非法 PHC 应报错")
	}
}

// TestVerifyPasswordCost 校验 cost 上限：防御恶意超参 PHC（DoS 面）。
func TestVerifyPasswordCost(t *testing.T) {
	// 构造一个超大 m 的 PHC（攻击者可能注入高 cost 参数拖垮校验）
	huge := "$argon2id$v=19$m=4294967295,t=999999,p=32$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	if _, err := VerifyPassword(huge, "x"); err == nil {
		t.Fatal("超高 cost 参数应被拒绝（DoS 防护）")
	}
}

// TestNeedsRehash 参数基线比对：低于基线 → true。
func TestNeedsRehash(t *testing.T) {
	phc, err := HashPassword("password 12345678")
	if err != nil {
		t.Fatal(err)
	}
	if NeedsRehash(phc) {
		t.Fatal("基线参数不应判定为需重哈希")
	}
	// 手工构造低参数 PHC（m 较小）
	weak := "$argon2id$v=19$m=1024,t=1,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	if !NeedsRehash(weak) {
		t.Fatal("低参数 Hash 应判定为需重哈希")
	}
	if NeedsRehash("garbage") == false {
		t.Fatal("无法解析的 PHC 应判定为需重哈希")
	}
}

// TestSessionTokenOnlyHash 会话 Token 只在响应中出现一次：库存 hash，无法反推 raw。
func TestSessionTokenOnlyHash(t *testing.T) {
	raw, hash, err := NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || hash == "" {
		t.Fatal("raw/hash 均不应为空")
	}
	if hash == raw {
		t.Fatal("库存 hash 不应等于 raw（只存摘要）")
	}
	if TokenHash(raw) != hash {
		t.Fatal("TokenHash(raw) 应等于签发时 hash")
	}
	// raw 长度 ≥ 256bit 随机（base64url 32 字节 → 43 字符）
	if len(raw) < 40 {
		t.Fatalf("Token 熵不足: len=%d", len(raw))
	}
}

// TestTokenHashDeterministic 同一 raw 的 hash 恒定。
func TestTokenHashDeterministic(t *testing.T) {
	a := TokenHash("some-token")
	b := TokenHash("some-token")
	if a != b {
		t.Fatal("TokenHash 应确定")
	}
}
