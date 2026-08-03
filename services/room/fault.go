// Package room 提供实时会话房间、字幕、岗位工具与故障控制能力。
// TASK-025 故障暂停计时、文字降级询问与额度联动挂接（FR-020）。
// 追踪：INTERVIEW-STATE-MACHINE.md 5.2（PAUSED_SYSTEM → DOWNGRADE_PROMPTED → TEXT_DEGRADED/ENDED）；
// realtime-events 7.2/7.7（timer.paused/resumed、avatar.downgrade_*）；TASK-061 额度预留为挂接点。
package room

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"miangedan/services/project"
)

// TimerPauseReason 为暂停计时原因（realtime-events 7.7 timer.paused）。
type TimerPauseReason string

// 暂停原因枚举。
const (
	PauseSystemFault     TimerPauseReason = "system_fault"
	PauseReconnect       TimerPauseReason = "reconnect"
	PauseAuthPaused      TimerPauseReason = "auth_paused"
	PauseDowngradePrompt TimerPauseReason = "downgrade_prompted"
)

// DowngradeStatus 为文字降级询问状态。
type DowngradeStatus string

// 降级状态枚举。
const (
	DowngradeNone     DowngradeStatus = "none"
	DowngradePrompted DowngradeStatus = "prompted"
	DowngradeAccepted DowngradeStatus = "accepted"
	DowngradeRejected DowngradeStatus = "rejected"
)

// EndReason 为会话结束原因（realtime-events 7.1 session.ended）。
type EndReason string

// 结束原因枚举。
const (
	EndCompleted         EndReason = "completed"
	EndUserExit          EndReason = "user_exit"
	EndUnrecoverable     EndReason = "unrecoverable"
	EndDowngradeRejected EndReason = "downgrade_rejected"
)

// 故障控制错误。
var (
	ErrTimerNotPaused   = errors.New("timer not paused")
	ErrDowngradeInvalid = errors.New("invalid downgrade")
	ErrSessionEnded     = errors.New("session ended")
)

func (s *Service) requireStatus(sess Session, allowed ...Status) error {
	for _, st := range allowed {
		if sess.RoomStatus == st {
			return nil
		}
	}
	return fmt.Errorf("%w: 会话当前状态 %s 不允许该操作", ErrStateConflict, sess.RoomStatus)
}

// PauseTimer 暂停计时（timer.paused）：LIVE → PAUSED_SYSTEM / AUTH_PAUSED / RECONNECTING。
// 幂等：同一次暂停重复触发只记录一次 paused_at。
func (s *Service) PauseTimer(_ context.Context, actor project.Actor, sessionID string, reason TimerPauseReason, idemKey string) (Session, error) {
	if err := s.validateActor(actor); err != nil {
		return Session{}, err
	}
	switch reason {
	case PauseSystemFault, PauseReconnect, PauseAuthPaused:
	default:
		return Session{}, fmt.Errorf("%w: 非法暂停原因", ErrInvalidInput)
	}
	return idempotent(s, "pausetimer|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Session, error) {
		sess, err := s.ownSession(actor, sessionID)
		if err != nil {
			return Session{}, err
		}
		if sess.RoomStatus == StatusEnded {
			return Session{}, ErrSessionEnded
		}
		now := s.now()
		switch reason {
		case PauseSystemFault:
			if sess.RoomStatus == StatusLive || sess.RoomStatus == StatusTextDegraded {
				sess.RoomStatus = StatusPausedSystem
				sess.PausedAt = &now
			}
		case PauseReconnect:
			if sess.RoomStatus == StatusLive || sess.RoomStatus == StatusTextDegraded {
				sess.RoomStatus = StatusReconnecting
				sess.PausedAt = &now
			}
		case PauseAuthPaused:
			if sess.RoomStatus == StatusLive || sess.RoomStatus == StatusTextDegraded {
				sess.RoomStatus = StatusAuthPaused
				sess.PausedAt = &now
			}
		}
		// TASK-061 挂接点：暂停段不计费（停止计量）。
		if err := s.billing.StopMetering(context.Background(), actor, sessionID); err != nil {
			return Session{}, err
		}
		if err := s.store.UpdateSession(sess); err != nil {
			return Session{}, err
		}
		return sess, nil
	})
}

// ResumeTimer 恢复计时（timer.resumed）：PAUSED_SYSTEM / AUTH_PAUSED → LIVE；
// RECONNECTING 由 ReconnectSession 处理。累计暂停秒数，恢复后计时继续。
func (s *Service) ResumeTimer(_ context.Context, actor project.Actor, sessionID string, idemKey string) (Session, error) {
	if err := s.validateActor(actor); err != nil {
		return Session{}, err
	}
	return idempotent(s, "resumetimer|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Session, error) {
		sess, err := s.ownSession(actor, sessionID)
		if err != nil {
			return Session{}, err
		}
		if sess.RoomStatus != StatusPausedSystem && sess.RoomStatus != StatusAuthPaused {
			return Session{}, fmt.Errorf("%w: 仅 PAUSED_SYSTEM/AUTH_PAUSED 可恢复计时", ErrTimerNotPaused)
		}
		now := s.now()
		if sess.PausedAt != nil {
			sess.PausedSeconds += int(now.Sub(*sess.PausedAt).Seconds())
		}
		sess.PausedAt = nil
		sess.RoomStatus = StatusLive
		// TASK-061 挂接点：恢复 LIVE 继续计量。
		if err := s.billing.StartMetering(context.Background(), actor, sessionID); err != nil {
			return Session{}, err
		}
		if err := s.store.UpdateSession(sess); err != nil {
			return Session{}, err
		}
		return sess, nil
	})
}

