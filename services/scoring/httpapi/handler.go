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
	// 正式复核（TASK-043）：当前为 501 占位，契约见 openapi /review。
	mux.HandleFunc("POST /v1/projects/{projectId}/rounds/{sequence}/review", h.reviewNotImplemented)
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

func (h *handler) reviewNotImplemented(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented",
		"正式复核由 TASK-043 实现（每次正式尝试仅一次）")
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}
