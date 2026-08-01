package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miangedan/services/identity"
	"miangedan/services/project"
)

type appAdapter struct {
	svc *project.Service
}

func (a *appAdapter) CreateProject(ctx context.Context, actor project.Actor, in project.CreateInput, key string) (project.Project, error) {
	return a.svc.CreateProject(ctx, actor, in, key)
}
func (a *appAdapter) GetProject(ctx context.Context, actor project.Actor, id string) (project.Project, error) {
	return a.svc.GetProject(ctx, actor, id)
}
func (a *appAdapter) ListProjects(ctx context.Context, actor project.Actor, f project.ListFilter) ([]project.Project, error) {
	return a.svc.ListProjects(ctx, actor, f)
}
func (a *appAdapter) RenameProject(ctx context.Context, actor project.Actor, id, name, key string) (project.Project, error) {
	return a.svc.RenameProject(ctx, actor, id, name, key)
}
func (a *appAdapter) DeleteProject(ctx context.Context, actor project.Actor, id, key string) (project.DeletionTask, error) {
	return a.svc.DeleteProject(ctx, actor, id, key)
}
func (a *appAdapter) DuplicateProject(ctx context.Context, actor project.Actor, id, lang, key string) (project.Project, error) {
	return a.svc.DuplicateProject(ctx, actor, id, lang, key)
}
func (a *appAdapter) GetPlan(ctx context.Context, actor project.Actor, id string) (project.PlanVersion, error) {
	return a.svc.GetPlan(ctx, actor, id)
}
func (a *appAdapter) EditPlan(ctx context.Context, actor project.Actor, id string, base int, rounds []project.RoundConfig, key string) (project.PlanVersion, error) {
	return a.svc.EditPlan(ctx, actor, id, base, rounds, key)
}
func (a *appAdapter) ConfirmPlan(ctx context.Context, actor project.Actor, id string, v int, acc []string, quote, key string) (project.Project, error) {
	return a.svc.ConfirmPlan(ctx, actor, id, v, acc, quote, key)
}

type stubAuth struct{}

func (stubAuth) Authenticate(_ string) (identity.Claims, error) {
	return identity.Claims{UserID: "user-001", DataRegion: "cn", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	flow, err := project.LoadFlowConfig("")
	if err != nil {
		t.Fatalf("加载流程配置失败: %v", err)
	}
	store := project.NewMemoryStore()
	svc, err := project.NewService(store, store, flow)
	if err != nil {
		t.Fatal(err)
	}
	h, err := New(&appAdapter{svc: svc}, stubAuth{}, "cn")
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func doJSON(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer synthetic-valid-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// 正常路径：创建项目 201。
func TestCreateProjectHTTP(t *testing.T) {
	h := newTestHandler(t)
	rec := doJSON(t, h, http.MethodPost, "/v1/projects", `{
		"interview_language": "zh-CN",
		"degraded_mode": "full",
		"resume_id": "11111111-1111-4111-8111-111111111111",
		"resume_version": 1
	}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建应 201，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "DRAFT" || body["project_id"] == "" {
		t.Fatalf("响应异常: %v", body)
	}
}

// 异常路径：降级模式缺同意 → 422 invalid_input。
func TestCreateDegradedWithoutConsentHTTP(t *testing.T) {
	h := newTestHandler(t)
	rec := doJSON(t, h, http.MethodPost, "/v1/projects", `{"interview_language": "zh-CN", "degraded_mode": "jd_only"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("应 422，实际 %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invalid_input") {
		t.Fatalf("错误码应为 invalid_input: %s", rec.Body.String())
	}
}

// 异常路径：未携带令牌 → 401。
func TestUnauthorizedHTTP(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("应 401，实际 %d", rec.Code)
	}
}

// 异常路径：查询不存在项目 → 404。
func TestGetProjectNotFoundHTTP(t *testing.T) {
	h := newTestHandler(t)
	rec := doJSON(t, h, http.MethodGet, "/v1/projects/00000000-0000-4000-8000-000000000000", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("应 404，实际 %d", rec.Code)
	}
}

// 异常路径：计划生成占位 → 501（TASK-033 落地前）。
func TestGeneratePlanPendingHTTP(t *testing.T) {
	h := newTestHandler(t)
	rec := doJSON(t, h, http.MethodPost, "/v1/projects/00000000-0000-4000-8000-000000000000/plan:generate", "{}")
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("应 501，实际 %d", rec.Code)
	}
}
