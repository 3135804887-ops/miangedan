package billing

import (
	"context"
	"errors"
	"testing"
	"time"
)

var testActor = Actor{UserID: "user-001", DataRegion: "cn"}

func newTestService(t *testing.T) (*Service, *MemoryStore) {
	t.Helper()
	store := NewMemoryStore()
	svc, err := NewService(store)
	if err != nil {
		t.Fatalf("创建服务失败: %v", err)
	}
	return svc, store
}

func samplePlan() PlanInput {
	return PlanInput{
		ProjectID: "p-1",
		Rounds: []RoundPlanInput{
			{Sequence: 1, DurationMinutes: 25, RetryEligible: true},
			{Sequence: 2, DurationMinutes: 30, RetryEligible: true},
			{Sequence: 3, DurationMinutes: 20, RetryEligible: true},
		},
	}
}

// 免费额度：首次登录 60 分钟；幂等（每人一份）。
func TestGrantFreeCreditIdempotent(t *testing.T) {
	svc, _ := newTestService(t)
	first, err := svc.GrantFreeCredit(context.Background(), testActor, "idem-free-1")
	if err != nil {
		t.Fatalf("发放免费额度失败: %v", err)
	}
	if first.TotalSeconds != FreeCreditSeconds {
		t.Fatalf("免费额度应为 3600 秒，实际 %d", first.TotalSeconds)
	}
	second, err := svc.GrantFreeCredit(context.Background(), testActor, "idem-free-1")
	if err != nil || second.EntitlementID != first.EntitlementID {
		t.Fatalf("免费额度幂等异常：%+v err=%v", second, err)
	}
	balance, err := svc.Balance(context.Background(), testActor)
	if err != nil || balance != FreeCreditSeconds {
		t.Fatalf("余额应为 3600，实际 %d err=%v", balance, err)
	}
}

// 四类权益：免费/项目包/加油包/Pro 叠加计入余额。
func TestEntitlementKindsAndBalance(t *testing.T) {
	svc, _ := newTestService(t)
	_, _ = svc.GrantFreeCredit(context.Background(), testActor, "idem-k1")
	_, _ = svc.GrantProjectPack(context.Background(), testActor, "p-1", 7200, "idem-k2")
	_, _ = svc.GrantTopup(context.Background(), testActor, 1800, "idem-k3")
	balance, err := svc.Balance(context.Background(), testActor)
	if err != nil {
		t.Fatalf("余额计算失败: %v", err)
	}
	if balance != 3600+7200+1800 {
		t.Fatalf("四类权益余额异常：%d", balance)
	}
	ok, err := svc.CanReserve(context.Background(), testActor, balance)
	if err != nil || !ok {
		t.Fatalf("等额预留应允许：ok=%v err=%v", ok, err)
	}
	ok, err = svc.CanReserve(context.Background(), testActor, balance+1)
	if err != nil || ok {
		t.Fatalf("超额预留应拒绝：ok=%v err=%v", ok, err)
	}
}

// Pro：结转 ≤1 账期且总余额 ≤2×月额度。
func TestProCarryoverCap(t *testing.T) {
	svc, store := newTestService(t)
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	period1 := now.AddDate(0, 0, 0)
	period2 := now.AddDate(0, 1, 0)
	period3 := now.AddDate(0, 2, 0)
	_, err := svc.ActivatePro(context.Background(), testActor, 3600, period1, period2, "idem-pro-1")
	if err != nil {
		t.Fatalf("激活 Pro 失败: %v", err)
	}
	// 第一账期剩余 3000 秒（模拟消耗 600）。
	items, _ := svc.GetEntitlements(context.Background(), testActor)
	for _, e := range items {
		if e.Kind == KindProSub {
			e.ConsumedSeconds = 600
			if err := store.UpdateEntitlement(e); err != nil {
				t.Fatalf("更新消耗失败: %v", err)
			}
		}
	}
	// 续费：结转 3000（≤月额度 3600），总 6600 ≤ 7200。
	_, err = svc.ActivatePro(context.Background(), testActor, 3600, period2, period3, "idem-pro-2")
	if err != nil {
		t.Fatalf("续费失败: %v", err)
	}
	balance, err := svc.Balance(context.Background(), testActor)
	if err != nil {
		t.Fatalf("余额计算失败: %v", err)
	}
	if balance != 6600 {
		t.Fatalf("续费后余额应为 6600（结转 3000 + 月 3600），实际 %d", balance)
	}
}

