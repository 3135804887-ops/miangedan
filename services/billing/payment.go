// Package billing 提供区域化支付集成：订单、签名回调、重放防护与幂等扣款
// （TASK-062；FR-033，US-06 场景 4；BILLING-STATE-MACHINE §5.2/§8/§9）。
// 红线：重复扣费为 0；支付状态不明保持处理中并禁止重复发起扣款；
// 同一订单只记一次权益和一次扣款；重复扣款自动识别并原路退回。
package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// 订单状态（BILLING-STATE-MACHINE 5.2）。
const (
	OrderCreated   = "ORDER_CREATED"
	OrderPending   = "PAYMENT_PENDING"
	OrderPaid      = "PAID"
	OrderFailed    = "PAYMENT_FAILED"
	OrderTimeout   = "PAYMENT_TIMEOUT"
	OrderCancelled = "ORDER_CANCELLED"
)

// 支付事件类型（openapi paymentCallback event_type 对齐）。
const (
	PaymentSucceeded = "payment_succeeded"
	PaymentFailed    = "payment_failed"
	RefundSucceeded  = "refund_succeeded"
)

// 事故类型。
const (
	IncidentDuplicateCharge = "duplicate_charge"
	IncidentPaymentFault    = "payment_fault"
)

// 回调验签重放窗口（±5 分钟）。
const callbackReplayWindow = 5 * time.Minute

// 支付相关错误（可被 HTTP 层映射为 401/409）。
var (
	ErrPaymentSignatureInvalid = errors.New("payment signature invalid")
	ErrPaymentReplay           = errors.New("payment callback replay")
	ErrPaymentPendingBlocked   = errors.New("payment pending: no duplicate charge")
)

// PaymentEventInput 为支付服务商回调体（服务端到服务端）。
type PaymentEventInput struct {
	PaymentEventID string         `json:"payment_event_id"`
	OrderID        string         `json:"order_id"`
	EventType      string         `json:"event_type"`
	DataRegion     string         `json:"data_region"`
	Payload        map[string]any `json:"payload"`
}

// PaymentVerifier 校验支付回调签名与时间戳重放窗口（供应商中立适配点）。
type PaymentVerifier interface {
	Verify(provider, timestamp, signature string, body []byte) error
}

// HMACPaymentVerifier 为 HMAC-SHA256 签名校验器；密钥按供应商从环境/密钥系统注入。
type HMACPaymentVerifier struct {
	secrets map[string][]byte
	now     func() time.Time
	window  time.Duration
}

// NewHMACPaymentVerifier 创建校验器（provider → 签名密钥）。
func NewHMACPaymentVerifier(secrets map[string]string) *HMACPaymentVerifier {
	raw := make(map[string][]byte, len(secrets))
	for provider, secret := range secrets {
		raw[provider] = []byte(secret)
	}
	return &HMACPaymentVerifier{
		secrets: raw,
		now:     time.Now,
		window:  callbackReplayWindow,
	}
}

// Verify 校验签名与时间戳重放窗口（fail-closed）。
func (v *HMACPaymentVerifier) Verify(provider, timestamp, signature string, body []byte) error {
	secret, ok := v.secrets[provider]
	if !ok {
		return fmt.Errorf("%w: 未注册供应商 %q", ErrPaymentSignatureInvalid, provider)
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: 时间戳非法", ErrPaymentSignatureInvalid)
	}
	now := v.now().Unix()
	delta := now - ts
	if delta < 0 {
		delta = -delta
	}
	if delta > int64(v.window.Seconds()) {
		return fmt.Errorf("%w: 时间戳超出重放窗口", ErrPaymentReplay)
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(provider))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("\n"))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(strings.ToLower(strings.TrimSpace(signature)))) {
		return fmt.Errorf("%w: 签名不匹配", ErrPaymentSignatureInvalid)
	}
	return nil
}

