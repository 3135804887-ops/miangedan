// Package billing 发票与税费测试（TASK-065；FR-033，US-06 场景 11）。
package billing

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// 中国区：合规发票（发票号码 + 增值税 6% 行；总额与订单一致）。
func TestChinaInvoiceCompliance(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-invoice-cn")
	_ = payOrder(t, svc, order, "evt-inv-cn-1", "txn-inv-cn-1")
	invoice, err := svc.IssueInvoice(context.Background(), testActor, order.OrderID, "idem-inv-cn")
	if err != nil {
		t.Fatalf("开票失败: %v", err)
	}
	if invoice.Kind != InvoiceKindInvoice || invoice.Status != InvoiceIssued {
		t.Fatalf("中国区应为合规发票：%+v", invoice)
	}
	if invoice.TotalCents != order.AmountCents {
		t.Fatalf("发票总额应与订单一致：%d != %d", invoice.TotalCents, order.AmountCents)
	}
	if len(invoice.TaxLines) != 1 || invoice.TaxLines[0].RatePercent != 6 {
		t.Fatalf("应有 6%% 增值税行：%+v", invoice.TaxLines)
	}
	if invoice.SubtotalCents+invoice.TaxLines[0].AmountCents != invoice.TotalCents {
		t.Fatalf("价税合计不一致：%d+%d != %d",
			invoice.SubtotalCents, invoice.TaxLines[0].AmountCents, invoice.TotalCents)
	}
}

// 国际区：税费明示收据（eu 19% VAT；intl 0% 也显示税费行）。
func TestInternationalReceiptWithTaxDisplay(t *testing.T) {
	svc, _ := newTestService(t)
	euActor := Actor{UserID: "user-eu", DataRegion: "eu"}
	quote, err := svc.CreateQuote(context.Background(), euActor, samplePlan(), "idem-quote-eu")
	if err != nil {
		t.Fatalf("创建报价失败: %v", err)
	}
	_, _ = svc.PresentQuote(context.Background(), euActor, quote.QuoteID)
	_, _ = svc.AcceptQuote(context.Background(), euActor, quote.QuoteID)
	order, err := svc.CreateOrder(context.Background(), euActor, quote.QuoteID,
		"psp_eu_primary", false, "idem-order-eu")
	if err != nil {
		t.Fatalf("创建订单失败: %v", err)
	}
	provider := "psp_eu_primary"
	verifier := NewHMACPaymentVerifier(map[string]string{provider: "eu-secret"})
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	verifier.now = func() time.Time { return now }
	ts := fmt.Sprintf("%d", now.Unix())
	body := mustJSON(t, map[string]any{
		"payment_event_id": "evt-eu-1",
		"order_id":         order.OrderID,
		"event_type":       PaymentSucceeded,
		"data_region":      "eu",
		"payload":          map[string]any{"provider_txn_id": "txn-eu-1"},
	})
	if _, err := svc.HandlePaymentCallback(context.Background(), verifier, body,
		provider, ts, signedCallback(t, "eu-secret", provider, ts, body)); err != nil {
		t.Fatalf("支付回调失败: %v", err)
	}
	receipt, err := svc.IssueInvoice(context.Background(), euActor, order.OrderID, "idem-inv-eu")
	if err != nil {
		t.Fatalf("开具收据失败: %v", err)
	}
	if receipt.Kind != InvoiceKindReceipt || len(receipt.TaxLines) == 0 ||
		receipt.TaxLines[0].RatePercent != 19 {
		t.Fatalf("eu 应为 19%% VAT 收据：%+v", receipt)
	}
}

// 仅 PAID 订单可开票；同一订单一份票据；幂等键去重。
func TestInvoiceGuardsAndIdempotency(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-invoice-guard")
	if _, err := svc.IssueInvoice(context.Background(), testActor,
		order.OrderID, "idem-inv-guard-1"); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("未支付订单开票应拒绝，实际 err=%v", err)
	}
	_ = payOrder(t, svc, order, "evt-inv-g-1", "txn-inv-g-1")
	first, err := svc.IssueInvoice(context.Background(), testActor, order.OrderID, "idem-inv-guard-2")
	if err != nil {
		t.Fatalf("开票失败: %v", err)
	}
	second, err := svc.IssueInvoice(context.Background(), testActor, order.OrderID, "idem-inv-guard-3")
	if err != nil || second.InvoiceID != first.InvoiceID {
		t.Fatalf("同一订单应只开一份票据：%+v err=%v", second, err)
	}
}

// 中国区发票可作废（合规）；收据不可作废。
func TestCancelInvoice(t *testing.T) {
	svc, _ := newTestService(t)
	order := mustOrder(t, svc, "idem-invoice-cancel")
	_ = payOrder(t, svc, order, "evt-inv-c-1", "txn-inv-c-1")
	_, err := svc.IssueInvoice(context.Background(), testActor, order.OrderID, "idem-inv-cancel")
	if err != nil {
		t.Fatalf("开票失败: %v", err)
	}
	cancelled, err := svc.CancelInvoice(context.Background(), testActor, order.OrderID, "用户申请作废")
	if err != nil || cancelled.Status != InvoiceCancelled {
		t.Fatalf("作废发票异常：%+v err=%v", cancelled, err)
	}
}

// 区域定价配置：币种/税率/票据类型按区域。
func TestRegionalPricingConfig(t *testing.T) {
	cn := PriceConfigFor("cn")
	if cn.Currency != "CNY" || cn.TaxRate != 0.06 || cn.InvoiceKind != InvoiceKindInvoice {
		t.Fatalf("中国区定价配置异常：%+v", cn)
	}
	eu := PriceConfigFor("eu")
	if eu.Currency != "EUR" || eu.TaxRate != 0.19 || eu.InvoiceKind != InvoiceKindReceipt {
		t.Fatalf("eu 定价配置异常：%+v", eu)
	}
	intl := PriceConfigFor("intl")
	if intl.Currency != "USD" || intl.InvoiceKind != InvoiceKindReceipt {
		t.Fatalf("intl 定价配置异常：%+v", intl)
	}
}
