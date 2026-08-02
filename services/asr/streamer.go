package asr

import (
	"context"
	"fmt"
	"time"
)

// Stream 为 ASR 双向流（§4.2：发送音频帧，接收 partial/final 事件）。
type Stream interface {
	SendAudio(AudioFrame) error
	Recv() (Event, error)
	Close() error
}

// Provider 为 ASR 供应商适配层契约（供应商中立；真实接入随选型）。
type Provider interface {
	OpenStream(context.Context, StreamConfig) (Stream, error)
}

// StubProvider 为合成 ASR（开发/测试）：按静音断点输出 partial → final。
type StubProvider struct{}

// OpenStream 返回合成识别流。
func (StubProvider) OpenStream(_ context.Context, cfg StreamConfig) (Stream, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	return &stubStream{cfg: cfg}, nil
}

type stubStream struct {
	cfg    StreamConfig
	seq    int
	closed bool
}

func (s *stubStream) SendAudio(_ AudioFrame) error {
	if s.closed {
		return fmt.Errorf("stream closed")
	}
	s.seq++
	return nil
}

func (s *stubStream) Recv() (Event, error) {
	if s.closed {
		return Event{}, fmt.Errorf("stream closed")
	}
	if s.seq == 0 {
		return Event{}, fmt.Errorf("no audio yet")
	}
	return Event{Kind: EventPartial, UtteranceID: "u-1", TextDelta: "合成", Confidence: 0.9, Language: s.cfg.Language, At: time.Now()}, nil
}

func (s *stubStream) Close() error {
	s.closed = true
	return nil
}

// ValidateConfig 校验识别配置（语言必选；静音断点必须为正）。
func ValidateConfig(cfg StreamConfig) error {
	if cfg.Language != "zh-CN" && cfg.Language != "en-US" {
		return fmt.Errorf("%w: 语言必须为 zh-CN | en-US", ErrInvalidConfig)
	}
	if cfg.SilenceEndpointMs <= 0 {
		return fmt.Errorf("%w: 静音断点必须为正", ErrInvalidConfig)
	}
	return nil
}
