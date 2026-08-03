// Package org 提供训练任务与模板（TASK-071；FR-035，US-07 场景 5；
// DOMAIN-MODEL Assignment；openapi /v1/orgs/{orgId}/assignments*）。
// 红线：60 分线/统一评分算法/保护属性/证据标准/跨轮解锁/正式复核在界面上不可配置，
// 服务端拒绝任何修改尝试并写审计。
package org

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 任务状态。
const (
	AssignmentDraft     = "draft"
	AssignmentPublished = "published"
	AssignmentClosed    = "closed"
)

// 成员任务状态（openapi Assignment.completion_summary 对齐）。
const (
	MemberNotStarted = "not_started"
	MemberInProgress = "in_progress"
	MemberCompleted  = "completed"
	MemberExited     = "exited"
)

// 模板禁止项（平台硬性限制；任何修改 60 分线/统一算法/保护属性/证据标准/
// 跨轮解锁/正式复核的尝试被拒并写审计）。
var protectedTemplateKeys = []string{
	"pass_line", "passing_score", "scoring_algorithm", "protected_attributes",
	"evidence_standard", "cross_round_unlock", "formal_review",
}

// 模板允许项（FR-035 规则 5：岗位/轮次/角色/时长/难度/语言/工具等）。
var allowedTemplateKeys = []string{
	"rounds", "roles", "duration_minutes", "difficulty", "language", "tools",
	"materials", "role", "sequence", "retry_eligible",
}

// Assignment 为训练任务（模板可配项/禁止项由服务端强制）。
type Assignment struct {
	AssignmentID     string
	OrgID            string
	Title            string
	JobCategory      string
	RoundTemplate    map[string]any
	DeadlineAt       time.Time
	MaxPracticeCount int
	OrgCreditSeconds int
	Status           string
	CreatedBy        string
	DataRegion       string
	CreatedAt        time.Time
}

// AssignmentMember 为任务-成员状态（默认最小可见）。
type AssignmentMember struct {
	AssignmentID         string
	OrgID                string
	UserID               string
	Status               string
	CompletedAt          *time.Time
	FaultFlag            bool
	OrgCreditUsedSeconds int
}

// CompletionSummary 为默认最小可见的完成情况（不含个人结果）。
type CompletionSummary struct {
	NotStarted           int
	InProgress           int
	Completed            int
	Quit                 int
	SystemFaultCount     int
	OrgCreditUsedSeconds int
	ResultShared         int
	ResultNotShared      int
}

// AssignmentInput 为创建任务输入。
type AssignmentInput struct {
	Title            string
	JobCategory      string
	RoundTemplate    map[string]any
	DeadlineAt       time.Time
	MaxPracticeCount int
	OrgCreditSeconds int
}

// CreateAssignment 创建训练任务：仅允许配置项可写；禁止项出现即拒绝并写审计。
func (s *Service) CreateAssignment(
	_ context.Context, actor Actor, orgID string, in AssignmentInput, idemKey string,
) (Assignment, error) {
	if err := s.require(actor, orgID, PermManageAssignments); err != nil {
		return Assignment{}, err
	}
	if strings.TrimSpace(in.Title) == "" || in.DeadlineAt.IsZero() ||
		strings.TrimSpace(idemKey) == "" {
		return Assignment{}, fmt.Errorf("%w: title、deadline_at 与幂等键必填", ErrInvalidInput)
	}
	if !s.now().UTC().Before(in.DeadlineAt) {
		return Assignment{}, fmt.Errorf("%w: 截止时间必须晚于当前时间", ErrInvalidInput)
	}
	if in.MaxPracticeCount < 0 || in.OrgCreditSeconds < 0 {
		return Assignment{}, fmt.Errorf("%w: 练习次数与机构额度不得为负", ErrInvalidInput)
	}
	if key := findProtectedKey(in.RoundTemplate); key != "" {
		_ = s.appendAudit(actor, orgID, "assignment.protected_config_attempt", key)
		return Assignment{}, fmt.Errorf("%w: 模板禁止配置 %s（60 分线/统一算法/证据规则等平台硬性限制）",
			ErrForbidden, key)
	}
	if cached, err := s.store.GetAssignmentByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Assignment{}, err
	}
	assignment := Assignment{
		AssignmentID:     newID(),
		OrgID:            orgID,
		Title:            strings.TrimSpace(in.Title),
		JobCategory:      strings.TrimSpace(in.JobCategory),
		RoundTemplate:    filterTemplate(in.RoundTemplate),
		DeadlineAt:       in.DeadlineAt,
		MaxPracticeCount: in.MaxPracticeCount,
		OrgCreditSeconds: in.OrgCreditSeconds,
		Status:           AssignmentDraft,
		CreatedBy:        actor.UserID,
		DataRegion:       actor.DataRegion,
		CreatedAt:        s.now().UTC(),
	}
	if err := s.store.SaveAssignment(assignment, idemKey); err != nil {
		return Assignment{}, err
	}
	_ = s.appendAudit(actor, orgID, "assignment.created", assignment.AssignmentID)
	return assignment, nil
}

