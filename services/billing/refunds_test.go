// Package billing 退款与补偿测试（TASK-063；FR-033，US-06 场景 3）。
package billing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

// 小额用户退款自动执行；账本冲正条目记录原因。
func TestSmallRefundAutoExecutes(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-refund-small")
	paid := payOrder(t, svc, order, "evt-rs-1", "txn-rs-1")
	if paid.Status != OrderPaid {
		t.Fatalf("订单应为 PAID，实际 %s", paid.Status)
	}
	refund, err := svc.RequestRefund(context.Background(), testActor,
		RefundInput{OrderID: order.OrderID, Reason: "未使用项目退款", Kind: RefundKindUserRequest},
		"idem-refund-1")
	if err != nil {
		t.Fatalf("申请退款失败: %v", err)
	}
	if refund.Status != Refunded {
		t.Fatalf("小额退款应自动执行，实际 %s", refund.Status)
	}
	ledger, _ := svc.GetLedger(context.Background(), testActor, order.ProjectID)
	hasReason := false
	for _, e := range ledger {
		if e.EntryType == EntryRefund && e.Reason != "" {
			hasReason = true
		}
	}
	if !hasReason {
		t.Fatal("账本应含退款冲正条目并记录原因")
	}
}

// 大额退款双人审批：同一审批人重复无效；两名不同审批人后自动执行。
func TestLargeRefundRequiresTwoApprovers(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-refund-large")
	paid := payOrder(t, svc, order, "evt-rl-1", "txn-rl-1")
	paid.AmountCents = LargeRefundThresholdCents + 1
	if err := svc.store.UpdateOrder(paid); err != nil {
		t.Fatalf("更新订单金额失败: %v", err)
	}
	refund, err := svc.RequestRefund(context.Background(), testActor,
		RefundInput{OrderID: order.OrderID, Reason: "大额退款", Kind: RefundKindUserRequest},
		"idem-refund-large-1")
	if err != nil {
		t.Fatalf("申请退款失败: %v", err)
	}
	if refund.Status != RefundReviewing {
		t.Fatalf("大额退款应进入双人审批，实际 %s", refund.Status)
	}
	approverA := Actor{UserID: "approver-a", DataRegion: "cn"}
	approverB := Actor{UserID: "approver-b", DataRegion: "cn"}
	afterA, err := svc.ApproveRefund(context.Background(), approverA, refund.RefundID, approverA.UserID)
	if err != nil {
		t.Fatalf("第一人审批失败: %v", err)
	}
	if afterA.Status != RefundReviewing {
		t.Fatalf("一人审批后应仍为审批中，实际 %s", afterA.Status)
	}
	dup, err := svc.ApproveRefund(context.Background(), approverA, refund.RefundID, approverA.UserID)
	if err != nil {
		t.Fatalf("重复审批报错: %v", err)
	}
	if len(dup.ApproverPair) != 1 {
		t.Fatalf("同一审批人不应重复计入：%v", dup.ApproverPair)
	}
	afterB, err := svc.ApproveRefund(context.Background(), approverB, refund.RefundID, approverB.UserID)
	if err != nil {
		t.Fatalf("第二人审批失败: %v", err)
	}
	if afterB.Status != Refunded {
		t.Fatalf("双人审批后应执行退款，实际 %s", afterB.Status)
	}
}

// 人工补偿必须双人审批；系统故障自动全额执行；本人不可自批。
func TestCompensationSystemFaultAndSelfApproval(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-refund-comp")
	_ = payOrder(t, svc, order, "evt-rc-1", "txn-rc-1")
	comp, err := svc.RequestRefund(context.Background(), testActor,
		RefundInput{OrderID: order.OrderID, Reason: "人工补偿", Kind: RefundKindCompensation},
		"idem-comp-1")
	if err != nil || comp.Status != RefundReviewing {
		t.Fatalf("补偿应双人审批：%+v err=%v", comp, err)
	}
	if _, err := svc.ApproveRefund(context.Background(), testActor, comp.RefundID, testActor.UserID); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("本人退款不可自批，实际 err=%v", err)
	}
	fault, err := svc.RequestRefund(context.Background(), testActor,
		RefundInput{OrderID: order.OrderID, Reason: "系统故障补偿", Kind: RefundKindSystemFault},
		"idem-fault-1")
	if err != nil || fault.Status != Refunded {
		t.Fatalf("系统故障应自动全额退款：%+v err=%v", fault, err)
	}
}

