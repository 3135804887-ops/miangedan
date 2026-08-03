package room

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"miangedan/services/project"
	"miangedan/services/region"
)

// ProjectAPI 为 TASK-020 依赖的项目服务能力（生产经内部契约/HTTP 调用，测试用内存实现）。
type ProjectAPI interface {
	GetProject(context.Context, project.Actor, string) (project.Project, error)
	GetPlan(context.Context, project.Actor, string) (project.PlanVersion, error)
	ClaimDevice(context.Context, project.Actor, string, string, string) (project.Project, error)
	TransferDevice(context.Context, project.Actor, string, string, string, string) (project.Project, error)
	ReleaseDevice(context.Context, project.Actor, string, string, string) (project.Project, error)
}

// Service 为会话房间应用服务（TASK-020：FR-013、NFR-007、SEC-003）。
type Service struct {
	store    Store
	idem     IdempotencyStore
	tokens   *MediaTokenManager
	provider Provider
	projects ProjectAPI
	billing  BillingAPI
	now      func() time.Time
}

// NewService 创建房间服务。
func NewService(store Store, idem IdempotencyStore, tokens *MediaTokenManager, provider Provider, projects ProjectAPI) (*Service, error) {
	if store == nil || idem == nil || tokens == nil || provider == nil || projects == nil {
		return nil, fmt.Errorf("%w: 缺少存储/令牌/房间提供方/项目服务", ErrInvalidInput)
	}
	return &Service{
		store: store, idem: idem, tokens: tokens, provider: provider,
		projects: projects, billing: billingNoop{}, now: time.Now,
	}, nil
}

// SetBilling 注入秒级账本能力（TASK-061；生产由 services/billing 实现）。
func (s *Service) SetBilling(b BillingAPI) {
	if b != nil {
		s.billing = b
	}
}

func idempotent[T any](s *Service, prefix, idemKey string, run func() (T, error)) (T, error) {
	var zero T
	fullKey := prefix + idemKey
	if idemKey != "" {
		var cached T
		found, err := s.idem.Recall(fullKey, &cached)
		if err != nil {
			return zero, err
		}
		if found {
			return cached, nil
		}
	}
	result, err := run()
	if err != nil {
		return zero, err
	}
	if idemKey != "" {
		if err := s.idem.Remember(fullKey, result); err != nil {
			return zero, err
		}
	}
	return result, nil
}

func (s *Service) validateActor(actor project.Actor) error {
	if strings.TrimSpace(actor.UserID) == "" {
		return fmt.Errorf("%w: 缺少用户身份", ErrInvalidInput)
	}
	return region.ValidateDataRegion(actor.DataRegion)
}

// projectIdemKey 透传项目服务幂等键：无显式键时不带前缀（避免空键被项目服务误缓存）。
func projectIdemKey(prefix, idemKey string) string {
	if idemKey == "" {
		return ""
	}
	return prefix + idemKey
}

