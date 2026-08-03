package billing

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// 计量状态。
const (
	MeterCapturing = "capturing"
	MeterStopped   = "stopped"
)

// Reserve 每轮开始前预留（余额不足阻止开始；已开始轮次不因余额变化中断）。
// 消费顺序：免费 → 项目包（限本项目）→ Pro → 加油包；账本追加 reserve 条目。
func (s *Service) Reserve(
	ctx context.Context, actor Actor, in ReserveInput,
) (LedgerEntry, error) {
	if err := validateActor(actor); err != nil {
		return LedgerEntry{}, err
	}
	if strings.TrimSpace(in.ProjectID) == "" ||
		strings.TrimSpace(in.AttemptID) == "" ||
		strings.TrimSpace(in.SessionID) == "" ||
		strings.TrimSpace(in.IdempotencyKey) == "" {
		return LedgerEntry{}, fmt.Errorf("%w: 预留入参不完整", ErrInvalidInput)
	}
	if in.RoundSequence < 1 || in.RoundSequence > 5 || in.EstimatedSeconds <= 0 {
		return LedgerEntry{}, fmt.Errorf("%w: 轮次与秒数非法", ErrInvalidInput)
	}
	cached, err := s.store.GetLedgerByIdempotencyKey(actor.DataRegion, in.IdempotencyKey)
	if err == nil {
		return cached, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return LedgerEntry{}, err
	}
	items, err := s.store.ListEntitlements(actor.DataRegion, actor.UserID)
	if err != nil {
		return LedgerEntry{}, err
	}
	consumed, err := consumeEntitlements(items, in.EstimatedSeconds, in.ProjectID)
	if err != nil {
		return LedgerEntry{}, fmt.Errorf("%w: %v", ErrInsufficient, err)
	}
	for entitlementID, seconds := range consumed {
		for i := range items {
			if items[i].EntitlementID == entitlementID {
				items[i].ConsumedSeconds += seconds
				if err := s.store.UpdateEntitlement(items[i]); err != nil {
					return LedgerEntry{}, err
				}
			}
		}
	}
	balanceAfter, err := s.Balance(ctx, actor)
	if err != nil {
		return LedgerEntry{}, err
	}
	now := s.now().UTC()
	entry := LedgerEntry{
		EntryID:        newID(),
		UserID:         actor.UserID,
		ProjectID:      in.ProjectID,
		RoundSequence:  in.RoundSequence,
		AttemptID:      in.AttemptID,
		SessionID:      in.SessionID,
		EntryType:      EntryReserve,
		Seconds:        in.EstimatedSeconds,
		Reason:         "round_reservation",
		BalanceAfter:   balanceAfter,
		IdempotencyKey: in.IdempotencyKey,
		DataRegion:     actor.DataRegion,
		CreatedAt:      now,
	}
	if err := s.store.AppendLedger(entry); err != nil {
		return LedgerEntry{}, err
	}
	meter := Meter{
		SessionID:          in.SessionID,
		AttemptID:          in.AttemptID,
		ProjectID:          in.ProjectID,
		RoundSequence:      in.RoundSequence,
		Status:             MeterStopped,
		ReservationSeconds: in.EstimatedSeconds,
		DataRegion:         actor.DataRegion,
	}
	if err := s.store.SaveMeter(meter); err != nil {
		return LedgerEntry{}, err
	}
	return entry, nil
}

// StartMetering 会话 LIVE 开始计量（usage.meter.started，幂等；只计 LIVE 秒）。
func (s *Service) StartMetering(
	_ context.Context, actor Actor, sessionID string, idemKey string,
) (Meter, error) {
	if err := validateActor(actor); err != nil {
		return Meter{}, err
	}
	if cached, err := s.store.GetLedgerByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		_ = cached
	} else if !errors.Is(err, ErrNotFound) {
		return Meter{}, err
	}
	meter, err := s.store.GetMeter(actor.DataRegion, sessionID)
	if err != nil {
		return Meter{}, err
	}
	if meter.Status == MeterCapturing {
		return meter, nil
	}
	if meter.Settled || meter.Refunded {
		return Meter{}, fmt.Errorf("%w: 会话已结算/返还，不能重新计量", ErrStateConflict)
	}
	now := s.now().UTC()
	meter.Status = MeterCapturing
	meter.StartedAt = &now
	meter.StoppedAt = nil
	if err := s.store.SaveMeter(meter); err != nil {
		return Meter{}, err
	}
	return meter, nil
}

