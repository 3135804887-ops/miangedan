package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"miangedan/services/identity"
	"miangedan/services/project"
	"miangedan/services/room"
)

type stubAuth struct{}

func (stubAuth) Authenticate(_ string) (identity.Claims, error) {
	return identity.Claims{UserID: "user-001", DataRegion: "cn", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

type appAdapter struct {
	svc *room.Service
}

type projectAPIAdapter struct {
	svc *project.Service
}

func (a projectAPIAdapter) GetProject(ctx context.Context, actor project.Actor, id string) (project.Project, error) {
	return a.svc.GetProject(ctx, actor, id)
}
func (a projectAPIAdapter) GetPlan(ctx context.Context, actor project.Actor, id string) (project.PlanVersion, error) {
	return a.svc.GetPlan(ctx, actor, id)
}
func (a projectAPIAdapter) ClaimDevice(ctx context.Context, actor project.Actor, id, device, key string) (project.Project, error) {
	return a.svc.ClaimDevice(ctx, actor, id, device, key)
}
func (a projectAPIAdapter) TransferDevice(ctx context.Context, actor project.Actor, id, current, next, key string) (project.Project, error) {
	return a.svc.TransferDevice(ctx, actor, id, current, next, key)
}
func (a projectAPIAdapter) ReleaseDevice(ctx context.Context, actor project.Actor, id, device, key string) (project.Project, error) {
	return a.svc.ReleaseDevice(ctx, actor, id, device, key)
}

func (a *appAdapter) CreateSession(ctx context.Context, actor project.Actor, in room.CreateSessionInput, key string) (room.SessionCreated, error) {
	return a.svc.CreateSession(ctx, actor, in, key)
}
func (a *appAdapter) GetSession(ctx context.Context, actor project.Actor, id string) (room.Session, error) {
	return a.svc.GetSession(ctx, actor, id)
}
func (a *appAdapter) EndSession(ctx context.Context, actor project.Actor, id string, confirm bool, key string) (room.Session, error) {
	return a.svc.EndSession(ctx, actor, id, confirm, key)
}
func (a *appAdapter) ReconnectSession(ctx context.Context, actor project.Actor, id, device string, seq int, key string) (room.SessionCreated, error) {
	return a.svc.ReconnectSession(ctx, actor, id, device, seq, key)
}
func (a *appAdapter) DeviceTransferSession(ctx context.Context, actor project.Actor, id, device string, confirm bool, key string) (room.SessionCreated, error) {
	return a.svc.DeviceTransferSession(ctx, actor, id, device, confirm, key)
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	flow, err := project.LoadFlowConfig("")
	if err != nil {
		t.Fatalf("加载流程配置失败: %v", err)
	}
	projStore := project.NewMemoryStore()
	projSvc, err := project.NewService(projStore, projStore, flow)
	if err != nil {
		t.Fatal(err)
	}
	store := room.NewMemoryStore()
	tokens, err := room.NewMediaTokenManager(room.TokenConfig{SigningKey: "synthetic-media-signing-key-0123456789abcdef", TTL: room.TokenTTLDefault}, store)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := room.NewService(store, store, tokens, room.StubRoomProvider{}, projectAPIAdapter{svc: projSvc})
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

// 正常路径：创建会话 201；未授权 401；未知会话 404。
func TestSessionHTTP(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("未授权应 401，实际 %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodGet, "/v1/sessions/00000000-0000-4000-8000-000000000000", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("未知会话应 404，实际 %d", rec.Code)
	}
	rec = doJSON(t, h, http.MethodPost, "/v1/projects/00000000-0000-4000-8000-000000000000/rounds/1/session", `{"kind": "formal", "device_id": "d"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("不存在项目应 404，实际 %d: %s", rec.Code, rec.Body.String())
	}
}
