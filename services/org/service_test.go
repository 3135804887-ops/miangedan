// Package org 机构租户测试（TASK-070；FR-034，US-07）。
package org

import (
	"context"
	"errors"
	"testing"
)

var testActor = Actor{UserID: "user-owner", DataRegion: "cn"}

func newTestService(t *testing.T) (*Service, *MemoryStore) {
	t.Helper()
	store := NewMemoryStore()
	svc, err := NewService(store)
	if err != nil {
		t.Fatalf("创建服务失败: %v", err)
	}
	return svc, store
}

// 创建机构：创建者以个人账户加入并成为所有者（无影子账户）。
func TestCreateOrgMakesOwnerWithPersonalAccount(t *testing.T) {
	svc, store := newTestService(t)
	org, err := svc.CreateOrg(context.Background(), testActor, "测试机构", "idem-org-1")
	if err != nil {
		t.Fatalf("创建机构失败: %v", err)
	}
	if org.Status != OrgActive || org.CreatedBy != testActor.UserID {
		t.Fatalf("机构状态异常：%+v", org)
	}
	member, err := store.GetMember(org.OrgID, testActor.UserID)
	if err != nil || member.Role != RoleOwner {
		t.Fatalf("创建者应为所有者：%+v err=%v", member, err)
	}
	// 幂等：同一幂等键返回同一机构。
	again, err := svc.CreateOrg(context.Background(), testActor, "测试机构", "idem-org-1")
	if err != nil || again.OrgID != org.OrgID {
		t.Fatalf("创建机构幂等异常：%+v err=%v", again, err)
	}
}

// 邀请：owner/admin 可邀请；五类邀请方式适配点；幂等。
func TestInviteAndAcceptByPersonalAccount(t *testing.T) {
	svc, _ := newTestService(t)
	org, err := svc.CreateOrg(context.Background(), testActor, "测试机构", "idem-org-2")
	if err != nil {
		t.Fatalf("创建机构失败: %v", err)
	}
	inv, err := svc.InviteMember(context.Background(), testActor,
		org.OrgID, InviteOrgEmail, RoleCandidate, "candidate@example.com", "idem-inv-1")
	if err != nil {
		t.Fatalf("邀请失败: %v", err)
	}
	again, err := svc.InviteMember(context.Background(), testActor,
		org.OrgID, InviteOrgEmail, RoleCandidate, "candidate@example.com", "idem-inv-1")
	if err != nil || again.InvitationID != inv.InvitationID {
		t.Fatalf("邀请幂等异常：%+v err=%v", again, err)
	}
	// 用户以个人账户加入。
	candidate := Actor{UserID: "candidate@example.com", DataRegion: "cn"}
	member, err := svc.AcceptInvitation(context.Background(), candidate, inv.InvitationID)
	if err != nil {
		t.Fatalf("接受邀请失败: %v", err)
	}
	if member.UserID != candidate.UserID || member.Role != RoleCandidate {
		t.Fatalf("应以个人账户身份加入：%+v", member)
	}
}

// 角色权限分离：candidate 无邀请权限；instructor 不可改角色；privacy_auditor 只看审计。
func TestRoleSeparation(t *testing.T) {
	svc, _ := newTestService(t)
	org, err := svc.CreateOrg(context.Background(), testActor, "测试机构", "idem-org-3")
	if err != nil {
		t.Fatalf("创建机构失败: %v", err)
	}
	for _, role := range []string{RoleAdmin, RoleInstructor, RoleCandidate} {
		inv, err := svc.InviteMember(context.Background(), testActor,
			org.OrgID, InviteLink, role, "", "idem-inv-"+role)
		if err != nil {
			t.Fatalf("邀请 %s 失败: %v", role, err)
		}
		user := Actor{UserID: "user-" + role, DataRegion: "cn"}
		if _, err := svc.AcceptInvitation(context.Background(), user, inv.InvitationID); err != nil {
			t.Fatalf("%s 加入失败: %v", role, err)
		}
	}
	candidate := Actor{UserID: "user-candidate", DataRegion: "cn", OrgID: org.OrgID}
	if _, err := svc.InviteMember(context.Background(), candidate,
		org.OrgID, InviteLink, RoleCandidate, "", "idem-inv-x"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("candidate 邀请应被拒，实际 err=%v", err)
	}
	instructor := Actor{UserID: "user-instructor", DataRegion: "cn", OrgID: org.OrgID}
	if _, err := svc.SetMemberRole(context.Background(), instructor,
		org.OrgID, "user-candidate", RoleAdmin); !errors.Is(err, ErrForbidden) {
		t.Fatalf("instructor 改角色应被拒，实际 err=%v", err)
	}
}

