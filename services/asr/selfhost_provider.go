package asr

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// SelfHostProvider 为自建 ASR 服务（mgd-selfhost）的回合级 Provider 实现
// （OD-01 自建矩阵；真实流式接入随媒体链路演进）。
type SelfHostProvider struct {
	client *SelfHostClient
}

// NewSelfHostProvider 构造自建 ASR Provider。
func NewSelfHostProvider(client *SelfHostClient) (*SelfHostProvider, error) {
	if client == nil {
		return nil, fmt.Errorf("%w: client 必填", ErrInvalidConfig)
	}
	return &SelfHostProvider{client: client}, nil
}

// OpenStream 打开回合级识别流：SendAudio 累积 PCM16 载荷，Close 时整段转写并返回 final 事件。
func (p *SelfHostProvider) OpenStream(ctx context.Context, cfg StreamConfig) (Stream, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	return &selfHostStream{
		ctx:    ctx,
		cfg:    cfg,
		client: p.client,
		events: make(chan Event, 1),
	}, nil
}

type selfHostStream struct {
	ctx    context.Context
	cfg    StreamConfig
	client *SelfHostClient

	mu     sync.Mutex
	closed bool
	pcm    []byte
	events chan Event
}

// SendAudio 累积 PCM16 音频载荷（不含载荷的元数据帧被忽略）。
func (s *selfHostStream) SendAudio(frame AudioFrame) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("stream closed")
	}
	if len(frame.PCM) > 0 {
		s.pcm = append(s.pcm, frame.PCM...)
	}
	return nil
}

// Recv 返回 Close 触发的 final 事件；流关闭后返回错误。
func (s *selfHostStream) Recv() (Event, error) {
	event, ok := <-s.events
	if !ok {
		return Event{}, errors.New("stream closed")
	}
	return event, nil
}

// Close 将累积音频组装为 16k 单声道 WAV 并提交自建 ASR 服务转写。
func (s *selfHostStream) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	pcm := s.pcm
	s.mu.Unlock()

	wav, err := buildWAV16kMono(pcm)
	if err != nil {
		close(s.events)
		return err
	}
	text, err := s.client.TranscribeWAV(s.ctx, wav, s.cfg.Language)
	if err != nil {
		close(s.events)
		return err
	}
	s.events <- Event{
		Kind:        EventFinal,
		UtteranceID: "u-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		FinalText:   text,
		Language:    s.cfg.Language,
		At:          time.Now(),
	}
	close(s.events)
	return nil
}

// buildWAV16kMono 将 PCM16 单声道采样组装为 16k WAV。
func buildWAV16kMono(pcm []byte) ([]byte, error) {
	const sampleRate = 16000
	if len(pcm)%2 != 0 {
		return nil, errors.New("pcm 必须为 16 位对齐")
	}
	const maxWavData = uint64(1<<32 - 1)
	if uint64(len(pcm)) > maxWavData {
		return nil, errors.New("pcm 超过 WAV 尺寸上限")
	}
	dataLen := uint32(len(pcm))
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	if err := binary.Write(&buf, binary.LittleEndian, uint32(36)+dataLen); err != nil {
		return nil, err
	}
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	if err := binary.Write(&buf, binary.LittleEndian, uint32(16)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint16(1)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint32(sampleRate)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint32(sampleRate*2)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint16(2)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.LittleEndian, uint16(16)); err != nil {
		return nil, err
	}
	buf.WriteString("data")
	if err := binary.Write(&buf, binary.LittleEndian, dataLen); err != nil {
		return nil, err
	}
	buf.Write(pcm)
	return buf.Bytes(), nil
}
