package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"miangedan/services/project"
	"miangedan/services/room"
)

var httpActor = project.Actor{UserID: "user-001", DataRegion: "cn"}

// readyHTTPProject 构造 READY 项目并返回 projectId（与 room 包测试等价的环境准备）。
func readyHTTPProject(t *testing.T, projSvc *project.Service, projStore *project.MemoryStore) string {
	t.Helper()
	proj, err := projSvc.CreateProject(context.Background(), httpActor, project.CreateInput{
		InterviewLanguage: "zh-CN",
		DegradedMode:      project.ModeFull,
		ResumeRef:         &project.MaterialRef{ID: "resume-1", Version: 1},
	}, "http-ready-key")
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}
	plan := project.PlanVersion{
		ProjectID:         proj.ProjectID,
		PlanVersion:       1,
		DataRegion:        proj.DataRegion,
		InterviewLanguage: proj.InterviewLanguage,
		ResumeRef:         proj.ResumeRef,
		DegradedMode:      proj.DegradedMode,
		RubricVersion:     "rubrics/v1/default",
		DimensionWeights: map[string]int{
			"professional_competence": 25, "problem_solving": 20, "communication": 15,
			"experience_evidence": 15, "behavioral_collaboration": 15, "learning_adaptability": 10,
		},
		Frozen: true,
		Rounds: []project.RoundConfig{{
			Sequence: 1, RoundType: project.RoundTypes[0], DurationMinutes: 30,
			Difficulty: "standard", CriticalDimensions: []string{project.DimensionKeys[0]},
		}},
		RoundWeights: []project.RoundWeight{{Sequence: 1, Weight: 100}},
	}
	if err := projStore.SavePlan(plan); err != nil {
		t.Fatalf("保存计划失败: %v", err)
	}
	if err := projSvc.SetRoundReadiness(proj.DataRegion, proj.ProjectID, 1, 1, true, true); err != nil {
		t.Fatalf("标记就绪失败: %v", err)
	}
	proj.Status = project.StatusReady
	proj.PlanVersion = 1
	if err := projStore.UpdateProject(proj); err != nil {
		t.Fatalf("更新项目失败: %v", err)
	}
	return proj.ProjectID
}

func createHTTPSession(t *testing.T, h http.Handler, projectID string) string {
	t.Helper()
	rec := doJSONWithKey(t, h, http.MethodPost,
		"/v1/projects/"+projectID+"/rounds/1/session",
		`{"kind": "formal", "device_id": "device-a"}`, "http-create-session")
	if rec.Code != http.StatusCreated {
		t.Fatalf("创建会话应 201，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var out struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.SessionID
}

// TASK-023 HTTP 正常路径：追加 → 修订 → 冻结 → 查询。
func TestTranscriptHTTPLifecycle(t *testing.T) {
	// 直接构造完整环境（与 newTestHandler 等价，便于拿到 projSvc 准备 READY 项目）。
	flow, _ := project.LoadFlowConfig("")
	projStore := project.NewMemoryStore()
	projSvc, err := project.NewService(projStore, projStore, flow)
	if err != nil {
		t.Fatal(err)
	}
	store := room.NewMemoryStore()
	tokens, _ := room.NewMediaTokenManager(room.TokenConfig{SigningKey: "synthetic-media-signing-key-0123456789abcdef", TTL: room.TokenTTLDefault}, store)
	svc, err := room.NewService(store, store, tokens, room.StubRoomProvider{}, projectAPIAdapter{svc: projSvc})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := New(&appAdapter{svc: svc}, stubAuth{}, "cn")
	if err != nil {
		t.Fatal(err)
	}
	projectID := readyHTTPProject(t, projSvc, projStore)
	sessionID := createHTTPSession(t, handler, projectID)

	rec := doJSON(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/transcripts",
		`{"turn_index": 1, "utterance_id": "utt-http-1", "kind": "final", "text": "我的回答", "language": "zh-CN", "confidence": 0.88}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("追加应 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	rec = doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/revisions",
		`{"revision_id": "rev-http-1", "utterance_id": "utt-http-1", "turn_index": 1, "revised_text": "修订后的回答"}`,
		"http-rev-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("修订应 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	rec = doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/turns/1/freeze", "", "http-freeze-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("冻结应 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var frozen map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &frozen)
	if frozen["final_count"].(float64) != 1 || frozen["revised_count"].(float64) != 1 {
		t.Fatalf("冻结统计错误: %v", frozen)
	}
	rec = doJSON(t, handler, http.MethodGet, "/v1/sessions/"+sessionID+"/transcripts", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("查询应 200，实际 %d", rec.Code)
	}
	var items []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &items)
	if len(items) != 1 || items[0]["revision_state"] != "accepted" || items[0]["frozen"] != true {
		t.Fatalf("冻结后转写应为 accepted+frozen: %v", items)
	}
}

// TASK-023 HTTP 异常路径：窗口关闭后修订返回 409。
func TestTranscriptHTTPWindowClosed(t *testing.T) {
	flow, _ := project.LoadFlowConfig("")
	projStore := project.NewMemoryStore()
	projSvc, _ := project.NewService(projStore, projStore, flow)
	store := room.NewMemoryStore()
	tokens, _ := room.NewMediaTokenManager(room.TokenConfig{SigningKey: "synthetic-media-signing-key-0123456789abcdef", TTL: room.TokenTTLDefault}, store)
	svc, _ := room.NewService(store, store, tokens, room.StubRoomProvider{}, projectAPIAdapter{svc: projSvc})
	handler, err := New(&appAdapter{svc: svc}, stubAuth{}, "cn")
	if err != nil {
		t.Fatal(err)
	}
	projectID := readyHTTPProject(t, projSvc, projStore)
	sessionID := createHTTPSession(t, handler, projectID)
	doJSON(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/transcripts",
		`{"turn_index": 1, "utterance_id": "utt-http-2", "kind": "final", "text": "回答", "language": "zh-CN", "confidence": 0.8}`)
	if rec := doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/turns/1/freeze", "", "http-freeze-2"); rec.Code != http.StatusOK {
		t.Fatalf("冻结应 200，实际 %d", rec.Code)
	}
	rec := doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/revisions",
		`{"revision_id": "rev-late", "utterance_id": "utt-http-2", "turn_index": 1, "revised_text": "迟到修订"}`,
		"http-rev-late")
	if rec.Code != http.StatusConflict {
		t.Fatalf("窗口关闭后修订应 409，实际 %d: %s", rec.Code, rec.Body.String())
	}
}