// 报价生命周期：DRAFT → PRESENTED → ACCEPTED + 计费版本冻结；开始后拒绝重报。
func TestQuoteLifecycleAndFreeze(t *testing.T) {
	svc, _ := newTestService(t)
	quote, err := svc.CreateQuote(context.Background(), testActor, samplePlan(), "idem-quote-1")
	if err != nil {
		t.Fatalf("创建报价失败: %v", err)
	}
	if quote.Status != QuoteDraft || quote.PlanVersion != 1 {
		t.Fatalf("初始报价异常：%+v", quote)
	}
	if quote.TotalMinutes != 75 || quote.FreeRetries != 3 || quote.AmountCents <= 0 {
		t.Fatalf("报价组成异常：%+v", quote)
	}
	quote, err = svc.PresentQuote(context.Background(), testActor, quote.QuoteID)
	if err != nil || quote.Status != QuotePresented {
		t.Fatalf("呈现报价失败：%+v err=%v", quote, err)
	}
	freeze, err := svc.AcceptQuote(context.Background(), testActor, quote.QuoteID)
	if err != nil {
		t.Fatalf("接受报价失败: %v", err)
	}
	if !freeze.Frozen || freeze.PlanVersion != 1 {
		t.Fatalf("冻结异常：%+v", freeze)
	}
	// 开始后（冻结）拒绝重新报价。
	if _, err := svc.RecalculateQuote(context.Background(), testActor, samplePlan(), "idem-quote-2"); !errors.Is(
		err, ErrQuoteFrozen) {
		t.Fatalf("冻结后重报应拒绝，实际 %v", err)
	}
}

// 开始前计划修改 → 重新报价（版本递增、差额语义由订单侧处理）。
func TestQuoteRecalculationBeforeStart(t *testing.T) {
	svc, _ := newTestService(t)
	first, err := svc.CreateQuote(context.Background(), testActor, samplePlan(), "idem-quote-3")
	if err != nil {
		t.Fatalf("创建报价失败: %v", err)
	}
	changed := samplePlan()
	changed.Rounds = append(changed.Rounds,
		RoundPlanInput{Sequence: 4, DurationMinutes: 20, RetryEligible: true})
	recalculated, err := svc.RecalculateQuote(context.Background(), testActor, changed, "idem-quote-4")
	if err != nil {
		t.Fatalf("重新报价失败: %v", err)
	}
	if recalculated.PlanVersion != first.PlanVersion+1 {
		t.Fatalf("重新报价版本应递增：%d vs %d", recalculated.PlanVersion, first.PlanVersion)
	}
	if recalculated.Status != QuotePresented {
		t.Fatalf("重新报价后应 PRESENTED：%s", recalculated.Status)
	}
	if recalculated.TotalMinutes != 95 {
		t.Fatalf("增加轮次后总时长应为 95，实际 %d", recalculated.TotalMinutes)
	}
}

// 异常：非法区域、负时长/秒数、未知报价、重复接受冲突必须拒绝。
func TestInvalidBillingRequests(t *testing.T) {
	svc, _ := newTestService(t)
	if _, err := svc.GrantFreeCredit(context.Background(),
		Actor{UserID: "u", DataRegion: "xx"}, "idem-bad-1"); err == nil {
		t.Fatal("非法区域必须拒绝")
	}
	if _, err := svc.GrantTopup(context.Background(), testActor, -1, "idem-bad-2"); err == nil {
		t.Fatal("负秒数必须拒绝")
	}
	if _, err := svc.CreateQuote(context.Background(), testActor,
		PlanInput{ProjectID: "p", Rounds: []RoundPlanInput{{Sequence: 1, DurationMinutes: 0}}},
		"idem-bad-3"); err == nil {
		t.Fatal("零时长轮次必须拒绝")
	}
	if _, err := svc.PresentQuote(context.Background(), testActor, "ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("未知报价应 not_found，实际 %v", err)
	}
	quote, _ := svc.CreateQuote(context.Background(), testActor, samplePlan(), "idem-bad-4")
	if _, err := svc.AcceptQuote(context.Background(), testActor, quote.QuoteID); !errors.Is(
		err, ErrStateConflict) {
		t.Fatalf("DRAFT 直接接受应冲突，实际 %v", err)
	}
}
