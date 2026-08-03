package room

import (
	"context"
	"errors"
	"testing"
	"time"
)

// liveSession 创建会话并将其状态直接置为 LIVE（模拟数字人已建连）。
func (e *testEnv) liveSession(t *testing.T, projectID string) Session {
	t.Helper()
	sess, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: projectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a",
	}, "t025-create")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	s, err := e.store.GetSession("cn", sess.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	s.RoomStatus = StatusLive
	if err := e.store.UpdateSession(s); err != nil {
		t.Fatal(err)
	}
	return s
}

// TASK-025 正常路径：故障暂停计时 → 恢复；暂停秒数累计且不计费秒数。
func TestTimerPauseResume(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 1)
	sess := e.liveSession(t, proj.ProjectID)
	start := e.now
	paused, err := e.svc.PauseTimer(context.Background(), testActor, sess.SessionID, PauseSystemFault, "pause-1")
	if err != nil {
		t.Fatalf("暂停失败: %v", err)
	}
	if paused.RoomStatus != StatusPausedSystem || paused.PausedAt == nil {
		t.Fatalf("暂停状态错误: %+v", paused)
	}
	// 前进 30 秒后恢复。
	e.svc.now = func() time.Time { return start.Add(30 * time.Second) }
	resumed, err := e.svc.ResumeTimer(context.Background(), testActor, sess.SessionID, "resume-1")
	if err != nil {
		t.Fatalf("恢复失败: %v", err)
	}
	if resumed.RoomStatus != StatusLive || resumed.PausedSeconds != 30 {
		t.Fatalf("恢复状态错误: %+v", resumed)
	}
	if resumed.BillableSeconds != 0 {
		t.Fatalf("暂停期间不应产生计费秒数: %d", resumed.BillableSeconds)
	}
	// 幂等重放。
	again, err := e.svc.ResumeTimer(context.Background(), testActor, sess.SessionID, "resume-1")
	if err != nil || again.PausedSeconds != 30 {
		t.Fatalf("幂等重放失败: %v %+v", err, again)
	}
}

// TASK-025 降级接受：PAUSED_SYSTEM → DOWNGRADE_PROMPTED → TEXT_DEGRADED（口语项按文字模式）。
func TestDowngradeAccepted(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 1)
	sess := e.liveSession(t, proj.ProjectID)
	if _, err := e.svc.PauseTimer(context.Background(), testActor, sess.SessionID, PauseSystemFault, "pause-2"); err != nil {
		t.Fatal(err)
	}
	promptID, err := e.svc.OfferDowngrade(context.Background(), testActor, sess.SessionID, "offer-1")
	if err != nil {
		t.Fatalf("发起降级询问失败: %v", err)
	}
	if promptID == "" {
		t.Fatal("prompt_id 不应为空")
	}
	// 重复发起幂等。
	same, err := e.svc.OfferDowngrade(context.Background(), testActor, sess.SessionID, "offer-1")
	if err != nil || same != promptID {
		t.Fatalf("重复发起应返回同一 prompt_id: %v %s", err, same)
	}
	s, err := e.svc.AcceptDowngrade(context.Background(), testActor, sess.SessionID, promptID, "accept-1")
	if err != nil {
		t.Fatalf("接受降级失败: %v", err)
	}
	if s.RoomStatus != StatusTextDegraded || s.DowngradeStatus != DowngradeAccepted || s.TextDegradedAt == nil {
		t.Fatalf("降级状态错误: %+v", s)
	}
}

// TASK-025 降级拒绝：ENDED + 评估未完成语义 + 设备释放（额度返还为 TASK-061 挂接点）。
func TestDowngradeDeclined(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 1)
	sess := e.liveSession(t, proj.ProjectID)
	if _, err := e.svc.PauseTimer(context.Background(), testActor, sess.SessionID, PauseSystemFault, "pause-3"); err != nil {
		t.Fatal(err)
	}
	promptID, err := e.svc.OfferDowngrade(context.Background(), testActor, sess.SessionID, "offer-2")
	if err != nil {
		t.Fatal(err)
	}
	s, err := e.svc.DeclineDowngrade(context.Background(), testActor, sess.SessionID, promptID, "decline-1")
	if err != nil {
		t.Fatalf("拒绝降级失败: %v", err)
	}
	if s.RoomStatus != StatusEnded || s.EndReason != EndDowngradeRejected || s.DowngradeStatus != DowngradeRejected {
		t.Fatalf("拒绝降级状态错误: %+v", s)
	}
	// 设备已释放：再次创建会话可成功（无 device_active 冲突）。
	if _, err := e.svc.CreateSession(context.Background(), testActor, CreateSessionInput{
		ProjectID: proj.ProjectID, RoundSequence: 1, Kind: KindFormal, DeviceID: "device-a",
	}, "t025-recreate"); err != nil {
		t.Fatalf("设备应已释放可重建会话: %v", err)
	}
}

// TASK-025 异常：非暂停状态恢复被拒；prompt_id 不匹配被拒；会话已结束后操作被拒。
func TestFaultControlsRejectInvalid(t *testing.T) {
	e := newTestEnv(t)
	proj := e.readyProject(t, 1)
	sess := e.liveSession(t, proj.ProjectID)
	if _, err := e.svc.ResumeTimer(context.Background(), testActor, sess.SessionID, "resume-x"); !errors.Is(err, ErrTimerNotPaused) {
		t.Fatalf("LIVE 状态恢复应被拒，got: %v", err)
	}
	if _, err := e.svc.PauseTimer(context.Background(), testActor, sess.SessionID, "bogus", "pause-x"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("非法暂停原因应被拒，got: %v", err)
	}
	if _, err := e.svc.PauseTimer(context.Background(), testActor, sess.SessionID, PauseSystemFault, "pause-4"); err != nil {
		t.Fatal(err)
	}
	promptID, err := e.svc.OfferDowngrade(context.Background(), testActor, sess.SessionID, "offer-3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.AcceptDowngrade(context.Background(), testActor, sess.SessionID, "wrong-prompt", "accept-x"); !errors.Is(err, ErrDowngradeInvalid) {
		t.Fatalf("prompt_id 不匹配应被拒，got: %v", err)
	}
	if _, err := e.svc.DeclineDowngrade(context.Background(), testActor, sess.SessionID, promptID, "decline-2"); err != nil {
		t.Fatalf("拒绝降级失败: %v", err)
	}
	if _, err := e.svc.PauseTimer(context.Background(), testActor, sess.SessionID, PauseSystemFault, "pause-5"); !errors.Is(err, ErrSessionEnded) {
		t.Fatalf("会话结束后暂停应被拒，got: %v", err)
	}
}