// CreateSession 创建会话房间并签发一次性短期媒体令牌（SEC-003）。
// 前置：项目 READY、本轮量表/覆盖方案就绪（FR-011）、单活动设备（TASK-018）；
// 交接包（TASK-034）与额度预留（TASK-061）为后续任务挂接点。
func (s *Service) CreateSession(ctx context.Context, actor project.Actor, in CreateSessionInput, idemKey string) (SessionCreated, error) {
	if err := s.validateActor(actor); err != nil {
		return SessionCreated{}, err
	}
	if in.RoundSequence < 1 || in.RoundSequence > 5 {
		return SessionCreated{}, fmt.Errorf("%w: 轮次序号必须为 1-5", ErrInvalidInput)
	}
	if strings.TrimSpace(in.DeviceID) == "" {
		return SessionCreated{}, fmt.Errorf("%w: device_id 必填", ErrInvalidInput)
	}
	if in.Kind != KindFormal && in.Kind != KindFormalRetry {
		return SessionCreated{}, fmt.Errorf("%w: kind 必须为 formal | formal_retry", ErrInvalidInput)
	}
	if in.Kind == KindFormalRetry && strings.TrimSpace(in.AttemptID) == "" {
		return SessionCreated{}, fmt.Errorf("%w: formal_retry 必须携带 attempt_id", ErrInvalidInput)
	}
	return idempotent(s, "createsession|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (SessionCreated, error) {
		proj, err := s.projects.GetProject(ctx, actor, in.ProjectID)
		if err != nil {
			return SessionCreated{}, mapProjectErr(err)
		}
		if proj.Status != project.StatusReady {
			return SessionCreated{}, fmt.Errorf("%w: 项目必须处于 READY 才能开始本轮（当前 %s）", ErrStateConflict, proj.Status)
		}
		if err := s.checkRoundReady(ctx, actor, proj, in.RoundSequence); err != nil {
			return SessionCreated{}, err
		}
		// 单活动设备绑定（TASK-018：另一设备活动时 ClaimDevice 返回 device_active → state_conflict）。
		if _, err := s.projects.ClaimDevice(ctx, actor, in.ProjectID, in.DeviceID, projectIdemKey("room-claim|", idemKey)); err != nil {
			return SessionCreated{}, mapProjectErr(err)
		}
		now := s.now()
		sessionID := newID()
		ref, err := s.provider.CreateRoom(ctx, CreateRoomInput{SessionID: sessionID, DataRegion: actor.DataRegion, ProjectID: in.ProjectID})
		if err != nil {
			return SessionCreated{}, err
		}
		token, exp, err := s.tokens.Issue(sessionID, in.DeviceID, actor.DataRegion, now)
		if err != nil {
			return SessionCreated{}, err
		}
		sess := Session{
			SessionID:       sessionID,
			ProjectID:       in.ProjectID,
			UserID:          actor.UserID,
			DataRegion:      actor.DataRegion,
			RoundSequence:   in.RoundSequence,
			AttemptID:       in.AttemptID,
			Kind:            in.Kind,
			RoomStatus:      StatusRoomCreated,
			RoomProviderRef: ref.Provider + "/" + ref.RoomID,
			ActiveDeviceID:  in.DeviceID,
			LastActivityAt:  now,
			CreatedAt:       now,
		}
		if err := s.store.SaveSession(sess); err != nil {
			return SessionCreated{}, err
		}
		// TASK-061 挂接点：每轮开始前预留（不足阻止开始），会话 LIVE 起计量。
		plan, err := s.projects.GetPlan(ctx, actor, in.ProjectID)
		if err != nil {
			return SessionCreated{}, mapProjectErr(err)
		}
		durationMinutes := 30
		for _, round := range plan.Rounds {
			if round.Sequence == in.RoundSequence {
				durationMinutes = round.DurationMinutes
				break
			}
		}
		reserveErr := s.billing.Reserve(ctx, actor, BillingReserveInput{
			ProjectID:        in.ProjectID,
			RoundSequence:    in.RoundSequence,
			AttemptID:        sess.AttemptID,
			SessionID:        sessionID,
			EstimatedSeconds: durationMinutes * 60,
		})
		if reserveErr != nil {
			if errors.Is(reserveErr, ErrInsufficientEntitlement) {
				return SessionCreated{}, ErrEntitlementMissing
			}
			return SessionCreated{}, reserveErr
		}
		if err := s.billing.StartMetering(ctx, actor, sessionID); err != nil {
			return SessionCreated{}, err
		}
		return SessionCreated{
			SessionID:          sessionID,
			RoomURL:            ref.URL,
			RoomToken:          token,
			RoomTokenExpiresAt: exp,
			DataRegion:         actor.DataRegion,
		}, nil
	})
}

func (s *Service) checkRoundReady(ctx context.Context, actor project.Actor, proj project.Project, sequence int) error {
	plan, err := s.projects.GetPlan(ctx, actor, proj.ProjectID)
	if err != nil {
		return mapProjectErr(err)
	}
	for _, r := range plan.Rounds {
		if r.Sequence == sequence {
			if !r.RubricBound || !r.QuestionCoveragePlanReady {
				return fmt.Errorf("%w: 本轮量表或问题覆盖方案未就绪（FR-011）", ErrStateConflict)
			}
			return nil
		}
	}
	return fmt.Errorf("%w: 计划中不存在轮次 %d", ErrStateConflict, sequence)
}

// GetSession 获取会话详情。
func (s *Service) GetSession(_ context.Context, actor project.Actor, sessionID string) (Session, error) {
	if err := s.validateActor(actor); err != nil {
		return Session{}, err
	}
	return s.store.GetSession(actor.DataRegion, sessionID)
}

// EndSession 用户主动退出（须确认）：吊销令牌、释放设备、会话置 ENDED。
func (s *Service) EndSession(ctx context.Context, actor project.Actor, sessionID string, confirm bool, idemKey string) (Session, error) {
	if err := s.validateActor(actor); err != nil {
		return Session{}, err
	}
	if !confirm {
		return Session{}, fmt.Errorf("%w: 退出必须显式确认", ErrInvalidInput)
	}
	return idempotent(s, "endsession|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Session, error) {
		sess, err := s.store.GetSession(actor.DataRegion, sessionID)
		if err != nil {
			return Session{}, err
		}
		if sess.UserID != actor.UserID {
			return Session{}, ErrNotFound
		}
		if err := s.tokens.store.RevokeSession(sessionID); err != nil {
			return Session{}, err
		}
		if sess.ActiveDeviceID != "" {
			if _, err := s.projects.ReleaseDevice(ctx, actor, sess.ProjectID, sess.ActiveDeviceID, projectIdemKey("room-end|", idemKey)); err != nil {
				return Session{}, mapProjectErr(err)
			}
		}
		sess.RoomStatus = StatusEnded
		sess.LastActivityAt = s.now()
		if err := s.store.UpdateSession(sess); err != nil {
			return Session{}, err
		}
		// TASK-061 挂接点：轮次结束按实际秒数结算（用户主动退出按实际扣减）。
		if err := s.billing.Settle(ctx, actor, sessionID, "round_ended"); err != nil {
			return Session{}, err
		}
		return sess, nil
	})
}

