// Package org 提供按任务细粒度结果授权（TASK-072；FR-035，US-07 场景 1/2；
// SCREEN-SPEC SCR-16；openapi /v1/assignments/{assignmentId}/shares*）。
// 红线：范围+有效期、可撤回；到期自动失效；"已完成未共享"只显示状态不显示失败。
package org

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 分享状态。
const (
	ShareActive    = "active"
	ShareWithdrawn = "withdrawn"
)

// 结果分享类别（六类封闭枚举；openapi ConsentGrant data_categories 对齐）。
var validShareCategories = map[string]bool{
	"total_score": true, "radar": true, "round_results": true,
	"full_report": true, "transcript": true, "media": true,
}

// Share 为按任务细粒度结果授权（范围+有效期+可撤回；到期自动失效）。
type Share struct {
	ShareID        string
	AssignmentID   string
	OrgID          string
	UserID         string
	DataCategories []string
	ExpiresAt      time.Time
	Status         string
	WithdrawnAt    *time.Time
	DataRegion     string
	IdempotencyKey string
	CreatedAt      time.Time
}

// GrantAssignmentShare 用户针对具体任务授权分享结果（范围+有效期；幂等）。
func (s *Service) GrantAssignmentShare(
	_ context.Context, actor Actor, assignmentID string, categories []string,
	expiresAt time.Time, idemKey string,
) (Share, error) {
	if err := validateActor(actor); err != nil {
		return Share{}, err
	}
	if len(categories) == 0 || expiresAt.IsZero() || strings.TrimSpace(idemKey) == "" {
		return Share{}, fmt.Errorf("%w: data_categories、expires_at 与幂等键必填", ErrInvalidInput)
	}
	for _, c := range categories {
		if !validShareCategories[c] {
			return Share{}, fmt.Errorf("%w: 非法分享类别 %s", ErrInvalidInput, c)
		}
	}
	if !s.now().UTC().Before(expiresAt) {
		return Share{}, fmt.Errorf("%w: 有效期必须晚于当前时间", ErrInvalidInput)
	}
	assignment, err := s.store.GetAssignmentByID(actor.DataRegion, assignmentID)
	if err != nil {
		return Share{}, err
	}
	if _, err := s.store.GetMember(assignment.OrgID, actor.UserID); err != nil {
		return Share{}, err
	}
	if cached, err := s.store.GetShareByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Share{}, err
	}
	share := Share{
		ShareID:        newID(),
		AssignmentID:   assignmentID,
		OrgID:          assignment.OrgID,
		UserID:         actor.UserID,
		DataCategories: append([]string(nil), categories...),
		ExpiresAt:      expiresAt,
		Status:         ShareActive,
		DataRegion:     actor.DataRegion,
		IdempotencyKey: idemKey,
		CreatedAt:      s.now().UTC(),
	}
	if err := s.store.SaveShare(share, idemKey); err != nil {
		return Share{}, err
	}
	return share, nil
}

// RevokeAssignmentShare 用户撤回分享授权（在线访问立即失效；写审计）。
func (s *Service) RevokeAssignmentShare(
	_ context.Context, actor Actor, assignmentID, shareID, _ string,
) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	share, err := s.store.GetShareByID(actor.DataRegion, shareID)
	if err != nil {
		return err
	}
	if share.AssignmentID != assignmentID || share.UserID != actor.UserID {
		return ErrNotFound
	}
	if share.Status == ShareWithdrawn {
		return nil
	}
	now := s.now().UTC()
	share.Status = ShareWithdrawn
	share.WithdrawnAt = &now
	if err := s.store.UpdateShare(share); err != nil {
		return err
	}
	return s.appendAudit(actor, share.OrgID, "org.share.withdrawn", shareID)
}

// ListShares 列出用户在任务上的分享授权。
func (s *Service) ListShares(
	_ context.Context, actor Actor, assignmentID string,
) ([]Share, error) {
	if err := validateActor(actor); err != nil {
		return nil, err
	}
	return s.store.ListSharesByUser(actor.DataRegion, actor.UserID, assignmentID)
}

// CheckShareEffective 机构侧访问校验：授权有效且未过期/未撤回；
// 有效则写 AccessAudit 并返回可见类别（"已完成未共享"由任务摘要展示）。
func (s *Service) CheckShareEffective(
	_ context.Context, actor Actor, orgID, assignmentID, userID, shareID string,
) ([]string, error) {
	if err := s.require(actor, orgID, PermViewAggregates); err != nil {
		return nil, err
	}
	org, err := s.store.GetOrgByID(actor.DataRegion, orgID)
	if err != nil || org.Status != OrgActive {
		return nil, ErrNotFound
	}
	share, err := s.store.GetShareByID(actor.DataRegion, shareID)
	if err != nil {
		return nil, err
	}
	if share.OrgID != orgID || share.AssignmentID != assignmentID || share.UserID != userID {
		return nil, ErrNotFound
	}
	if share.Status != ShareActive || s.now().UTC().After(share.ExpiresAt) {
		return nil, nil // 撤回/到期自动失效。
	}
	_ = s.appendAudit(actor, orgID, "org.result.accessed", shareID)
	return append([]string(nil), share.DataCategories...), nil
}

// ExpireShares 到期失效扫描（供定时任务调用；幂等）。
func (s *Service) ExpireShares(_ context.Context, dataRegion string) (int, error) {
	shares, err := s.store.ListActiveShares(dataRegion)
	if err != nil {
		return 0, err
	}
	now := s.now().UTC()
	count := 0
	for i := range shares {
		if now.After(shares[i].ExpiresAt) {
			shares[i].Status = ShareWithdrawn
			at := now
			shares[i].WithdrawnAt = &at
			if err := s.store.UpdateShare(shares[i]); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}
