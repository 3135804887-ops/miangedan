// Package billing 提供退款与补偿流程：大额/人工补偿双人审批、系统故障自动全额、
// 账本冲正条目记录原因（TASK-063；FR-033，US-06 场景 3；BILLING-STATE-MACHINE §5.4）。
// 红线：退款、付费状态与争议不得影响评分、复核或解锁；账本追加式。
package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// LargeRefundThresholdCents 为双人审批大额门槛（¥500 等值；区域定价配置可调）。
const LargeRefundThresholdCents = 50000

// RefundInput 为退款申请输入。
type RefundInput struct {
	OrderID string
	Reason  string
	Kind    string
}

// RequestRefund 申请退款：小额用户退款自动执行；大额或人工补偿双人审批；
// 系统故障自动全额执行；账本冲正条目记录原因（不影响评分/复核/解锁）。
func (s *Service) RequestRefund(
	_ context.Context, actor Actor, in RefundInput, idemKey string,
) (Refund, error) {
	if err := validateActor(actor); err != nil {
		return Refund{}, err
	}
	if strings.TrimSpace(in.OrderID) == "" || strings.TrimSpace(in.Reason) == "" ||
		strings.TrimSpace(idemKey) == "" {
		return Refund{}, fmt.Errorf("%w: order_id、reason 与幂等键必填", ErrInvalidInput)
	}
	if in.Kind == "" {
		in.Kind = RefundKindUserRequest
	}
	if in.Kind != RefundKindUserRequest && in.Kind != RefundKindSystemFault &&
		in.Kind != RefundKindCompensation {
		return Refund{}, fmt.Errorf("%w: 未知退款类型 %q", ErrInvalidInput, in.Kind)
	}
	if cached, err := s.store.GetRefundByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Refund{}, err
	}
	order, err := s.getOwnedOrder(actor, in.OrderID)
	if err != nil {
		return Refund{}, err
	}
	if order.Status != OrderPaid {
		return Refund{}, fmt.Errorf("%w: 仅 PAID 订单可退款（当前 %s）", ErrStateConflict, order.Status)
	}
	amount := order.AmountCents - order.RefundedCents
	if amount <= 0 {
		return Refund{}, fmt.Errorf("%w: 订单已全额退款", ErrStateConflict)
	}
	refund := Refund{
		RefundID:       newID(),
		OrderID:        order.OrderID,
		UserID:         actor.UserID,
		AmountCents:    amount,
		Currency:       order.Currency,
		Reason:         in.Reason,
		Kind:           in.Kind,
		Status:         RefundRequested,
		DataRegion:     actor.DataRegion,
		IdempotencyKey: idemKey,
		CreatedAt:      s.now().UTC(),
	}
	automatic := false
	switch in.Kind {
	case RefundKindSystemFault:
		automatic = true // 系统故障：自动全额，无需审批。
	case RefundKindCompensation:
		refund.Status = RefundReviewing
	case RefundKindUserRequest:
		if amount >= LargeRefundThresholdCents {
			refund.Status = RefundReviewing
		} else {
			automatic = true
		}
	}
	if err := s.store.SaveRefund(refund, idemKey); err != nil {
		return Refund{}, err
	}
	if automatic {
		if err := s.executeRefund(refund); err != nil {
			return Refund{}, err
		}
		refund.Status = Refunded
		now := s.now().UTC()
		refund.RefundedAt = &now
	}
	return refund, nil
}

// ApproveRefund 双人审批：同一审批人重复提交无效；两名不同审批人后自动执行。
func (s *Service) ApproveRefund(
	_ context.Context, actor Actor, refundID, approverID string,
) (Refund, error) {
	if err := validateActor(actor); err != nil {
		return Refund{}, err
	}
	if strings.TrimSpace(approverID) == "" {
		return Refund{}, fmt.Errorf("%w: 审批人必填", ErrInvalidInput)
	}
	refund, err := s.store.GetRefundByID(actor.DataRegion, refundID)
	if err != nil {
		return Refund{}, err
	}
	if refund.UserID == approverID {
		return Refund{}, fmt.Errorf("%w: 本人退款不可自批", ErrStateConflict)
	}
	if refund.Status == Refunded || refund.Status == RefundRejected {
		return refund, nil
	}
	pair, err := s.store.AppendRefundApproval(actor.DataRegion, refundID, approverID)
	if err != nil {
		return Refund{}, err
	}
	refund.ApproverPair = pair
	if len(pair) == 2 {
		if err := s.executeRefund(refund); err != nil {
			return Refund{}, err
		}
		return s.store.GetRefundByID(actor.DataRegion, refundID)
	}
	return refund, nil
}

// RejectRefund 拒绝退款并说明原因（用户可申诉）。
func (s *Service) RejectRefund(
	_ context.Context, actor Actor, refundID, reason string,
) (Refund, error) {
	if err := validateActor(actor); err != nil {
		return Refund{}, err
	}
	refund, err := s.store.GetRefundByID(actor.DataRegion, refundID)
	if err != nil {
		return Refund{}, err
	}
	if refund.Status != RefundRequested && refund.Status != RefundReviewing {
		return Refund{}, fmt.Errorf("%w: 当前状态不可拒绝（%s）", ErrStateConflict, refund.Status)
	}
	refund.Status = RefundRejected
	refund.RejectReason = reason
	if err := s.store.UpdateRefund(refund); err != nil {
		return Refund{}, err
	}
	return refund, nil
}

// AppealRefund 被拒后申诉（重新进入审批；仅本人可申诉）。
func (s *Service) AppealRefund(
	_ context.Context, actor Actor, refundID, reason string,
) (Refund, error) {
	if err := validateActor(actor); err != nil {
		return Refund{}, err
	}
	refund, err := s.store.GetRefundByID(actor.DataRegion, refundID)
	if err != nil {
		return Refund{}, err
	}
	if refund.UserID != actor.UserID {
		return Refund{}, ErrNotFound
	}
	if refund.Status != RefundRejected {
		return Refund{}, fmt.Errorf("%w: 仅 REFUND_REJECTED 可申诉", ErrStateConflict)
	}
	refund.Status = RefundRequested
	refund.RejectReason = ""
	refund.Reason += "（申诉：" + reason + "）"
	if err := s.store.UpdateRefund(refund); err != nil {
		return Refund{}, err
	}
	return refund, nil
}

// GetRefund 查询退款（用户本人或审批方可见）。
func (s *Service) GetRefund(_ context.Context, actor Actor, refundID string) (Refund, error) {
	if err := validateActor(actor); err != nil {
		return Refund{}, err
	}
	return s.store.GetRefundByID(actor.DataRegion, refundID)
}

// ListRefunds 列出用户退款（逐笔账单可见）。
func (s *Service) ListRefunds(_ context.Context, actor Actor) ([]Refund, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	return s.store.ListRefundsByUser(actor.DataRegion, actor.UserID)
}

// executeRefund 执行退款：幂等（MarkRefundExecuted 原子），账本冲正条目记录原因。
func (s *Service) executeRefund(refund Refund) error {
	executed, fresh, err := s.store.MarkRefundExecuted(refund.DataRegion, refund.RefundID, s.now().UTC())
	if err != nil {
		return err
	}
	if !fresh {
		return nil
	}
	order, err := s.store.GetOrderByID(executed.DataRegion, executed.OrderID)
	if err != nil {
		return err
	}
	if err := s.appendRefundLedger(order, executed, "refund_executed"); err != nil {
		return err
	}
	order.RefundedCents += executed.AmountCents
	order.ProgressNote = "退款已执行（" + executed.Reason + "）"
	return s.store.UpdateOrder(order)
}
