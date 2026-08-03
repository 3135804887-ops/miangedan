// Package adminapi 提供禁止改分的系统级约束与破窗访问流程（TASK-082；
// FR-039，US-08 场景 3；AGENTS.md §2 禁止项）。
// 红线：后台无编辑分数/解锁控件（与前端 control-registry 呼应）；正式复核唯一入口；
//
//	破窗访问限重大安全/法律事件，限定理由与时长并事后复核；存储只 SELECT/INSERT。
package adminapi

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// 破窗访问时长上限（≤8 小时）与事后复核期限（72 小时）。
const (
	MaxBreakGlassMinutes = 480
	ReviewWindowHours    = 72
)

// FormalReviewOnlyEntry 为个体结果唯一入口说明（正式复核服务产生新版本）。
const FormalReviewOnlyEntry = "services/scoring 正式复核是产生新分数版本的唯一入口；后台无编辑分数/解锁路径"

// 破窗访问状态（由追加式评审事件推导）。
const (
	GlassOpen     = "open"
	GlassReviewed = "reviewed"
	GlassExpired  = "expired"
)

// BreakGlass 为一次破窗访问（限重大安全/法律事件；限定理由与时长）。
type BreakGlass struct {
	GlassID         string
	TargetUserID    string
	Reason          string
	DurationMinutes int
	TargetRef       string
	DataRegion      string
	OpenedBy        string
	OpenedAt        time.Time
	ExpiresAt       time.Time
}

// BreakGlassReview 为破窗事后复核（追加式；与开启者必须不同人）。
type BreakGlassReview struct {
	ReviewID   string
	GlassID    string
	ReviewerID string
	Decision   string
	Note       string
	ReviewedAt time.Time
}

// ScoreWriteAction 为被禁止的后台改分动作（与前端 control-registry 红线呼应）。
func ScoreWriteAction(action string) bool {
	switch action {
	case "edit_score", "unlock_round", "edit_evidence":
		return true
	}
	return false
}

// AttemptScoreWrite 后台改分系统级约束：任何编辑分数/解锁/改证据的尝试一律拒绝并写审计。
func (s *Service) AttemptScoreWrite(
	_ context.Context, actor Actor, action, target string,
) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if !ScoreWriteAction(action) {
		return fmt.Errorf("%w: 非法动作", ErrInvalidInput)
	}
	_ = s.appendAudit(actor, "score_write."+action+"_blocked", target)
	return fmt.Errorf("%w: %s；个体结果只能经正式复核产生新版本", ErrForbidden, FormalReviewOnlyEntry)
}

// OpenBreakGlass 破窗访问：限重大安全或法律事件；限定理由与时长；通知用户由上层触发。
func (s *Service) OpenBreakGlass(
	_ context.Context, actor Actor, targetUserID, reason, targetRef string, durationMinutes int,
) (BreakGlass, error) {
	if err := requireRole(actor, RolePrivacySecurity); err != nil {
		return BreakGlass{}, err
	}
	if strings.TrimSpace(reason) == "" || strings.TrimSpace(targetUserID) == "" {
		return BreakGlass{}, fmt.Errorf("%w: 理由与目标用户必填", ErrInvalidInput)
	}
	if durationMinutes < 1 || durationMinutes > MaxBreakGlassMinutes {
		return BreakGlass{}, fmt.Errorf("%w: 破窗时长须为 1-%d 分钟", ErrInvalidInput, MaxBreakGlassMinutes)
	}
	now := s.now().UTC()
	glass := BreakGlass{
		GlassID:         newID(),
		TargetUserID:    targetUserID,
		Reason:          reason,
		DurationMinutes: durationMinutes,
		TargetRef:       targetRef,
		DataRegion:      actor.DataRegion,
		OpenedBy:        actor.StaffID,
		OpenedAt:        now,
		ExpiresAt:       now.Add(time.Duration(durationMinutes) * time.Minute),
	}
	if err := s.store.SaveBreakGlass(glass); err != nil {
		return BreakGlass{}, err
	}
	_ = s.appendAudit(actor, "break_glass.opened", glass.GlassID)
	return glass, nil
}

// ReviewBreakGlass 破窗事后复核（72 小时内；复核人不得与开启者相同）。
func (s *Service) ReviewBreakGlass(
	_ context.Context, actor Actor, glassID, decision, note string,
) (BreakGlassReview, error) {
	if err := requireRole(actor, RolePrivacySecurity); err != nil {
		return BreakGlassReview{}, err
	}
	if decision != "approved" && decision != "rejected" {
		return BreakGlassReview{}, fmt.Errorf("%w: 复核结论非法", ErrInvalidInput)
	}
	glass, err := s.store.GetBreakGlass(actor.DataRegion, glassID)
	if err != nil {
		return BreakGlassReview{}, err
	}
	if glass.OpenedBy == actor.StaffID {
		return BreakGlassReview{}, fmt.Errorf("%w: 开启者不可自审", ErrForbidden)
	}
	reviews, err := s.store.ListBreakGlassReviews(glassID)
	if err != nil {
		return BreakGlassReview{}, err
	}
	if len(reviews) > 0 {
		return BreakGlassReview{}, fmt.Errorf("%w: 已复核，不可重复复核", ErrStateConflict)
	}
	if s.now().UTC().After(glass.OpenedAt.Add(ReviewWindowHours * time.Hour)) {
		return BreakGlassReview{}, fmt.Errorf("%w: 超出 72 小时事后复核窗口", ErrStateConflict)
	}
	review := BreakGlassReview{
		ReviewID:   newID(),
		GlassID:    glassID,
		ReviewerID: actor.StaffID,
		Decision:   decision,
		Note:       note,
		ReviewedAt: s.now().UTC(),
	}
	if err := s.store.AppendBreakGlassReview(review); err != nil {
		return BreakGlassReview{}, err
	}
	_ = s.appendAudit(actor, "break_glass.reviewed", glassID)
	return review, nil
}

// GetBreakGlass 查询破窗访问及其有效状态（open → reviewed/expired）。
func (s *Service) GetBreakGlass(
	_ context.Context, actor Actor, glassID string,
) (BreakGlass, string, error) {
	if err := validateActor(actor); err != nil {
		return BreakGlass{}, "", err
	}
	glass, err := s.store.GetBreakGlass(actor.DataRegion, glassID)
	if err != nil {
		return BreakGlass{}, "", err
	}
	reviews, err := s.store.ListBreakGlassReviews(glassID)
	if err != nil {
		return BreakGlass{}, "", err
	}
	status := GlassOpen
	if len(reviews) > 0 {
		status = GlassReviewed
	} else if s.now().UTC().After(glass.ExpiresAt) {
		status = GlassExpired
	}
	return glass, status, nil
}
