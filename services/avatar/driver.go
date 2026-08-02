package avatar

import (
	"context"
	"fmt"
	"time"
)

// Driver 为数字人驱动适配层（PROVIDER-ADAPTERS §4.4：start_session / drive / stop）。
// 真实媒体接入随供应商选型；本任务落地契约、口型预算与授权角色约束。
type Driver interface {
	Start(ctx context.Context, in StartInput) (DriverSession, error)
}

// StubDriver 为合成驱动（开发/测试；不产生真实音视频）。
type StubDriver struct {
	Library *CharacterLibrary
}

// Start 校验角色授权与人格后返回合成会话。
func (d StubDriver) Start(_ context.Context, in StartInput) (DriverSession, error) {
	if err := d.Library.Validate(in.CharacterID); err != nil {
		return nil, err
	}
	if err := ValidatePersona(in.Persona); err != nil {
		return nil, err
	}
	if err := ValidateVideoProfile(in.VideoProfile); err != nil {
		return nil, err
	}
	return &stubSession{}, nil
}

type stubSession struct{}

// Drive 返回零偏差（合成）并验证预算（NFR-011）。
func (s *stubSession) Drive(_ context.Context, chunks []AudioChunk) (LipSyncReport, error) {
	for _, c := range chunks {
		if c.Seq < 0 {
			return LipSyncReport{}, fmt.Errorf("%w: 分片序号非法", ErrLipSyncOverBudget)
		}
	}
	return LipSyncReport{MaxDeviationMs: 0, CheckedAt: time.Now()}, nil
}

// Stop 停止驱动（打断场景）。
func (s *stubSession) Stop(context.Context) error { return nil }
