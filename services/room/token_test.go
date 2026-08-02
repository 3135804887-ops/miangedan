package room

import (
	"errors"
	"testing"
	"time"
)

func newTokenManager(t *testing.T, store MediaTokenStore) *MediaTokenManager {
	t.Helper()
	m, err := NewMediaTokenManager(TokenConfig{SigningKey: "synthetic-media-signing-key-0123456789abcdef", TTL: TokenTTLDefault}, store)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// 正常路径：签发后可校验一次；重复使用同一令牌被拒（一次性，SEC-003）。
func TestMediaTokenOneTime(t *testing.T) {
	store := NewMemoryStore()
	m := newTokenManager(t, store)
	now := time.Now()
	token, exp, err := m.Issue("s1", "device-a", "cn", now)
	if err != nil || !exp.After(now) {
		t.Fatalf("签发失败: %v", err)
	}
	claims, err := m.Verify(token, now.Add(time.Minute))
	if err != nil || claims.SessionID != "s1" || claims.DeviceID != "device-a" {
		t.Fatalf("首次校验应通过: %v %+v", err, claims)
	}
	if _, err := m.Verify(token, now.Add(time.Minute)); !errors.Is(err, ErrTokenUsed) {
		t.Fatalf("重复使用必须拒绝，实际 %v", err)
	}
}

// 异常路径：篡改签名/过期/吊销必须拒绝。
func TestMediaTokenRejected(t *testing.T) {
	store := NewMemoryStore()
	m := newTokenManager(t, store)
	now := time.Now()
	token, _, err := m.Issue("s1", "device-a", "cn", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Verify(token+"x", now.Add(time.Minute)); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("篡改令牌必须拒绝，实际 %v", err)
	}
	// 过期。
	if _, err := m.Verify(token, now.Add(TokenTTLDefault+time.Minute)); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("过期令牌必须拒绝，实际 %v", err)
	}
	// 吊销。
	token2, _, err := m.Issue("s2", "device-a", "cn", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeSession("s2"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Verify(token2, now.Add(time.Minute)); !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("已吊销令牌必须拒绝，实际 %v", err)
	}
}

// 异常路径：签名密钥过短必须拒绝（与业务令牌隔离，SEC-003）。
func TestTokenConfigRejected(t *testing.T) {
	if _, err := NewMediaTokenManager(TokenConfig{SigningKey: "short"}, NewMemoryStore()); err == nil {
		t.Fatal("短密钥必须拒绝")
	}
}
