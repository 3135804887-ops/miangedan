// Package org 提供机构租户与六类角色（TASK-070；FR-034，US-07；
// DOMAIN-MODEL §6.16；openapi /v1/orgs*）。
// 红线：用户以个人账户加入（无影子账户）；财务/审计/教学/管理权限默认分离。
package org

import "time"

// 六类角色（DOMAIN-MODEL §6.16；openapi OrgRole 对齐）。
const (
	RoleOwner          = "owner"
	RoleAdmin          = "admin"
	RoleInstructor     = "instructor"
	RolePrivacyAuditor = "privacy_auditor"
	RoleFinance        = "finance"
	RoleCandidate      = "candidate"
)

// 邀请方式（FR-034：邀请链接/机构邮箱/批量名单/SSO/SCIM 适配点）。
const (
	InviteLink     = "link"
	InviteOrgEmail = "org_email"
	InviteBulkList = "bulk_list"
	InviteSSO      = "sso"
	InviteSCIM     = "scim"
)

// 机构状态。
const (
	OrgActive      = "active"
	OrgSuspended   = "suspended"
	OrgDeactivated = "deactivated"
)

// 邀请状态。
const (
	InvitePending  = "pending"
	InviteAccepted = "accepted"
	InviteRevoked  = "revoked"
)

// 权限枚举（六类角色权限分离矩阵）。
const (
	PermManageOrg         = "manage_org"
	PermInviteMembers     = "invite_members"
	PermManageRoles       = "manage_roles"
	PermManageAssignments = "manage_assignments"
	PermViewAggregates    = "view_aggregates"
	PermViewAudit         = "view_audit"
	PermViewFinance       = "view_finance"
	PermParticipate       = "participate"
)

// Actor 为调用方身份（个人账户；机构上下文）。
type Actor struct {
	UserID     string
	DataRegion string
	OrgID      string
	Role       string
}

// Org 为机构租户（用户以个人账户加入，无影子账户）。
type Org struct {
	OrgID          string
	Name           string
	DataRegion     string
	Status         string
	CreatedBy      string
	CreatedAt      time.Time
	IdempotencyKey string
}

// Member 为机构成员（六类角色之一；left_at 非空表示已退出/移除）。
type Member struct {
	OrgID        string
	UserID       string
	Role         string
	InviteMethod string
	JoinedAt     time.Time
	LeftAt       *time.Time
}

// Invitation 为成员邀请（加入前展示机构名称、可见数据、期限与退出影响）。
type Invitation struct {
	InvitationID   string
	OrgID          string
	Email          string
	Role           string
	InviteMethod   string
	Status         string
	ExpiresAt      time.Time
	DataRegion     string
	IdempotencyKey string
	CreatedAt      time.Time
}

// AuditEntry 为追加式访问审计（SELECT/INSERT only；无更新/删除路径）。
type AuditEntry struct {
	AuditID    string
	OrgID      string
	ActorID    string
	ActorRole  string
	Action     string
	TargetRef  string
	DataRegion string
	CreatedAt  time.Time
}
