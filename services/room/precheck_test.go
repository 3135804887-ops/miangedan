package room

import (
	"context"
	"errors"
	"testing"
)

// TASK-027 正常路径：冻结输入模式与便利设置；会话进入 AVATAR_CONNECTING。
func TestFreezePreCheckOK(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 1)
	sess, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a",
	}, "t027-create")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	pc, err := e.svc.FreezePreCheck(context.Background(), testActor, sess.SessionID, FreezePreCheckInput{
		InputModes:     []InputMode{ModeVoice, ModeText, ModeCamera},
		Accommodations: []string{"reduced_motion"},
		DeviceReport:   DeviceReport{CameraOK: true, MicOK: true, NetworkRated: "good"},
	}, "precheck-1")
	if err != nil {
		t.Fatalf("冻结会前配置失败: %v", err)
	}
	if !pc.Frozen || pc.FrozenAt == nil || len(pc.InputModes) != 3 {
		t.Fatalf("冻结结果错误: %+v", pc)
	}
	s, err := e.svc.GetSession(context.Background(), testActor, sess.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if s.RoomStatus != StatusAvatarConnecting {
		t.Fatalf("冻结后应进入 AVATAR_CONNECTING，实际 %s", s.RoomStatus)
	}
	// 幂等重放：相同 idemKey 返回首次结果。
	again, err := e.svc.FreezePreCheck(context.Background(), testActor, sess.SessionID, FreezePreCheckInput{
		InputModes: []InputMode{ModeVoice}, Accommodations: nil,
		DeviceReport: DeviceReport{NetworkRated: "good"},
	}, "precheck-1")
	if err != nil {
		t.Fatalf("幂等重放失败: %v", err)
	}
	if len(again.InputModes) != 3 {
		t.Fatalf("幂等重放应返回首次结果: %+v", again)
	}
}

// TASK-027 异常：重复冻结（不同键）被拒；未知模式/便利设置被拒；摄像头可关但数字人无关闭选项。
func TestFreezePreCheckRejectsInvalid(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 1)
	sess, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a",
	}, "t027-create-2")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	if _, err := e.svc.FreezePreCheck(context.Background(), testActor, sess.SessionID, FreezePreCheckInput{
		InputModes: []InputMode{ModeVoice}, Accommodations: nil,
		DeviceReport: DeviceReport{NetworkRated: "good"},
	}, "precheck-2"); err != nil {
		t.Fatalf("首次冻结失败: %v", err)
	}
	if _, err := e.svc.FreezePreCheck(context.Background(), testActor, sess.SessionID, FreezePreCheckInput{
		InputModes: []InputMode{ModeText}, Accommodations: nil,
		DeviceReport: DeviceReport{NetworkRated: "good"},
	}, "precheck-2b"); !errors.Is(err, ErrPreCheckFrozen) {
		t.Fatalf("重复冻结应被拒，got: %v", err)
	}
	// 未知模式。
	proj2 := e.readyProject(t, 1)
	sess2, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj2.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-b",
	}, "t027-create-3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.FreezePreCheck(context.Background(), testActor, sess2.SessionID, FreezePreCheckInput{
		InputModes: []InputMode{"cheat_mode"}, Accommodations: nil,
		DeviceReport: DeviceReport{NetworkRated: "good"},
	}, "precheck-3"); !errors.Is(err, ErrPreCheckInvalid) {
		t.Fatalf("未知模式应被拒，got: %v", err)
	}
	// 未知便利设置。
	if _, err := e.svc.FreezePreCheck(context.Background(), testActor, sess2.SessionID, FreezePreCheckInput{
		InputModes: []InputMode{ModeVoice}, Accommodations: []string{"unlimited_answers"},
		DeviceReport: DeviceReport{NetworkRated: "good"},
	}, "precheck-4"); !errors.Is(err, ErrPreCheckInvalid) {
		t.Fatalf("未知便利设置应被拒，got: %v", err)
	}
	// 非法网络评级。
	if _, err := e.svc.FreezePreCheck(context.Background(), testActor, sess2.SessionID, FreezePreCheckInput{
		InputModes: []InputMode{ModeVoice}, Accommodations: nil,
		DeviceReport: DeviceReport{NetworkRated: "excellent"},
	}, "precheck-5"); !errors.Is(err, ErrPreCheckInvalid) {
		t.Fatalf("非法网络评级应被拒，got: %v", err)
	}
}

// TASK-027 规则：仅摄像头/麦克风可关，数字人音视频始终开启（模式枚举无关闭数字人）。
func TestPreCheckAvatarAlwaysOn(t *testing.T) {
	if validInputModes[ModeCamera] != true {
		t.Fatal("camera 应为合法模式（可关）")
	}
	if _, ok := validInputModes["avatar_off"]; ok {
		t.Fatal("禁止出现关闭数字人模式")
	}
}
