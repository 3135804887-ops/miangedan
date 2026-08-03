package export

import (
	"context"
	"errors"
	"testing"
	"time"
)

var testActor = Actor{UserID: "user-001", DataRegion: "cn"}

func newTestService(t *testing.T) (*Service, *MemoryStore) {
	t.Helper()
	store := NewMemoryStore()
	svc, err := NewService(store)
	if err != nil {
		t.Fatalf("创建服务失败: %v", err)
	}
	return svc, store
}

// 导出：异步任务创建 → 执行 → 进度可查；导出物必带训练用途标记。
func TestTaskLifecycle(t *testing.T) {
	svc, store := newTestService(t)
	task, err := svc.CreateExport(context.Background(), testActor, "", "account", "idem-export-1")
	if err != nil {
		t.Fatalf("创建导出任务失败: %v", err)
	}
	if task.Status != TaskQueued || !task.TrainingMarker {
		t.Fatalf("初始状态异常：%+v", task)
	}
	executed, err := svc.ExecuteExport(context.Background(), testActor, task.TaskID)
	if err != nil {
		t.Fatalf("执行导出失败: %v", err)
	}
	if executed.Status != TaskSucceeded {
		t.Fatalf("导出应 succeeded：%s", executed.Status)
	}
	if executed.ExportContentRef == "" {
		t.Fatal("导出内容引用缺失")
	}
	if executed.ProgressNote == nil || !contains(*executed.ProgressNote, TrainingUseDisclaimer) {
		t.Fatalf("导出说明必须含训练用途标记：%v", executed.ProgressNote)
	}
	queried, err := svc.GetTask(context.Background(), testActor, task.TaskID)
	if err != nil || queried.Status != TaskSucceeded {
		t.Fatalf("进度可查异常：%+v err=%v", queried, err)
	}
	// 幂等：重复创建返回同一任务。
	again, err := svc.CreateExport(context.Background(), testActor, "", "account", "idem-export-1")
	if err != nil || again.TaskID != task.TaskID {
		t.Fatalf("导出幂等异常：%+v err=%v", again, err)
	}
	_ = store
}

// 项目报告导出：scope=project 必须携带 project_id；标记强制。
func TestProjectExportRequiresProjectID(t *testing.T) {
	svc, _ := newTestService(t)
	if _, err := svc.CreateExport(context.Background(), testActor, "", "project", "idem-export-2"); err == nil {
		t.Fatal("project 导出缺 project_id 必须拒绝")
	}
	task, err := svc.CreateExport(context.Background(), testActor, "p-1", "project", "idem-export-3")
	if err != nil {
		t.Fatalf("创建项目导出失败: %v", err)
	}
	if task.ProjectID != "p-1" || !task.TrainingMarker {
		t.Fatalf("项目导出异常：%+v", task)
	}
}

// 删除任务：级联执行六层真实进度；完成状态；财务记录保留说明。
func TestDeletionTaskLifecycle(t *testing.T) {
	svc, _ := newTestService(t)
	task, err := svc.CreateDeletionTask(context.Background(), testActor,
		DeletionRequest{TargetType: TargetProject, TargetID: "p-1"}, "idem-del-1")
	if err != nil {
		t.Fatalf("创建删除任务失败: %v", err)
	}
	if task.Status != DeletionRequested {
		t.Fatalf("初始状态应为 REQUESTED：%s", task.Status)
	}
	if task.LegalRetentionNote == nil || !contains(*task.LegalRetentionNote, "解除内容关联") {
		t.Fatalf("财务记录保留说明缺失：%v", task.LegalRetentionNote)
	}
	executed, err := svc.ExecuteDeletion(context.Background(), testActor, task.TaskID)
	if err != nil {
		t.Fatalf("执行删除失败: %v", err)
	}
	if executed.Status != DeletionCompleted {
		t.Fatalf("删除应完成：%s", executed.Status)
	}
	progress := executed.Progress
	for name, status := range map[string]string{
		"database": progress.Database, "cache": progress.Cache,
		"search_index": progress.SearchIndex, "object_storage": progress.ObjectStorage,
		"backups": progress.Backups, "third_party_processors": progress.ThirdPartyProcessors,
	} {
		if status != LayerDone {
			t.Fatalf("层 %s 应为 done，实际 %s", name, status)
		}
	}
	if executed.CompletedAt == nil {
		t.Fatal("完成时间缺失")
	}
}

