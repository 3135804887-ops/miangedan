package billing

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeClock struct {
	current time.Time
}

func (c *fakeClock) now() time.Time { return c.current }

func (c *fakeClock) advance(seconds int) {
	c.current = c.current.Add(time.Duration(seconds) * time.Second)
}

func newLedgerService(t *testing.T) (*Service, *MemoryStore, *fakeClock) {
	t.Helper()
	store := NewMemoryStore()
	svc, err := NewService(store)
	if err != nil {
		t.Fatalf("创建服务失败: %v", err)
	}
	clock := &fakeClock{current: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)}
	svc.now = clock.now
	return svc, store, clock
}

func seedBalance(t *testing.T, svc *Service, seconds int) {
	t.Helper()
	if _, err := svc.GrantFreeCredit(context.Background(), testActor, "idem-seed-1"); err != nil {
		t.Fatalf("发放免费额度失败: %v", err)
	}
	if seconds > FreeCreditSeconds {
		if _, err := svc.GrantTopup(context.Background(), testActor,
			seconds-FreeCreditSeconds, "idem-seed-2"); err != nil {
			t.Fatalf("发放加油包失败: %v", err)
		}
	}
}

func reserveInput(sessionID string) ReserveInput {
	return ReserveInput{
		ProjectID:        "p-1",
		RoundSequence:    1,
		AttemptID:        "a-1",
		SessionID:        sessionID,
		EstimatedSeconds: 1800,
		IdempotencyKey:   "reserve-" + sessionID,
	}
}

// 预留：按免费→加油包顺序消费；账本 reserve 条目；幂等。
func TestReserveConsumesAndIdempotent(t *testing.T) {
	svc, _, _ := newLedgerService(t)
	seedBalance(t, svc, 3600)
	entry, err := svc.Reserve(context.Background(), testActor, reserveInput("s-1"))
	if err != nil {
		t.Fatalf("预留失败: %v", err)
	}
	if entry.EntryType != EntryReserve || entry.Seconds != 1800 {
		t.Fatalf("预留条目异常：%+v", entry)
	}
	balance, _ := svc.Balance(context.Background(), testActor)
	if balance != 1800 {
		t.Fatalf("预留后余额应为 1800，实际 %d", balance)
	}
	again, err := svc.Reserve(context.Background(), testActor, reserveInput("s-1"))
	if err != nil || again.EntryID != entry.EntryID {
		t.Fatalf("预留幂等异常：%+v err=%v", again, err)
	}
}

// 余额不足：阻止开始（ErrInsufficient）。
func TestReserveInsufficient(t *testing.T) {
	svc, _, _ := newLedgerService(t)
	if _, err := svc.GrantTopup(context.Background(), testActor, 600, "idem-insufficient-1"); err != nil {
		t.Fatalf("发放加油包失败: %v", err)
	}
	if _, err := svc.Reserve(context.Background(), testActor, reserveInput("s-2")); !errors.Is(
		err, ErrInsufficient) {
		t.Fatalf("余额不足应拒绝，实际 %v", err)
	}
}

// 计量：只计 LIVE 秒；故障/等待暂停段不计。
func TestMeteringCountsOnlyLiveSeconds(t *testing.T) {
	svc, _, clock := newLedgerService(t)
	seedBalance(t, svc, 3600)
	_, _ = svc.Reserve(context.Background(), testActor, reserveInput("s-3"))
	_, err := svc.StartMetering(context.Background(), testActor, "s-3", "start-3")
	if err != nil {
		t.Fatalf("开始计量失败: %v", err)
	}
	clock.advance(120) // LIVE 120 秒
	_, _ = svc.StopMetering(context.Background(), testActor, "s-3")
	clock.advance(300) // 故障/等待 300 秒（不计）
	_, _ = svc.StartMetering(context.Background(), testActor, "s-3", "start-3b")
	clock.advance(60) // 再 LIVE 60 秒
	meter, err := svc.StopMetering(context.Background(), testActor, "s-3")
	if err != nil {
		t.Fatalf("停止计量失败: %v", err)
	}
	if meter.AccumulatedSeconds != 180 {
		t.Fatalf("只应计 180 秒 LIVE，实际 %d", meter.AccumulatedSeconds)
	}
}

