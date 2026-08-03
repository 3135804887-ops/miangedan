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
	latest       scoring.Result
	items        []scoring.Result
	next         string
	err          error
	review       scoring.ReviewResult
	retryAttempt scoring.RetryAttempt
	queried      []string
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

func (s *stubApp) Review(
	_ context.Context, _ scoring.Actor, _ scoring.ReviewRequest,
) (scoring.ReviewResult, error) {
	if s.err != nil {
		return scoring.ReviewResult{}, s.err
	}
	return s.review, nil
}

func (s *stubApp) BeginRetry(
	_ context.Context, _ scoring.Actor, _ scoring.BeginRetryRequest,
) (scoring.RetryAttempt, error) {
	if s.err != nil {
		return scoring.RetryAttempt{}, s.err
	}
	return s.retryAttempt, nil
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

func TestReviewAccepted(t *testing.T) {
	app := &stubApp{review: scoring.ReviewResult{
		Reason: "正式复核：重算结果与原始一致；历史版本保留",
		Review: scoring.Result{
			ScoreID: "review-1", ScoreVersion: 2,
			ProjectID: "p1", RoundSequence: 1, DataRegion: "cn",
			ResultStatus: scoring.ResultPass,
		},
	}}
	handler := newTestHandler(app)
	req := httptest.NewRequest(http.MethodPost,
		"/v1/projects/p1/rounds/1/review",
		strings.NewReader(`{"attempt_id":"00000000-0000-4000-8000-00000000a001","scope":"round"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Idempotency-Key", "review-idem-001")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("复核应 202，实际 %d：%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON: %v", err)
	}
	if body["task_type"] != "review" || body["status"] != "succeeded" {
		t.Fatalf("异步任务字段异常：%v", body)
	}
	if body["review_result"] == nil {
		t.Fatal("202 响应必须包含 review_result（前后对比）")
	}
}

func TestReviewConflictAlreadyReviewed(t *testing.T) {
	handler := newTestHandler(&stubApp{err: scoring.ErrReviewLimit})
	req := httptest.NewRequest(http.MethodPost,
		"/v1/projects/p1/rounds/1/review",
		strings.NewReader(`{"attempt_id":"00000000-0000-4000-8000-00000000a001","scope":"round"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Idempotency-Key", "review-idem-002")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("二次复核应 409，实际 %d", rec.Code)
	}
}

func TestReviewRequiresIdempotencyKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost,
		"/v1/projects/p1/rounds/1/review",
		strings.NewReader(`{"attempt_id":"00000000-0000-4000-8000-00000000a001","scope":"round"}`))
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	newTestHandler(&stubApp{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("缺幂等键应 400，实际 %d", rec.Code)
	}
}

func TestStartRetryCreated(t *testing.T) {
	app := &stubApp{retryAttempt: scoring.RetryAttempt{
		AttemptID: "retry-1", ProjectID: "p1", RoundSequence: 1,
		Status: scoring.RetryStatusScheduled, DataRegion: "cn",
	}}
	req := httptest.NewRequest(http.MethodPost,
		"/v1/projects/p1/rounds/1/retry", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Idempotency-Key", "retry-idem-001")
	rec := httptest.NewRecorder()
	newTestHandler(app).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("发起重试应 201，实际 %d：%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON: %v", err)
	}
	if body["status"] != "RETRY_SCHEDULED" || body["attempt_id"] != "retry-1" {
		t.Fatalf("RetryAttempt 响应异常：%v", body)
	}
}

func TestStartRetryConflict(t *testing.T) {
	handler := newTestHandler(&stubApp{err: scoring.ErrStateConflict})
	req := httptest.NewRequest(http.MethodPost,
		"/v1/projects/p1/rounds/1/retry", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Idempotency-Key", "retry-idem-002")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("PASS 轮次重试应 409，实际 %d", rec.Code)
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
