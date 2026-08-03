// Package asr 提供流式语音识别、回合检测与打断/防重叠（TASK-022，EPIC-03）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-022；FR-017；NFR-008 ~ NFR-010；
// docs/ai/PROVIDER-ADAPTERS.md §4.2；docs/api/realtime-events.md（asr.partial/final、user.interrupt.*）。
package asr

import (
	"errors"
	"time"
)

// ASRFinalBudget 为 ASR 最终文本时延预算（NFR-010：P95 ≤1s）。
const ASRFinalBudget = time.Second

// StopLatencyBudget 为打断至数字人停止发声预算（NFR-009：P95 ≤500ms）。
const StopLatencyBudget = 500 * time.Millisecond

// EventKind 为 ASR 事件类型（realtime-events asr.partial / asr.final）。
type EventKind string

// 事件类型。
const (
	EventPartial EventKind = "partial"
	EventFinal   EventKind = "final"
)

// StreamConfig 为识别配置（§4.2：语言、口音/领域提示、标点与逆规范化开关）。
type StreamConfig struct {
	Language                 string
	AccentHint               string
	PunctuationEnabled       bool
	InverseTextNormalization bool
	SilenceEndpointMs        int
}

// Event 为 ASR 事件（partial 仅展示不入证据；final 为评分证据输入，NFR-010）。
type Event struct {
	Kind        EventKind
	UtteranceID string
	TextDelta   string
	FinalText   string
	Confidence  float64
	Language    string
	At          time.Time
}

// AudioFrame 为音频帧（WebRTC 音轨转码后输入）。
type AudioFrame struct {
	Seq      int
	Duration time.Duration
	RmsLevel float64
	Speech   bool
	PCM      []byte // 16k 单声道 PCM16 音频载荷（自建 ASR 回合级转写输入）
}

// 服务错误。
var (
	ErrInvalidConfig  = errors.New("invalid asr stream config")
	ErrBudgetExceeded = errors.New("asr latency budget exceeded")
	ErrOverlap        = errors.New("overlapping speech denied")
)
