// Package org 提供机构租户服务（TASK-070；FR-034，US-07；DOMAIN-MODEL §6.16）。
package org

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"miangedan/services/region"
)

// 默认邀请有效期。
const invitationTTL = 14 * 24 * time.Hour

// 六类角色权限矩阵（默认分离：owner 全量；admin 不含财务/审计；
// instructor 只管任务与聚合；privacy_auditor 只看审计；finance 只看财务；
// candidate 只参与任务）。
var rolePermissions = map[string]map[string]bool{
	RoleOwner: {
		PermManageOrg: true, PermInviteMembers: true, PermManageRoles: true,
		PermManageAssignments: true, PermViewAggregates: true, PermViewAudit: true,
		PermViewFinance: true, PermParticipate: true,
	},
	RoleAdmin: {
		PermManageOrg: true, PermInviteMembers: true, PermManageRoles: true,
		PermManageAssignments: true, PermViewAggregates: true, PermParticipate: true,
	},
	RoleInstructor: {
		PermManageAssignments: true, PermViewAggregates: true, PermParticipate: true,
	},
	RolePrivacyAuditor: {
		PermViewAudit: true,
	},
	RoleFinance: {
		PermViewFinance: true,
	},
	RoleCandidate: {
		PermParticipate: true,
	},
}

// Service 为机构租户服务。
type Service struct {
	store Store
	now   func() time.Time
}

// NewService 创建机构服务。
func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: 缺少存储", ErrInvalidInput)
	}
	return &Service{store: store, now: time.Now}, nil
}

// Can 查询角色是否拥有权限。
func Can(role, permission string) bool {
	return rolePermissions[role][permission]
}

// CreateOrg 创建机构租户：创建者以个人账户加入并成为所有者（无影子账户）。
func (s *Service) CreateOrg(
	_ context.Context, actor Actor, name, idemKey string,
) (Org, error) {
	if err := validateActor(actor); err != nil {
		return Org{}, err
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(idemKey) == "" {
		return Org{}, fmt.Errorf("%w: name 与幂等键必填", ErrInvalidInput)
	}
	if cached, err := s.store.GetOrgByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Org{}, err
	}
	org := Org{
		OrgID:          newID(),
		Name:           strings.TrimSpace(name),
		DataRegion:     actor.DataRegion,
		Status:         OrgActive,
		CreatedBy:      actor.UserID,
		CreatedAt:      s.now().UTC(),
		IdempotencyKey: idemKey,
	}
	if err := s.store.SaveOrg(org, idemKey); err != nil {
		return Org{}, err
	}
	member := Member{
		OrgID:        org.OrgID,
		UserID:       actor.UserID,
		Role:         RoleOwner,
		InviteMethod: InviteLink,
		JoinedAt:     s.now().UTC(),
	}
	if err := s.store.SaveMember(member); err != nil {
		return Org{}, err
	}
	if err := s.appendAudit(actor, org.OrgID, "org.created", org.OrgID); err != nil {
		return Org{}, err
	}
	return org, nil
}

// GetOrg 返回机构详情（调用者须为有效成员；按角色过滤由上层实现）。
func (s *Service) GetOrg(
	_ context.Context, actor Actor, orgID string,
) (Org, error) {
	if err := validateActor(actor); err != nil {
		return Org{}, err
	}
	member, err := s.store.GetMember(orgID, actor.UserID)
	if err != nil || member.LeftAt != nil {
		return Org{}, ErrNotFound
	}
	return s.store.GetOrgByID(actor.DataRegion, orgID)
}

// InviteMember 邀请成员（owner/admin；link/org_email/bulk_list/sso/scim 适配点）。
func (s *Service) InviteMember(
	_ context.Context, actor Actor, orgID, method, role, email, idemKey string,
) (Invitation, error) {
	if err := s.require(actor, orgID, PermInviteMembers); err != nil {
		return Invitation{}, err
	}
	if !validRole(role) || !validMethod(method) || strings.TrimSpace(idemKey) == "" {
		return Invitation{}, fmt.Errorf("%w: 角色/邀请方式非法", ErrInvalidInput)
	}
	if method == InviteOrgEmail && strings.TrimSpace(email) == "" {
		return Invitation{}, fmt.Errorf("%w: 机构邮箱邀请必须提供 email", ErrInvalidInput)
	}
	if cached, err := s.store.GetInvitationByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Invitation{}, err
	}
	org, err := s.store.GetOrgByID(actor.DataRegion, orgID)
	if err != nil {
		return Invitation{}, err
	}
	if org.Status != OrgActive {
		return Invitation{}, fmt.Errorf("%w: 机构非 active", ErrStateConflict)
	}
	invitation := Invitation{
		InvitationID:   newID(),
		OrgID:          orgID,
		Email:          strings.TrimSpace(email),
		Role:           role,
		InviteMethod:   method,
		Status:         InvitePending,
		ExpiresAt:      s.now().UTC().Add(invitationTTL),
		DataRegion:     actor.DataRegion,
		IdempotencyKey: idemKey,
		CreatedAt:      s.now().UTC(),
	}
	if err := s.store.SaveInvitation(invitation, idemKey); err != nil {
		return Invitation{}, err
	}
	if err := s.appendAudit(actor, orgID, "org.member.invited", invitation.InvitationID); err != nil {
		return Invitation{}, err
	}
	return invitation, nil
}