// StopMetering 停止计量（故障/等待/重连/降级接受后；累计仅 LIVE 段）。
func (s *Service) StopMetering(
	_ context.Context, actor Actor, sessionID string,
) (Meter, error) {
	if err := validateActor(actor); err != nil {
		return Meter{}, err
	}
	meter, err := s.store.GetMeter(actor.DataRegion, sessionID)
	if err != nil {
		return Meter{}, err
	}
	if meter.Status == MeterStopped {
		return meter, nil
	}
	if meter.StartedAt != nil {
		now := s.now().UTC()
		meter.AccumulatedSeconds += int(now.Sub(*meter.StartedAt).Seconds())
		meter.StoppedAt = &now
	}
	meter.Status = MeterStopped
	if err := s.store.SaveMeter(meter); err != nil {
		return Meter{}, err
	}
	return meter, nil
}

// Settle 轮次正常结束：按实际使用扣减，多余预留以冲正释放（用户主动退出同规则）。
func (s *Service) Settle(
	ctx context.Context, actor Actor, sessionID, reason string, idemKey string,
) ([]LedgerEntry, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	if cached, err := s.store.GetLedgerByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return []LedgerEntry{cached}, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	meter, err := s.store.GetMeter(actor.DataRegion, sessionID)
	if err != nil {
		return nil, err
	}
	if meter.Settled {
		return nil, fmt.Errorf("%w: 会话已结算", ErrStateConflict)
	}
	// 停止计量（若仍在 LIVE）。
	if meter.Status == MeterCapturing {
		if _, err := s.StopMetering(ctx, actor, sessionID); err != nil {
			return nil, err
		}
		meter.AccumulatedSeconds = s.currentAccumulated(actor, sessionID)
	}
	balanceAfter, err := s.Balance(ctx, actor)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	consumeEntry := LedgerEntry{
		EntryID:        newID(),
		UserID:         actor.UserID,
		ProjectID:      meter.ProjectID,
		RoundSequence:  meter.RoundSequence,
		AttemptID:      meter.AttemptID,
		SessionID:      sessionID,
		EntryType:      EntryConsume,
		Seconds:        meter.AccumulatedSeconds,
		Reason:         reason,
		BalanceAfter:   balanceAfter,
		IdempotencyKey: idemKey,
		DataRegion:     actor.DataRegion,
		CreatedAt:      now,
	}
	if err := s.store.AppendLedger(consumeEntry); err != nil {
		return nil, err
	}
	entries := []LedgerEntry{consumeEntry}
	unused := meter.ReservationSeconds - meter.AccumulatedSeconds
	if unused > 0 {
		// 冲正释放未使用预留（余额恢复）。
		if err := s.restoreEntitlements(ctx, actor, unused); err != nil {
			return nil, err
		}
		reversal := LedgerEntry{
			EntryID:        newID(),
			UserID:         actor.UserID,
			ProjectID:      meter.ProjectID,
			RoundSequence:  meter.RoundSequence,
			AttemptID:      meter.AttemptID,
			SessionID:      sessionID,
			EntryType:      EntryReversal,
			Seconds:        unused,
			Reason:         "release_unused",
			BalanceAfter:   balanceAfter + unused,
			IdempotencyKey: idemKey + "-release",
			DataRegion:     actor.DataRegion,
			CreatedAt:      s.now().UTC(),
		}
		if err := s.store.AppendLedger(reversal); err != nil {
			return nil, err
		}
		entries = append(entries, reversal)
	}
	meter.Settled = true
	if err := s.store.SaveMeter(meter); err != nil {
		return nil, err
	}
	return entries, nil
}