// CreateOrder 创建订单（PAYMENT_PENDING；幂等键去重，重复创建返回同一订单）。
func (s *Service) CreateOrder(
	_ context.Context, actor Actor, quoteID, paymentMethod string, autoRenewConsent bool, idemKey string,
) (Order, error) {
	if err := validateActor(actor); err != nil {
		return Order{}, err
	}
	if strings.TrimSpace(quoteID) == "" || strings.TrimSpace(paymentMethod) == "" ||
		strings.TrimSpace(idemKey) == "" {
		return Order{}, fmt.Errorf("%w: quote_id、payment_method 与幂等键必填", ErrInvalidInput)
	}
	if cached, err := s.store.GetOrderByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Order{}, err
	}
	quote, err := s.store.GetQuote(actor.DataRegion, quoteID)
	if err != nil {
		return Order{}, err
	}
	if quote.Status != QuoteAccepted {
		return Order{}, fmt.Errorf("%w: 仅 QUOTE_ACCEPTED 报价可创建订单（当前 %s）",
			ErrStateConflict, quote.Status)
	}
	now := s.now().UTC()
	order := Order{
		OrderID:          newID(),
		UserID:           actor.UserID,
		QuoteID:          quote.QuoteID,
		ProjectID:        quote.ProjectID,
		Status:           OrderPending,
		AmountCents:      quote.AmountCents,
		Currency:         quote.Currency,
		PaymentMethod:    paymentMethod,
		ProgressNote:     "支付处理中：状态不明时禁止重复扣款",
		AutoRenewConsent: autoRenewConsent,
		DataRegion:       actor.DataRegion,
		CreatedAt:        now,
		IdempotencyKey:   idemKey,
	}
	if err := s.store.SaveOrder(order, idemKey); err != nil {
		return Order{}, err
	}
	return order, nil
}

// GetOrder 返回订单（含处理中状态的真实进度说明）。
func (s *Service) GetOrder(_ context.Context, actor Actor, orderID string) (Order, error) {
	if err := validateActor(actor); err != nil {
		return Order{}, err
	}
	return s.getOwnedOrder(actor, orderID)
}

// ListOrders 列出用户订单（逐笔账单入口之一）。
func (s *Service) ListOrders(_ context.Context, actor Actor) ([]Order, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	return s.store.ListOrdersByUser(actor.DataRegion, actor.UserID)
}

// InitiateCharge 发起支付（幂等入口）：支付状态不明（PAYMENT_PENDING/PAYMENT_TIMEOUT）
// 禁止重复发起扣款；支付失败保留计划，可重新发起。
func (s *Service) InitiateCharge(_ context.Context, actor Actor, orderID string) (Order, error) {
	order, err := s.getOwnedOrder(actor, orderID)
	if err != nil {
		return Order{}, err
	}
	switch order.Status {
	case OrderPending, OrderTimeout:
		return Order{}, fmt.Errorf("%w: 订单 %s 支付处理中", ErrPaymentPendingBlocked, orderID)
	case OrderPaid:
		return Order{}, fmt.Errorf("%w: 订单已支付", ErrStateConflict)
	case OrderCancelled:
		return Order{}, fmt.Errorf("%w: 订单已取消", ErrStateConflict)
	case OrderCreated, OrderFailed:
		order.Status = OrderPending
		order.ProgressNote = "支付处理中：状态不明时禁止重复扣款"
		if err := s.store.UpdateOrder(order); err != nil {
			return Order{}, err
		}
	}
	return order, nil
}

