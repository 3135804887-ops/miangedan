package avatar

import (
	"context"
	"errors"
	"testing"
	"time"

	"miangedan/services/provider"
)

func testDriver(t *testing.T) (Driver, *CharacterLibrary) {
	t.Helper()
	lib, err := NewCharacterLibrary(SyntheticCharacters()...)
	if err != nil {
		t.Fatal(err)
	}
	return StubDriver{Library: lib}, lib
}

// 正常路径：授权角色可启动；未知角色被拒（禁止每场生成新脸）。
func TestCharacterLibrary(t *testing.T) {
	_, lib := testDriver(t)
	if err := lib.Validate("avatar-zh-01"); err != nil {
		t.Fatalf("授权角色应可校验: %v", err)
	}
	if err := lib.Validate("avatar-unknown"); !errors.Is(err, ErrCharacterNotFound) {
		t.Fatalf("未知角色必须拒绝，实际 %v", err)
	}
}

// TASK-090 补测（TC-NFR-007-N01/TC-NFR-012-N01）：建连 ≤8s 预算冒烟；默认档位 ≥720p/24fps。
func TestDriverStartWithinConnectBudget(t *testing.T) {
	if DefaultVideoProfile.Width < 1280 || DefaultVideoProfile.Height < 720 || DefaultVideoProfile.FPS < 24 {
		t.Fatalf("默认档位必须 ≥720p/24fps，实际 %+v", DefaultVideoProfile)
	}
	driver, _ := testDriver(t)
	start := time.Now()
	sess, err := driver.Start(context.Background(), StartInput{
		CharacterID:  "avatar-zh-01",
		Persona:      DefaultPersona(),
		VideoProfile: DefaultVideoProfile,
	})
	if err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 8*time.Second {
		t.Fatalf("建连超过 8s 预算（NFR-007 P95）: %v", elapsed)
	}
	if err := sess.Stop(context.Background()); err != nil {
		t.Fatalf("停止失败: %v", err)
	}
}

// 正常/异常路径：驱动启动校验人格与视频档位。
func TestDriverStart(t *testing.T) {
	driver, _ := testDriver(t)
	in := StartInput{CharacterID: "avatar-zh-01", Persona: DefaultPersona(), VideoProfile: DefaultVideoProfile}
	sess, err := driver.Start(context.Background(), in)
	if err != nil {
		t.Fatalf("启动应成功: %v", err)
	}
	report, err := sess.Drive(context.Background(), []AudioChunk{{Seq: 1, Duration: 0}})
	if err != nil || report.MaxDeviationMs > LipSyncBudgetMs {
		t.Fatalf("口型偏差应在预算内: %+v %v", report, err)
	}
	bad := in
	bad.CharacterID = "unknown"
	if _, err := driver.Start(context.Background(), bad); !errors.Is(err, ErrCharacterNotFound) {
		t.Fatalf("未知角色必须拒绝: %v", err)
	}
	badPersona := in
	badPersona.Persona.Tone = "aggressive"
	if _, err := driver.Start(context.Background(), badPersona); !errors.Is(err, ErrInvalidPersona) {
		t.Fatalf("越界人格必须拒绝: %v", err)
	}
	badVideo := in
	badVideo.VideoProfile = VideoProfile{}
	if _, err := driver.Start(context.Background(), badVideo); !errors.Is(err, ErrInvalidVideoProfile) {
		t.Fatalf("非法视频档位必须拒绝: %v", err)
	}
}

// 正常路径：注册 CapAvatar 供应商并可被适配层路由（TASK-030 集成）。
func TestRegisterAndRoute(t *testing.T) {
	driver, _ := testDriver(t)
	reg := provider.NewRegistry(nil)
	if err := RegisterDriver(reg, driver, "cn", provider.RolePrimary, "1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := RegisterDriver(reg, driver, "cn", provider.RoleSecondary, "1.0.0"); err != nil {
		t.Fatal(err)
	}
	rt := provider.NewRouter(reg)
	got, err := rt.Route(provider.CapAvatar, "cn", provider.RouteOptions{})
	if err != nil || got.ProviderID != "avatar_cn_primary" {
		t.Fatalf("应路由到 avatar_cn_primary: %v %+v", err, got)
	}
	// 主熔断切 secondary。
	for i := 0; i < 5; i++ {
		rt.RecordFailure("avatar_cn_primary")
	}
	got, err = rt.Route(provider.CapAvatar, "cn", provider.RouteOptions{})
	if err != nil || got.ProviderID != "avatar_cn_secondary" {
		t.Fatalf("主 open 应切 secondary: %v %+v", err, got)
	}
}