// RefundFull 系统责任导致评估未完成：自动全额返还本轮预留（冲正条目）。
func (s *Service) RefundFull(
	ctx context.Context, actor Actor, sessionID, reason string, idemKey string,
) (LedgerEntry, error) {
	if err := validateActor(actor); err != nil {
		return LedgerEntry{}, err
	}
	if cached, err := s.store.GetLedgerByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return LedgerEntry{}, err
	}
	meter, err := s.store.GetMeter(actor.DataRegion, sessionID)
	if err != nil {
		return LedgerEntry{}, err
	}
	if meter.Refunded {
		return LedgerEntry{}, fmt.Errorf("%w: 会话已全额返还", ErrStateConflict)
	}
	if err := s.restoreEntitlements(ctx, actor, meter.ReservationSeconds); err != nil {
		return LedgerEntry{}, err
	}
	balanceAfter, err := s.Balance(ctx, actor)
	if err != nil {
		return LedgerEntry{}, err
	}
	entry := LedgerEntry{
		EntryID:        newID(),
		UserID:         actor.UserID,
		ProjectID:      meter.ProjectID,
		RoundSequence:  meter.RoundSequence,
		AttemptID:      meter.AttemptID,
		SessionID:      sessionID,
		EntryType:      EntryReversal,
		Seconds:        meter.ReservationSeconds,
		Reason:         reason,
		BalanceAfter:   balanceAfter,
		IdempotencyKey: idemKey,
		DataRegion:     actor.DataRegion,
		CreatedAt:      s.now().UTC(),
	}
	if err := s.store.AppendLedger(entry); err != nil {
		return LedgerEntry{}, err
	}
	meter.Refunded = true
	if err := s.store.SaveMeter(meter); err != nil {
		return LedgerEntry{}, err
	}
	return entry, nil
}

// GetLedger 列出项目账本（逐笔账单，FR-033）。
func (s *Service) GetLedger(
	_ context.Context, actor Actor, projectID string,
) ([]LedgerEntry, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	return s.store.GetLedgerByProject(actor.DataRegion, projectID)
}

func (s *Service) currentAccumulated(actor Actor, sessionID string) int {
	meter, err := s.store.GetMeter(actor.DataRegion, sessionID)
	if err != nil {
		return 0
	}
	return meter.AccumulatedSeconds
}

func (s *Service) restoreEntitlements(_ context.Context, actor Actor, seconds int) error {
	if seconds <= 0 {
		return nil
	}
	items, err := s.store.ListEntitlements(actor.DataRegion, actor.UserID)
	if err != nil {
		return err
	}
	remaining := seconds
	for i := range items {
		if remaining <= 0 {
			break
		}
		if items[i].Status != "active" || items[i].ConsumedSeconds <= 0 {
			continue
		}
		restore := items[i].ConsumedSeconds
		if restore > remaining {
			restore = remaining
		}
		items[i].ConsumedSeconds -= restore
		remaining -= restore
		if err := s.store.UpdateEntitlement(items[i]); err != nil {
			return err
		}
	}
	return nil
}

// consumeEntitlements 按顺序消费：免费 → 项目包（限本项目）→ Pro → 加油包。
func consumeEntitlements(
	items []Entitlement, seconds int, projectID string,
) (map[string]int, error) {
	order := []string{KindFreeCredit, KindProjectPack, KindProSub, KindTopup}
	consumed := make(map[string]int)
	remaining := seconds
	for _, kind := range order {
		if remaining <= 0 {
			break
		}
		for _, e := range items {
			if e.Status != "active" || e.Kind != kind {
				continue
			}
			if kind == KindProjectPack && e.ScopeProjectID != "" && e.ScopeProjectID != projectID {
				continue
			}
			available := e.Remaining()
			if available <= 0 {
				continue
			}
			take := available
			if take > remaining {
				take = remaining
			}
			consumed[e.EntitlementID] += take
			remaining -= take
		}
	}
	if remaining > 0 {
		return nil, fmt.Errorf("余额不足（缺 %d 秒）", remaining)
	}
	return consumed, nil
}