// HandlePaymentCallback 处理支付服务商回调：验签 + 时间戳重放窗口 + payment_event_id 去重。
// 重复事件无副作用；重复扣款自动识别并原路退回（写 Incident + 通知记录）。
func (s *Service) HandlePaymentCallback(
	_ context.Context, verifier PaymentVerifier, body []byte, provider, timestamp, signature string,
) (Order, error) {
	if verifier == nil || len(body) == 0 {
		return Order{}, fmt.Errorf("%w: 缺少验签器或回调体", ErrInvalidInput)
	}
	if err := verifier.Verify(provider, timestamp, signature, body); err != nil {
		return Order{}, err
	}
	var in PaymentEventInput
	if err := json.Unmarshal(body, &in); err != nil {
		return Order{}, fmt.Errorf("%w: 回调体 JSON 非法: %v", ErrInvalidInput, err)
	}
	if strings.TrimSpace(in.PaymentEventID) == "" || strings.TrimSpace(in.OrderID) == "" ||
		strings.TrimSpace(in.DataRegion) == "" {
		return Order{}, fmt.Errorf("%w: 回调缺少 payment_event_id/order_id/data_region", ErrInvalidInput)
	}
	if in.EventType != PaymentSucceeded && in.EventType != PaymentFailed && in.EventType != RefundSucceeded {
		return Order{}, fmt.Errorf("%w: 未知事件类型 %q", ErrInvalidInput, in.EventType)
	}
	if processed, err := s.store.GetPaymentEvent(provider, in.PaymentEventID); err == nil {
		// 同一 payment_event_id 重复回调：只处理一次，返回当前订单（无副作用）。
		return s.store.GetOrderByID(processed.DataRegion, in.OrderID)
	} else if !errors.Is(err, ErrNotFound) {
		return Order{}, err
	}
	order, err := s.store.GetOrderByID(in.DataRegion, in.OrderID)
	if err != nil {
		return Order{}, err
	}
	hash := contentHash(body)
	switch in.EventType {
	case PaymentSucceeded:
		order, err = s.applyPaymentSucceeded(order, in)
		if err != nil {
			return Order{}, err
		}
	case PaymentFailed:
		if order.Status != OrderPaid {
			order.Status = OrderFailed
			order.ProgressNote = "支付失败：计划已保留，可重新支付"
			if err := s.store.UpdateOrder(order); err != nil {
				return Order{}, err
			}
		}
	case RefundSucceeded:
		order, err = s.applyRefundSucceeded(order, in)
		if err != nil {
			return Order{}, err
		}
	}
	event := PaymentEvent{
		PaymentEventID: in.PaymentEventID,
		Provider:       provider,
		OrderID:        order.OrderID,
		EventType:      in.EventType,
		PayloadHash:    hash,
		DataRegion:     order.DataRegion,
		ProcessedAt:    s.now().UTC(),
	}
	if err := s.store.SavePaymentEvent(event); err != nil {
		return Order{}, err
	}
	return order, nil
}

// ReconcileOrder 对账任务按支付侧流水号收敛：支付成功未到账的订单补记权益与扣款
// （幂等；同一订单只记一次）。
func (s *Service) ReconcileOrder(
	_ context.Context, actor Actor, orderID, providerTxnID, idemKey string,
) (Order, error) {
	order, err := s.getOwnedOrder(actor, orderID)
	if err != nil {
		return Order{}, err
	}
	if strings.TrimSpace(providerTxnID) == "" {
		return Order{}, fmt.Errorf("%w: provider_txn_id 必填", ErrInvalidInput)
	}
	if order.Status == OrderPaid {
		return order, nil
	}
	if order.Status != OrderPending {
		return Order{}, fmt.Errorf("%w: 仅 PAYMENT_PENDING 可对账收敛（当前 %s）",
			ErrStateConflict, order.Status)
	}
	order, err = s.markPaidAndGrant(order, providerTxnID, idemKey)
	if err != nil {
		return Order{}, err
	}
	return order, nil
}

// applyPaymentSucceeded 处理支付成功：首次成功记一次权益和一次扣款；
// 已支付再收到成功事件视为重复扣款，自动原路退回并写 Incident。
func (s *Service) applyPaymentSucceeded(order Order, in PaymentEventInput) (Order, error) {
	if order.Status == OrderPaid {
		return s.autoRefundDuplicateCharge(order, in.PaymentEventID)
	}
	txnID, _ := in.Payload["provider_txn_id"].(string)
	if strings.TrimSpace(txnID) == "" {
		// 支付成功未到账：保持处理中，由对账任务按流水号收敛，禁止重复扣款。
		order.Status = OrderPending
		order.ProgressNote = "支付成功未到账：对账中，不重复扣款"
		if err := s.store.UpdateOrder(order); err != nil {
			return Order{}, err
		}
		return order, nil
	}
	return s.markPaidAndGrant(order, txnID, "order-"+order.OrderID)
}

// markPaidAndGrant 将订单置为 PAID 并发放一次权益（幂等键绑定订单）。
func (s *Service) markPaidAndGrant(order Order, txnID, idemKey string) (Order, error) {
	quote, err := s.store.GetQuote(order.DataRegion, order.QuoteID)
	if err != nil {
		return Order{}, err
	}
	actor := Actor{UserID: order.UserID, DataRegion: order.DataRegion}
	if _, err := s.GrantProjectPack(context.Background(), actor, order.ProjectID,
		quote.TotalMinutes*60, idemKey); err != nil {
		return Order{}, err
	}
	now := s.now().UTC()
	order.Status = OrderPaid
	order.Provider = providerForOrder(order.PaymentMethod)
	order.ProviderTxnID = txnID
	order.ProgressNote = "支付成功：权益已发放"
	order.PaidAt = &now
	if err := s.store.UpdateOrder(order); err != nil {
		return Order{}, err
	}
	return order, nil
}

