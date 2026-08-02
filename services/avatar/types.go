// Package avatar 提供数字人驱动接入（TASK-021，EPIC-03）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-021；FR-013、FR-014；NFR-007、NFR-011、NFR-012；
// docs/ai/PROVIDER-ADAPTERS.md §4.4；SECURITY-REQUIREMENTS SEC-014。
package avatar

import (
	"context"
	"errors"
	"time"
)

// LipSyncBudgetMs 为口型与音频偏差上限（NFR-011：≤200ms）。
const LipSyncBudgetMs = 200

// DefaultVideoProfile 为默认数字人视频档位（NFR-012：≥720p、24fps）。
var DefaultVideoProfile = VideoProfile{Width: 1280, Height: 720, FPS: 24, AudioContinuous: false}

// VideoProfile 为分辨率/帧率档位（NFR-012；弱网优先音频连续）。
type VideoProfile struct {
	Width           int
	Height          int
	FPS             int
	AudioContinuous bool
}

// Character 为固定授权写实 2D 角色库条目（FR-014；禁止每场生成新脸，§4.4 约束）。
type Character struct {
	ID          string
	LicenseRef  string
	ProfileHash string
}

// Persona 为动态面试官人格（interview-flows style_parameters；有界、与候选人保护属性无关）。
type Persona struct {
	Tone                      string
	Pace                      string
	FollowupIntensity         string
	PoliteInterruptionAllowed bool
	HintLevel                 string
	PressureLevel             string
}

// StartInput 为启动数字人会话入参。
type StartInput struct {
	CharacterID  string
	Persona      Persona
	VideoProfile VideoProfile
}

// LipSyncReport 为口型同步结果（NFR-011）。
type LipSyncReport struct {
	MaxDeviationMs int
	CheckedAt      time.Time
}

// DriverSession 为单场数字人会话（实时音视频；打断可立即停驱）。
type DriverSession interface {
	Drive(ctx context.Context, audioChunks []AudioChunk) (LipSyncReport, error)
	Stop(ctx context.Context) error
}

// AudioChunk 为 TTS 音频分片（与服务端时间戳对齐；不含用户内容持久化）。
type AudioChunk struct {
	Seq      int
	Duration time.Duration
}

// 服务错误。
var (
	ErrCharacterNotFound   = errors.New("character not found in licensed library")
	ErrInvalidPersona      = errors.New("invalid persona")
	ErrLipSyncOverBudget   = errors.New("lip sync deviation exceeds budget")
	ErrInvalidVideoProfile = errors.New("invalid video profile")
)
