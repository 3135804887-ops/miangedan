// Package httpapi exposes TASK-040/041/042 under the /v1/projects/{projectId}/rounds/{sequence}
// scores prefixes（openapi scores 标签）。
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"miangedan/services/region"
	"miangedan/services/scoring"
)

// Application 为传输无关的评分应用接口（openapi scores 标签）。
type Application interface {
	GetLatest(context.Context, scoring.Actor, string, int) (scoring.Result, error)
	ListVersions(
		context.Context, scoring.Actor, string, int, int, string,
	) ([]scoring.Result, string, error)
	// TASK-043 正式复核（每次正式尝试仅一次）。
	Review(context.Context, scoring.Actor, scoring.ReviewRequest) (scoring.ReviewResult, error)
	// TASK-053 正式重试（新题/维度锁定/矛盾解锁重评）。
	BeginRetry(context.Context, scoring.Actor, scoring.BeginRetryRequest) (scoring.RetryAttempt, error)
}

// Authenticator 由 TASK-010 identity 服务实现。
type Authenticator interface {
	Authenticate(string) (scoring.Actor, error)
}

// New 构建区域绑定的评分 HTTP 处理器。
func New(app Application, authenticator Authenticator, dataRegion string) (http.Handler, error) {
	if app == nil || authenticator == nil || region.ValidateDataRegion(dataRegion) != nil {
		return nil, errors.New("scoring http handler requires application, authenticator and valid data region")
	}
	h := &handler{app: app, authenticator: authenticator, dataRegion: dataRegion}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/projects/{projectId}/rounds/{sequence}/result", h.getRoundResult)
	mux.HandleFunc("GET /v1/projects/{projectId}/rounds/{sequence}/scores", h.listScoreVersions)
	mux.HandleFunc("POST /v1/projects/{projectId}/rounds/{sequence}/review", h.review)
	mux.HandleFunc("POST /v1/projects/{projectId}/rounds/{sequence}/retry", h.startRetry)
	return mux, nil
}

type handler struct {
	app           Application
	authenticator Authenticator
	dataRegion    string
}

func (h *handler) actor(r *http.Request) (scoring.Actor, error) {
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if auth == "" || token == auth {
		return scoring.Actor{}, errors.New("unauthorized")
	}
	actor, err := h.authenticator.Authenticate(token)
	if err != nil {
		return scoring.Actor{}, err
	}
	if actor.DataRegion != h.dataRegion {
		return scoring.Actor{}, errors.New("region_mismatch")
	}
	return actor, nil
}

func (h *handler) getRoundResult(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "缺少有效身份")
		return
	}
	sequence, err := roundSequence(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_parameter", err.Error())
		return
	}
	result, err := h.app.GetLatest(r.Context(), actor, r.PathValue("projectId"), sequence)
	if err != nil {
		writeScoringError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handler) listScoreVersions(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "缺少有效身份")
		return
	}
	sequence, err := roundSequence(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_parameter", err.Error())
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "invalid_parameter", "limit 必须为 1-100")
			return
		}
		limit = parsed
	}
	items, next, err := h.app.ListVersions(
		r.Context(), actor, r.PathValue("projectId"), sequence, limit, r.URL.Query().Get("cursor"),
	)
	if err != nil {
		writeScoringError(w, err)
		return
	}
	if items == nil {
		items = []scoring.Result{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data_region": actor.DataRegion,
		"items":       items,
		"next_cursor": next,
	})
}

// review 处理正式复核请求（202 AsyncTask + 前后对比；409 表示已复核过）。
func (h *handler) review(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "缺少有效身份")
		return
	}
	sequence, err := roundSequence(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_parameter", err.Error())
		return
	}
	idemKey := r.Header.Get("Idempotency-Key")
	if len(idemKey) < 8 || len(idemKey) > 128 {
		writeError(w, http.StatusBadRequest, "invalid_parameter",
			"Idempotency-Key 必填（8-128 字符）")
		return
	}
	var body struct {
		AttemptID string `json:"attempt_id"`
		Scope     string `json:"scope"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "请求体非法")
		return
	}
	reviewResult, err := h.app.Review(r.Context(), actor, scoring.ReviewRequest{
		ProjectID:      r.PathValue("projectId"),
		RoundSequence:  sequence,
		AttemptID:      body.AttemptID,
		Scope:          body.Scope,
		IdempotencyKey: idemKey,
	})
	if err != nil {
		writeReviewError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"task_id":       reviewResult.Review.ScoreID,
		"task_type":     "review",
		"status":        "succeeded",
		"progress_note": "复核完成：产生新 ScoreVersion，前后对比见 review_result",
		"data_region":   actor.DataRegion,
		"review_result": reviewResult,
	})
}

// startRetry 发起正式重试（201 RetryAttempt；409 当前状态不允许重试）。
func (h *handler) startRetry(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "缺少有效身份")
		return
	}
	sequence, err := roundSequence(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_parameter", err.Error())
		return
	}
	idemKey := r.Header.Get("Idempotency-Key")
	if len(idemKey) < 8 || len(idemKey) > 128 {
		writeError(w, http.StatusBadRequest, "invalid_parameter",
			"Idempotency-Key 必填（8-128 字符）")
		return
	}
	attempt, err := h.app.BeginRetry(r.Context(), actor, scoring.BeginRetryRequest{
		ProjectID:      r.PathValue("projectId"),
		RoundSequence:  sequence,
		IdempotencyKey: idemKey,
	})
	if err != nil {
		writeRetryError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, attempt)
}

func decode(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func roundSequence(r *http.Request) (int, error) {
	sequence, err := strconv.Atoi(r.PathValue("sequence"))
	if err != nil || sequence < 1 || sequence > 5 {
		return 0, errors.New("sequence 必须为 1-5")
	}
	return sequence, nil
}

func writeScoringError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, scoring.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "评分结果不存在")
	case errors.Is(err, scoring.ErrInvalidCursor):
		writeError(w, http.StatusBadRequest, "invalid_cursor", "游标非法")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "评分服务错误")
	}
}

func writeReviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, scoring.ErrReviewLimit):
		writeError(w, http.StatusConflict, "state_conflict", "本次正式尝试已复核过（仅一次）")
	case errors.Is(err, scoring.ErrEvidenceMismatch):
		writeError(w, http.StatusConflict, "evidence_mismatch", "冻结证据散列不一致（疑似篡改，触发安全审计）")
	case errors.Is(err, scoring.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "该正式尝试不存在评分结果")
	case errors.Is(err, scoring.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_parameter", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "复核服务错误")
	}
}

func writeRetryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, scoring.ErrStateConflict):
		writeError(w, http.StatusConflict, "state_conflict", "当前轮状态不允许正式重试")
	case errors.Is(err, scoring.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "该轮不存在可重试的正式评分结果")
	case errors.Is(err, scoring.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_parameter", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "重试服务错误")
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}