// autoRefundDuplicateCharge 识别重复扣款：全额原路退回、写 Incident、记冲正原因。
func (s *Service) autoRefundDuplicateCharge(order Order, eventID string) (Order, error) {
	if order.RefundedCents >= order.AmountCents {
		return order, nil // 已全额退回，仅记录事件。
	}
	now := s.now().UTC()
	refund := Refund{
		RefundID:       newID(),
		OrderID:        order.OrderID,
		UserID:         order.UserID,
		AmountCents:    order.AmountCents - order.RefundedCents,
		Currency:       order.Currency,
		Reason:         "重复扣款自动识别，原路退回（payment_event_id=" + eventID + "）",
		Kind:           RefundKindDuplicateCharge,
		Status:         Refunded,
		DataRegion:     order.DataRegion,
		CreatedAt:      now,
		IdempotencyKey: "dup-refund-" + eventID,
	}
	if err := s.store.SaveRefund(refund, refund.IdempotencyKey); err != nil {
		return Order{}, err
	}
	if err := s.appendRefundLedger(order, refund, "duplicate_charge_auto_refund"); err != nil {
		return Order{}, err
	}
	order.RefundedCents += refund.AmountCents
	order.ProgressNote = "检测到重复扣款：已自动原路退回，并通知用户"
	if err := s.store.UpdateOrder(order); err != nil {
		return Order{}, err
	}
	incident := Incident{
		IncidentID: newID(),
		Kind:       IncidentDuplicateCharge,
		Severity:   "high",
		Region:     order.DataRegion,
		Summary:    "重复扣款已自动识别并原路退回（order_id=" + order.OrderID + "）",
		DataRegion: order.DataRegion,
		CreatedAt:  now,
	}
	if err := s.store.SaveIncident(incident); err != nil {
		return Order{}, err
	}
	return order, nil
}

// appendRefundLedger 追加退款冲正条目（账本追加式；记录原因）。
func (s *Service) appendRefundLedger(order Order, refund Refund, reason string) error {
	balance, err := s.Balance(context.Background(),
		Actor{UserID: order.UserID, DataRegion: order.DataRegion})
	if err != nil {
		return err
	}
	entry := LedgerEntry{
		EntryID:        newID(),
		UserID:         order.UserID,
		ProjectID:      order.ProjectID,
		EntryType:      EntryRefund,
		Reason:         reason + "（refund_id=" + refund.RefundID + "）",
		BalanceAfter:   balance,
		IdempotencyKey: "ledger-" + refund.RefundID,
		DataRegion:     order.DataRegion,
		CreatedAt:      s.now().UTC(),
	}
	return s.store.AppendLedger(entry)
}

// applyRefundSucceeded 处理退款成功回调：将订单对应待退款标记为已退款。
func (s *Service) applyRefundSucceeded(order Order, _ PaymentEventInput) (Order, error) {
	refunds, err := s.store.ListRefundsByOrder(order.DataRegion, order.OrderID)
	if err != nil {
		return Order{}, err
	}
	for i := range refunds {
		if refunds[i].Status == RefundApproved || refunds[i].Status == RefundRequested ||
			refunds[i].Status == RefundReviewing {
			refunds[i].Status = Refunded
			now := s.now().UTC()
			refunds[i].RefundedAt = &now
			if err := s.store.UpdateRefund(refunds[i]); err != nil {
				return Order{}, err
			}
			order.RefundedCents += refunds[i].AmountCents
			order.ProgressNote = "退款已到账"
			if err := s.store.UpdateOrder(order); err != nil {
				return Order{}, err
			}
			return order, nil
		}
	}
	return order, nil
}

func (s *Service) getOwnedOrder(actor Actor, orderID string) (Order, error) {
	if err := validateActor(actor); err != nil {
		return Order{}, err
	}
	order, err := s.store.GetOrderByID(actor.DataRegion, orderID)
	if err != nil {
		return Order{}, err
	}
	if order.UserID != actor.UserID {
		return Order{}, ErrNotFound
	}
	return order, nil
}

// providerForOrder 将区域化支付方式标识映射为供应商中立提供方（演示确定性映射）。
func providerForOrder(paymentMethod string) string {
	if strings.Contains(paymentMethod, "psp_") {
		return paymentMethod
	}
	return "psp_" + paymentMethod
}

func contentHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
