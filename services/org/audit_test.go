// Package org 机构访问审计与退出即时失效测试（TASK-074；FR-034/035，US-07 场景 4）。
package org

import (
	"context"
	"testing"
	"time"
)

func sharedCandidate(t *testing.T, svc *Service, orgID string) (string, string) {
	t.Helper()
	assignment, err := svc.CreateAssignment(context.Background(), testActor,
		orgID, sampleAssignmentInput(), "idem-assign-audit")
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	candidate := Actor{UserID: "user-audit-candidate", DataRegion: "cn"}
	inv, err := svc.InviteMember(context.Background(), testActor,
		orgID, InviteLink, RoleCandidate, "", "idem-inv-audit")
	if err != nil {
		t.Fatalf("邀请失败: %v", err)
	}
	if _, err := svc.AcceptInvitation(context.Background(), candidate, inv.InvitationID); err != nil {
		t.Fatalf("加入失败: %v", err)
	}
	share, err := svc.GrantAssignmentShare(context.Background(), candidate,
		assignment.AssignmentID, []string{"radar"}, time.Now().Add(24*time.Hour), "idem-share-audit")
	if err != nil {
		t.Fatalf("授权失败: %v", err)
	}
	return candidate.UserID, share.ShareID
}

// 退出机构：共享链接立即失效；审计记录继续存在。
func TestLeaveOrgInvalidatesShareImmediately(t *testing.T) {
	svc, store := newTestService(t)
	org := mustOrg(t, svc, "idem-org-a5")
	candidateID, shareID := sharedCandidate(t, svc, org.OrgID)
	if err := svc.LeaveOrg(context.Background(), Actor{UserID: candidateID, DataRegion: "cn"}, org.OrgID); err != nil {
		t.Fatalf("退出失败: %v", err)
	}
	if svc.IsMemberAccessValid(org.OrgID, candidateID) {
		t.Fatal("退出后成员访问应失效")
	}
	share, _ := svc.store.GetShareByID("cn", shareID)
	if share.Status != ShareWithdrawn {
		t.Fatalf("退出后共享链接应立即失效：%s", share.Status)
	}
	audits, _ := store.ListAudits("cn", org.OrgID)
	if len(audits) == 0 {
		t.Fatal("审计记录应继续存在")
	}
}

// 移除成员：owner 操作后访问立即失效。
func TestRemoveMemberInvalidatesAccess(t *testing.T) {
	svc, _ := newTestService(t)
	org := mustOrg(t, svc, "idem-org-a6")
	candidateID, shareID := sharedCandidate(t, svc, org.OrgID)
	if err := svc.RemoveMember(context.Background(), testActor, org.OrgID, candidateID); err != nil {
		t.Fatalf("移除成员失败: %v", err)
	}
	share, _ := svc.store.GetShareByID("cn", shareID)
	if share.Status != ShareWithdrawn {
		t.Fatalf("移除后共享应立即失效：%s", share.Status)
	}
	if svc.IsMemberAccessValid(org.OrgID, candidateID) {
		t.Fatal("移除后成员访问应失效")
	}
}

// 机构停用：全部共享链接失效；审计记录失效原因。
func TestSuspendOrgInvalidatesAllShares(t *testing.T) {
	svc, store := newTestService(t)
	org := mustOrg(t, svc, "idem-org-a7")
	_, _ = sharedCandidate(t, svc, org.OrgID)
	if _, err := svc.SetOrgStatus(context.Background(), testActor, org.OrgID, OrgSuspended); err != nil {
		t.Fatalf("停用机构失败: %v", err)
	}
	count, err := svc.InvalidateOrgAccess(context.Background(), testActor, org.OrgID)
	if err != nil || count != 1 {
		t.Fatalf("失效共享异常：count=%d err=%v", count, err)
	}
	audits, _ := store.ListAudits("cn", org.OrgID)
	hasInvalidate := false
	for _, a := range audits {
		if a.Action == "org.access.invalidated" {
			hasInvalidate = true
		}
	}
	if !hasInvalidate {
		t.Fatal("停用失效应写审计")
	}
}

// 审计可见性：privacy_auditor 可看；instructor 不可看。
func TestAuditVisibility(t *testing.T) {
	svc, _ := newTestService(t)
	org := mustOrg(t, svc, "idem-org-a8")
	auditor := Actor{UserID: "user-auditor", DataRegion: "cn", OrgID: org.OrgID}
	inv, _ := svc.InviteMember(context.Background(), testActor, org.OrgID, InviteLink, RolePrivacyAuditor, "", "idem-inv-auditor")
	_, _ = svc.AcceptInvitation(context.Background(), auditor, inv.InvitationID)
	if _, err := svc.ListAccessAudits(context.Background(), auditor, org.OrgID); err != nil {
		t.Fatalf("privacy_auditor 应可看审计: %v", err)
	}
	instructor := Actor{UserID: "user-instr", DataRegion: "cn", OrgID: org.OrgID}
	inv2, _ := svc.InviteMember(context.Background(), testActor, org.OrgID, InviteLink, RoleInstructor, "", "idem-inv-instr")
	_, _ = svc.AcceptInvitation(context.Background(), instructor, inv2.InvitationID)
	if _, err := svc.ListAccessAudits(context.Background(), instructor, org.OrgID); err == nil {
		t.Fatal("instructor 不可查看审计")
	}
}
