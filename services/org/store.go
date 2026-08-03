// Package org 提供机构租户存储接口（TASK-070）。
package org

// Store 为机构租户存储（生产 PostgreSQL；审计仅 SELECT/INSERT）。
type Store interface {
	SaveOrg(Org, string) error
	GetOrgByID(dataRegion, orgID string) (Org, error)
	GetOrgByIdempotencyKey(dataRegion, key string) (Org, error)
	UpdateOrg(Org) error
	SaveMember(Member) error
	GetMember(orgID, userID string) (Member, error)
	ListMembers(orgID string) ([]Member, error)
	UpdateMember(Member) error
	SaveInvitation(Invitation, string) error
	GetInvitationByID(dataRegion, invitationID string) (Invitation, error)
	GetInvitationByIdempotencyKey(dataRegion, key string) (Invitation, error)
	UpdateInvitation(Invitation) error
	ListInvitations(dataRegion, orgID string) ([]Invitation, error)
	AppendAudit(AuditEntry) error
	ListAudits(dataRegion, orgID string) ([]AuditEntry, error)
	// TASK-071 训练任务与模板（禁止项由服务层强制；成员状态默认最小可见）。
	SaveAssignment(Assignment, string) error
	GetAssignmentByID(dataRegion, assignmentID string) (Assignment, error)
	GetAssignmentByIdempotencyKey(dataRegion, key string) (Assignment, error)
	UpdateAssignment(Assignment) error
	ListAssignments(dataRegion, orgID string) ([]Assignment, error)
	SaveAssignmentMember(AssignmentMember) error
	GetAssignmentMember(assignmentID, userID string) (AssignmentMember, error)
	UpdateAssignmentMember(AssignmentMember) error
	ListAssignmentMembers(assignmentID string) ([]AssignmentMember, error)
	// TASK-072 按任务细粒度结果授权（范围+有效期+可撤回）。
	SaveShare(Share, string) error
	GetShareByID(dataRegion, shareID string) (Share, error)
	GetShareByIdempotencyKey(dataRegion, key string) (Share, error)
	UpdateShare(Share) error
	ListSharesByUser(dataRegion, userID, assignmentID string) ([]Share, error)
	ListSharesByAssignment(dataRegion, assignmentID string) ([]Share, error)
	ListActiveShares(dataRegion string) ([]Share, error)
	// TASK-074 退出/移除即时失效。
	ListActiveSharesForUserOrg(userID, orgID string) ([]Share, error)
}