// OfferDowngrade 发起文字降级询问（avatar.downgrade_prompted）：PAUSED_SYSTEM → DOWNGRADE_PROMPTED。
// 返回 prompt_id；重复触发返回同一 prompt_id。
func (s *Service) OfferDowngrade(_ context.Context, actor project.Actor, sessionID string, idemKey string) (string, error) {
	if err := s.validateActor(actor); err != nil {
		return "", err
	}
	sess, err := idempotent(s, "downgrade-offer|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Session, error) {
		sess, err := s.ownSession(actor, sessionID)
		if err != nil {
			return Session{}, err
		}
		if sess.DowngradeStatus == DowngradePrompted || sess.DowngradeStatus == DowngradeAccepted {
			return sess, nil
		}
		if err := s.requireStatus(sess, StatusPausedSystem, StatusLive); err != nil {
			return Session{}, err
		}
		now := s.now()
		sess.RoomStatus = StatusDowngradePrompted
		sess.DowngradeStatus = DowngradePrompted
		sess.DowngradePromptID = newID()
		sess.PausedAt = &now
		if err := s.store.UpdateSession(sess); err != nil {
			return Session{}, err
		}
		return sess, nil
	})
	if err != nil {
		return "", err
	}
	return sess.DowngradePromptID, nil
}

// AcceptDowngrade 同意文字降级（avatar.downgrade_accepted）：DOWNGRADE_PROMPTED → TEXT_DEGRADED。
// 故障点起不再消耗数字人额度（TASK-061 额度联动挂接点）；口语项按文字模式规则处理。
func (s *Service) AcceptDowngrade(ctx context.Context, actor project.Actor, sessionID, promptID string, idemKey string) (Session, error) {
	if err := s.validateActor(actor); err != nil {
		return Session{}, err
	}
	if strings.TrimSpace(promptID) == "" {
		return Session{}, fmt.Errorf("%w: prompt_id 必填", ErrDowngradeInvalid)
	}
	return idempotent(s, "downgrade-accept|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Session, error) {
		sess, err := s.ownSession(actor, sessionID)
		if err != nil {
			return Session{}, err
		}
		if err := s.requireStatus(sess, StatusDowngradePrompted); err != nil {
			return Session{}, err
		}
		if sess.DowngradePromptID != promptID {
			return Session{}, fmt.Errorf("%w: prompt_id 不匹配", ErrDowngradeInvalid)
		}
		now := s.now()
		if sess.PausedAt != nil {
			sess.PausedSeconds += int(now.Sub(*sess.PausedAt).Seconds())
		}
		sess.PausedAt = nil
		sess.RoomStatus = StatusTextDegraded
		sess.DowngradeStatus = DowngradeAccepted
		sess.TextDegradedAt = &now
		// TASK-061 挂接点：故障点起不计数字人额度（停止计量，文字面试继续）。
		if err := s.billing.StopMetering(ctx, actor, sessionID); err != nil {
			return Session{}, err
		}
		if err := s.store.UpdateSession(sess); err != nil {
			return Session{}, err
		}
		return sess, nil
	})
}

// DeclineDowngrade 拒绝降级（avatar.downgrade_rejected）：DOWNGRADE_PROMPTED → ENDED。
// 项目 → EVALUATION_INCOMPLETE（评估未完成 ≠ 失败）；系统责任全额返还（TASK-061 挂接点）。
func (s *Service) DeclineDowngrade(ctx context.Context, actor project.Actor, sessionID, promptID string, idemKey string) (Session, error) {
	if err := s.validateActor(actor); err != nil {
		return Session{}, err
	}
	if strings.TrimSpace(promptID) == "" {
		return Session{}, fmt.Errorf("%w: prompt_id 必填", ErrDowngradeInvalid)
	}
	return idempotent(s, "downgrade-decline|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Session, error) {
		sess, err := s.ownSession(actor, sessionID)
		if err != nil {
			return Session{}, err
		}
		if err := s.requireStatus(sess, StatusDowngradePrompted); err != nil {
			return Session{}, err
		}
		if sess.DowngradePromptID != promptID {
			return Session{}, fmt.Errorf("%w: prompt_id 不匹配", ErrDowngradeInvalid)
		}
		now := s.now()
		if sess.PausedAt != nil {
			sess.PausedSeconds += int(now.Sub(*sess.PausedAt).Seconds())
		}
		sess.PausedAt = nil
		sess.RoomStatus = StatusEnded
		sess.DowngradeStatus = DowngradeRejected
		sess.EndReason = EndDowngradeRejected
		sess.EndedAt = &now
		if err := s.tokens.store.RevokeSession(sessionID); err != nil {
			return Session{}, err
		}
		if sess.ActiveDeviceID != "" {
			if _, err := s.projects.ReleaseDevice(ctx, actor, sess.ProjectID, sess.ActiveDeviceID, projectIdemKey("room-decline|", idemKey)); err != nil {
				return Session{}, mapProjectErr(err)
			}
		}
		// TASK-061 挂接点：拒绝降级 = 系统责任，全额返还本轮预留额度。
		if err := s.billing.RefundFull(ctx, actor, sessionID,
			"downgrade_rejected_full_refund"); err != nil {
			return Session{}, err
		}
		if err := s.store.UpdateSession(sess); err != nil {
			return Session{}, err
		}
		return sess, nil
	})
}
