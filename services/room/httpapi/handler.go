// Package httpapi exposes TASK-020 under the /v1/sessions and project round session prefixes.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"miangedan/services/identity"
	"miangedan/services/project"
	"miangedan/services/region"
	"miangedan/services/room"
)

const maxRequestBodyBytes int64 = 64 << 10

// Application 为传输无关的会话应用接口（openapi sessions 标签）。
type Application interface {
	CreateSession(context.Context, project.Actor, room.CreateSessionInput, string) (room.SessionCreated, error)
	GetSession(context.Context, project.Actor, string) (room.Session, error)
	EndSession(context.Context, project.Actor, string, bool, string) (room.Session, error)
	ReconnectSession(context.Context, project.Actor, string, string, int, string) (room.SessionCreated, error)
	DeviceTransferSession(context.Context, project.Actor, string, string, bool, string) (room.SessionCreated, error)
}

// Authenticator 由 TASK-010 identity 服务实现。
type Authenticator interface {
	Authenticate(string) (identity.Claims, error)
}

// New 构建区域绑定的会话 HTTP 处理器。
func New(app Application, authenticator Authenticator, dataRegion string) (http.Handler, error) {
	if app == nil || authenticator == nil || region.ValidateDataRegion(dataRegion) != nil {
		return nil, errors.New("room http handler requires application, authenticator and valid data region")
	}
	h := &handler{app: app, authenticator: authenticator, dataRegion: dataRegion}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/projects/{projectId}/rounds/{sequence}/session", h.createSession)
	mux.HandleFunc("GET /v1/sessions/{sessionId}", h.getSession)
	mux.HandleFunc("POST /v1/sessions/{sessionId}/end", h.endSession)
	mux.HandleFunc("POST /v1/sessions/{sessionId}/reconnect", h.reconnectSession)
	mux.HandleFunc("POST /v1/sessions/{sessionId}/device-transfer", h.transferDevice)
	return mux, nil
}

type handler struct {
	app           Application
	authenticator Authenticator
	dataRegion    string
}

var errRegionMismatch = errors.New("region_mismatch")

func (h *handler) actor(r *http.Request) (project.Actor, error) {
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if auth == "" || token == auth {
		return project.Actor{}, errors.New("unauthorized")
	}
	claims, err := h.authenticator.Authenticate(token)
	if err != nil {
		return project.Actor{}, err
	}
	if claims.DataRegion != h.dataRegion {
		return project.Actor{}, errRegionMismatch
	}
	return project.Actor{UserID: claims.UserID, DataRegion: claims.DataRegion}, nil
}

func (h *handler) decode(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must be a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}

func mapError(err error) (int, string, string) {
	switch {
	case errors.Is(err, room.ErrInvalidInput):
		return http.StatusUnprocessableEntity, "invalid_input", err.Error()
	case errors.Is(err, room.ErrStateConflict):
		return http.StatusConflict, "state_conflict", err.Error()
	case errors.Is(err, room.ErrReconnectExpired):
		return http.StatusGone, "reconnect_expired", err.Error()
	case errors.Is(err, room.ErrEntitlementMissing):
		return http.StatusPaymentRequired, "insufficient_entitlement", err.Error()
	case errors.Is(err, room.ErrNotFound):
		return http.StatusNotFound, "not_found", err.Error()
	case errors.Is(err, errRegionMismatch):
		return http.StatusForbidden, "region_mismatch", "跨区请求被拒（ADR-0005）"
	default:
		return http.StatusUnauthorized, "unauthorized", err.Error()
	}
}

type sessionCreatedJSON struct {
	SessionID          string    `json:"session_id"`
	RoomURL            string    `json:"room_url"`
	RoomToken          string    `json:"room_token"`
	RoomTokenExpiresAt time.Time `json:"room_token_expires_at"`
	DataRegion         string    `json:"data_region"`
}

func toSessionCreatedJSON(s room.SessionCreated) sessionCreatedJSON {
	return sessionCreatedJSON{
		SessionID:          s.SessionID,
		RoomURL:            s.RoomURL,
		RoomToken:          s.RoomToken,
		RoomTokenExpiresAt: s.RoomTokenExpiresAt,
		DataRegion:         s.DataRegion,
	}
}

type sessionJSON struct {
	SessionID       string  `json:"session_id"`
	ProjectID       string  `json:"project_id"`
	RoundSequence   int     `json:"round_sequence"`
	AttemptID       *string `json:"attempt_id"`
	Kind            string  `json:"kind"`
	RoomStatus      string  `json:"room_status"`
	BillableSeconds int     `json:"billable_seconds"`
	ActiveDeviceID  *string `json:"active_device_id"`
	DataRegion      string  `json:"data_region"`
	CreatedAt       string  `json:"created_at"`
}

func toSessionJSON(s room.Session) sessionJSON {
	out := sessionJSON{
		SessionID:       s.SessionID,
		ProjectID:       s.ProjectID,
		RoundSequence:   s.RoundSequence,
		Kind:            string(s.Kind),
		RoomStatus:      string(s.RoomStatus),
		BillableSeconds: s.BillableSeconds,
		DataRegion:      s.DataRegion,
		CreatedAt:       s.CreatedAt.UTC().Format(time.RFC3339),
	}
	if s.AttemptID != "" {
		out.AttemptID = &s.AttemptID
	}
	if s.ActiveDeviceID != "" {
		out.ActiveDeviceID = &s.ActiveDeviceID
	}
	return out
}

func (h *handler) createSession(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	sequence, err := strconv.Atoi(r.PathValue("sequence"))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", "sequence 必须为整数")
		return
	}
	var req struct {
		Kind      string  `json:"kind"`
		AttemptID *string `json:"attempt_id"`
		DeviceID  string  `json:"device_id"`
	}
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	attemptID := ""
	if req.AttemptID != nil {
		attemptID = *req.AttemptID
	}
	result, err := h.app.CreateSession(r.Context(), actor, room.CreateSessionInput{
		ProjectID:     r.PathValue("projectId"),
		RoundSequence: sequence,
		Kind:          room.SessionKind(req.Kind),
		AttemptID:     attemptID,
		DeviceID:      req.DeviceID,
	}, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusCreated, toSessionCreatedJSON(result))
}

func (h *handler) getSession(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	sess, err := h.app.GetSession(r.Context(), actor, r.PathValue("sessionId"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, toSessionJSON(sess))
}

func (h *handler) endSession(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	sess, err := h.app.EndSession(r.Context(), actor, r.PathValue("sessionId"), req.Confirm, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, toSessionJSON(sess))
}

func (h *handler) reconnectSession(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req struct {
		DeviceID             string `json:"device_id"`
		LastConfirmedRoomSeq int    `json:"last_confirmed_room_seq"`
	}
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	result, err := h.app.ReconnectSession(r.Context(), actor, r.PathValue("sessionId"), req.DeviceID, req.LastConfirmedRoomSeq, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, toSessionCreatedJSON(result))
}

func (h *handler) transferDevice(w http.ResponseWriter, r *http.Request) {
	actor, err := h.actor(r)
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	var req struct {
		NewDeviceID string `json:"new_device_id"`
		Confirm     bool   `json:"confirm"`
	}
	if err := h.decode(r, &req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid_input", err.Error())
		return
	}
	result, err := h.app.DeviceTransferSession(r.Context(), actor, r.PathValue("sessionId"), req.NewDeviceID, req.Confirm, r.Header.Get("Idempotency-Key"))
	if err != nil {
		status, code, msg := mapError(err)
		writeError(w, status, code, msg)
		return
	}
	writeJSON(w, http.StatusOK, toSessionCreatedJSON(result))
}
