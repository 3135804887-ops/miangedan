// Package billing 支付订单与回调测试（TASK-062；FR-033，US-06 场景 4）。
package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

const testProvider = "psp_cn_primary"

func signedCallback(t *testing.T, secret, provider, ts string, body []byte) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(provider))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(ts))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func newCallbackVerifier(t *testing.T, secret string) (*HMACPaymentVerifier, time.Time) {
	t.Helper()
	verifier := NewHMACPaymentVerifier(map[string]string{testProvider: secret})
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	verifier.now = func() time.Time { return now }
	return verifier, now
}

// 创建已接受报价对应的订单（幂等键去重；支付处理中禁止重复发起扣款）。
func TestCreateOrderIdempotentAndPendingBlocksCharge(t *testing.T) {
	svc, _ := newTestService(t)
	quote, err := svc.CreateQuote(context.Background(), testActor, samplePlan(), "idem-quote-pay")
	if err != nil {
		t.Fatalf("创建报价失败: %v", err)
	}
	if _, err := svc.PresentQuote(context.Background(), testActor, quote.QuoteID); err != nil {
		t.Fatalf("呈现报价失败: %v", err)
	}
	if _, err := svc.AcceptQuote(context.Background(), testActor, quote.QuoteID); err != nil {
		t.Fatalf("接受报价失败: %v", err)
	}
	order, err := svc.CreateOrder(context.Background(), testActor,
		quote.QuoteID, "psp_cn_primary", false, "idem-order-1")
	if err != nil {
		t.Fatalf("创建订单失败: %v", err)
	}
	if order.Status != OrderPending {
		t.Fatalf("订单初始状态应为 PAYMENT_PENDING，实际 %s", order.Status)
	}
	again, err := svc.CreateOrder(context.Background(), testActor,
		quote.QuoteID, "psp_cn_primary", false, "idem-order-1")
	if err != nil || again.OrderID != order.OrderID {
		t.Fatalf("订单幂等异常：%+v err=%v", again, err)
	}
	if _, err := svc.InitiateCharge(context.Background(), testActor, order.OrderID); !errors.Is(err, ErrPaymentPendingBlocked) {
		t.Fatalf("处理中订单应禁止重复扣款，实际 err=%v", err)
	}
}

// 支付回调：验签通过后置 PAID 并发放一次权益；同一 payment_event_id 重复无副作用。
func TestPaymentCallbackSettlesAndDedup(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-order-settle")
	verifier, now := newCallbackVerifier(t, "secret-1")
	ts := fmt.Sprintf("%d", now.Unix())
	body := mustJSON(t, map[string]any{
		"payment_event_id": "evt-settle-1",
		"order_id":         order.OrderID,
		"event_type":       PaymentSucceeded,
		"data_region":      "cn",
		"payload":          map[string]any{"provider_txn_id": "txn-settle-1"},
	})
	sig := signedCallback(t, "secret-1", testProvider, ts, body)
	got, err := svc.HandlePaymentCallback(context.Background(), verifier, body,
		testProvider, ts, sig)
	if err != nil {
		t.Fatalf("支付回调失败: %v", err)
	}
	if got.Status != OrderPaid {
		t.Fatalf("回调后订单应为 PAID，实际 %s", got.Status)
	}
	balance, _ := svc.Balance(context.Background(), testActor)
	if balance != 75*60 {
		t.Fatalf("支付成功应发放 4500 秒权益，实际 %d", balance)
	}
	// 同一事件重复回调：无副作用。
	again, err := svc.HandlePaymentCallback(context.Background(), verifier, body,
		testProvider, ts, sig)
	if err != nil || again.Status != OrderPaid {
		t.Fatalf("重复回调异常：%+v err=%v", again, err)
	}
	balance2, _ := svc.Balance(context.Background(), testActor)
	if balance2 != balance {
		t.Fatalf("重复回调不得重复发放权益：%d -> %d", balance, balance2)
	}
}

// 重复扣款（同一订单第二笔成功扣款）：自动原路退回 + Incident，权益不重复发放。
func TestDuplicateChargeAutoRefund(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-order-dup")
	verifier, now := newCallbackVerifier(t, "secret-2")
	ts := fmt.Sprintf("%d", now.Unix())
	first := mustJSON(t, map[string]any{
		"payment_event_id": "evt-dup-1",
		"order_id":         order.OrderID,
		"event_type":       PaymentSucceeded,
		"data_region":      "cn",
		"payload":          map[string]any{"provider_txn_id": "txn-dup-1"},
	})
	if _, err := svc.HandlePaymentCallback(context.Background(), verifier, first,
		testProvider, ts, signedCallback(t, "secret-2", testProvider, ts, first)); err != nil {
		t.Fatalf("首次支付回调失败: %v", err)
	}
	balance, _ := svc.Balance(context.Background(), testActor)
	second := mustJSON(t, map[string]any{
		"payment_event_id": "evt-dup-2",
		"order_id":         order.OrderID,
		"event_type":       PaymentSucceeded,
		"data_region":      "cn",
		"payload":          map[string]any{"provider_txn_id": "txn-dup-2"},
	})
	got, err := svc.HandlePaymentCallback(context.Background(), verifier, second,
		testProvider, ts, signedCallback(t, "secret-2", testProvider, ts, second))
	if err != nil {
		t.Fatalf("重复扣款回调处理失败: %v", err)
	}
	if got.RefundedCents != got.AmountCents {
		t.Fatalf("重复扣款应全额退回：refunded=%d amount=%d", got.RefundedCents, got.AmountCents)
	}
	refunds, _ := svc.store.ListRefundsByOrder("cn", order.OrderID)
	if len(refunds) != 1 || refunds[0].Status != Refunded {
		t.Fatalf("应有 1 笔已退款的重复扣款记录：%+v", refunds)
	}
	incidents, _ := svc.store.ListIncidents("cn")
	found := false
	for _, inc := range incidents {
		if inc.Kind == IncidentDuplicateCharge {
			found = true
		}
	}
	if !found {
		t.Fatal("重复扣款应写入 Incident")
	}
	balance2, _ := svc.Balance(context.Background(), testActor)
	if balance2 != balance {
		t.Fatalf("重复扣款不得重复发放权益：%d -> %d", balance, balance2)
	}
}

