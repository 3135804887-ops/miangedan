package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"miangedan/services/project"
)

var genHTTPActor = project.Actor{UserID: "user-001", DataRegion: "cn"}

// TASK-033 HTTP 正常路径：材料确认后 plan:generate 返回 201 草稿。
func TestGeneratePlanHTTP(t *testing.T) {
	// 直接构造完整环境（便于将项目置为 MATERIAL_REVIEW）。
	flow, _ := project.LoadFlowConfig("")
	store := project.NewMemoryStore()
	svc, err := project.NewService(store, store, flow)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(&appAdapter{svc: svc}, stubAuth{}, "cn")
	if err != nil {
		t.Fatal(err)
	}
	proj, err := svc.CreateProject(context.Background(), genHTTPActor, project.CreateInput{
		InterviewLanguage: "zh-CN", DegradedMode: project.ModeFull,
		ResumeRef: &project.MaterialRef{ID: "resume-1", Version: 1},
	}, "http-gen-create")
	if err != nil {
		t.Fatal(err)
	}
	proj.Status = project.StatusMaterialReview
	if err := store.UpdateProject(proj); err != nil {
		t.Fatal(err)
	}
	rec := doJSON(t, handler, http.MethodPost, "/v1/projects/"+proj.ProjectID+"/plan:generate", "{}")
	if rec.Code != http.StatusCreated {
		t.Fatalf("生成应 201，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var plan map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &plan)
	if plan["frozen"] != false || plan["rubric_version"] == "" {
		t.Fatalf("草稿响应错误: %v", plan)
	}
	rounds, ok := plan["rounds"].([]any)
	if !ok || len(rounds) != 3 {
		t.Fatalf("应 3 轮: %v", plan["rounds"])
	}
}
