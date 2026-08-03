package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"miangedan/services/scoring"
)

type stubApp struct {
	latest  scoring.Result
	items   []scoring.Result
	next    string
	err     error
	queried []string
}

func (s *stubApp) GetLatest(
	_ context.Context, _ scoring.Actor, projectID string, _ int,
) (scoring.Result, error) {
	s.queried = append(s.queried, projectID)
	if s.err != nil {
		return scoring.Result{}, s.err
	}
	return s.latest, nil
}

func (s *stubApp) ListVersions(
	_ context.Context, _ scoring.Actor, projectID string, _ int, _ int, cursor string,
) ([]scoring.Result, string, error) {
	s.queried = append(s.queried, projectID+"|"+cursor)
	if s.err != nil {
		return nil, "", s.err
	}
	return s.items, s.next, nil
}

type stubAuth struct{}

func (stubAuth) Authenticate(token string) (scoring.Actor, error) {
	if token != "test-token" {
		return scoring.Actor{}, errors.New("invalid token")
	}
	return scoring.Actor{UserID: "user-001", DataRegion: "cn"}, nil
}

func newTestHandler(app Application) http.Handler {
	h, err := New(app, stubAuth{}, "cn")
	if err != nil {
		panic(err)
	}
	return h
}

func get(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestGetRoundResultOK(t *testing.T) {
	app := &stubApp{latest: scoring.Result{
		ScoreID: "score-1", ScoreVersion: 1, ProjectID: "p1",
		RoundSequence: 1, DataRegion: "cn", ResultStatus: scoring.ResultPass,
	}}
	rec := get(t, newTestHandler(app),
		"/v1/projects/p1/rounds/1/result")
	if rec.Code != http.StatusOK {
		t.Fatalf("应 200，实际 %d：%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON: %v", err)
	}
	if body["result_status"] != "PASS" || body["score_id"] != "score-1" {
		t.Fatalf("响应字段异常：%v", body)
	}
}

func TestGetRoundResultNotFound(t *testing.T) {
	app := &stubApp{err: scoring.ErrNotFound}
	rec := get(t, newTestHandler(app),
		"/v1/projects/p1/rounds/1/result")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("应 404，实际 %d", rec.Code)
	}
}

func TestListScoreVersionsPaged(t *testing.T) {
	app := &stubApp{
		items: []scoring.Result{{ScoreID: "s1"}, {ScoreID: "s2"}},
		next:  "2",
	}
	rec := get(t, newTestHandler(app),
		"/v1/projects/p1/rounds/1/scores?limit=2&cursor=0")
	if rec.Code != http.StatusOK {
		t.Fatalf("应 200，实际 %d", rec.Code)
	}
	var body struct {
		DataRegion string `json:"data_region"`
		Items      []any  `json:"items"`
		NextCursor string `json:"next_cursor"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if body.DataRegion != "cn" || len(body.Items) != 2 || body.NextCursor != "2" {
		t.Fatalf("分页响应异常：%+v", body)
	}
}

func TestListScoreVersionsInvalidLimit(t *testing.T) {
	rec := get(t, newTestHandler(&stubApp{}),
		"/v1/projects/p1/rounds/1/scores?limit=0")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("非法 limit 应 400，实际 %d", rec.Code)
	}
}

func TestReviewNotImplemented(t *testing.T) {
	handler := newTestHandler(&stubApp{})
	req := httptest.NewRequest(http.MethodPost,
		"/v1/projects/p1/rounds/1/review", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("正式复核未实现应 501，实际 %d", rec.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet,
		"/v1/projects/p1/rounds/1/result", nil)
	rec := httptest.NewRecorder()
	newTestHandler(&stubApp{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("未授权应 401，实际 %d", rec.Code)
	}
}
