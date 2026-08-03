// Package billing 提供 Pro 订阅生命周期：自动续费单独勾选、扣款前提醒、
// 月度时长/结转 ≤1 账期/总余额 ≤2×月额度、到期当前轮可完成（TASK-064；
// FR-033，US-06 场景 5；BILLING-STATE-MACHINE §5.5）。
package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"miangedan/services/region"
)

// 续费事件状态。
const (
	RenewalReminded = "reminded"
	RenewalCharged  = "charged"
	RenewalFailed   = "failed"
)

// renewalReminderWindow 为扣款前提醒有效窗口（提醒后 7 天内可扣款）。
const renewalReminderWindow = 7 * 24 * time.Hour

// ErrReconsentRequired 为续费条款变化需重新同意的错误。
var ErrReconsentRequired = errors.New("subscription renewal requires re-consent")

// RenewalRecord 为一次续费事件（自动续费单独勾选 + 扣款前提醒）。
type RenewalRecord struct {
	RenewalID      string
	SubscriptionID string
	UserID         string
	PeriodStart    time.Time
	PeriodEnd      time.Time
	MonthlySeconds int
	PriceCents     int
	Status         string
	RemindedAt     *time.Time
	ChargedAt      *time.Time
	DataRegion     string
	IdempotencyKey string
}

// GetSubscription 返回当前订阅（月度时长、结转、账期、自动续费）。
func (s *Service) GetSubscription(
	_ context.Context, actor Actor,
) (ProSubscription, error) {
	if err := validateActor(actor); err != nil {
		return ProSubscription{}, err
	}
	return s.store.GetSubscription(actor.DataRegion, actor.UserID)
}

// SetAutoRenew 单独勾选自动续费：必须明确同意；记录同意时的月额度与价格条款。
// 关闭自动续费不改变当期权益。
func (s *Service) SetAutoRenew(
	_ context.Context, actor Actor, enabled, consentGiven bool,
	monthlySeconds, priceCents int, idemKey string,
) (ProSubscription, error) {
	if err := validateActor(actor); err != nil {
		return ProSubscription{}, err
	}
	if strings.TrimSpace(idemKey) == "" {
		return ProSubscription{}, fmt.Errorf("%w: 幂等键必填", ErrInvalidInput)
	}
	if cached, err := s.store.GetSubscriptionByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return ProSubscription{}, err
	}
	sub, err := s.store.GetSubscription(actor.DataRegion, actor.UserID)
	if err != nil {
		return ProSubscription{}, err
	}
	if enabled {
		if !consentGiven {
			return ProSubscription{}, fmt.Errorf("%w: 自动续费必须单独勾选并同意", ErrInvalidInput)
		}
		if monthlySeconds <= 0 || priceCents < 0 {
			return ProSubscription{}, fmt.Errorf("%w: 续费条款非法", ErrInvalidInput)
		}
		sub.AutoRenew = true
		sub.Status = "SUB_ACTIVE"
		sub.ConsentMonthlySeconds = monthlySeconds
		sub.ConsentPriceCents = priceCents
		sub.IdempotencyKey = idemKey
	} else {
		sub.AutoRenew = false
		sub.IdempotencyKey = idemKey
	}
	if err := s.store.UpdateSubscription(sub); err != nil {
		return ProSubscription{}, err
	}
	return sub, nil
}

// CancelAutoRenew 取消续费：权益保留至账期结束；历史与删除/导出权利不受影响。
func (s *Service) CancelAutoRenew(
	_ context.Context, actor Actor, idemKey string,
) (ProSubscription, error) {
	if err := validateActor(actor); err != nil {
		return ProSubscription{}, err
	}
	if cached, err := s.store.GetSubscriptionByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return ProSubscription{}, err
	}
	sub, err := s.store.GetSubscription(actor.DataRegion, actor.UserID)
	if err != nil {
		return ProSubscription{}, err
	}
	sub.AutoRenew = false
	sub.Status = "SUB_CANCELLED"
	sub.IdempotencyKey = idemKey
	if err := s.store.UpdateSubscription(sub); err != nil {
		return ProSubscription{}, err
	}
	return sub, nil
}

