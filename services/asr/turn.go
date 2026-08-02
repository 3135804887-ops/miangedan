package asr

import (
	"fmt"
	"time"
)

// TurnDetector 为回合检测：语音结束后经静音窗口输出最终文本（NFR-010：断点→final ≤1s）。
type TurnDetector struct {
	silenceMs  int
	lastSpeech time.Time
	hasSpeech  bool
	finalAt    time.Time
	now        func() time.Time
}

// NewTurnDetector 创建回合检测器。
func NewTurnDetector(silenceMs int, now func() time.Time) (*TurnDetector, error) {
	if silenceMs <= 0 {
		return nil, ErrInvalidConfig
	}
	if now == nil {
		now = time.Now
	}
	return &TurnDetector{silenceMs: silenceMs, now: now}, nil
}

// Feed 逐帧喂入；语音帧刷新静音窗口，静音达到窗口即判定回合结束。
func (d *TurnDetector) Feed(f AudioFrame) (final bool, latency time.Duration) {
	if f.Speech {
		d.hasSpeech = true
		d.lastSpeech = d.now()
		return false, 0
	}
	if !d.hasSpeech {
		return false, 0
	}
	elapsed := d.now().Sub(d.lastSpeech)
	if elapsed >= time.Duration(d.silenceMs)*time.Millisecond {
		d.finalAt = d.now()
		return true, d.finalAt.Sub(d.lastSpeech)
	}
	return false, 0
}

// CheckBudget 校验断点→final 时延在 NFR-010 预算内。
func CheckBudget(latency time.Duration) error {
	if latency > ASRFinalBudget {
		return fmt.Errorf("%w: 断点→final %s 超过 %s", ErrBudgetExceeded, latency, ASRFinalBudget)
	}
	return nil
}

// Speaker 为说话方。
type Speaker string

// 说话方。
const (
	SpeakerUser   Speaker = "user"
	SpeakerAvatar Speaker = "avatar"
)

// TurnGate 为单说话方闸门（FR-017：避免重叠说话；打断后数字人停止发声预算 NFR-009）。
type TurnGate struct {
	speaking   Speaker
	stopAt     time.Time
	stopBudget time.Duration
	now        func() time.Time
}

// NewTurnGate 创建闸门。
func NewTurnGate(stopBudget time.Duration, now func() time.Time) *TurnGate {
	if stopBudget <= 0 {
		stopBudget = StopLatencyBudget
	}
	if now == nil {
		now = time.Now
	}
	return &TurnGate{stopBudget: stopBudget, now: now}
}

// CanStart 判断说话方是否可开始（数字人不得在用户说话/停止确认期间输出，避免重叠）。
func (g *TurnGate) CanStart(who Speaker) bool {
	if g.speaking == who {
		return false
	}
	if who == SpeakerAvatar && g.speaking == SpeakerUser {
		return false
	}
	return true
}

// Start 开始说话。
func (g *TurnGate) Start(who Speaker) error {
	if !g.CanStart(who) {
		return ErrOverlap
	}
	g.speaking = who
	return nil
}

// Interrupt 打断数字人（语音 VAD 或停止按钮）；记录停止时刻用于 NFR-009 预算校验。
func (g *TurnGate) Interrupt() {
	g.stopAt = g.now()
	g.speaking = SpeakerUser
}

// StopConfirmed 数字人确认停止；校验打断→停止时延预算（NFR-009：P95 ≤500ms）。
func (g *TurnGate) StopConfirmed() error {
	if g.stopAt.IsZero() {
		return nil
	}
	latency := g.now().Sub(g.stopAt)
	if latency > g.stopBudget {
		return fmt.Errorf("%w: 打断→停止 %s 超过 %s", ErrBudgetExceeded, latency, g.stopBudget)
	}
	g.stopAt = time.Time{}
	return nil
}

// Speaking 返回当前说话方。
func (g *TurnGate) Speaking() Speaker {
	return g.speaking
}
