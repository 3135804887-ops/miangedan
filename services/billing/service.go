package billing

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"miangedan/services/region"
)

// Service 为报价与权益服务（TASK-060；FR-031）。
type Service struct {
	store Store
	now   func() time.Time
}

// NewService 创建服务。
func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: 缺少存储", ErrInvalidInput)
	}
	return &Service{store: store, now: time.Now}, nil
}

// ---- 权益 ----

// GrantFreeCredit 首次登录发放免费 60 分钟（幂等：每人一份）。
func (s *Service) GrantFreeCredit(
	_ context.Context, actor Actor, idemKey string,
) (Entitlement, error) {
	if err := validateActor(actor); err != nil {
		return Entitlement{}, err
	}
	cached, err := s.store.GetEntitlementByIdempotencyKey(actor.DataRegion, idemKey)
	if err == nil {
		return cached, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Entitlement{}, err
	}
	now := s.now().UTC()
	entitlement := Entitlement{
		EntitlementID: newID(),
		UserID:        actor.UserID,
		Kind:          KindFreeCredit,
		TotalSeconds:  FreeCreditSeconds,
		Status:        "active",
		ValidFrom:     now,
		DataRegion:    actor.DataRegion,
	}
	if err := s.store.SaveEntitlement(entitlement, idemKey); err != nil {
		return Entitlement{}, err
	}
	return entitlement, nil
}

// GrantProjectPack 发放单项目包（覆盖已确认轮次 + 每失败轮一次正式重试的估算时长）。
func (s *Service) GrantProjectPack(
	_ context.Context, actor Actor, projectID string, estimatedSeconds int, idemKey string,
) (Entitlement, error) {
	if err := validateActor(actor); err != nil {
		return Entitlement{}, err
	}
	if strings.TrimSpace(projectID) == "" || estimatedSeconds <= 0 {
		return Entitlement{}, fmt.Errorf(
			"%w: project_id 与 estimated_seconds 必填", ErrInvalidInput)
	}
	cached, err := s.store.GetEntitlementByIdempotencyKey(actor.DataRegion, idemKey)
	if err == nil {
		return cached, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Entitlement{}, err
	}
	now := s.now().UTC()
	entitlement := Entitlement{
		EntitlementID:  newID(),
		UserID:         actor.UserID,
		Kind:           KindProjectPack,
		ScopeProjectID: projectID,
		TotalSeconds:   estimatedSeconds,
		Status:         "active",
		ValidFrom:      now,
		DataRegion:     actor.DataRegion,
	}
	if err := s.store.SaveEntitlement(entitlement, idemKey); err != nil {
		return Entitlement{}, err
	}
	return entitlement, nil
}

// GrantTopup 发放时长加油包。
func (s *Service) GrantTopup(
	_ context.Context, actor Actor, seconds int, idemKey string,
) (Entitlement, error) {
	if err := validateActor(actor); err != nil {
		return Entitlement{}, err
	}
	if seconds <= 0 {
		return Entitlement{}, fmt.Errorf("%w: seconds 必须 >0", ErrInvalidInput)
	}
	cached, err := s.store.GetEntitlementByIdempotencyKey(actor.DataRegion, idemKey)
	if err == nil {
		return cached, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Entitlement{}, err
	}
	now := s.now().UTC()
	entitlement := Entitlement{
		EntitlementID: newID(),
		UserID:        actor.UserID,
		Kind:          KindTopup,
		TotalSeconds:  seconds,
		Status:        "active",
		ValidFrom:     now,
		DataRegion:    actor.DataRegion,
	}
	if err := s.store.SaveEntitlement(entitlement, idemKey); err != nil {
		return Entitlement{}, err
	}
	return entitlement, nil
}