// PublishAssignment 发布任务（draft → published）。
func (s *Service) PublishAssignment(
	_ context.Context, actor Actor, orgID, assignmentID string,
) (Assignment, error) {
	if err := s.require(actor, orgID, PermManageAssignments); err != nil {
		return Assignment{}, err
	}
	assignment, err := s.getOwnedAssignment(actor, orgID, assignmentID)
	if err != nil {
		return Assignment{}, err
	}
	if assignment.Status != AssignmentDraft {
		return Assignment{}, fmt.Errorf("%w: 仅 draft 可发布（当前 %s）", ErrStateConflict, assignment.Status)
	}
	assignment.Status = AssignmentPublished
	if err := s.store.UpdateAssignment(assignment); err != nil {
		return Assignment{}, err
	}
	_ = s.appendAudit(actor, orgID, "assignment.published", assignmentID)
	return assignment, nil
}

// CloseAssignment 关闭任务（published → closed）。
func (s *Service) CloseAssignment(
	_ context.Context, actor Actor, orgID, assignmentID string,
) (Assignment, error) {
	if err := s.require(actor, orgID, PermManageAssignments); err != nil {
		return Assignment{}, err
	}
	assignment, err := s.getOwnedAssignment(actor, orgID, assignmentID)
	if err != nil {
		return Assignment{}, err
	}
	if assignment.Status != AssignmentPublished {
		return Assignment{}, fmt.Errorf("%w: 仅 published 可关闭（当前 %s）", ErrStateConflict, assignment.Status)
	}
	assignment.Status = AssignmentClosed
	if err := s.store.UpdateAssignment(assignment); err != nil {
		return Assignment{}, err
	}
	_ = s.appendAudit(actor, orgID, "assignment.closed", assignmentID)
	return assignment, nil
}

// GetAssignment 返回任务（默认最小可见：仅完成状态与计数，不含个人结果）。
func (s *Service) GetAssignment(
	_ context.Context, actor Actor, orgID, assignmentID string,
) (Assignment, CompletionSummary, error) {
	if err := s.require(actor, orgID, PermViewAggregates); err != nil {
		return Assignment{}, CompletionSummary{}, err
	}
	assignment, err := s.getOwnedAssignment(actor, orgID, assignmentID)
	if err != nil {
		return Assignment{}, CompletionSummary{}, err
	}
	members, err := s.store.ListAssignmentMembers(assignmentID)
	if err != nil {
		return Assignment{}, CompletionSummary{}, err
	}
	summary := CompletionSummary{}
	for _, m := range members {
		switch m.Status {
		case MemberNotStarted:
			summary.NotStarted++
		case MemberInProgress:
			summary.InProgress++
		case MemberCompleted:
			summary.Completed++
		case MemberExited:
			summary.Quit++
		}
		if m.FaultFlag {
			summary.SystemFaultCount++
		}
		summary.OrgCreditUsedSeconds += m.OrgCreditUsedSeconds
	}
	// "已完成未共享"展示（TASK-072）：仅显示计数，不显示失败。
	shares, err := s.store.ListSharesByAssignment(actor.DataRegion, assignmentID)
	if err != nil {
		return Assignment{}, CompletionSummary{}, err
	}
	sharedUsers := make(map[string]bool)
	for _, sh := range shares {
		if sh.Status == ShareActive && !s.now().UTC().After(sh.ExpiresAt) {
			sharedUsers[sh.UserID] = true
		}
	}
	for _, m := range members {
		if m.Status == MemberCompleted {
			if sharedUsers[m.UserID] {
				summary.ResultShared++
			} else {
				summary.ResultNotShared++
			}
		}
	}
	return assignment, summary, nil
}

// SetAssignmentMemberStatus 更新任务-成员状态（instructor/admin；写审计）。
func (s *Service) SetAssignmentMemberStatus(
	_ context.Context, actor Actor, orgID, assignmentID, userID, status string,
) (AssignmentMember, error) {
	if err := s.require(actor, orgID, PermManageAssignments); err != nil {
		return AssignmentMember{}, err
	}
	switch status {
	case MemberNotStarted, MemberInProgress, MemberCompleted, MemberExited:
	default:
		return AssignmentMember{}, fmt.Errorf("%w: 任务状态非法", ErrInvalidInput)
	}
	member, err := s.store.GetAssignmentMember(assignmentID, userID)
	if err != nil {
		return AssignmentMember{}, err
	}
	member.Status = status
	if status == MemberCompleted {
		now := s.now().UTC()
		member.CompletedAt = &now
	}
	if err := s.store.UpdateAssignmentMember(member); err != nil {
		return AssignmentMember{}, err
	}
	_ = s.appendAudit(actor, orgID, "assignment.member_status_changed", assignmentID+":"+userID)
	return member, nil
}

func (s *Service) getOwnedAssignment(actor Actor, orgID, assignmentID string) (Assignment, error) {
	assignment, err := s.store.GetAssignmentByID(actor.DataRegion, assignmentID)
	if err != nil {
		return Assignment{}, err
	}
	if assignment.OrgID != orgID {
		return Assignment{}, ErrNotFound
	}
	return assignment, nil
}

// findProtectedKey 查找禁止配置键（大小写不敏感）。
func findProtectedKey(template map[string]any) string {
	for _, key := range protectedTemplateKeys {
		for tKey := range template {
			if strings.EqualFold(tKey, key) {
				return tKey
			}
		}
	}
	return ""
}

// filterTemplate 仅保留允许的模板键（其余忽略）。
func filterTemplate(template map[string]any) map[string]any {
	out := make(map[string]any)
	for _, key := range allowedTemplateKeys {
		for tKey, value := range template {
			if strings.EqualFold(tKey, key) {
				out[key] = value
			}
		}
	}
	return out
}