// 结算：按实际使用扣减 + 冲正释放未使用预留；用户主动退出同规则。
func TestSettleActualUsageAndRelease(t *testing.T) {
	svc, store, clock := newLedgerService(t)
	seedBalance(t, svc, 3600)
	_, _ = svc.Reserve(context.Background(), testActor, reserveInput("s-4"))
	_, _ = svc.StartMetering(context.Background(), testActor, "s-4", "start-4")
	clock.advance(600)
	entries, err := svc.Settle(context.Background(), testActor, "s-4", "user_exit", "settle-4")
	if err != nil {
		t.Fatalf("结算失败: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("应有 consume + release 两条，实际 %d", len(entries))
	}
	if entries[0].EntryType != EntryConsume || entries[0].Seconds != 600 {
		t.Fatalf("消费条目异常：%+v", entries[0])
	}
	if entries[1].EntryType != EntryReversal || entries[1].Seconds != 1200 {
		t.Fatalf("释放冲正异常：%+v", entries[1])
	}
	balance, _ := svc.Balance(context.Background(), testActor)
	if balance != 3600-600 {
		t.Fatalf("结算后余额应为 3000，实际 %d", balance)
	}
	ledger, err := store.GetLedgerByProject("cn", "p-1")
	if err != nil || len(ledger) != 3 {
		t.Fatalf("账本应含 reserve+consume+release 三条：%d err=%v", len(ledger), err)
	}
}

// 系统责任全额返还：冲正条目恢复整轮预留（降级拒绝路径）。
func TestRefundFullRestoresReservation(t *testing.T) {
	svc, _, clock := newLedgerService(t)
	seedBalance(t, svc, 3600)
	_, _ = svc.Reserve(context.Background(), testActor, reserveInput("s-5"))
	_, _ = svc.StartMetering(context.Background(), testActor, "s-5", "start-5")
	clock.advance(120)
	entry, err := svc.RefundFull(context.Background(), testActor, "s-5",
		"downgrade_rejected_full_refund", "refund-5")
	if err != nil {
		t.Fatalf("全额返还失败: %v", err)
	}
	if entry.EntryType != EntryReversal || entry.Seconds != 1800 {
		t.Fatalf("返还冲正异常：%+v", entry)
	}
	balance, _ := svc.Balance(context.Background(), testActor)
	if balance != 3600 {
		t.Fatalf("全额返还后余额应为 3600，实际 %d", balance)
	}
	again, err := svc.RefundFull(context.Background(), testActor, "s-5",
		"downgrade_rejected_full_refund", "refund-5")
	if err != nil || again.EntryID != entry.EntryID {
		t.Fatalf("返还幂等异常：%+v err=%v", again, err)
	}
}

// 项目包范围：其他项目的预留不消耗本项目包。
func TestProjectPackScopedToProject(t *testing.T) {
	svc, _, _ := newLedgerService(t)
	if _, err := svc.GrantProjectPack(context.Background(), testActor, "p-1", 3600, "idem-pack-1"); err != nil {
		t.Fatalf("发放项目包失败: %v", err)
	}
	other := reserveInput("s-6")
	other.ProjectID = "p-2"
	if _, err := svc.Reserve(context.Background(), testActor, other); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("其他项目不得消耗本项目包，实际 %v", err)
	}
	entry, err := svc.Reserve(context.Background(), testActor, reserveInput("s-7"))
	if err != nil || entry.EntryType != EntryReserve {
		t.Fatalf("本项目预留应成功：%+v err=%v", entry, err)
	}
}