// PrepareRenewal 续费扣款前提醒：校验条款与同意一致（变化须重新同意）；
// 幂等键去重；同一账期只生成一条续费事件。
func (s *Service) PrepareRenewal(
	_ context.Context, actor Actor, idemKey string,
) (RenewalRecord, error) {
	if err := validateActor(actor); err != nil {
		return RenewalRecord{}, err
	}
	if cached, err := s.store.GetRenewalByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return RenewalRecord{}, err
	}
	sub, err := s.store.GetSubscription(actor.DataRegion, actor.UserID)
	if err != nil {
		return RenewalRecord{}, err
	}
	if !sub.AutoRenew || sub.Status == "SUB_CANCELLED" {
		return RenewalRecord{}, fmt.Errorf("%w: 未勾选自动续费", ErrStateConflict)
	}
	price := currentProPrice(sub.MonthlySeconds, actor.DataRegion)
	if sub.ConsentMonthlySeconds != sub.MonthlySeconds || sub.ConsentPriceCents != price {
		return RenewalRecord{}, fmt.Errorf("%w: 续费价格或权益已变化（同意 %d，当前 %d）",
			ErrReconsentRequired, sub.ConsentPriceCents, price)
	}
	records, err := s.store.ListRenewalsBySubscription(actor.DataRegion, sub.SubscriptionID)
	if err != nil {
		return RenewalRecord{}, err
	}
	for _, r := range records {
		if r.PeriodStart.Equal(sub.PeriodEnd) && r.Status == RenewalCharged {
			return r, nil
		}
	}
	now := s.now().UTC()
	record := RenewalRecord{
		RenewalID:      newID(),
		SubscriptionID: sub.SubscriptionID,
		UserID:         actor.UserID,
		PeriodStart:    sub.PeriodEnd,
		PeriodEnd:      sub.PeriodEnd.AddDate(0, 1, 0),
		MonthlySeconds: sub.MonthlySeconds,
		PriceCents:     price,
		Status:         RenewalReminded,
		RemindedAt:     &now,
		DataRegion:     actor.DataRegion,
		IdempotencyKey: idemKey,
	}
	if err := s.store.SaveRenewalRecord(record, idemKey); err != nil {
		return RenewalRecord{}, err
	}
	return record, nil
}

// ChargeRenewal 执行续费扣款：必须有扣款前提醒（7 天窗口内）且条款未变化；
// 结转 ≤1 账期、总余额 ≤2×月额度由 ActivatePro 保证；到期当前轮可完成。
func (s *Service) ChargeRenewal(
	ctx context.Context, actor Actor, renewalID, idemKey string,
) (ProSubscription, error) {
	if err := validateActor(actor); err != nil {
		return ProSubscription{}, err
	}
	if cached, err := s.store.GetRenewalByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		_ = cached
	} else if !errors.Is(err, ErrNotFound) {
		return ProSubscription{}, err
	}
	record, err := s.store.GetRenewalByID(actor.DataRegion, renewalID)
	if err != nil {
		return ProSubscription{}, err
	}
	if record.UserID != actor.UserID {
		return ProSubscription{}, ErrNotFound
	}
	if record.Status != RenewalReminded || record.RemindedAt == nil {
		return ProSubscription{}, fmt.Errorf("%w: 尚无扣款前提醒", ErrStateConflict)
	}
	if s.now().UTC().Sub(*record.RemindedAt) > renewalReminderWindow {
		return ProSubscription{}, fmt.Errorf("%w: 提醒已过期，需重新提醒", ErrStateConflict)
	}
	sub, err := s.store.GetSubscription(actor.DataRegion, actor.UserID)
	if err != nil {
		return ProSubscription{}, err
	}
	if !sub.AutoRenew {
		return ProSubscription{}, fmt.Errorf("%w: 自动续费已取消", ErrStateConflict)
	}
	price := currentProPrice(record.MonthlySeconds, actor.DataRegion)
	if sub.ConsentPriceCents != price || sub.ConsentMonthlySeconds != record.MonthlySeconds {
		return ProSubscription{}, fmt.Errorf("%w: 条款已变化", ErrReconsentRequired)
	}
	if _, err := s.ActivatePro(ctx, actor, record.MonthlySeconds,
		record.PeriodStart, record.PeriodEnd, "renew-"+renewalID); err != nil {
		record.Status = RenewalFailed
		_ = s.store.UpdateRenewalRecord(record)
		return ProSubscription{}, err
	}
	now := s.now().UTC()
	record.Status = RenewalCharged
	record.ChargedAt = &now
	if err := s.store.UpdateRenewalRecord(record); err != nil {
		return ProSubscription{}, err
	}
	return s.store.GetSubscription(actor.DataRegion, actor.UserID)
}

// ExpireDueSubscriptions 到期任务：账期结束未续费 → SUB_EXPIRED；
// 历史不删除，不影响导出与删除权利；进行中的正式轮次不受影响
// （余额校验只发生在每轮开始前）。
func (s *Service) ExpireDueSubscriptions(
	_ context.Context, dataRegion string,
) ([]ProSubscription, error) {
	if err := region.ValidateDataRegion(dataRegion); err != nil {
		return nil, err
	}
	subs, err := s.store.ListSubscriptions(dataRegion)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	var expired []ProSubscription
	for i := range subs {
		if subs[i].Status == "SUB_EXPIRED" || !now.After(subs[i].PeriodEnd) {
			continue
		}
		subs[i].Status = "SUB_EXPIRED"
		subs[i].AutoRenew = false
		if err := s.store.UpdateSubscription(subs[i]); err != nil {
			return nil, err
		}
		items, err := s.store.ListEntitlements(dataRegion, subs[i].UserID)
		if err != nil {
			return nil, err
		}
		for j := range items {
			if items[j].Kind == KindProSub && items[j].Status == "active" {
				items[j].Status = "expired"
				if err := s.store.UpdateEntitlement(items[j]); err != nil {
					return nil, err
				}
			}
		}
		expired = append(expired, subs[i])
	}
	return expired, nil
}

// currentProPrice 按区域定价配置计算月度价格（分）。
func currentProPrice(monthlySeconds int, dataRegion string) int {
	price := PriceConfigFor(dataRegion)
	return monthlySeconds / 60 * price.PerMinuteCents
}
