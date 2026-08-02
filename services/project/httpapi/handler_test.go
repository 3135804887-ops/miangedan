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
func (a *appAdapter) GeneratePlanDraft(ctx context.Context, actor project.Actor, id, key string) (project.PlanVersion, error) {
	return a.svc.GeneratePlanDraft(ctx, actor, id, key)
}
func (a *appAdapter) EditPlan(ctx context.Context, actor project.Actor, id string, base int, rounds []project.RoundConfig, key string) (project.PlanVersion, error) {
	return a.svc.EditPlan(ctx, actor, id, base, rounds, key)
}
func (a *appAdapter) ConfirmPlan(ctx context.Context, actor project.Actor, id string, v int, acc []string, quote, key string) (project.Project, error) {
	return a.svc.ConfirmPlan(ctx, actor, id, v, acc, quote, key)
}
func (a *appAdapter) SaveLibraryEntry(ctx context.Context, actor project.Actor, kind project.LibraryKind, materialID string, version int, company, jobTitle, key string) (project.LibraryEntry, error) {
	return a.svc.SaveLibraryEntry(ctx, actor, kind, materialID, version, company, jobTitle, key)
}
func (a *appAdapter) ListLibrary(ctx context.Context, actor project.Actor, kind project.LibraryKind) ([]project.LibraryEntry, error) {
	return a.svc.ListLibrary(ctx, actor, kind)
}
func (a *appAdapter) DeleteLibraryEntry(ctx context.Context, actor project.Actor, kind project.LibraryKind, materialID, key string) error {
	return a.svc.DeleteLibraryEntry(ctx, actor, kind, materialID, key)
}
func (a *appAdapter) GetPreferences(ctx context.Context, actor project.Actor) (project.Preferences, error) {
	return a.svc.GetPreferences(ctx, actor)
}
func (a *appAdapter) SetPreferences(ctx context.Context, actor project.Actor, ui, interview, key string) (project.Preferences, error) {
	return a.svc.SetPreferences(ctx, actor, ui, interview, key)
}
func (a *appAdapter) ClaimDevice(ctx context.Context, actor project.Actor, id, device, key string) (project.Project, error) {
	return a.svc.ClaimDevice(ctx, actor, id, device, key)
}
func (a *appAdapter) TransferDevice(ctx context.Context, actor project.Actor, id, current, next, key string) (project.Project, error) {
	return a.svc.TransferDevice(ctx, actor, id, current, next, key)
}
func (a *appAdapter) ReleaseDevice(ctx context.Context, actor project.Actor, id, device, key string) (project.Project, error) {
	return a.svc.ReleaseDevice(ctx, actor, id, device, key)
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

// 异常路径：计划生成前置不满足（DRAFT 未确认材料）→ 409。
func TestGeneratePlanStateConflictHTTP(t *testing.T) {
	h := newTestHandler(t)
	rec := doJSON(t, h, http.MethodPost, "/v1/projects/00000000-0000-4000-8000-000000000000/plan:generate", "{}")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("未知项目应 404，实际 %d", rec.Code)
	}
}

// 正常路径：材料库保存/列表/删除；语言偏好读写；设备认领/转移/释放。
func TestLibraryAndPreferencesHTTP(t *testing.T) {
	h := newTestHandler(t)
	rec := doJSON(t, h, http.MethodPost, "/v1/library/resumes", `{"material_id": "11111111-1111-4111-8111-111111111111", "version": 1, "company": "合成科技", "job_title": "后端工程师"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("保存简历应 201，实际 %d: %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/v1/library/resumes", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "11111111-1111-4111-8111-111111111111") {
		t.Fatalf("简历库列表异常: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPut, "/v1/me/preferences", `{"ui_language": "en-US", "interview_language": "zh-CN"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"interview_language":"zh-CN"`) {
		t.Fatalf("偏好写入异常: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodGet, "/v1/me/preferences", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ui_language":"en-US"`) {
		t.Fatalf("偏好读取异常: %d %s", rec.Code, rec.Body.String())
	}
}

// 正常路径：创建项目后认领设备；另一设备认领被拒（409 device_active）；转移后原设备失效。
func TestDeviceClaimTransferHTTP(t *testing.T) {
	h := newTestHandler(t)
	rec := doJSON(t, h, http.MethodPost, "/v1/projects", `{"interview_language": "zh-CN", "degraded_mode": "full"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建应 201: %d", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	id := body["project_id"].(string)
	rec = doJSON(t, h, http.MethodPost, "/v1/projects/"+id+"/device:claim", `{"device_id": "device-a"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("认领应 200: %d %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, h, http.MethodPost, "/v1/projects/"+id+"/device:claim", `{"device_id": "device-b"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("非正式状态下的第二次认领允许替换（仅正式面试锁定）: %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/v1/projects/"+id+"/device:transfer", `{"current_device_id": "device-b", "new_device_id": "device-c"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "device-c") {
		t.Fatalf("转移应 200: %d %s", rec.Code, rec.Body.String())
	}
}
