// Package billing 提供发票与税费：中国区合规发票、国际区税费明示收据、
// 区域定价配置（TASK-065；FR-033，US-06 场景 11；BILLING-STATE-MACHINE §7）。
package billing

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// 票据类型：中国区合规发票；国际区税费明示的收据。
const (
	InvoiceKindInvoice = "invoice"
	InvoiceKindReceipt = "receipt"
	InvoiceIssued      = "issued"
	InvoiceCancelled   = "cancelled"
)

// TaxLine 为一行税费（国际区税费明示）。
type TaxLine struct {
	Name        string
	RatePercent float64
	AmountCents int
}

// Invoice 为发票（cn）或收据（eu/intl）；区域定价配置决定币种与税率。
type Invoice struct {
	InvoiceID     string
	OrderID       string
	UserID        string
	Kind          string
	Number        string
	Currency      string
	SubtotalCents int
	TaxLines      []TaxLine
	TotalCents    int
	Status        string
	DataRegion    string
	CreatedAt     time.Time
	CancelledAt   *time.Time
}

// IssueInvoice 为 PAID 订单开票/开收据（幂等；同一订单只开一份）。
// 中国区：合规发票（发票号码 + 增值税行）；国际区：税费明示收据。
func (s *Service) IssueInvoice(
	_ context.Context, actor Actor, orderID, idemKey string,
) (Invoice, error) {
	if err := validateActor(actor); err != nil {
		return Invoice{}, err
	}
	if strings.TrimSpace(orderID) == "" || strings.TrimSpace(idemKey) == "" {
		return Invoice{}, fmt.Errorf("%w: order_id 与幂等键必填", ErrInvalidInput)
	}
	if cached, err := s.store.GetInvoiceByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Invoice{}, err
	}
	if existing, err := s.store.GetInvoiceByOrder(actor.DataRegion, orderID); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Invoice{}, err
	}
	order, err := s.getOwnedOrder(actor, orderID)
	if err != nil {
		return Invoice{}, err
	}
	if order.Status != OrderPaid {
		return Invoice{}, fmt.Errorf("%w: 仅 PAID 订单可开票（当前 %s）", ErrStateConflict, order.Status)
	}
	price := PriceConfigFor(order.DataRegion)
	subtotal := roundCents(float64(order.AmountCents) / (1 + price.TaxRate))
	tax := order.AmountCents - subtotal
	kind := price.InvoiceKind
	if kind == "" {
		kind = InvoiceKindReceipt
	}
	invoice := Invoice{
		InvoiceID:     newID(),
		OrderID:       order.OrderID,
		UserID:        order.UserID,
		Kind:          kind,
		Number:        s.newInvoiceNumber(kind, order.DataRegion, order.OrderID),
		Currency:      order.Currency,
		SubtotalCents: subtotal,
		TaxLines: []TaxLine{{
			Name:        price.TaxLabel,
			RatePercent: price.TaxRate * 100,
			AmountCents: tax,
		}},
		TotalCents: order.AmountCents,
		Status:     InvoiceIssued,
		DataRegion: order.DataRegion,
		CreatedAt:  s.now().UTC(),
	}
	if err := s.store.SaveInvoice(invoice, idemKey); err != nil {
		return Invoice{}, err
	}
	return invoice, nil
}

// GetInvoice 按订单查询票据。
func (s *Service) GetInvoice(_ context.Context, actor Actor, orderID string) (Invoice, error) {
	if err := validateActor(actor); err != nil {
		return Invoice{}, err
	}
	return s.store.GetInvoiceByOrder(actor.DataRegion, orderID)
}

// ListInvoices 列出用户票据。
func (s *Service) ListInvoices(_ context.Context, actor Actor) ([]Invoice, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	return s.store.ListInvoicesByUser(actor.DataRegion, actor.UserID)
}

// CancelInvoice 作废发票（中国区合规：作废后不得再次开票，需红冲流程；仅 issued 可作废）。
func (s *Service) CancelInvoice(
	_ context.Context, actor Actor, orderID, reason string,
) (Invoice, error) {
	if err := validateActor(actor); err != nil {
		return Invoice{}, err
	}
	invoice, err := s.store.GetInvoiceByOrder(actor.DataRegion, orderID)
	if err != nil {
		return Invoice{}, err
	}
	if invoice.UserID != actor.UserID {
		return Invoice{}, ErrNotFound
	}
	if invoice.Kind != InvoiceKindInvoice {
		return Invoice{}, fmt.Errorf("%w: 收据不可作废", ErrStateConflict)
	}
	if invoice.Status != InvoiceIssued {
		return Invoice{}, fmt.Errorf("%w: 当前状态不可作废（%s）", ErrStateConflict, invoice.Status)
	}
	now := s.now().UTC()
	invoice.Status = InvoiceCancelled
	invoice.CancelledAt = &now
	invoice.Number += "-VOID(" + reason + ")"
	if err := s.store.UpdateInvoice(invoice); err != nil {
		return Invoice{}, err
	}
	return invoice, nil
}

// newInvoiceNumber 生成发票/收据号码（区域化；确定性可测）。
func (s *Service) newInvoiceNumber(kind, dataRegion, orderID string) string {
	prefix := "RCP"
	if kind == InvoiceKindInvoice {
		prefix = "MGD-INV"
	}
	short := strings.ToUpper(strings.ReplaceAll(orderID, "-", ""))
	if len(short) > 10 {
		short = short[:10]
	}
	return fmt.Sprintf("%s-%s-%s-%d", prefix, strings.ToUpper(dataRegion), short, s.now().UTC().Year())
}

func roundCents(v float64) int {
	return int(math.Round(v))
}