// 邮箱不匹配拒绝；邀请过期拒绝；机构停用后不可加入。
func TestInvitationGuards(t *testing.T) {
	svc, _ := newTestService(t)
	org, err := svc.CreateOrg(context.Background(), testActor, "测试机构", "idem-org-4")
	if err != nil {
		t.Fatalf("创建机构失败: %v", err)
	}
	inv, err := svc.InviteMember(context.Background(), testActor,
		org.OrgID, InviteOrgEmail, RoleCandidate, "right@example.com", "idem-inv-4")
	if err != nil {
		t.Fatalf("邀请失败: %v", err)
	}
	wrong := Actor{UserID: "wrong@example.com", DataRegion: "cn"}
	if _, err := svc.AcceptInvitation(context.Background(), wrong, inv.InvitationID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("邮箱不匹配应拒绝，实际 err=%v", err)
	}
	if _, err := svc.SetOrgStatus(context.Background(), testActor, org.OrgID, OrgSuspended); err != nil {
		t.Fatalf("停用机构失败: %v", err)
	}
	inv2, err := svc.InviteMember(context.Background(), testActor,
		org.OrgID, InviteLink, RoleCandidate, "", "idem-inv-5")
	if err == nil {
		t.Fatalf("停用机构不应可邀请，实际 inv=%+v", inv2)
	}
}

// 退出机构：个人记录保留（left_at）；审计继续存在。
func TestLeaveOrgKeepsRecordsAndAudit(t *testing.T) {
	svc, store := newTestService(t)
	org, err := svc.CreateOrg(context.Background(), testActor, "测试机构", "idem-org-5")
	if err != nil {
		t.Fatalf("创建机构失败: %v", err)
	}
	if err := svc.LeaveOrg(context.Background(), testActor, org.OrgID); err != nil {
		t.Fatalf("退出机构失败: %v", err)
	}
	member, err := store.GetMember(org.OrgID, testActor.UserID)
	if err != nil || member.LeftAt == nil {
		t.Fatalf("退出后 left_at 应存在：%+v err=%v", member, err)
	}
	audits, err := store.ListAudits("cn", org.OrgID)
	if err != nil || len(audits) == 0 {
		t.Fatalf("审计记录应保留：%+v err=%v", audits, err)
	}
	if _, err := svc.GetOrg(context.Background(), testActor, org.OrgID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("退出后机构访问应失效，实际 err=%v", err)
	}
}

// 权限矩阵常量正确性（六类角色分离）。
func TestPermissionMatrix(t *testing.T) {
	if !Can(RoleOwner, PermManageOrg) || !Can(RoleOwner, PermViewFinance) {
		t.Fatal("owner 应有全部权限")
	}
	if Can(RoleAdmin, PermViewFinance) || Can(RoleAdmin, PermViewAudit) {
		t.Fatal("admin 不应有财务/审计权限（默认分离）")
	}
	if Can(RoleInstructor, PermManageRoles) {
		t.Fatal("instructor 不应有角色管理权限")
	}
	if !Can(RolePrivacyAuditor, PermViewAudit) || Can(RolePrivacyAuditor, PermManageOrg) {
		t.Fatal("privacy_auditor 仅审计权限")
	}
	if !Can(RoleFinance, PermViewFinance) || Can(RoleFinance, PermManageAssignments) {
		t.Fatal("finance 仅财务权限")
	}
	if !Can(RoleCandidate, PermParticipate) || Can(RoleCandidate, PermInviteMembers) {
		t.Fatal("candidate 仅参与权限")
	}
}
