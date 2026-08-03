// Package org 按任务细粒度结果授权测试（TASK-072；FR-035，US-07 场景 1/2）。
package org

import (
	"context"
	"errors"
	"testing"
	"time"
)

func setupShareScenario(t *testing.T, svc *Service, orgID string) (Assignment, Actor) {
	t.Helper()
	assignment, err := svc.CreateAssignment(context.Background(), testActor,
		orgID, sampleAssignmentInput(), "idem-assign-share")
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	candidate := Actor{UserID: "user-share", DataRegion: "cn"}
	inv, err := svc.InviteMember(context.Background(), testActor,
		orgID, InviteLink, RoleCandidate, "", "idem-inv-share")
	if err != nil {
		t.Fatalf("邀请失败: %v", err)
	}
	if _, err := svc.AcceptInvitation(context.Background(), candidate, inv.InvitationID); err != nil {
		t.Fatalf("加入失败: %v", err)
	}
	return assignment, candidate
}

// 授权：范围+有效期；幂等；非法类别拒绝；仅成员可授权。
func TestGrantShareScopeExpiryIdempotent(t *testing.T) {
	svc, _ := newTestService(t)
	org := mustOrg(t, svc, "idem-org-s1")
	assignment, candidate := setupShareScenario(t, svc, org.OrgID)
	expires := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	share, err := svc.GrantAssignmentShare(context.Background(), candidate,
		assignment.AssignmentID, []string{"radar"}, expires, "idem-share-1")
	if err != nil {
		t.Fatalf("授权失败: %v", err)
	}
	if share.Status != ShareActive || len(share.DataCategories) != 1 ||
		share.DataCategories[0] != "radar" {
		t.Fatalf("授权内容异常：%+v", share)
	}
	again, err := svc.GrantAssignmentShare(context.Background(), candidate,
		assignment.AssignmentID, []string{"radar"}, expires, "idem-share-1")
	if err != nil || again.ShareID != share.ShareID {
		t.Fatalf("授权幂等异常：%+v err=%v", again, err)
	}
	if _, err := svc.GrantAssignmentShare(context.Background(), candidate,
		assignment.AssignmentID, []string{"rank"}, expires, "idem-share-2"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("非法类别应拒绝，实际 err=%v", err)
	}
}

// 机构访问：仅有效期内返回已授权类别并写审计；到期自动失效。
func TestShareEffectiveAndExpiry(t *testing.T) {
	svc, store := newTestService(t)
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	org := mustOrg(t, svc, "idem-org-s2")
	assignment, candidate := setupShareScenario(t, svc, org.OrgID)
	expires := now.Add(30 * 24 * time.Hour)
	share, err := svc.GrantAssignmentShare(context.Background(), candidate,
		assignment.AssignmentID, []string{"radar"}, expires, "idem-share-s2")
	if err != nil {
		t.Fatalf("授权失败: %v", err)
	}
	instructor := Actor{UserID: "user-instructor", DataRegion: "cn", OrgID: org.OrgID}
	inv, _ := svc.InviteMember(context.Background(), testActor, org.OrgID, InviteLink, RoleInstructor, "", "idem-inv-s2")
	_, _ = svc.AcceptInvitation(context.Background(), instructor, inv.InvitationID)
	categories, err := svc.CheckShareEffective(context.Background(), instructor,
		org.OrgID, assignment.AssignmentID, candidate.UserID, share.ShareID)
	if err != nil || len(categories) != 1 || categories[0] != "radar" {
		t.Fatalf("有效期内应可访问：%v err=%v", categories, err)
	}
	audits, _ := store.ListAudits("cn", org.OrgID)
	hasAccessAudit := false
	for _, a := range audits {
		if a.Action == "org.result.accessed" {
			hasAccessAudit = true
		}
	}
	if !hasAccessAudit {
		t.Fatal("访问应写审计")
	}
	// 到期自动失效。
	svc.now = func() time.Time { return now.Add(31 * 24 * time.Hour) }
	count, err := svc.ExpireShares(context.Background(), "cn")
	if err != nil || count != 1 {
		t.Fatalf("到期失效异常：count=%d err=%v", count, err)
	}
	categories, err = svc.CheckShareEffective(context.Background(), instructor,
		org.OrgID, assignment.AssignmentID, candidate.UserID, share.ShareID)
	if err != nil || categories != nil {
		t.Fatalf("到期后应不可访问：%v err=%v", categories, err)
	}
}

// 撤回：在线访问立即失效；审计记录继续存在。
func TestRevokeShareImmediate(t *testing.T) {
	svc, store := newTestService(t)
	org := mustOrg(t, svc, "idem-org-s3")
	assignment, candidate := setupShareScenario(t, svc, org.OrgID)
	share, err := svc.GrantAssignmentShare(context.Background(), candidate,
		assignment.AssignmentID, []string{"total_score"}, time.Now().Add(24*time.Hour), "idem-share-s3")
	if err != nil {
		t.Fatalf("授权失败: %v", err)
	}
	if err := svc.RevokeAssignmentShare(context.Background(), candidate,
		assignment.AssignmentID, share.ShareID, "idem-revoke-s3"); err != nil {
		t.Fatalf("撤回失败: %v", err)
	}
	instructor := Actor{UserID: "user-instructor", DataRegion: "cn", OrgID: org.OrgID}
	inv, _ := svc.InviteMember(context.Background(), testActor, org.OrgID, InviteLink, RoleInstructor, "", "idem-inv-s3")
	_, _ = svc.AcceptInvitation(context.Background(), instructor, inv.InvitationID)
	categories, err := svc.CheckShareEffective(context.Background(), instructor,
		org.OrgID, assignment.AssignmentID, candidate.UserID, share.ShareID)
	if err != nil || categories != nil {
		t.Fatalf("撤回后在线访问应立即失效：%v err=%v", categories, err)
	}
	audits, _ := store.ListAudits("cn", org.OrgID)
	hasWithdrawAudit := false
	for _, a := range audits {
		if a.Action == "org.share.withdrawn" {
			hasWithdrawAudit = true
		}
	}
	if !hasWithdrawAudit {
		t.Fatal("撤回应写审计")
	}
}

// "已完成未共享"展示：仅计数，不显示失败。
func TestCompletionSummarySharedNotShared(t *testing.T) {
	svc, store := newTestService(t)
	org := mustOrg(t, svc, "idem-org-s4")
	assignment, candidate := setupShareScenario(t, svc, org.OrgID)
	for i, userID := range []string{candidate.UserID, "user-b", "user-c"} {
		now := time.Now().UTC()
		member := AssignmentMember{
			AssignmentID: assignment.AssignmentID,
			OrgID:        org.OrgID,
			UserID:       userID,
			Status:       MemberCompleted,
			CompletedAt:  &now,
		}
		if err := store.SaveAssignmentMember(member); err != nil {
			t.Fatalf("保存成员失败: %v", err)
		}
		_ = i
	}
	_, _ = svc.GrantAssignmentShare(context.Background(), candidate,
		assignment.AssignmentID, []string{"radar"}, time.Now().Add(24*time.Hour), "idem-share-s4")
	_, summary, err := svc.GetAssignment(context.Background(), testActor, org.OrgID, assignment.AssignmentID)
	if err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if summary.ResultShared != 1 || summary.ResultNotShared != 2 {
		t.Fatalf("已完成未共享计数异常：%+v", summary)
	}
}
