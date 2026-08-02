// Package room 提供实时会话房间、短期媒体令牌与会话单活动设备绑定（TASK-020）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-020；FR-013、NFR-007；SEC-003；SEC-008；
// docs/domain/INTERVIEW-STATE-MACHINE.md 6.2（Session 状态）；docs/api/realtime-events.md。
package room

import (
	"errors"
	"time"
)

// Status 为会话房间状态（INTERVIEW-STATE-MACHINE.md 6.2，禁止自创状态）。
type Status string

// 房间状态全集。
const (
	StatusRoomCreated       Status = "ROOM_CREATED"
	StatusPreCheck          Status = "PRE_CHECK"
	StatusAvatarConnecting  Status = "AVATAR_CONNECTING"
	StatusLive              Status = "LIVE"
	StatusPausedSystem      Status = "PAUSED_SYSTEM"
	StatusReconnecting      Status = "RECONNECTING"
	StatusDowngradePrompted Status = "DOWNGRADE_PROMPTED"
	StatusTextDegraded      Status = "TEXT_DEGRADED"
	StatusAuthPaused        Status = "AUTH_PAUSED"
	StatusEnded             Status = "ENDED"
)

// SessionKind 为会话类型（openapi createSession.kind）。
type SessionKind string

// 会话类型。
const (
	KindFormal      SessionKind = "formal"
	KindFormalRetry SessionKind = "formal_retry"
)

// Session 为实时会话（DOMAIN-MODEL 6.10；DATA-MODEL sessions）。
type Session struct {
	SessionID       string
	ProjectID       string
	UserID          string
	DataRegion      string
	RoundSequence   int
	AttemptID       string
	Kind            SessionKind
	RoomStatus      Status
	RoomProviderRef string
	ActiveDeviceID  string
	// TASK-025 故障控制：暂停计时、重连截止、降级状态、结束原因。
	PausedAt          *time.Time
	PausedSeconds     int
	DowngradeStatus   DowngradeStatus
	DowngradePromptID string
	TextDegradedAt    *time.Time
	EndReason         EndReason
	EndedAt           *time.Time
	LastActivityAt    time.Time
	BillableSeconds   int
	CreatedAt         time.Time
}

// SessionCreated 为建连响应（openapi SessionCreated）。
type SessionCreated struct {
	SessionID          string
	RoomURL            string
	RoomToken          string
	RoomTokenExpiresAt time.Time
	DataRegion         string
}

// CreateSessionInput 为创建会话入参。
type CreateSessionInput struct {
	ProjectID     string
	RoundSequence int
	Kind          SessionKind
	AttemptID     string
	DeviceID      string
}

// 服务错误（httpapi 映射开放 API 错误码）。
var (
	ErrNotFound           = errors.New("session not found")
	ErrStateConflict      = errors.New("session state conflict")
	ErrInvalidInput       = errors.New("invalid input")
	ErrReconnectExpired   = errors.New("reconnect window expired")
	ErrTokenInvalid       = errors.New("media token invalid")
	ErrTokenUsed          = errors.New("media token already used")
	ErrTokenRevoked       = errors.New("media token revoked")
	ErrEntitlementMissing = errors.New("insufficient entitlement")
)

// TokenTTLDefault 为短期房间令牌默认有效期（分钟级，SEC-003）。
const TokenTTLDefault = 5 * time.Minute

// ReconnectWindow 为 3 分钟重连窗口（INTERVIEW-STATE-MACHINE 6.2）。
const ReconnectWindow = 3 * time.Minute