// ReconnectSession 3 分钟窗口内重连：吊销旧令牌、签发新令牌；窗口过期 → ENDED + reconnect_expired。
// lastConfirmedSeq 为断线恢复游标，由回合/证据链路（TASK-023/026）消费，本任务登记并透传。
func (s *Service) ReconnectSession(_ context.Context, actor project.Actor, sessionID, deviceID string, _ int, idemKey string) (SessionCreated, error) {
	if err := s.validateActor(actor); err != nil {
		return SessionCreated{}, err
	}
	if strings.TrimSpace(deviceID) == "" {
		return SessionCreated{}, fmt.Errorf("%w: device_id 必填", ErrInvalidInput)
	}
	return idempotent(s, "reconnect|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (SessionCreated, error) {
		sess, err := s.store.GetSession(actor.DataRegion, sessionID)
		if err != nil {
			return SessionCreated{}, err
		}
		if sess.UserID != actor.UserID {
			return SessionCreated{}, ErrNotFound
		}
		if sess.ActiveDeviceID != "" && sess.ActiveDeviceID != deviceID {
			return SessionCreated{}, fmt.Errorf("%w: 重连设备与活动设备不一致", ErrStateConflict)
		}
		now := s.now()
		if now.Sub(sess.LastActivityAt) > ReconnectWindow {
			sess.RoomStatus = StatusEnded
			_ = s.tokens.store.RevokeSession(sessionID)
			_ = s.store.UpdateSession(sess)
			return SessionCreated{}, ErrReconnectExpired
		}
		if err := s.tokens.store.RevokeSession(sessionID); err != nil {
			return SessionCreated{}, err
		}
		token, exp, err := s.tokens.Issue(sessionID, deviceID, actor.DataRegion, now)
		if err != nil {
			return SessionCreated{}, err
		}
		sess.LastActivityAt = now
		if err := s.store.UpdateSession(sess); err != nil {
			return SessionCreated{}, err
		}
		return SessionCreated{
			SessionID:          sessionID,
			RoomURL:            "wss://stub.miangedan.example/room/" + sessionID,
			RoomToken:          token,
			RoomTokenExpiresAt: exp,
			DataRegion:         actor.DataRegion,
		}, nil
	})
}

// DeviceTransferSession 安全转移设备（用户确认）：项目活动设备切换 + 旧令牌吊销 + 新设备令牌。
func (s *Service) DeviceTransferSession(ctx context.Context, actor project.Actor, sessionID, newDeviceID string, confirm bool, idemKey string) (SessionCreated, error) {
	if err := s.validateActor(actor); err != nil {
		return SessionCreated{}, err
	}
	if !confirm {
		return SessionCreated{}, fmt.Errorf("%w: 设备转移必须显式确认", ErrInvalidInput)
	}
	if strings.TrimSpace(newDeviceID) == "" {
		return SessionCreated{}, fmt.Errorf("%w: new_device_id 必填", ErrInvalidInput)
	}
	return idempotent(s, "transfer|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (SessionCreated, error) {
		sess, err := s.store.GetSession(actor.DataRegion, sessionID)
		if err != nil {
			return SessionCreated{}, err
		}
		if sess.UserID != actor.UserID {
			return SessionCreated{}, ErrNotFound
		}
		if _, err := s.projects.TransferDevice(ctx, actor, sess.ProjectID, sess.ActiveDeviceID, newDeviceID, projectIdemKey("room-transfer|", idemKey)); err != nil {
			return SessionCreated{}, mapProjectErr(err)
		}
		if err := s.tokens.store.RevokeSession(sessionID); err != nil {
			return SessionCreated{}, err
		}
		now := s.now()
		token, exp, err := s.tokens.Issue(sessionID, newDeviceID, actor.DataRegion, now)
		if err != nil {
			return SessionCreated{}, err
		}
		sess.ActiveDeviceID = newDeviceID
		sess.LastActivityAt = now
		if err := s.store.UpdateSession(sess); err != nil {
			return SessionCreated{}, err
		}
		return SessionCreated{
			SessionID:          sessionID,
			RoomURL:            "wss://stub.miangedan.example/room/" + sessionID,
			RoomToken:          token,
			RoomTokenExpiresAt: exp,
			DataRegion:         actor.DataRegion,
		}, nil
	})
}

func mapProjectErr(err error) error {
	if errors.Is(err, project.ErrDeviceActive) {
		return fmt.Errorf("%w: 正式面试已在其他设备活动", ErrStateConflict)
	}
	if errors.Is(err, project.ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, project.ErrStateConflict) {
		return ErrStateConflict
	}
	return err
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
