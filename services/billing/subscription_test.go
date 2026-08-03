// Package billing Pro 订阅生命周期测试（TASK-064；FR-033，US-06 场景 5）。
package billing

import (
	"context"
	"errors"
	"testing"
	"time"
)

func activateProForTest(t *testing.T, svc *Service, start, end time.Time, idem string) ProSubscription {
	t.Helper()
	sub, err := svc.ActivatePro(context.Background(), testActor, 3600, start, end, idem)
	if err != nil {
		t.Fatalf("激活 Pro 失败: %v", err)
	}
	return sub
}

// 自动续费必须单独勾选并同意；同意条款被记录。
func TestSetAutoRenewRequiresConsent(t *testing.T) {
	svc, _ := newTestService(t)
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	_ = activateProForTest(t, svc, now, now.AddDate(0, 1, 0), "idem-sub-consent")
	if _, err := svc.SetAutoRenew(context.Background(), testActor, true, false,
		3600, 3000, "idem-consent-1"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("未同意应拒绝自动续费，实际 err=%v", err)
	}
	sub, err := svc.SetAutoRenew(context.Background(), testActor, true, true,
		3600, 3000, "idem-consent-2")
	if err != nil {
		t.Fatalf("设置自动续费失败: %v", err)
	}
	if !sub.AutoRenew || sub.ConsentPriceCents != 3000 {
		t.Fatalf("自动续费条款异常：%+v", sub)
	}
}

// 取消续费：权益保留至账期结束（余额不变）。
func TestCancelAutoRenewKeepsEntitlement(t *testing.T) {
	svc, _ := newTestService(t)
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	_ = activateProForTest(t, svc, now, now.AddDate(0, 1, 0), "idem-sub-cancel")
	balance, _ := svc.Balance(context.Background(), testActor)
	sub, err := svc.CancelAutoRenew(context.Background(), testActor, "idem-cancel-1")
	if err != nil {
		t.Fatalf("取消续费失败: %v", err)
	}
	if sub.Status != "SUB_CANCELLED" || sub.AutoRenew {
		t.Fatalf("取消续费状态异常：%+v", sub)
	}
	balanceAfter, _ := svc.Balance(context.Background(), testActor)
	if balanceAfter != balance {
		t.Fatalf("取消续费不得影响当期权益：%d -> %d", balance, balanceAfter)
	}
}

// 扣款前提醒：未提醒不可扣款；提醒后 7 天窗口内扣款成功并结转。
func TestRenewalReminderAndCharge(t *testing.T) {
	svc, store := newTestService(t)
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	sub := activateProForTest(t, svc, now, now.AddDate(0, 1, 0), "idem-sub-renew")
	sub.AutoRenew = true
	sub.ConsentMonthlySeconds = 3600
	sub.ConsentPriceCents = 3000
	if err := store.UpdateSubscription(sub); err != nil {
		t.Fatalf("更新订阅失败: %v", err)
	}
	if _, err := svc.ChargeRenewal(context.Background(), testActor, "missing-renewal", "idem-charge-0"); err == nil {
		t.Fatal("无提醒事件应拒绝扣款")
	}
	record, err := svc.PrepareRenewal(context.Background(), testActor, "idem-renew-1")
	if err != nil {
		t.Fatalf("准备续费失败: %v", err)
	}
	if record.Status != RenewalReminded || record.RemindedAt == nil {
		t.Fatalf("续费事件应为 reminded：%+v", record)
	}
	renewed, err := svc.ChargeRenewal(context.Background(), testActor, record.RenewalID, "idem-charge-1")
	if err != nil {
		t.Fatalf("扣款续费失败: %v", err)
	}
	if renewed.Status != "SUB_ACTIVE" || !renewed.PeriodStart.Equal(record.PeriodStart) {
		t.Fatalf("续费后订阅异常：%+v", renewed)
	}
	charged, _ := svc.store.GetRenewalByID("cn", record.RenewalID)
	if charged.Status != RenewalCharged || charged.ChargedAt == nil {
		t.Fatalf("续费事件应已扣款：%+v", charged)
	}
	balance, _ := svc.Balance(context.Background(), testActor)
	if balance > 2*3600 {
		t.Fatalf("总余额不得超过 2×月额度：%d", balance)
	}
}

// 条款变化（价格或权益）必须重新同意。
func TestRenewalTermsChangeRequiresReconsent(t *testing.T) {
	svc, store := newTestService(t)
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	sub := activateProForTest(t, svc, now, now.AddDate(0, 1, 0), "idem-sub-terms")
	sub.AutoRenew = true
	sub.ConsentMonthlySeconds = 3600
	sub.ConsentPriceCents = 99999
	if err := store.UpdateSubscription(sub); err != nil {
		t.Fatalf("更新订阅失败: %v", err)
	}
	if _, err := svc.PrepareRenewal(context.Background(), testActor, "idem-terms-1"); !errors.Is(err, ErrReconsentRequired) {
		t.Fatalf("条款变化应要求重新同意，实际 err=%v", err)
	}
}

// 到期：SUB_EXPIRED；历史保留；进行中的正式轮次可正常结束。
func TestExpireDueSubscriptionKeepsHistoryAndCompletesRound(t *testing.T) {
	svc, _ := newTestService(t)
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	start := now.AddDate(0, -1, 0)
	end := now.AddDate(0, 0, 0)
	_ = activateProForTest(t, svc, start, end, "idem-sub-expire")
	if _, err := svc.Reserve(context.Background(), testActor, ReserveInput{
		ProjectID: "p-1", RoundSequence: 1, AttemptID: "a-1", SessionID: "s-1",
		EstimatedSeconds: 600, IdempotencyKey: "idem-expire-reserve",
	}); err != nil {
		t.Fatalf("预留失败: %v", err)
	}
	expired, err := svc.ExpireDueSubscriptions(context.Background(), "cn")
	if err != nil {
		t.Fatalf("到期任务失败: %v", err)
	}
	if len(expired) != 1 || expired[0].Status != "SUB_EXPIRED" {
		t.Fatalf("到期订阅应 SUB_EXPIRED：%+v", expired)
	}
	if _, err := svc.Settle(context.Background(), testActor, "s-1", "round_completed", "idem-expire-settle"); err != nil {
		t.Fatalf("进行中轮次应可正常结束: %v", err)
	}
	if _, err := svc.GetSubscription(context.Background(), testActor); err != nil {
		t.Fatalf("历史订阅应保留: %v", err)
	}
}

// 提醒超过 7 天窗口后禁止扣款。
func TestRenewalReminderExpires(t *testing.T) {
	svc, store := newTestService(t)
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	sub := activateProForTest(t, svc, now, now.AddDate(0, 1, 0), "idem-sub-window")
	sub.AutoRenew = true
	sub.ConsentMonthlySeconds = 3600
	sub.ConsentPriceCents = 3000
	if err := store.UpdateSubscription(sub); err != nil {
		t.Fatalf("更新订阅失败: %v", err)
	}
	record, err := svc.PrepareRenewal(context.Background(), testActor, "idem-window-1")
	if err != nil {
		t.Fatalf("准备续费失败: %v", err)
	}
	svc.now = func() time.Time { return now.Add(8 * 24 * time.Hour) }
	if _, err := svc.ChargeRenewal(context.Background(), testActor, record.RenewalID, "idem-window-charge"); err == nil {
		t.Fatal("提醒超过窗口应禁止扣款")
	}
}