// 签名错误与时间戳超出重放窗口均拒绝（fail-closed）。
func TestCallbackRejectsBadSignatureAndReplay(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-order-sig")
	verifier, now := newCallbackVerifier(t, "secret-3")
	ts := fmt.Sprintf("%d", now.Unix())
	body := mustJSON(t, map[string]any{
		"payment_event_id": "evt-sig-1",
		"order_id":         order.OrderID,
		"event_type":       PaymentSucceeded,
		"data_region":      "cn",
	})
	if _, err := svc.HandlePaymentCallback(context.Background(), verifier, body,
		testProvider, ts, "deadbeef"); !errors.Is(err, ErrPaymentSignatureInvalid) {
		t.Fatalf("错误签名应拒绝，实际 err=%v", err)
	}
	oldTS := fmt.Sprintf("%d", now.Add(-10*time.Minute).Unix())
	sig := signedCallback(t, "secret-3", testProvider, oldTS, body)
	if _, err := svc.HandlePaymentCallback(context.Background(), verifier, body,
		testProvider, oldTS, sig); !errors.Is(err, ErrPaymentReplay) {
		t.Fatalf("超出重放窗口应拒绝，实际 err=%v", err)
	}
}

// 支付成功未到账：保持 PAYMENT_PENDING 且不发放权益；对账任务按流水号收敛后补记。
func TestSuccessNotCreditedStaysPendingThenReconcile(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-order-recon")
	verifier, now := newCallbackVerifier(t, "secret-4")
	ts := fmt.Sprintf("%d", now.Unix())
	body := mustJSON(t, map[string]any{
		"payment_event_id": "evt-recon-1",
		"order_id":         order.OrderID,
		"event_type":       PaymentSucceeded,
		"data_region":      "cn",
		"payload":          map[string]any{},
	})
	got, err := svc.HandlePaymentCallback(context.Background(), verifier, body,
		testProvider, ts, signedCallback(t, "secret-4", testProvider, ts, body))
	if err != nil {
		t.Fatalf("回调处理失败: %v", err)
	}
	if got.Status != OrderPending {
		t.Fatalf("未到账应保持 PAYMENT_PENDING，实际 %s", got.Status)
	}
	balance, _ := svc.Balance(context.Background(), testActor)
	if balance != 0 {
		t.Fatalf("未到账不得发放权益，实际余额 %d", balance)
	}
	reconciled, err := svc.ReconcileOrder(context.Background(), testActor,
		order.OrderID, "txn-recon-1", "recon-1")
	if err != nil {
		t.Fatalf("对账收敛失败: %v", err)
	}
	if reconciled.Status != OrderPaid {
		t.Fatalf("对账后应为 PAID，实际 %s", reconciled.Status)
	}
	balance, _ = svc.Balance(context.Background(), testActor)
	if balance != 75*60 {
		t.Fatalf("对账收敛应补记 4500 秒，实际 %d", balance)
	}
}

// 支付失败保留计划；可重新发起扣款。
func TestPaymentFailedThenReinitiate(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-order-fail")
	verifier, now := newCallbackVerifier(t, "secret-5")
	ts := fmt.Sprintf("%d", now.Unix())
	body := mustJSON(t, map[string]any{
		"payment_event_id": "evt-fail-1",
		"order_id":         order.OrderID,
		"event_type":       PaymentFailed,
		"data_region":      "cn",
	})
	got, err := svc.HandlePaymentCallback(context.Background(), verifier, body,
		testProvider, ts, signedCallback(t, "secret-5", testProvider, ts, body))
	if err != nil || got.Status != OrderFailed {
		t.Fatalf("支付失败回调异常：%+v err=%v", got, err)
	}
	retry, err := svc.InitiateCharge(context.Background(), testActor, order.OrderID)
	if err != nil || retry.Status != OrderPending {
		t.Fatalf("失败订单应可重新发起，实际 %+v err=%v", retry, err)
	}
}

func mustOrder(t *testing.T, svc *Service, idemKey string) Order {
	t.Helper()
	quote, err := svc.CreateQuote(context.Background(), testActor, samplePlan(), idemKey+"-quote")
	if err != nil {
		t.Fatalf("创建报价失败: %v", err)
	}
	if _, err := svc.PresentQuote(context.Background(), testActor, quote.QuoteID); err != nil {
		t.Fatalf("呈现报价失败: %v", err)
	}
	if _, err := svc.AcceptQuote(context.Background(), testActor, quote.QuoteID); err != nil {
		t.Fatalf("接受报价失败: %v", err)
	}
	order, err := svc.CreateOrder(context.Background(), testActor,
		quote.QuoteID, testProvider, false, idemKey)
	if err != nil {
		t.Fatalf("创建订单失败: %v", err)
	}
	return order
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("序列化回调失败: %v", err)
	}
	return body
}
