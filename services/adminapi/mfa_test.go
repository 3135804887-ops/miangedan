// Package adminapi 追加式审计与抗钓鱼 MFA 测试（TASK-084；FR-037/040）。
package adminapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

const testPublicKey = "device-public-key-0123456789abcdef"

func mfaSignature(t *testing.T, nonce string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(testPublicKey))
	_, _ = mac.Write([]byte(nonce))
	return hex.EncodeToString(mac.Sum(nil))
}

// 抗钓鱼 MFA：挑战一次性、5 分钟有效；错误签名拒绝。
func TestMFAChallengeVerifyAndReplay(t *testing.T) {
	svc, _ := newTestService(t)
	device, err := svc.RegisterMFADevice(context.Background(), securityActor, "yubikey", testPublicKey)
	if err != nil {
		t.Fatalf("登记设备失败: %v", err)
	}
	challenge, err := svc.CreateMFAChallenge(context.Background(), securityActor)
	if err != nil {
		t.Fatalf("创建挑战失败: %v", err)
	}
	if _, err := svc.VerifyMFA(context.Background(), securityActor,
		challenge.ChallengeID, device.DeviceID, "badsig"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("错误签名应拒绝，实际 err=%v", err)
	}
	verification, err := svc.VerifyMFA(context.Background(), securityActor,
		challenge.ChallengeID, device.DeviceID, mfaSignature(t, challenge.Nonce))
	if err != nil {
		t.Fatalf("验证失败: %v", err)
	}
	if !verification.ExpiresAt.After(verification.VerifiedAt) {
		t.Fatalf("验证窗口异常：%+v", verification)
	}
	// 挑战不可重放。
	if _, err := svc.VerifyMFA(context.Background(), securityActor,
		challenge.ChallengeID, device.DeviceID, mfaSignature(t, challenge.Nonce)); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("挑战重放应被拒绝，实际 err=%v", err)
	}
	// 挑战过期拒绝。
	expired, _ := svc.CreateMFAChallenge(context.Background(), securityActor)
	svc.now = func() time.Time { return time.Now().UTC().Add(6 * time.Minute) }
	if _, err := svc.VerifyMFA(context.Background(), securityActor,
		expired.ChallengeID, device.DeviceID, mfaSignature(t, expired.Nonce)); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("过期挑战应拒绝，实际 err=%v", err)
	}
}

// 高风险操作重新验证：15 分钟窗口外拒绝。
func TestHighRiskRecheck(t *testing.T) {
	svc, _ := newTestService(t)
	if err := svc.RequireHighRiskMFA(context.Background(), securityActor,
		"break_glass.open"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("无验证应拒绝高风险操作，实际 err=%v", err)
	}
	device, _ := svc.RegisterMFADevice(context.Background(), securityActor, "yubikey", testPublicKey)
	challenge, _ := svc.CreateMFAChallenge(context.Background(), securityActor)
	if _, err := svc.VerifyMFA(context.Background(), securityActor,
		challenge.ChallengeID, device.DeviceID, mfaSignature(t, challenge.Nonce)); err != nil {
		t.Fatalf("验证失败: %v", err)
	}
	if err := svc.RequireHighRiskMFA(context.Background(), securityActor, "rubric.deprecate"); err != nil {
		t.Fatalf("窗口内应允许高风险操作，实际 err=%v", err)
	}
	svc.now = func() time.Time { return time.Now().UTC().Add(16 * time.Minute) }
	if err := svc.RequireHighRiskMFA(context.Background(), securityActor,
		"rubric.deprecate"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("超窗应重新验证，实际 err=%v", err)
	}
}

// 审计日志：追加式分页查询；管理员不可删除（存储仅 SELECT/INSERT）。
func TestAuditAppendOnlyPaged(t *testing.T) {
	svc, _ := newTestService(t)
	for i := 0; i < 5; i++ {
		_ = svc.AttemptScoreWrite(context.Background(), securityActor, "edit_score", "p-1")
	}
	audits, err := svc.ListAuditLogs(context.Background(), securityActor, "cn", 2, 0)
	if err != nil {
		t.Fatalf("分页查询失败: %v", err)
	}
	if len(audits) != 2 {
		t.Fatalf("第一页应有 2 条，实际 %d", len(audits))
	}
	// 存储接口无审计修改路径（追加式：仅 Append/List）。
	storeType := reflect.TypeOf((*Store)(nil)).Elem()
	for i := 0; i < storeType.NumMethod(); i++ {
		name := storeType.Method(i).Name
		if (strings.HasPrefix(name, "Audit") || name == "ListAuditsPaged") &&
			!strings.HasPrefix(name, "AppendAudit") &&
			!strings.HasPrefix(name, "ListAudit") {
			t.Fatalf("审计存储存在修改路径: %s", name)
		}
	}
}
