// Package httpapi exposes TASK-055 under the /v1/me/export、/v1/deletion-tasks
// and /v1/projects/{projectId}/report/export prefixes（openapi account/deletion 标签）。
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"miangedan/services/export"
	"miangedan/services/region"
)

// Application 为传输无关的导出/删除应用接口。
type Application interface {
	CreateExport(context.Context, export.Actor, string, string, string) (export.Task, error)
	ExecuteExport(context.Context, export.Actor, string) (export.Task, error)
	GetTask(context.Context, export.Actor, string) (export.Task, error)
	CreateDeletionTask(context.Context, export.Actor, export.DeletionRequest, string) (export.DeletionTask, error)
	ExecuteDeletion(context.Context, export.Actor, string) (export.DeletionTask, error)
	RetryDeletionTask(context.Context, export.Actor, string) (export.DeletionTask, error)
	GetDeletionTask(context.Context, export.Actor, string) (export.DeletionTask, error)
}

// Authenticator 由 TASK-010 identity 服务实现。
type Authenticator interface {
	Authenticate(string) (export.Actor, error)
}

// New 构建区域绑定的导出/删除 HTTP 处理器。
func New(app Application, authenticator Authenticator, dataRegion string) (http.Handler, error) {
	if app == nil || authenticator == nil || region.ValidateDataRegion(dataRegion) != nil {
		return nil, errors.New("export http handler requires application, authenticator and valid data region")
	}
	h := &handler{app: app, authenticator: authenticator, dataRegion: dataRegion}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/me/export", h.requestExport)
	mux.HandleFunc("POST /v1/deletion-tasks", h.createDeletionTask)
	mux.HandleFunc("GET /v1/deletion-tasks/{taskId}", h.getDeletionTask)
	mux.HandleFunc("POST /v1/projects/{projectId}/report/export", h.exportReport)
	return mux, nil
}

type handler struct {
	app           Application
	authenticator Authenticator
	dataRegion    string
}

func (h *handler) actor(r *http.Request) (export.Actor, error) {
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if auth == "" || token == auth {
		return export.Actor{}, errors.New("unauthorized")
	}
	actor, err := h.authenticator.Authenticate(token)
	if err != nil {
		return export.Actor{}, err
	}
	if actor.DataRegion != h.dataRegion {
		return export.Actor{}, errors.New("region_mismatch")
	}
	return actor, nil
}

func (h *handler) requestExport(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "缺少有效身份")
		return
	}
	idemKey := idemKey(r)
	task, err := h.app.CreateExport(r.Context(), actor, "", "account", idemKey)
	if err != nil {
		writeTaskError(w, err)
		return
	}
	task, err = h.app.ExecuteExport(r.Context(), actor, task.TaskID)
	if err != nil {
		writeTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, asyncView(task))
}

func (h *handler) exportReport(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "缺少有效身份")
		return
	}
	task, err := h.app.CreateExport(r.Context(), actor,
		r.PathValue("projectId"), "project", idemKey(r))
	if err != nil {
		writeTaskError(w, err)
		return
	}
	task, err = h.app.ExecuteExport(r.Context(), actor, task.TaskID)
	if err != nil {
		writeTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, asyncView(task))
}

func (h *handler) createDeletionTask(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "缺少有效身份")
		return
	}
	var body struct {
		TargetType string `json:"target_type"`
		TargetID   string `json:"target_id"`
	}
	if err := decode(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "请求体非法")
		return
	}
	task, err := h.app.CreateDeletionTask(r.Context(), actor, export.DeletionRequest{
		TargetType: body.TargetType,
		TargetID:   body.TargetID,
		UserID:     actor.UserID,
	}, idemKey(r))
	if err != nil {
		writeTaskError(w, err)
		return
	}
	task, err = h.app.ExecuteDeletion(r.Context(), actor, task.TaskID)
	if err != nil {
		writeTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, task)
}

func (h *handler) getDeletionTask(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "缺少有效身份")
		return
	}
	task, err := h.app.GetDeletionTask(r.Context(), actor, r.PathValue("taskId"))
	if err != nil {
		writeTaskError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func idemKey(r *http.Request) string {
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		key = "default-" + r.URL.Path
	}
	return key
}

func asyncView(task export.Task) map[string]any {
	note := ""
	if task.ProgressNote != nil {
		note = *task.ProgressNote
	}
	return map[string]any{
		"task_id":       task.TaskID,
		"task_type":     task.TaskType,
		"status":        task.Status,
		"progress_note": note,
		"data_region":   task.DataRegion,
	}
}

func decode(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeTaskError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, export.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "任务不存在")
	case errors.Is(err, export.ErrStateConflict):
		writeError(w, http.StatusConflict, "state_conflict", err.Error())
	case errors.Is(err, export.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_parameter", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "任务服务错误")
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
