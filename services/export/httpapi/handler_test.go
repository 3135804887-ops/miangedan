package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miangedan/services/export"
)

type stubApp struct {
	Task         export.Task
	deletionTask export.DeletionTask
	err          error
}

func (s *stubApp) CreateExport(
	_ context.Context, _ export.Actor, _ string, _ string, _ string,
) (export.Task, error) {
	if s.err != nil {
		return export.Task{}, s.err
	}
	return s.Task, nil
}

func (s *stubApp) ExecuteExport(
	_ context.Context, _ export.Actor, taskID string,
) (export.Task, error) {
	if s.err != nil {
		return export.Task{}, s.err
	}
	task := s.Task
	task.TaskID = taskID
	task.Status = export.TaskSucceeded
	return task, nil
}

func (s *stubApp) GetTask(
	_ context.Context, _ export.Actor, _ string,
) (export.Task, error) {
	if s.err != nil {
		return export.Task{}, s.err
	}
	return s.Task, nil
}

func (s *stubApp) CreateDeletionTask(
	_ context.Context, _ export.Actor, _ export.DeletionRequest, _ string,
) (export.DeletionTask, error) {
	if s.err != nil {
		return export.DeletionTask{}, s.err
	}
	return s.deletionTask, nil
}

func (s *stubApp) ExecuteDeletion(
	_ context.Context, _ export.Actor, _ string,
) (export.DeletionTask, error) {
	if s.err != nil {
		return export.DeletionTask{}, s.err
	}
	task := s.deletionTask
	task.Status = export.DeletionCompleted
	return task, nil
}

func (s *stubApp) RetryDeletionTask(
	_ context.Context, _ export.Actor, _ string,
) (export.DeletionTask, error) {
	return s.ExecuteDeletion(context.Background(), export.Actor{}, "")
}

func (s *stubApp) GetDeletionTask(
	_ context.Context, _ export.Actor, _ string,
) (export.DeletionTask, error) {
	if s.err != nil {
		return export.DeletionTask{}, s.err
	}
	return s.deletionTask, nil
}

type stubAuth struct{}

func (stubAuth) Authenticate(token string) (export.Actor, error) {
	if token != "test-token" {
		return export.Actor{}, errors.New("invalid token")
	}
	return export.Actor{UserID: "user-001", DataRegion: "cn"}, nil
}

func newTestHandler(app Application) http.Handler {
	h, err := New(app, stubAuth{}, "cn")
	if err != nil {
		panic(err)
	}
	return h
}

func authed(req *http.Request) *http.Request {
	req.Header.Set("Authorization", "Bearer test-token")
	return req
}

func TestRequestExportAccepted(t *testing.T) {
	app := &stubApp{Task: export.Task{
		TaskID: "export-1", TaskType: "export", Status: export.TaskQueued, DataRegion: "cn",
	}}
	req := authed(httptest.NewRequest(http.MethodGet, "/v1/me/export", nil))
	rec := httptest.NewRecorder()
	newTestHandler(app).ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("导出应 202，实际 %d：%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON: %v", err)
	}
	if body["task_type"] != "export" || body["status"] != "succeeded" {
		t.Fatalf("导出任务响应异常：%v", body)
	}
}

func TestExportReportAccepted(t *testing.T) {
	app := &stubApp{Task: export.Task{
		TaskID: "export-2", TaskType: "export", Status: export.TaskQueued, DataRegion: "cn",
	}}
	req := authed(httptest.NewRequest(http.MethodPost,
		"/v1/projects/p1/report/export", strings.NewReader(`{}`)))
	rec := httptest.NewRecorder()
	newTestHandler(app).ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("报告导出应 202，实际 %d", rec.Code)
	}
}

func TestCreateDeletionTaskAccepted(t *testing.T) {
	app := &stubApp{deletionTask: export.DeletionTask{
		TaskID: "del-1", TargetType: export.TargetProject, TargetID: "p1",
		Status: export.DeletionInProgress, DataRegion: "cn",
	}}
	req := authed(httptest.NewRequest(http.MethodPost, "/v1/deletion-tasks",
		strings.NewReader(`{"target_type":"project","target_id":"p1"}`)))
	req.Header.Set("Idempotency-Key", "del-idem-001")
	rec := httptest.NewRecorder()
	newTestHandler(app).ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("删除任务应 202，实际 %d：%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON: %v", err)
	}
	if body["status"] != "COMPLETED" || body["target_type"] != "project" {
		t.Fatalf("DeletionTask 响应异常：%v", body)
	}
}

func TestGetDeletionTaskProgress(t *testing.T) {
	app := &stubApp{deletionTask: export.DeletionTask{
		TaskID: "del-2", TargetType: export.TargetAccount, TargetID: "u1",
		Status: export.DeletionInProgress, DataRegion: "cn",
		Progress: export.DeletionProgress{
			Database: export.LayerDone, Cache: export.LayerInProgress,
		},
	}}
	req := authed(httptest.NewRequest(http.MethodGet, "/v1/deletion-tasks/del-2", nil))
	rec := httptest.NewRecorder()
	newTestHandler(app).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("进度查询应 200，实际 %d", rec.Code)
	}
	var body struct {
		Progress export.DeletionProgress `json:"progress"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if body.Progress.Database != "done" || body.Progress.Cache != "in_progress" {
		t.Fatalf("真实进度异常：%+v", body.Progress)
	}
}
