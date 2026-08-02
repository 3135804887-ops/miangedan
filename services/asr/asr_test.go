package asr

import (
	"context"
	"errors"
	"testing"
	"time"

	"miangedan/services/provider"
)

// 正常/异常路径：回合检测在静音窗口后输出 final，断点→final 时延在 NFR-010 预算内。
func TestTurnDetector(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	d, err := NewTurnDetector(700, clock)
	if err != nil {
		t.Fatal(err)
	}
	if final, _ := d.Feed(AudioFrame{Seq: 1, Speech: true}); final {
		t.Fatal("语音帧不应触发 final")
	}
	now = now.Add(500 * time.Millisecond)
	if final, _ := d.Feed(AudioFrame{Seq: 2, Speech: false}); final {
		t.Fatal("静音未到窗口不应触发 final")
	}
	now = now.Add(300 * time.Millisecond)
	final, latency := d.Feed(AudioFrame{Seq: 3, Speech: false})
	if !final {
		t.Fatal("静音达到窗口应触发 final")
	}
	if err := CheckBudget(latency); err != nil {
		t.Fatalf("断点→final 应在预算内: %v", err)
	}
	if _, err := NewTurnDetector(0, nil); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal("非法静音窗口必须拒绝")
	}
}

// 正常/异常路径：单说话方闸门避免重叠；打断→停止在 NFR-009 预算内。
func TestTurnGate(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	g := NewTurnGate(StopLatencyBudget, clock)
	if err := g.Start(SpeakerAvatar); err != nil {
		t.Fatal(err)
	}
	if err := g.Start(SpeakerAvatar); !errors.Is(err, ErrOverlap) {
		t.Fatalf("重复开始必须拒绝，实际 %v", err)
	}
	g.Interrupt()
	if g.Speaking() != SpeakerUser {
		t.Fatal("打断后应切换为用户说话")
	}
	now = now.Add(100 * time.Millisecond)
	if err := g.StopConfirmed(); err != nil {
		t.Fatalf("100ms 停止应在预算内: %v", err)
	}
	g.Interrupt()
	now = now.Add(600 * time.Millisecond)
	if err := g.StopConfirmed(); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("600ms 停止必须超预算，实际 %v", err)
	}
}

// 正常/异常路径：流式识别配置校验与合成流。
func TestStreamConfig(t *testing.T) {
	if err := ValidateConfig(StreamConfig{Language: "zh-CN", SilenceEndpointMs: 700}); err != nil {
		t.Fatalf("合法配置应通过: %v", err)
	}
	if err := ValidateConfig(StreamConfig{Language: "fr-FR", SilenceEndpointMs: 700}); err == nil {
		t.Fatal("非法语言必须拒绝")
	}
	s, err := StubProvider{}.OpenStream(context.Background(), StreamConfig{Language: "zh-CN", SilenceEndpointMs: 700})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SendAudio(AudioFrame{Seq: 1, Speech: true}); err != nil {
		t.Fatal(err)
	}
	ev, err := s.Recv()
	if err != nil || ev.Kind != EventPartial {
		t.Fatalf("合成流应输出 partial: %v %+v", err, ev)
	}
}

// 正常路径：注册 asr_{region}_{role} 并可被适配层路由与熔断。
func TestRegisterAndRoute(t *testing.T) {
	reg := provider.NewRegistry(nil)
	if err := RegisterProvider(reg, StubProvider{}, "cn", provider.RolePrimary, "1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := RegisterProvider(reg, StubProvider{}, "cn", provider.RoleSecondary, "1.0.0"); err != nil {
		t.Fatal(err)
	}
	rt := provider.NewRouter(reg)
	got, err := rt.Route(provider.CapASR, "cn", provider.RouteOptions{})
	if err != nil || got.ProviderID != "asr_cn_primary" {
		t.Fatalf("应路由到 asr_cn_primary: %v %+v", err, got)
	}
	for i := 0; i < 5; i++ {
		rt.RecordFailure("asr_cn_primary")
	}
	got, err = rt.Route(provider.CapASR, "cn", provider.RouteOptions{})
	if err != nil || got.ProviderID != "asr_cn_secondary" {
		t.Fatalf("主 open 应切 secondary: %v %+v", err, got)
	}
}