// 拒绝后说明原因并可申诉（重新进入审批）。
func TestRejectThenAppeal(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-refund-reject")
	paid := payOrder(t, svc, order, "evt-rr-1", "txn-rr-1")
	paid.AmountCents = LargeRefundThresholdCents + 1
	_ = svc.store.UpdateOrder(paid)
	refund, err := svc.RequestRefund(context.Background(), testActor,
		RefundInput{OrderID: order.OrderID, Reason: "大额申请", Kind: RefundKindUserRequest},
		"idem-reject-1")
	if err != nil {
		t.Fatalf("申请退款失败: %v", err)
	}
	rejected, err := svc.RejectRefund(context.Background(), testActor, refund.RefundID, "不符合退款条件")
	if err != nil || rejected.Status != RefundRejected || rejected.RejectReason == "" {
		t.Fatalf("拒绝退款异常：%+v err=%v", rejected, err)
	}
	appealed, err := svc.AppealRefund(context.Background(), testActor, refund.RefundID, "补充材料")
	if err != nil || appealed.Status != RefundRequested {
		t.Fatalf("申诉异常：%+v err=%v", appealed, err)
	}
}

// 并发双人审批：恰好两名审批人，退款只执行一次（幂等）。
func TestConcurrentApprovals(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-refund-conc")
	paid := payOrder(t, svc, order, "evt-rn-1", "txn-rn-1")
	paid.AmountCents = LargeRefundThresholdCents + 1
	_ = svc.store.UpdateOrder(paid)
	refund, err := svc.RequestRefund(context.Background(), testActor,
		RefundInput{OrderID: order.OrderID, Reason: "大额", Kind: RefundKindUserRequest},
		"idem-conc-1")
	if err != nil {
		t.Fatalf("申请退款失败: %v", err)
	}
	approvers := []Actor{
		{UserID: "conc-a", DataRegion: "cn"},
		{UserID: "conc-b", DataRegion: "cn"},
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range approvers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = svc.ApproveRefund(context.Background(), approvers[i], refund.RefundID, approvers[i].UserID)
		}(i)
	}
	wg.Wait()
	for i, e := range errs {
		if e != nil {
			t.Fatalf("并发审批 %d 失败: %v", i, e)
		}
	}
	final, err := svc.store.GetRefundByID("cn", refund.RefundID)
	if err != nil {
		t.Fatalf("查询退款失败: %v", err)
	}
	if final.Status != Refunded || len(final.ApproverPair) != 2 {
		t.Fatalf("并发审批结果异常：status=%s pair=%v", final.Status, final.ApproverPair)
	}
	ledger, _ := svc.GetLedger(context.Background(), testActor, order.ProjectID)
	refundEntries := 0
	for _, e := range ledger {
		if e.EntryType == EntryRefund && e.Reason != "" {
			refundEntries++
		}
	}
	if refundEntries != 1 {
		t.Fatalf("退款冲正条目应恰好 1 条，实际 %d", refundEntries)
	}
}

// 未支付订单不可退款；幂等键重复返回同一退款。
func TestRefundGuardsAndIdempotency(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-refund-guard")
	if _, err := svc.RequestRefund(context.Background(), testActor,
		RefundInput{OrderID: order.OrderID, Reason: "未支付", Kind: RefundKindUserRequest},
		"idem-guard-1"); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("未支付订单退款应拒绝，实际 err=%v", err)
	}
	_ = payOrder(t, svc, order, "evt-rg-1", "txn-rg-1")
	first, err := svc.RequestRefund(context.Background(), testActor,
		RefundInput{OrderID: order.OrderID, Reason: "退款", Kind: RefundKindUserRequest},
		"idem-guard-2")
	if err != nil {
		t.Fatalf("申请退款失败: %v", err)
	}
	second, err := svc.RequestRefund(context.Background(), testActor,
		RefundInput{OrderID: order.OrderID, Reason: "退款", Kind: RefundKindUserRequest},
		"idem-guard-2")
	if err != nil || second.RefundID != first.RefundID {
		t.Fatalf("退款幂等异常：%+v err=%v", second, err)
	}
}

// payOrder 通过合法回调将订单置为 PAID。
func payOrder(t *testing.T, svc *Service, order Order, eventID, txnID string) Order {
	t.Helper()
	verifier, now := newCallbackVerifier(t, "pay-secret")
	ts := fmt.Sprintf("%d", now.Unix())
	body := mustJSON(t, map[string]any{
		"payment_event_id": eventID,
		"order_id":         order.OrderID,
		"event_type":       PaymentSucceeded,
		"data_region":      "cn",
		"payload":          map[string]any{"provider_txn_id": txnID},
	})
	paid, err := svc.HandlePaymentCallback(context.Background(), verifier, body,
		testProvider, ts, signedCallback(t, "pay-secret", testProvider, ts, body))
	if err != nil {
		t.Fatalf("支付回调失败: %v", err)
	}
	return paid
}
