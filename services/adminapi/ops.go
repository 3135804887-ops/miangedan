// Package adminapi 提供运营后台：区域/房间/供应商/SLO 监控，默认匿名技术指标
// （TASK-080；FR-037，US-08 场景 1；SCREEN-SPEC SCR-17）。
// 红线：运营不可旁听/代答（无加入路径，尝试即拒绝并写审计）。
package adminapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"miangedan/services/region"
)

// Service 为运营后台服务。
type Service struct {
	store Store
	now   func() time.Time
}

// NewService 创建运营后台服务。
func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: 缺少存储", ErrInvalidInput)
	}
	return &Service{store: store, now: time.Now}, nil
}

// RecordRoomSnapshot 记录匿名会话技术快照（仅匿名会话编号与技术指标）。
func (s *Service) RecordRoomSnapshot(
	_ context.Context, actor Actor, snapshot RoomSnapshot,
) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if strings.TrimSpace(snapshot.AnonymousSessionID) == "" || snapshot.Region == "" {
		return fmt.Errorf("%w: 匿名会话编号与区域必填", ErrInvalidInput)
	}
	snapshot.SnapshotID = newID()
	snapshot.CreatedAt = s.now().UTC()
	return s.store.SaveRoomSnapshot(snapshot)
}

// RecordProviderHealth 记录供应商匿名技术指标。
func (s *Service) RecordProviderHealth(
	_ context.Context, actor Actor, health ProviderHealth,
) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	provider, err := s.store.GetProvider(actor.DataRegion, health.ProviderID)
	if err != nil {
		return err
	}
	provider.LatencyP95Ms = health.LatencyP95Ms
	provider.ErrorRate = health.ErrorRate
	provider.CircuitBreaker = health.CircuitBreaker
	provider.Status = health.Status
	return s.store.UpdateProvider(provider)
}

// ListRegionStatus 返回区域监控（在线房间/排队/容量/供应商健康/SLO/错误预算）。
func (s *Service) ListRegionStatus(
	_ context.Context, actor Actor, dataRegion string,
) (RegionOpsStatus, error) {
	if err := s.requireOps(actor, dataRegion); err != nil {
		return RegionOpsStatus{}, err
	}
	return s.store.GetRegionStatus(dataRegion)
}

// ListRooms 返回区域匿名房间列表（不含身份与内容）。
func (s *Service) ListRooms(
	_ context.Context, actor Actor, dataRegion string,
) ([]RoomSnapshot, error) {
	if err := s.requireOps(actor, dataRegion); err != nil {
		return nil, err
	}
	return s.store.ListRoomSnapshots(dataRegion)
}

// UpdateProviderStatus 供应商状态变更（灰度/禁用；故障切换必须记录原因并写审计）。
func (s *Service) UpdateProviderStatus(
	_ context.Context, actor Actor, providerID, status string, rampPercent int, reason string,
) (ProviderInfo, error) {
	if err := s.requireOps(actor, ""); err != nil {
		return ProviderInfo{}, err
	}
	if status != ProviderActive && status != ProviderRamping && status != ProviderDisabled {
		return ProviderInfo{}, fmt.Errorf("%w: 状态非法", ErrInvalidInput)
	}
	if status == ProviderDisabled && strings.TrimSpace(reason) == "" {
		return ProviderInfo{}, fmt.Errorf("%w: 故障切换必须记录原因", ErrInvalidInput)
	}
	provider, err := s.store.GetProvider(actor.DataRegion, providerID)
	if err != nil {
		return ProviderInfo{}, err
	}
	provider.Status = status
	provider.RampPercent = rampPercent
	provider.Note = reason
	if err := s.store.UpdateProvider(provider); err != nil {
		return ProviderInfo{}, err
	}
	if err := s.appendAudit(actor, "provider.status_changed", providerID); err != nil {
		return ProviderInfo{}, err
	}
	return provider, nil
}

// OperatorSessionGuard 运营会话红线：加入/旁听/代答一律拒绝并写审计。
func (s *Service) OperatorSessionGuard(
	_ context.Context, actor Actor, action string,
) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	switch action {
	case "join", "eavesdrop", "answer":
	default:
		return fmt.Errorf("%w: 非法动作", ErrInvalidInput)
	}
	_ = s.appendAudit(actor, "operator.session."+action+"_blocked", action)
	return fmt.Errorf("%w: 运营不可加入/旁听/代答", ErrForbidden)
}

// ListAudits 查询追加式审计日志（管理员不可删除；默认脱敏）。
func (s *Service) ListAudits(
	_ context.Context, actor Actor, dataRegion string,
) ([]AuditEntry, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	return s.store.ListAudits(dataRegion)
}

func (s *Service) requireOps(actor Actor, dataRegion string) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if actor.Role != RoleOps && actor.Role != RoleSuperAdmin {
		return fmt.Errorf("%w: 需要实时运营角色", ErrForbidden)
	}
	if dataRegion != "" && dataRegion != actor.DataRegion {
		return fmt.Errorf("%w: 跨区访问拒绝", ErrForbidden)
	}
	return nil
}

func (s *Service) appendAudit(actor Actor, action, target string) error {
	return s.store.AppendAudit(AuditEntry{
		AuditID:    newID(),
		StaffID:    actor.StaffID,
		Role:       actor.Role,
		Action:     action,
		TargetRef:  target,
		DataRegion: actor.DataRegion,
		CreatedAt:  s.now().UTC(),
	})
}

func validateActor(actor Actor) error {
	if strings.TrimSpace(actor.StaffID) == "" {
		return fmt.Errorf("%w: 缺少后台身份", ErrInvalidInput)
	}
	return region.ValidateDataRegion(actor.DataRegion)
}
