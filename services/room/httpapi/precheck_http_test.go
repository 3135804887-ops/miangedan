package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"miangedan/services/project"
	"miangedan/services/room"
)

// TASK-027 HTTP 正常路径：冻结会前配置 → 查询一致。
func TestPreCheckHTTPFreezeAndGet(t *testing.T) {
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
	handler, _ := New(&appAdapter{svc: svc}, stubAuth{}, "cn")
	projectID := readyHTTPProject(t, projSvc, projStore)
	sessionID := createHTTPSession(t, handler, projectID)

	rec := doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/precheck/freeze",
		`{"input_modes": ["voice", "text", "camera"], "accommodations": ["reduced_motion"], "device_report": {"camera_ok": true, "mic_ok": true, "network_rated": "good"}}`,
		"http-precheck-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("冻结应 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var pc map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &pc)
	if pc["frozen"] != true || len(pc["input_modes"].([]any)) != 3 {
		t.Fatalf("冻结结果错误: %v", pc)
	}
	rec = doJSON(t, handler, http.MethodGet, "/v1/sessions/"+sessionID+"/precheck", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("查询应 200，实际 %d", rec.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["frozen"] != true {
		t.Fatalf("查询结果错误: %v", got)
	}
}

// TASK-027 HTTP 异常：重复冻结返回 409。
func TestPreCheckHTTPAlreadyFrozen(t *testing.T) {
	flow, _ := project.LoadFlowConfig("")
	projStore := project.NewMemoryStore()
	projSvc, _ := project.NewService(projStore, projStore, flow)
	store := room.NewMemoryStore()
	tokens, _ := room.NewMediaTokenManager(room.TokenConfig{SigningKey: "synthetic-media-signing-key-0123456789abcdef", TTL: room.TokenTTLDefault}, store)
	svc, _ := room.NewService(store, store, tokens, room.StubRoomProvider{}, projectAPIAdapter{svc: projSvc})
	handler, _ := New(&appAdapter{svc: svc}, stubAuth{}, "cn")
	projectID := readyHTTPProject(t, projSvc, projStore)
	sessionID := createHTTPSession(t, handler, projectID)
	body := `{"input_modes": ["voice"], "accommodations": [], "device_report": {"camera_ok": false, "mic_ok": true, "network_rated": "fair"}}`
	if rec := doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/precheck/freeze", body, "http-precheck-2"); rec.Code != http.StatusOK {
		t.Fatalf("首次冻结应 200，实际 %d", rec.Code)
	}
	rec := doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/precheck/freeze", body, "http-precheck-2b")
	if rec.Code != http.StatusConflict {
		t.Fatalf("重复冻结应 409，实际 %d: %s", rec.Code, rec.Body.String())
	}
}