// ActivatePro 激活 Pro 订阅（月额度；余额 ≤2×月额度；结转 ≤1 账期）。
func (s *Service) ActivatePro(
	_ context.Context, actor Actor, monthlySeconds int, periodStart, periodEnd time.Time, idemKey string,
) (ProSubscription, error) {
	if err := validateActor(actor); err != nil {
		return ProSubscription{}, err
	}
	if monthlySeconds <= 0 || !periodEnd.After(periodStart) {
		return ProSubscription{}, fmt.Errorf("%w: 月额度与账期非法", ErrInvalidInput)
	}
	cached, err := s.store.GetSubscription(actor.DataRegion, actor.UserID)
	carryover := 0
	if err == nil {
		// 续费：结转上一账期 Pro 未使用时长（≤1 账期），总余额 ≤ 2×月额度。
		previous := s.proRemaining(actor)
		if previous > 0 {
			carryover = previous
			if carryover > monthlySeconds {
				carryover = monthlySeconds
			}
		}
		_ = cached
	} else if !errors.Is(err, ErrNotFound) {
		return ProSubscription{}, err
	}
	// 旧 Pro 权益到期（新账期权益另立），避免余额重复计算。
	items, err := s.store.ListEntitlements(actor.DataRegion, actor.UserID)
	if err != nil {
		return ProSubscription{}, err
	}
	for _, e := range items {
		if e.Kind == KindProSub && e.Status == "active" {
			e.Status = "expired"
			if err := s.store.UpdateEntitlement(e); err != nil {
				return ProSubscription{}, err
			}
		}
	}
	pro := ProSubscription{
		SubscriptionID:   newID(),
		UserID:           actor.UserID,
		Status:           "SUB_ACTIVE",
		MonthlySeconds:   monthlySeconds,
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		CarryoverSeconds: carryover,
		AutoRenew:        false,
		DataRegion:       actor.DataRegion,
	}
	if err := s.store.SaveSubscription(pro, idemKey); err != nil {
		return ProSubscription{}, err
	}
	total := monthlySeconds + carryover
	if total > 2*monthlySeconds {
		total = 2 * monthlySeconds
	}
	entitlement := Entitlement{
		EntitlementID: newID(),
		UserID:        actor.UserID,
		Kind:          KindProSub,
		TotalSeconds:  total,
		Status:        "active",
		ValidFrom:     periodStart,
		ValidTo:       periodEnd,
		DataRegion:    actor.DataRegion,
	}
	if err := s.store.SaveEntitlement(entitlement, idemKey+"-ent"); err != nil {
		return ProSubscription{}, err
	}
	return pro, nil
}

// Balance 返回用户全部可用权益余额（秒）。
func (s *Service) Balance(_ context.Context, actor Actor) (int, error) {
	if err := validateActor(actor); err != nil {
		return 0, err
	}
	items, err := s.store.ListEntitlements(actor.DataRegion, actor.UserID)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, e := range items {
		if e.Status != "active" {
			continue
		}
		now := s.now()
		if !e.ValidTo.IsZero() && now.After(e.ValidTo) {
			continue
		}
		total += e.Remaining()
	}
	return total, nil
}

// CanReserve 校验开始前预留是否足够（余额校验只在每轮开始前）。
func (s *Service) CanReserve(ctx context.Context, actor Actor, seconds int) (bool, error) {
	balance, err := s.Balance(ctx, actor)
	if err != nil {
		return false, err
	}
	return balance >= seconds, nil
}

// ---- 报价 ----

// CreateQuote 创建报价（QUOTE_DRAFT；开始后计费版本冻结则拒绝）。
func (s *Service) CreateQuote(
	_ context.Context, actor Actor, plan PlanInput, idemKey string,
) (Quote, error) {
	if err := validateActor(actor); err != nil {
		return Quote{}, err
	}
	if strings.TrimSpace(plan.ProjectID) == "" || len(plan.Rounds) == 0 {
		return Quote{}, fmt.Errorf("%w: project_id 与轮次必填", ErrInvalidInput)
	}
	if len(plan.Rounds) > 5 {
		return Quote{}, fmt.Errorf("%w: 轮次数必须为 1-5", ErrInvalidInput)
	}
	for _, round := range plan.Rounds {
		if round.DurationMinutes < 10 || round.DurationMinutes > 60 {
			return Quote{}, fmt.Errorf(
				"%w: 轮次时长必须为 10-60 分钟", ErrInvalidInput)
		}
	}
	if _, err := s.store.GetFreeze(actor.DataRegion, plan.ProjectID); err == nil {
		return Quote{}, ErrQuoteFrozen
	} else if !errors.Is(err, ErrNotFound) {
		return Quote{}, err
	}
	cached, err := s.store.GetQuoteByIdempotencyKey(actor.DataRegion, idemKey)
	if err == nil {
		return cached, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Quote{}, err
	}
	price := PriceConfigFor(actor.DataRegion)
	totalMinutes := 0
	retries := 0
	for _, round := range plan.Rounds {
		totalMinutes += round.DurationMinutes
		if round.RetryEligible {
			retries++
		}
	}
	amount := math.Round(float64(totalMinutes*price.PerMinuteCents) * (1 + price.TaxRate))
	now := s.now().UTC()
	quote := Quote{
		QuoteID:        newID(),
		ProjectID:      plan.ProjectID,
		PlanVersion:    1,
		Status:         QuoteDraft,
		TotalMinutes:   totalMinutes,
		FreeRetries:    retries,
		AmountCents:    int(amount),
		Currency:       price.Currency,
		TaxDescription: fmt.Sprintf("含税（税率 %.0f%%）", price.TaxRate*100),
		ValidUntil:     now.Add(72 * time.Hour),
		DataRegion:     actor.DataRegion,
		CreatedAt:      now,
	}
	if err := s.store.SaveQuote(quote, idemKey); err != nil {
		return Quote{}, err
	}
	return quote, nil
}

