// Package adminapi 提供追加式审计日志与抗钓鱼 MFA（TASK-084；FR-037/FR-040，
// US-08 规则 12；SCREEN-SPEC SCR-17）。
// 红线：管理员不可删除审计（存储只 SELECT/INSERT）；抗钓鱼 MFA（挑战绑定设备）；
// 高风险操作重新验证；审计写入无更新/删除路径。
package adminapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// MFA 有效期常量。
const (
	ChallengeTTL       = 5 * time.Minute
	VerificationWindow = 15 * time.Minute
)

// MFADevice 为抗钓鱼 MFA 设备（公钥绑定；不可跨区）。
type MFADevice struct {
	DeviceID     string
	StaffID      string
	Name         string
	PublicKey    string
	DataRegion   string
	RegisteredAt time.Time
	RevokedAt    *time.Time
}

// MFAChallenge 为一次挑战（随机 nonce；5 分钟有效；一次使用）。
type MFAChallenge struct {
	ChallengeID string
	StaffID     string
	Nonce       string
	DataRegion  string
	ExpiresAt   time.Time
	UsedAt      *time.Time
}

// MFAVerification 为一次成功验证（15 分钟高风险操作窗口）。
type MFAVerification struct {
	VerificationID string
	StaffID        string
	ChallengeID    string
	DeviceID       string
	DataRegion     string
	VerifiedAt     time.Time
	ExpiresAt      time.Time
}

// RegisterMFADevice 登记抗钓鱼设备（WebAuthn 适配点：公钥绑定员工）。
func (s *Service) RegisterMFADevice(
	_ context.Context, actor Actor, name, publicKey string,
) (MFADevice, error) {
	if err := validateActor(actor); err != nil {
		return MFADevice{}, err
	}
	if strings.TrimSpace(name) == "" || len(publicKey) < 16 {
		return MFADevice{}, fmt.Errorf("%w: 设备名与公钥必填（公钥至少 16 字节）", ErrInvalidInput)
	}
	device := MFADevice{
		DeviceID:     newID(),
		StaffID:      actor.StaffID,
		Name:         name,
		PublicKey:    publicKey,
		DataRegion:   actor.DataRegion,
		RegisteredAt: s.now().UTC(),
	}
	if err := s.store.SaveMFADevice(device); err != nil {
		return MFADevice{}, err
	}
	_ = s.appendAudit(actor, "mfa.device_registered", device.DeviceID)
	return device, nil
}

// CreateMFAChallenge 创建挑战（随机 nonce；5 分钟有效；抗钓鱼：挑战不可重放）。
func (s *Service) CreateMFAChallenge(
	_ context.Context, actor Actor,
) (MFAChallenge, error) {
	if err := validateActor(actor); err != nil {
		return MFAChallenge{}, err
	}
	challenge := MFAChallenge{
		ChallengeID: newID(),
		StaffID:     actor.StaffID,
		Nonce:       newID() + newID(),
		DataRegion:  actor.DataRegion,
		ExpiresAt:   s.now().UTC().Add(ChallengeTTL),
	}
	if err := s.store.SaveMFAChallenge(challenge); err != nil {
		return MFAChallenge{}, err
	}
	return challenge, nil
}

// VerifyMFA 验证挑战签名（HMAC-SHA256(publicKey, nonce)）；成功后 15 分钟窗口。
func (s *Service) VerifyMFA(
	_ context.Context, actor Actor, challengeID, deviceID, signature string,
) (MFAVerification, error) {
	if err := validateActor(actor); err != nil {
		return MFAVerification{}, err
	}
	challenge, err := s.store.GetMFAChallenge(actor.DataRegion, challengeID)
	if err != nil {
		return MFAVerification{}, err
	}
	if challenge.StaffID != actor.StaffID {
		return MFAVerification{}, ErrNotFound
	}
	if challenge.UsedAt != nil {
		return MFAVerification{}, fmt.Errorf("%w: 挑战已使用（抗重放）", ErrStateConflict)
	}
	if s.now().UTC().After(challenge.ExpiresAt) {
		return MFAVerification{}, fmt.Errorf("%w: 挑战已过期", ErrStateConflict)
	}
	device, err := s.store.GetMFADevice(actor.DataRegion, deviceID)
	if err != nil {
		return MFAVerification{}, err
	}
	if device.StaffID != actor.StaffID || device.RevokedAt != nil {
		return MFAVerification{}, ErrNotFound
	}
	mac := hmac.New(sha256.New, []byte(device.PublicKey))
	_, _ = mac.Write([]byte(challenge.Nonce))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(strings.ToLower(signature))) {
		_ = s.appendAudit(actor, "mfa.verify_failed", challengeID)
		return MFAVerification{}, fmt.Errorf("%w: 签名不匹配", ErrForbidden)
	}
	now := s.now().UTC()
	challenge.UsedAt = &now
	if err := s.store.UpdateMFAChallenge(challenge); err != nil {
		return MFAVerification{}, err
	}
	verification := MFAVerification{
		VerificationID: newID(),
		StaffID:        actor.StaffID,
		ChallengeID:    challengeID,
		DeviceID:       deviceID,
		DataRegion:     actor.DataRegion,
		VerifiedAt:     now,
		ExpiresAt:      now.Add(VerificationWindow),
	}
	if err := s.store.SaveMFAVerification(verification); err != nil {
		return MFAVerification{}, err
	}
	_ = s.appendAudit(actor, "mfa.verified", challengeID)
	return verification, nil
}

// RequireHighRiskMFA 高风险操作再验证：15 分钟内无有效验证则拒绝。
func (s *Service) RequireHighRiskMFA(
	_ context.Context, actor Actor, action string,
) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	latest, err := s.store.GetLatestMFAVerification(actor.DataRegion, actor.StaffID)
	if err != nil || s.now().UTC().After(latest.ExpiresAt) {
		_ = s.appendAudit(actor, "mfa.high_risk_blocked", action)
		return fmt.Errorf("%w: 高风险操作需重新 MFA 验证（15 分钟窗口）", ErrForbidden)
	}
	_ = s.appendAudit(actor, "mfa.high_risk_allowed", action)
	return nil
}

// ListAuditLogs 追加式审计分页查询（管理员不可删除；默认脱敏）。
func (s *Service) ListAuditLogs(
	_ context.Context, actor Actor, dataRegion string, limit, offset int,
) ([]AuditEntry, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 || offset < 0 {
		return nil, fmt.Errorf("%w: limit 1-100，offset ≥0", ErrInvalidInput)
	}
	return s.store.ListAuditsPaged(dataRegion, limit, offset)
}
