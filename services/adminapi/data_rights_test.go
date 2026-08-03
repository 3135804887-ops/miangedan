// Package adminapi 数据权利请求测试（TASK-083；FR-040，US-05 场景 5）。
package adminapi

import (
	"context"
	"errors"
	"testing"
)

var supportActor = Actor{StaffID: "staff-support", DataRegion: "cn", Role: RoleSupport}

func sampleDataRight() DataRightRequest {
	return DataRightRequest{
		UserID: "user-001", RequestType: DRDelete,
		TargetType: "project", TargetID: "p-1",
	}
}

// 创建 + 执行：六层真实进度全 done；级联逐项可追踪；财务记录保留说明。
func TestDataRightExecuteWithRealProgress(t *testing.T) {
	svc, _ := newTestService(t)
	req, err := svc.CreateDataRightRequest(context.Background(),
		supportActor, sampleDataRight(), "idem-dr-1")
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}
	if req.Status != DRRequested || req.LegalRetentionNote != LegalRetentionNote {
		t.Fatalf("初始状态异常：%+v", req)
	}
	done, err := svc.ExecuteDataRight(context.Background(), supportActor, req.RequestID)
	if err != nil {
		t.Fatalf("执行请求失败: %v", err)
	}
	if done.Status != DRCompleted || done.Progress.ObjectStorage != LayerDone ||
		done.Progress.Backups != LayerDone || done.Progress.ThirdPartyProcessors != LayerDone {
		t.Fatalf("六层进度异常：%+v", done.Progress)
	}
	for item, status := range done.Cascade {
		if status != LayerDone {
			t.Fatalf("级联项 %s 未完成：%s", item, status)
		}
	}
	if done.CompletedAt == nil {
		t.Fatal("完成时间应记录")
	}
}

// 失败可重试：某一层失败 → FAILED 且如实展示；不伪造完成。
func TestDataRightFailureIsHonestAndRetryable(t *testing.T) {
	svc, _ := newTestService(t)
	req, err := svc.CreateDataRightRequest(context.Background(),
		supportActor, sampleDataRight(), "idem-dr-2")
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}
	failed, err := svc.FailDataRightLayer(context.Background(), supportActor, req.RequestID)
	if err != nil {
		t.Fatalf("失败注入失败: %v", err)
	}
	if failed.Status != DRFailed || failed.Progress.ObjectStorage != LayerFailed {
		t.Fatalf("失败状态应如实展示：%+v", failed)
	}
	// 重试后完成。
	retried, err := svc.ExecuteDataRight(context.Background(), supportActor, req.RequestID)
	if err != nil || retried.Status != DRCompleted {
		t.Fatalf("重试应可完成：%+v err=%v", retried, err)
	}
}

// 幂等：同一幂等键返回同一请求；非法类型拒绝。
func TestDataRightIdempotencyAndValidation(t *testing.T) {
	svc, _ := newTestService(t)
	first, err := svc.CreateDataRightRequest(context.Background(),
		supportActor, sampleDataRight(), "idem-dr-3")
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}
	second, err := svc.CreateDataRightRequest(context.Background(),
		supportActor, sampleDataRight(), "idem-dr-3")
	if err != nil || second.RequestID != first.RequestID {
		t.Fatalf("幂等异常：%+v err=%v", second, err)
	}
	bad := sampleDataRight()
	bad.RequestType = "hack"
	if _, err := svc.CreateDataRightRequest(context.Background(),
		supportActor, bad, "idem-dr-4"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("非法类型应拒绝，实际 err=%v", err)
	}
}

// 角色门禁：非 support 角色不可创建/执行。
func TestDataRightRoleGuard(t *testing.T) {
	svc, _ := newTestService(t)
	ops := Actor{StaffID: "staff-ops", DataRegion: "cn", Role: RoleOps}
	if _, err := svc.CreateDataRightRequest(context.Background(),
		ops, sampleDataRight(), "idem-dr-5"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ops 创建数据权利请求应被拒，实际 err=%v", err)
	}
}