// AcceptInvitation 用户以个人账户加入机构（无影子账户；过期/停用拒绝）。
func (s *Service) AcceptInvitation(
	_ context.Context, actor Actor, invitationID string,
) (Member, error) {
	if err := validateActor(actor); err != nil {
		return Member{}, err
	}
	invitation, err := s.store.GetInvitationByID(actor.DataRegion, invitationID)
	if err != nil {
		return Member{}, err
	}
	if invitation.Status != InvitePending {
		return Member{}, fmt.Errorf("%w: 邀请已使用或撤销", ErrStateConflict)
	}
	if s.now().UTC().After(invitation.ExpiresAt) {
		return Member{}, fmt.Errorf("%w: 邀请已过期", ErrStateConflict)
	}
	org, err := s.store.GetOrgByID(actor.DataRegion, invitation.OrgID)
	if err != nil {
		return Member{}, err
	}
	if org.Status != OrgActive {
		return Member{}, fmt.Errorf("%w: 机构未启用", ErrStateConflict)
	}
	if invitation.Email != "" && !strings.EqualFold(invitation.Email, actor.UserID) {
		return Member{}, fmt.Errorf("%w: 邮箱不匹配", ErrForbidden)
	}
	member := Member{
		OrgID:        invitation.OrgID,
		UserID:       actor.UserID,
		Role:         invitation.Role,
		InviteMethod: invitation.InviteMethod,
		JoinedAt:     s.now().UTC(),
	}
	if err := s.store.SaveMember(member); err != nil {
		return Member{}, err
	}
	invitation.Status = InviteAccepted
	if err := s.store.UpdateInvitation(invitation); err != nil {
		return Member{}, err
	}
	if err := s.appendAudit(actor, invitation.OrgID, "org.member.joined", invitation.OrgID); err != nil {
		return Member{}, err
	}
	return member, nil
}

// ListMembers 列出机构成员（owner/admin/instructor）。
func (s *Service) ListMembers(
	_ context.Context, actor Actor, orgID string,
) ([]Member, error) {
	if err := s.require(actor, orgID, PermManageAssignments); err != nil {
		return nil, err
	}
	return s.store.ListMembers(orgID)
}

// SetMemberRole 调整成员角色（owner 仅 owner 本人可变更；默认分离不可越权）。
func (s *Service) SetMemberRole(
	_ context.Context, actor Actor, orgID, userID, role string,
) (Member, error) {
	if err := s.require(actor, orgID, PermManageRoles); err != nil {
		return Member{}, err
	}
	if !validRole(role) {
		return Member{}, fmt.Errorf("%w: 角色非法", ErrInvalidInput)
	}
	member, err := s.store.GetMember(orgID, userID)
	if err != nil {
		return Member{}, err
	}
	if member.Role == RoleOwner && actor.UserID != userID {
		return Member{}, fmt.Errorf("%w: 仅所有者本人可变更所有者角色", ErrForbidden)
	}
	member.Role = role
	if err := s.store.UpdateMember(member); err != nil {
		return Member{}, err
	}
	if err := s.appendAudit(actor, orgID, "org.member.role_changed", userID); err != nil {
		return Member{}, err
	}
	return member, nil
}

// LeaveOrg 用户退出机构（个人记录保留；机构访问失效由 TASK-074 挂接）。
func (s *Service) LeaveOrg(
	_ context.Context, actor Actor, orgID string,
) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	member, err := s.store.GetMember(orgID, actor.UserID)
	if err != nil {
		return ErrNotFound
	}
	if member.LeftAt != nil {
		return nil
	}
	now := s.now().UTC()
	member.LeftAt = &now
	if err := s.store.UpdateMember(member); err != nil {
		return err
	}
	return s.appendAudit(actor, orgID, "org.member.left", actor.UserID)
}

// SetOrgStatus 停用/启用/注销机构（owner；停用后成员令牌与共享链接失效由 074 挂接）。
func (s *Service) SetOrgStatus(
	_ context.Context, actor Actor, orgID, status string,
) (Org, error) {
	if err := s.require(actor, orgID, PermManageOrg); err != nil {
		return Org{}, err
	}
	if status != OrgActive && status != OrgSuspended && status != OrgDeactivated {
		return Org{}, fmt.Errorf("%w: 状态非法", ErrInvalidInput)
	}
	org, err := s.store.GetOrgByID(actor.DataRegion, orgID)
	if err != nil {
		return Org{}, err
	}
	org.Status = status
	if err := s.store.UpdateOrg(org); err != nil {
		return Org{}, err
	}
	if err := s.appendAudit(actor, orgID, "org.status_changed", status); err != nil {
		return Org{}, err
	}
	return org, nil
}

// require 校验机构成员身份与权限。
func (s *Service) require(actor Actor, orgID, permission string) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	member, err := s.store.GetMember(orgID, actor.UserID)
	if err != nil || member.LeftAt != nil {
		return ErrNotFound
	}
	if !Can(member.Role, permission) {
		return fmt.Errorf("%w: 角色 %s 无权限 %s", ErrForbidden, member.Role, permission)
	}
	return nil
}

func (s *Service) appendAudit(actor Actor, orgID, action, target string) error {
	return s.store.AppendAudit(AuditEntry{
		AuditID:    newID(),
		OrgID:      orgID,
		ActorID:    actor.UserID,
		ActorRole:  actor.Role,
		Action:     action,
		TargetRef:  target,
		DataRegion: actor.DataRegion,
		CreatedAt:  s.now().UTC(),
	})
}

func validateActor(actor Actor) error {
	if strings.TrimSpace(actor.UserID) == "" {
		return fmt.Errorf("%w: 缺少个人账户身份", ErrInvalidInput)
	}
	return region.ValidateDataRegion(actor.DataRegion)
}

func validRole(role string) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleInstructor, RolePrivacyAuditor, RoleFinance, RoleCandidate:
		return true
	}
	return false
}

func validMethod(method string) bool {
	switch method {
	case InviteLink, InviteOrgEmail, InviteBulkList, InviteSSO, InviteSCIM:
		return true
	}
	return false
}
