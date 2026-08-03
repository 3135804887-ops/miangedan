package room

import (
	"context"
	"errors"
	"testing"

	"miangedan/services/project"
)

// recordingBilling 记录 TASK-061 挂接调用（用于闭环断言）。
type recordingBilling struct {
	calls      []string
	reserveErr error
}

func (b *recordingBilling) Reserve(_ context.Context, _ project.Actor, in BillingReserveInput) error {
	b.calls = append(b.calls, "reserve:"+in.SessionID)
	return b.reserveErr
}

func (b *recordingBilling) StartMetering(_ context.Context, _ project.Actor, sessionID string) error {
	b.calls = append(b.calls, "start:"+sessionID)
	return nil
}

func (b *recordingBilling) StopMetering(_ context.Context, _ project.Actor, sessionID string) error {
	b.calls = append(b.calls, "stop:"+sessionID)
	return nil
}

func (b *recordingBilling) Settle(_ context.Context, _ project.Actor, sessionID, _ string) error {
	b.calls = append(b.calls, "settle:"+sessionID)
	return nil
}

func (b *recordingBilling) RefundFull(_ context.Context, _ project.Actor, sessionID, _ string) error {
	b.calls = append(b.calls, "refund:"+sessionID)
	return nil
}

func createSessionForBilling(t *testing.T, env *testEnv) (project.Project, SessionCreated) {
	t.Helper()
	proj := env.readyProject(t, 1)
	created, err := env.svc.CreateSession(context.Background(), testActor,
		CreateSessionInput{
			ProjectID:     proj.ProjectID,
			RoundSequence: 1,
			DeviceID:      "device-billing-1",
			Kind:          KindFormal,
		}, "k-billing-1")
	if err != nil {
		t.Fatalf("创建会话失败: %v", err)
	}
	return proj, created
}

// 闭环：创建会话 → 预留 + 开始计量；结束会话 → 按实际结算。
func TestBillingHooksCreateAndEnd(t *testing.T) {
	env := newTestEnv(t)
	billing := &recordingBilling{}
	env.svc.SetBilling(billing)
	_, created := createSessionForBilling(t, env)
	if _, err := env.svc.EndSession(context.Background(), testActor, created.SessionID, true, "k-end-1"); err != nil {
		t.Fatalf("结束会话失败: %v", err)
	}
	if containsCall(billing.calls, "reserve:"+created.SessionID) == false ||
		containsCall(billing.calls, "start:"+created.SessionID) == false {
		t.Fatalf("创建会话必须预留并开始计量：%v", billing.calls)
	}
	if containsCall(billing.calls, "settle:"+created.SessionID) == false {
		t.Fatalf("结束会话必须结算：%v", billing.calls)
	}
}

// 降级接受：故障点起停止计量（文字面试不再消耗数字人额度）。
func TestBillingHookDowngradeAcceptedStopsMetering(t *testing.T) {
	env := newTestEnv(t)
	billing := &recordingBilling{}
	env.svc.SetBilling(billing)
	proj := env.readyProject(t, 1)
	sess := env.liveSession(t, proj.ProjectID)
	promptID, err := env.svc.OfferDowngrade(context.Background(), testActor, sess.SessionID, "k-offer-1")
	if err != nil {
		t.Fatalf("发起降级询问失败: %v", err)
	}
	if _, err := env.svc.AcceptDowngrade(context.Background(), testActor,
		sess.SessionID, promptID, "k-accept-1"); err != nil {
		t.Fatalf("接受降级失败: %v", err)
	}
	if containsCall(billing.calls, "stop:"+sess.SessionID) == false {
		t.Fatalf("降级接受必须停止计量：%v", billing.calls)
	}
}

// 降级拒绝：系统责任全额返还本轮预留（闭环：拒绝 → 全额返还）。
func TestBillingHookDowngradeDeclinedRefundsFull(t *testing.T) {
	env := newTestEnv(t)
	billing := &recordingBilling{}
	env.svc.SetBilling(billing)
	proj := env.readyProject(t, 1)
	sess := env.liveSession(t, proj.ProjectID)
	promptID, err := env.svc.OfferDowngrade(context.Background(), testActor, sess.SessionID, "k-offer-2")
	if err != nil {
		t.Fatalf("发起降级询问失败: %v", err)
	}
	if _, err := env.svc.DeclineDowngrade(context.Background(), testActor,
		sess.SessionID, promptID, "k-decline-1"); err != nil {
		t.Fatalf("拒绝降级失败: %v", err)
	}
	if containsCall(billing.calls, "refund:"+sess.SessionID) == false {
		t.Fatalf("拒绝降级必须全额返还预留：%v", billing.calls)
	}
}

// 故障暂停：暂停段停止计量；恢复后重新计量。
func TestBillingHooksTimerPauseResume(t *testing.T) {
	env := newTestEnv(t)
	billing := &recordingBilling{}
	env.svc.SetBilling(billing)
	proj := env.readyProject(t, 1)
	sess := env.liveSession(t, proj.ProjectID)
	if _, err := env.svc.PauseTimer(context.Background(), testActor,
		sess.SessionID, PauseSystemFault, "k-pause-1"); err != nil {
		t.Fatalf("暂停失败: %v", err)
	}
	if _, err := env.svc.ResumeTimer(context.Background(), testActor,
		sess.SessionID, "k-resume-1"); err != nil {
		t.Fatalf("恢复失败: %v", err)
	}
	if containsCall(billing.calls, "stop:"+sess.SessionID) == false ||
		containsCall(billing.calls, "start:"+sess.SessionID) == false {
		t.Fatalf("暂停/恢复必须停止/开始计量：%v", billing.calls)
	}
}

// 余额不足：预留失败 → 阻止开始（402 insufficient_entitlement）。
func TestCreateSessionInsufficientEntitlement(t *testing.T) {
	env := newTestEnv(t)
	env.svc.SetBilling(&recordingBilling{reserveErr: ErrInsufficientEntitlement})
	proj := env.readyProject(t, 1)
	_, err := env.svc.CreateSession(context.Background(), testActor,
		CreateSessionInput{
			ProjectID:     proj.ProjectID,
			RoundSequence: 1,
			DeviceID:      "device-billing-2",
			Kind:          KindFormal,
		}, "k-billing-2")
	if err == nil || !errors.Is(err, ErrEntitlementMissing) {
		t.Fatalf("余额不足应返回 ErrEntitlementMissing，实际 %v", err)
	}
}

func containsCall(calls []string, target string) bool {
	for _, call := range calls {
		if call == target {
			return true
		}
	}
	return false
}