// 删除失败 → FAILED 可重试；重试不伪造完成。
func TestDeletionFailureAndRetry(t *testing.T) {
	svc, _ := newTestService(t)
	task, err := svc.CreateDeletionTask(context.Background(), testActor,
		DeletionRequest{TargetType: TargetAccount, TargetID: "u-1"}, "idem-del-2")
	if err != nil {
		t.Fatalf("创建删除任务失败: %v", err)
	}
	failed, err := svc.FailDeletionLayer(context.Background(), testActor, task.TaskID)
	if err != nil {
		t.Fatalf("注入失败失败: %v", err)
	}
	if failed.Status != DeletionFailed || failed.Progress.ObjectStorage != LayerFailed {
		t.Fatalf("失败状态异常：%+v", failed)
	}
	if _, err := svc.RetryDeletionTask(context.Background(), testActor, task.TaskID); err != nil {
		t.Fatalf("重试失败: %v", err)
	}
	retried, err := svc.GetDeletionTask(context.Background(), testActor, task.TaskID)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if retried.Status != DeletionCompleted || retried.Progress.ObjectStorage != LayerDone {
		t.Fatalf("重试后应完成：%+v", retried)
	}
	// 非 FAILED 任务不可重试。
	if _, err := svc.RetryDeletionTask(context.Background(), testActor, task.TaskID); !errors.Is(
		err, ErrStateConflict) {
		t.Fatalf("已完成任务重试应拒绝，实际 %v", err)
	}
}

// 删除幂等：同一幂等键返回同一任务。
func TestDeletionIdempotent(t *testing.T) {
	svc, _ := newTestService(t)
	req := DeletionRequest{TargetType: TargetResume, TargetID: "r-1"}
	first, err := svc.CreateDeletionTask(context.Background(), testActor, req, "idem-del-3")
	if err != nil {
		t.Fatalf("创建失败: %v", err)
	}
	second, err := svc.CreateDeletionTask(context.Background(), testActor, req, "idem-del-3")
	if err != nil || second.TaskID != first.TaskID {
		t.Fatalf("删除幂等异常：%+v err=%v", second, err)
	}
}

// 到期提醒扫描（RETENTION-MATRIX §5：30 天/7 天窗口）。
func TestExpiringItemsScan(t *testing.T) {
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	items := []ExpiringItem{
		{Kind: "report", ItemID: "a", ExpiresAt: now.Add(40 * 24 * time.Hour)}, // 不在窗口
		{Kind: "report", ItemID: "b", ExpiresAt: now.Add(20 * 24 * time.Hour)}, // 30 天窗口
		{Kind: "media", ItemID: "c", ExpiresAt: now.Add(5 * 24 * time.Hour)},   // 7 天窗口
		{Kind: "report", ItemID: "d", ExpiresAt: now.Add(-1 * time.Hour)},      // 已到期不提醒
	}
	expiring := ExpiringItems(items, now, DefaultExpiryWindows)
	if len(expiring) != 2 {
		t.Fatalf("应命中 2 项提醒：%v", expiring)
	}
	ids := map[string]bool{}
	for _, item := range expiring {
		ids[item.ItemID] = true
	}
	if !ids["b"] || !ids["c"] {
		t.Fatalf("提醒窗口命中异常：%v", expiring)
	}
}

// 异常：非法 target_type、缺身份/区域必须拒绝。
func TestInvalidRequestsRejected(t *testing.T) {
	svc, _ := newTestService(t)
	if _, err := svc.CreateDeletionTask(context.Background(), testActor,
		DeletionRequest{TargetType: "other", TargetID: "x"}, "idem-del-bad"); err == nil {
		t.Fatal("非法 target_type 必须拒绝")
	}
	if _, err := svc.CreateDeletionTask(context.Background(), testActor,
		DeletionRequest{TargetType: TargetProject, TargetID: ""}, "idem-del-bad2"); err == nil {
		t.Fatal("缺 target_id 必须拒绝")
	}
	if _, err := svc.CreateExport(context.Background(),
		Actor{UserID: "", DataRegion: "cn"}, "", "account", "idem-exp-bad"); err == nil {
		t.Fatal("缺身份必须拒绝")
	}
}

func contains(text, fragment string) bool {
	return len(text) >= len(fragment) && indexOf(text, fragment) >= 0
}

func indexOf(text, fragment string) int {
	for i := 0; i+len(fragment) <= len(text); i++ {
		if text[i:i+len(fragment)] == fragment {
			return i
		}
	}
	return -1
}
