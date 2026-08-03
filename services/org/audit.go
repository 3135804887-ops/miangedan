// Package org 提供机构访问审计与退出即时失效（TASK-074；FR-034/FR-035，
// US-07 场景 4；DOMAIN-MODEL §6.17）。
// 红线：退出/被移除/机构停用 → 成员令牌与共享链接立即失效；审计继续存在。
package org

import (
	"context"
	"fmt"
)

// ListAccessAudits 机构访问审计（谁/何时/访问了什么；privacy_auditor/owner 可见）。
func (s *Service) ListAccessAudits(
	_ context.Context, actor Actor, orgID string,
) ([]AuditEntry, error) {
	if err := s.require(actor, orgID, PermViewAudit); err != nil {
		return nil, err
	}
	return s.store.ListAudits(actor.DataRegion, orgID)
}

// RemoveMember 移除成员：机构访问立即失效（撤回其全部分享授权）；个人记录保留。
func (s *Service) RemoveMember(
	_ context.Context, actor Actor, orgID, userID string,
) error {
	if err := s.require(actor, orgID, PermManageRoles); err != nil {
		return err
	}
	member, err := s.store.GetMember(orgID, userID)
	if err != nil {
		return err
	}
	if member.Role == RoleOwner && actor.UserID != userID {
		return fmt.Errorf("%w: 仅所有者本人可移除所有者", ErrForbidden)
	}
	now := s.now().UTC()
	member.LeftAt = &now
	if err := s.store.UpdateMember(member); err != nil {
		return err
	}
	if err := s.revokeSharesForUser(orgID, userID); err != nil {
		return err
	}
	return s.appendAudit(actor, orgID, "org.member.removed", userID)
}

// InvalidateOrgAccess 机构停用/注销时调用：全部成员令牌与共享链接立即失效。
func (s *Service) InvalidateOrgAccess(
	_ context.Context, actor Actor, orgID string,
) (int, error) {
	if err := s.require(actor, orgID, PermManageOrg); err != nil {
		return 0, err
	}
	shares, err := s.store.ListActiveShares(actor.DataRegion)
	if err != nil {
		return 0, err
	}
	now := s.now().UTC()
	count := 0
	for i := range shares {
		if shares[i].OrgID == orgID {
			shares[i].Status = ShareWithdrawn
			shares[i].WithdrawnAt = &now
			if err := s.store.UpdateShare(shares[i]); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, s.appendAudit(actor, orgID, "org.access.invalidated", orgID)
}

// IsMemberAccessValid 授权层判定：成员存在且未退出（令牌失效判定）。
func (s *Service) IsMemberAccessValid(orgID, userID string) bool {
	member, err := s.store.GetMember(orgID, userID)
	if err != nil || member.LeftAt != nil {
		return false
	}
	return true
}

// revokeSharesForUser 撤回用户在机构全部任务上的分享授权（即时失效）。
func (s *Service) revokeSharesForUser(orgID, userID string) error {
	shares, err := s.store.ListActiveSharesForUserOrg(userID, orgID)
	if err != nil {
		return err
	}
	now := s.now().UTC()
	for i := range shares {
		shares[i].Status = ShareWithdrawn
		shares[i].WithdrawnAt = &now
		if err := s.store.UpdateShare(shares[i]); err != nil {
			return err
		}
	}
	return nil
}
