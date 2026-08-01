// Package httpapi exposes TASK-011 under the /v1/consent prefix.
package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"miangedan/services/consent"
	"miangedan/services/identity"
	"miangedan/services/region"
)

const maxRequestBodyBytes int64 = 64 << 10

// Application is the transport-independent consent application surface.
type Application interface {
	List(context.Context, consent.Actor) ([]consent.State, error)
	History(context.Context, consent.Actor, consent.Type) ([]consent.Grant, error)
	Grant(context.Context, consent.Actor, consent.Type, consent.GrantInput, string) (consent.Grant, error)
	Withdraw(context.Context, consent.Actor, consent.Type, consent.WithdrawalInput, string) (consent.Grant, error)
	Decide(context.Context, consent.Actor, consent.AccessRequest) (consent.AccessDecision, error)
}

// Authenticator is satisfied by the TASK-010 identity service.
type Authenticator interface {
	Authenticate(string) (identity.Claims, error)
}

// New builds a region-pinned consent handler using TASK-010 business tokens.
func New(app Application, authenticator Authenticator, dataRegion string) (http.Handler, error) {
	if app == nil || authenticator == nil || region.ValidateDataRegion(dataRegion) != nil {
		return nil, errors.New("consent http handler requires application, identity authenticator and valid data region")
	}
	h := &handler{app: app, authenticator: authenticator, dataRegion: dataRegion}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/consent/grants", h.list)
	mux.HandleFunc("PUT /v1/consent/grants/{consentType}", h.grant)
	mux.HandleFunc("POST /v1/consent/grants/{consentType}/withdrawals", h.withdraw)
	mux.HandleFunc("GET /v1/consent/grants/{consentType}/history", h.history)
	mux.HandleFunc("POST /v1/consent/access-decisions", h.decide)
	return mux, nil
}

type handler struct {
	app           Application
	authenticator Authenticator
	dataRegion    string
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	items, err := h.app.List(r.Context(), actor)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		DataRegion string          `json:"data_region"`
		Items      []consent.State `json:"items"`
	}{actor.DataRegion, items})
}

func (h *handler) history(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	items, err := h.app.History(r.Context(), actor, consent.Type(r.PathValue("consentType")))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		DataRegion string          `json:"data_region"`
		Items      []consent.Grant `json:"items"`
	}{actor.DataRegion, items})
}

func (h *handler) grant(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var body struct {
		Scope     *consent.Scope        `json:"scope"`
		ExpiresAt *time.Time            `json:"expires_at"`
		Evidence  consent.EvidenceInput `json:"evidence"`
	}
	if err := decodeJSON(w, r, &body); err != nil || body.Scope == nil {
		h.writeError(w, consent.NewValidationError())
		return
	}
	result, err := h.app.Grant(
		r.Context(), actor, consent.Type(r.PathValue("consentType")),
		consent.GrantInput{Scope: *body.Scope, ExpiresAt: body.ExpiresAt, Evidence: body.Evidence},
		r.Header.Get("Idempotency-Key"),
	)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *handler) withdraw(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var body struct {
		Scope    *consent.Scope        `json:"scope"`
		Evidence consent.EvidenceInput `json:"evidence"`
	}
	if err := decodeJSON(w, r, &body); err != nil || body.Scope == nil {
		h.writeError(w, consent.NewValidationError())
		return
	}
	result, err := h.app.Withdraw(
		r.Context(), actor, consent.Type(r.PathValue("consentType")),
		consent.WithdrawalInput{Scope: *body.Scope, Evidence: body.Evidence},
		r.Header.Get("Idempotency-Key"),
	)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *handler) decide(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var body struct {
		Type  consent.Type   `json:"consent_type"`
		Scope *consent.Scope `json:"scope"`
	}
	if err := decodeJSON(w, r, &body); err != nil || body.Scope == nil {
		h.writeError(w, consent.NewValidationError())
		return
	}
	result, err := h.app.Decide(r.Context(), actor, consent.AccessRequest{Type: body.Type, Scope: *body.Scope})
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handler) authenticate(w http.ResponseWriter, r *http.Request) (consent.Actor, bool) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		h.writeError(w, consent.NewUnauthorizedError())
		return consent.Actor{}, false
	}
	claims, err := h.authenticator.Authenticate(parts[1])
	if err != nil || claims.UserID == "" || claims.SessionID == "" {
		h.writeError(w, consent.NewUnauthorizedError())
		return consent.Actor{}, false
	}
	if claims.DataRegion != h.dataRegion {
		h.writeError(w, consent.NewRegionMismatchError())
		return consent.Actor{}, false
	}
	return consent.Actor{UserID: claims.UserID, SessionID: claims.SessionID, DataRegion: claims.DataRegion}, true
}

func (h *handler) writeError(w http.ResponseWriter, err error) {
	domain := consent.AsDomainError(err)
	status := http.StatusInternalServerError
	switch domain.Code {
	case consent.CodeUnauthorized:
		status = http.StatusUnauthorized
	case consent.CodeForbidden, consent.CodeRegionMismatch:
		status = http.StatusForbidden
	case consent.CodeNotFound:
		status = http.StatusNotFound
	case consent.CodeConflict, consent.CodeIdempotencyConflict:
		status = http.StatusConflict
	case consent.CodeValidationFailed:
		status = http.StatusUnprocessableEntity
	case consent.CodeInternal:
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, struct {
		Error struct {
			Code       consent.Code   `json:"code"`
			Message    string         `json:"message"`
			Details    map[string]any `json:"details"`
			TraceID    string         `json:"trace_id"`
			DataRegion string         `json:"data_region"`
		} `json:"error"`
	}{Error: struct {
		Code       consent.Code   `json:"code"`
		Message    string         `json:"message"`
		Details    map[string]any `json:"details"`
		TraceID    string         `json:"trace_id"`
		DataRegion string         `json:"data_region"`
	}{domain.Code, domain.Message, domain.Details, newTraceID(), h.dataRegion}})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func newTraceID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "trace-unavailable"
	}
	return hex.EncodeToString(raw[:])
}
