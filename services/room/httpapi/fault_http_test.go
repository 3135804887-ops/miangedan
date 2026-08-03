package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"miangedan/services/project"
	"miangedan/services/room"
)

// TASK-025 HTTP 正常路径：暂停 → 降级询问 → 接受 → TEXT_DEGRADED。
func TestFaultHTTPDowngradeFlow(t *testing.T) {
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
	// 置 LIVE。
	s, err := svc.GetSession(context.Background(), httpActor, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	s.RoomStatus = room.StatusLive
	_ = store.UpdateSession(s)

	rec := doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/timer/pause",
		`{"reason": "system_fault"}`, "http-pause-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("暂停应 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	rec = doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/downgrade/offer", "", "http-offer-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("降级询问应 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var offer struct {
		PromptID string `json:"prompt_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &offer)
	if offer.PromptID == "" {
		t.Fatal("prompt_id 为空")
	}
	rec = doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/downgrade/accept",
		`{"prompt_id": "`+offer.PromptID+`"}`, "http-accept-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("接受降级应 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var sess map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &sess)
	if sess["room_status"] != "TEXT_DEGRADED" {
		t.Fatalf("降级后状态错误: %v", sess)
	}
}

// TASK-025 HTTP 异常：未暂停会话恢复返回 409。
func TestFaultHTTPResumeInvalid(t *testing.T) {
	flow, _ := project.LoadFlowConfig("")
	projStore := project.NewMemoryStore()
	projSvc, _ := project.NewService(projStore, projStore, flow)
	store := room.NewMemoryStore()
	tokens, _ := room.NewMediaTokenManager(room.TokenConfig{SigningKey: "synthetic-media-signing-key-0123456789abcdef", TTL: room.TokenTTLDefault}, store)
	svc, _ := room.NewService(store, store, tokens, room.StubRoomProvider{}, projectAPIAdapter{svc: projSvc})
	handler, _ := New(&appAdapter{svc: svc}, stubAuth{}, "cn")
	projectID := readyHTTPProject(t, projSvc, projStore)
	sessionID := createHTTPSession(t, handler, projectID)
	rec := doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/timer/resume", "", "http-resume-x")
	if rec.Code != http.StatusConflict {
		t.Fatalf("未暂停恢复应 409，实际 %d: %s", rec.Code, rec.Body.String())
	}
}
