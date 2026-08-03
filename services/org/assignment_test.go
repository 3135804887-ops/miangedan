// Package org 训练任务与模板测试（TASK-071；FR-035，US-07 场景 5）。
package org

import (
	"context"
	"errors"
	"testing"
	"time"
)

func mustOrg(t *testing.T, svc *Service, idem string) Org {
	t.Helper()
	org, err := svc.CreateOrg(context.Background(), testActor, "任务机构", idem)
	if err != nil {
		t.Fatalf("创建机构失败: %v", err)
	}
	return org
}

func sampleAssignmentInput() AssignmentInput {
	return AssignmentInput{
		Title:            "后端岗位训练",
		JobCategory:      "backend",
		DeadlineAt:       time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		MaxPracticeCount: 3,
		OrgCreditSeconds: 7200,
		RoundTemplate: map[string]any{
			"rounds":   map[string]any{"count": 3},
			"language": "zh-CN",
			"tools":    []string{"whiteboard"},
		},
	}
}

// 创建任务：允许项可写；发布/关闭状态机；幂等。
func TestCreatePublishCloseAssignment(t *testing.T) {
	svc, _ := newTestService(t)
	org := mustOrg(t, svc, "idem-org-a1")
	assignment, err := svc.CreateAssignment(context.Background(), testActor,
		org.OrgID, sampleAssignmentInput(), "idem-assign-1")
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	if assignment.Status != AssignmentDraft {
		t.Fatalf("初始状态应为 draft：%s", assignment.Status)
	}
	again, err := svc.CreateAssignment(context.Background(), testActor,
		org.OrgID, sampleAssignmentInput(), "idem-assign-1")
	if err != nil || again.AssignmentID != assignment.AssignmentID {
		t.Fatalf("创建任务幂等异常：%+v err=%v", again, err)
	}
	published, err := svc.PublishAssignment(context.Background(), testActor, org.OrgID, assignment.AssignmentID)
	if err != nil || published.Status != AssignmentPublished {
		t.Fatalf("发布任务异常：%+v err=%v", published, err)
	}
	if _, err := svc.PublishAssignment(context.Background(), testActor, org.OrgID, assignment.AssignmentID); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("重复发布应拒绝，实际 err=%v", err)
	}
	closed, err := svc.CloseAssignment(context.Background(), testActor, org.OrgID, assignment.AssignmentID)
	if err != nil || closed.Status != AssignmentClosed {
		t.Fatalf("关闭任务异常：%+v err=%v", closed, err)
	}
}

// 禁止项：修改 60 分线/量表/证据规则被拒并写审计。
func TestProtectedTemplateRejectedWithAudit(t *testing.T) {
	svc, store := newTestService(t)
	org := mustOrg(t, svc, "idem-org-a2")
	input := sampleAssignmentInput()
	input.RoundTemplate = map[string]any{"pass_line": 50}
	if _, err := svc.CreateAssignment(context.Background(), testActor,
		org.OrgID, input, "idem-assign-2"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("修改 60 分线应被拒绝，实际 err=%v", err)
	}
	audits, _ := store.ListAudits("cn", org.OrgID)
	found := false
	for _, a := range audits {
		if a.Action == "assignment.protected_config_attempt" {
			found = true
		}
	}
	if !found {
		t.Fatal("违规操作应写审计")
	}
	// 其他禁止键同样拒绝。
	for _, key := range []string{"scoring_algorithm", "evidence_standard", "formal_review"} {
		input2 := sampleAssignmentInput()
		input2.RoundTemplate = map[string]any{key: "x"}
		if _, err := svc.CreateAssignment(context.Background(), testActor,
			org.OrgID, input2, "idem-assign-"+key); !errors.Is(err, ErrForbidden) {
			t.Fatalf("禁止键 %s 应被拒绝，实际 err=%v", key, err)
		}
	}
}

// 默认最小可见：完成情况仅计数，不暴露个人结果。
func TestCompletionSummaryMinimalVisibility(t *testing.T) {
	svc, store := newTestService(t)
	org := mustOrg(t, svc, "idem-org-a3")
	assignment, err := svc.CreateAssignment(context.Background(), testActor,
		org.OrgID, sampleAssignmentInput(), "idem-assign-3")
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	for i, status := range []string{MemberNotStarted, MemberInProgress, MemberCompleted, MemberExited} {
		userID := "candidate-" + status
		member := AssignmentMember{
			AssignmentID: assignment.AssignmentID,
			OrgID:        org.OrgID,
			UserID:       userID,
			Status:       status,
		}
		if i == 2 {
			now := time.Now().UTC()
			member.CompletedAt = &now
			member.OrgCreditUsedSeconds = 600
		}
		if err := store.SaveAssignmentMember(member); err != nil {
			t.Fatalf("保存成员状态失败: %v", err)
		}
	}
	_, summary, err := svc.GetAssignment(context.Background(), testActor, org.OrgID, assignment.AssignmentID)
	if err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if summary.NotStarted != 1 || summary.InProgress != 1 || summary.Completed != 1 || summary.Quit != 1 {
		t.Fatalf("完成情况计数异常：%+v", summary)
	}
	if summary.OrgCreditUsedSeconds != 600 {
		t.Fatalf("机构额度消耗计数异常：%d", summary.OrgCreditUsedSeconds)
	}
}

// 权限：candidate 不可创建任务；截止时间必须晚于当前。
func TestAssignmentGuards(t *testing.T) {
	svc, _ := newTestService(t)
	org := mustOrg(t, svc, "idem-org-a4")
	candidate := Actor{UserID: "user-candidate", DataRegion: "cn", OrgID: org.OrgID}
	if _, err := svc.CreateAssignment(context.Background(), candidate,
		org.OrgID, sampleAssignmentInput(), "idem-assign-4"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("candidate 创建任务应被拒（非成员），实际 err=%v", err)
	}
	input := sampleAssignmentInput()
	input.DeadlineAt = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := svc.CreateAssignment(context.Background(), testActor,
		org.OrgID, input, "idem-assign-5"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("过去截止时间应被拒，实际 err=%v", err)
	}
}