// PresentQuote 报价呈现（DRAFT → PRESENTED）。
func (s *Service) PresentQuote(_ context.Context, actor Actor, quoteID string) (Quote, error) {
	quote, err := s.getOwnedQuote(actor, quoteID)
	if err != nil {
		return Quote{}, err
	}
	if quote.Status != QuoteDraft {
		return Quote{}, fmt.Errorf("%w: 仅 QUOTE_DRAFT 可呈现", ErrStateConflict)
	}
	quote.Status = QuotePresented
	if err := s.store.UpdateQuote(quote); err != nil {
		return Quote{}, err
	}
	return quote, nil
}

// AcceptQuote 接受报价并冻结计费版本（开始后不可再报价；幂等）。
func (s *Service) AcceptQuote(_ context.Context, actor Actor, quoteID string) (Freeze, error) {
	quote, err := s.getOwnedQuote(actor, quoteID)
	if err != nil {
		return Freeze{}, err
	}
	if quote.Status == QuoteAccepted {
		return s.store.GetFreeze(actor.DataRegion, quote.ProjectID)
	}
	if quote.Status != QuotePresented {
		return Freeze{}, fmt.Errorf(
			"%w: 仅 QUOTE_PRESENTED 可接受（当前 %s）", ErrStateConflict, quote.Status)
	}
	quote.Status = QuoteAccepted
	if err := s.store.UpdateQuote(quote); err != nil {
		return Freeze{}, err
	}
	now := s.now().UTC()
	freeze := Freeze{
		ProjectID:   quote.ProjectID,
		QuoteID:     quote.QuoteID,
		PlanVersion: quote.PlanVersion,
		Frozen:      true,
		FrozenAt:    now,
		DataRegion:  actor.DataRegion,
	}
	if err := s.store.SaveFreeze(freeze, "freeze-"+quote.QuoteID); err != nil {
		return Freeze{}, err
	}
	return freeze, nil
}

// RecalculateQuote 计划修改（开始前）→ 新版本报价（QUOTE_RECALCULATED → PRESENTED）。
func (s *Service) RecalculateQuote(
	_ context.Context, actor Actor, plan PlanInput, idemKey string,
) (Quote, error) {
	if _, err := s.store.GetFreeze(actor.DataRegion, plan.ProjectID); err == nil {
		return Quote{}, ErrQuoteFrozen
	} else if !errors.Is(err, ErrNotFound) {
		return Quote{}, err
	}
	quotes, err := s.store.ListQuotes(actor.DataRegion, plan.ProjectID)
	if err != nil {
		return Quote{}, err
	}
	nextVersion := 1
	if len(quotes) > 0 {
		nextVersion = quotes[len(quotes)-1].PlanVersion + 1
	}
	quote, err := s.CreateQuote(context.Background(), actor, plan, idemKey)
	if err != nil {
		return Quote{}, err
	}
	quote.PlanVersion = nextVersion
	quote.Status = QuoteRecalculated
	if err := s.store.UpdateQuote(quote); err != nil {
		return Quote{}, err
	}
	quote.Status = QuotePresented
	if err := s.store.UpdateQuote(quote); err != nil {
		return Quote{}, err
	}
	return quote, nil
}

// GetFreeze 查询计费版本冻结。
func (s *Service) GetFreeze(
	_ context.Context, actor Actor, projectID string,
) (Freeze, error) {
	if err := validateActor(actor); err != nil {
		return Freeze{}, err
	}
	return s.store.GetFreeze(actor.DataRegion, projectID)
}

// GetEntitlements 列出用户权益。
func (s *Service) GetEntitlements(
	_ context.Context, actor Actor,
) ([]Entitlement, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	return s.store.ListEntitlements(actor.DataRegion, actor.UserID)
}

func (s *Service) getOwnedQuote(actor Actor, quoteID string) (Quote, error) {
	if err := validateActor(actor); err != nil {
		return Quote{}, err
	}
	quote, err := s.store.GetQuote(actor.DataRegion, quoteID)
	if err != nil {
		return Quote{}, err
	}
	return quote, nil
}

func (s *Service) proRemaining(actor Actor) int {
	items, err := s.store.ListEntitlements(actor.DataRegion, actor.UserID)
	if err != nil {
		return 0
	}
	total := 0
	for _, e := range items {
		if e.Status == "active" && e.Kind == KindProSub {
			total += e.Remaining()
		}
	}
	return total
}

func validateActor(actor Actor) error {
	if strings.TrimSpace(actor.UserID) == "" {
		return fmt.Errorf("%w: 缺少用户身份", ErrInvalidInput)
	}
	return region.ValidateDataRegion(actor.DataRegion)
}
