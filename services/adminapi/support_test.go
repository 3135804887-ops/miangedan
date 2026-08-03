// Package adminapi 客服工单测试（TASK-085；FR-039，US-08 场景 4）。
package adminapi

import (
	"context"
	"errors"
	"testing"
	"time"
)

func mustTicket(t *testing.T, svc *Service, idem string) Ticket {
	t.Helper()
	ticket, err := svc.CreateTicket(context.Background(), supportActor,
		"user-001", "额度问题", TicketEntitle, idem)
	if err != nil {
		t.Fatalf("创建工单失败: %v", err)
	}
	return ticket
}

// 默认最小可见：工单不含逐字稿/媒体内容；未授权不可访问逐字稿。
func TestDefaultMinimalVisibility(t *testing.T) {
	svc, _ := newTestService(t)
	ticket := mustTicket(t, svc, "idem-ticket-1")
	got, err := svc.GetTicket(context.Background(), supportActor, ticket.TicketID)
	if err != nil || got.Visibility != "minimal" || got.Category != TicketEntitle {
		t.Fatalf("默认最小可见异常：%+v err=%v", got, err)
	}
	visible, err := svc.CheckTranscriptAccess(context.Background(),
		supportActor, ticket.TicketID, "session-1")
	if err != nil || visible {
		t.Fatalf("未授权逐字稿应不可见：visible=%v err=%v", visible, err)
	}
}

// 逐字稿：用户授权后可见（会话范围+有效期）；过期自动失效。
func TestTranscriptAuthorizationScopeAndExpiry(t *testing.T) {
	svc, _ := newTestService(t)
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	ticket := mustTicket(t, svc, "idem-ticket-2")
	_, err := svc.GrantTranscriptAuthorization(context.Background(), supportActor,
		"user-001", ticket.TicketID, "session-1", now.Add(24*time.Hour), "idem-tr-1")
	if err != nil {
		t.Fatalf("授权失败: %v", err)
	}
	visible, err := svc.CheckTranscriptAccess(context.Background(),
		supportActor, ticket.TicketID, "session-1")
	if err != nil || !visible {
		t.Fatalf("授权后应可见：visible=%v err=%v", visible, err)
	}
	// 其他会话不可见（会话范围）。
	other, _ := svc.CheckTranscriptAccess(context.Background(),
		supportActor, ticket.TicketID, "session-2")
	if other {
		t.Fatal("其他会话不可见")
	}
	// 过期后不可见。
	svc.now = func() time.Time { return now.Add(25 * time.Hour) }
	visible, _ = svc.CheckTranscriptAccess(context.Background(),
		supportActor, ticket.TicketID, "session-1")
	if visible {
		t.Fatal("过期授权应失效")
	}
}

// 媒体访问：用户申请 + 双人审批；本人不可自批；两人后放行。
func TestMediaAccessDualApproval(t *testing.T) {
	svc, _ := newTestService(t)
	ticket := mustTicket(t, svc, "idem-ticket-3")
	req, err := svc.RequestMediaAccess(context.Background(), supportActor,
		"user-001", ticket.TicketID, "session-1", "idem-media-1")
	if err != nil {
		t.Fatalf("申请媒体失败: %v", err)
	}
	if req.Status != MediaRequested {
		t.Fatalf("申请初始状态异常：%+v", req)
	}
	if _, err := svc.ApproveMediaAccess(context.Background(), supportActor,
		req.AccessRequestID, "user-001"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("用户自批应拒绝，实际 err=%v", err)
	}
	approverA := Actor{StaffID: "staff-a", DataRegion: "cn", Role: RoleSupport}
	approverB := Actor{StaffID: "staff-b", DataRegion: "cn", Role: RoleSupport}
	afterA, err := svc.ApproveMediaAccess(context.Background(), approverA,
		req.AccessRequestID, approverA.StaffID)
	if err != nil || afterA.Status != MediaRequested {
		t.Fatalf("一人审批后应仍为 requested：%+v err=%v", afterA, err)
	}
	afterB, err := svc.ApproveMediaAccess(context.Background(), approverB,
		req.AccessRequestID, approverB.StaffID)
	if err != nil || afterB.Status != MediaApproved {
		t.Fatalf("双人审批后应放行：%+v err=%v", afterB, err)
	}
}

// 媒体申请可拒绝；工单可解决；角色门禁。
func TestMediaRejectResolveAndRoleGuard(t *testing.T) {
	svc, _ := newTestService(t)
	ticket := mustTicket(t, svc, "idem-ticket-4")
	req, err := svc.RequestMediaAccess(context.Background(), supportActor,
		"user-001", ticket.TicketID, "session-1", "idem-media-2")
	if err != nil {
		t.Fatalf("申请媒体失败: %v", err)
	}
	rejected, err := svc.RejectMediaAccess(context.Background(), supportActor, req.AccessRequestID)
	if err != nil || rejected.Status != MediaRejected {
		t.Fatalf("拒绝媒体异常：%+v err=%v", rejected, err)
	}
	resolved, err := svc.ResolveTicket(context.Background(), supportActor, ticket.TicketID)
	if err != nil || resolved.Status != TicketResolved {
		t.Fatalf("解决工单异常：%+v err=%v", resolved, err)
	}
	ops := Actor{StaffID: "staff-ops", DataRegion: "cn", Role: RoleOps}
	if _, err := svc.CreateTicket(context.Background(), ops,
		"user-001", "x", TicketOther, "idem-ticket-5"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ops 建工单应被拒，实际 err=%v", err)
	}
}
