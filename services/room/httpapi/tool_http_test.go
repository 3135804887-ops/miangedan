package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"miangedan/services/project"
	"miangedan/services/room"
)

// readyHTTPProjectWithTools 构造第 1 轮已配置 code_editor 的 READY 项目。
func readyHTTPProjectWithTools(t *testing.T, projSvc *project.Service, projStore *project.MemoryStore) string {
	t.Helper()
	id := readyHTTPProject(t, projSvc, projStore)
	plan, err := projSvc.GetPlan(context.Background(), httpActor, id)
	if err != nil {
		t.Fatal(err)
	}
	plan.Rounds[0].Tools = []string{"code_editor"}
	if err := projStore.SavePlan(plan); err != nil {
		t.Fatal(err)
	}
	return id
}

func newToolHandler(t *testing.T) (http.Handler, *project.Service, *project.MemoryStore) {
	t.Helper()
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
	return handler, projSvc, projStore
}

// TASK-024 HTTP 正常路径：激活已配置工具 → 记录事件 → 查询。
func TestToolHTTPLifecycle(t *testing.T) {
	handler, projSvc, projStore := newToolHandler(t)
	projectID := readyHTTPProjectWithTools(t, projSvc, projStore)
	sessionID := createHTTPSession(t, handler, projectID)

	rec := doJSON(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/tools/code_editor/activate", `{"preconfig_ref": "preconfig/tool/v1/code"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("激活应 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var act map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &act)
	if act["tool_key"] != "code_editor" {
		t.Fatalf("激活结果错误: %v", act)
	}
	rec = doJSONWithKey(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/tools/code_editor/events",
		`{"tool_event_id": "ev-http-1", "event_type": "run", "content_ref": "s3://region/session/tool/1"}`, "http-tool-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("记录事件应 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, handler, http.MethodGet, "/v1/sessions/"+sessionID+"/tools", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("查询应 200，实际 %d", rec.Code)
	}
	var items []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &items)
	if len(items) != 1 || items[0]["tool_event_id"] != "ev-http-1" {
		t.Fatalf("工具事件列表错误: %v", items)
	}
}

// TASK-024 HTTP 异常路径：未配置工具激活返回 409。
func TestToolHTTPNotConfigured(t *testing.T) {
	handler, projSvc, projStore := newToolHandler(t)
	projectID := readyHTTPProject(t, projSvc, projStore) // 未配置工具
	sessionID := createHTTPSession(t, handler, projectID)
	rec := doJSON(t, handler, http.MethodPost, "/v1/sessions/"+sessionID+"/tools/code_editor/activate", `{}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("未配置工具应 409，实际 %d: %s", rec.Code, rec.Body.String())
	}
}
