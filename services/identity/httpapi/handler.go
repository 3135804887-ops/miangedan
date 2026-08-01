// Package httpapi exposes the TASK-010 identity use cases under the
// /v1/identity prefix declared in docs/api/openapi.yaml.
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

	"miangedan/services/identity"
	"miangedan/services/region"
)

const maxRequestBodyBytes int64 = 64 << 10

// Application is the transport-independent identity application surface.
// Keeping the HTTP layer behind this interface makes route behavior testable
// without vendor SDKs, a database, or access to secret material.
type Application interface {
	RequestEmailChallenge(context.Context, identity.RequestEmailChallengeInput, string) (identity.VerificationChallenge, error)
	VerifyEmailChallenge(context.Context, identity.VerifyEmailChallengeInput, string) (identity.VerificationProof, error)
	VerifyOAuth(context.Context, identity.VerifyOAuthInput, string) (identity.VerificationProof, error)
	CreateSession(context.Context, identity.CreateSessionInput, string) (identity.Session, error)
	RefreshSession(context.Context, identity.RefreshSessionInput, string) (identity.Session, error)
	Authenticate(string) (identity.Claims, error)
	GetAccount(context.Context, identity.Claims) (identity.Account, error)
	UpdateAccount(context.Context, identity.Claims, identity.UpdateAccountInput, string) (identity.Account, error)
	BindIdentity(context.Context, identity.Claims, identity.BindIdentityInput, string) (identity.IdentityBinding, error)
}

// New builds a region-pinned handler. A request cannot select another region;
// this prevents a control-plane deployment from becoming a cross-region data
// path even when a body or token contains a different data_region.
func New(app Application, dataRegion string) (http.Handler, error) {
	if app == nil || region.ValidateDataRegion(dataRegion) != nil {
		return nil, errors.New("identity http handler requires an application and valid data region")
	}
	h := &handler{app: app, dataRegion: dataRegion}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/identity/email/challenges", h.requestEmailChallenge)
	mux.HandleFunc("POST /v1/identity/email/challenges/{challengeId}/verify", h.verifyEmailChallenge)
	mux.HandleFunc("POST /v1/identity/oauth/{provider}/verify", h.verifyOAuth)
	mux.HandleFunc("POST /v1/identity/sessions", h.createSession)
	mux.HandleFunc("POST /v1/identity/sessions/refresh", h.refreshSession)
	mux.HandleFunc("GET /v1/identity/account", h.getAccount)
	mux.HandleFunc("PATCH /v1/identity/account", h.updateAccount)
	mux.HandleFunc("POST /v1/identity/bindings", h.bindIdentity)
	return mux, nil
}

type handler struct {
	app        Application
	dataRegion string
}

func (h *handler) requestEmailChallenge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email      string `json:"email"`
		DataRegion string `json:"data_region"`
	}
	if !h.decodeRegionBody(w, r, &body, &body.DataRegion) {
		return
	}
	result, err := h.app.RequestEmailChallenge(r.Context(), identity.RequestEmailChallengeInput(body), r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (h *handler) verifyEmailChallenge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code       string `json:"code"`
		DataRegion string `json:"data_region"`
	}
	if !h.decodeRegionBody(w, r, &body, &body.DataRegion) {
		return
	}
	result, err := h.app.VerifyEmailChallenge(r.Context(), identity.VerifyEmailChallengeInput{
		ChallengeID: r.PathValue("challengeId"),
		Code:        body.Code,
		DataRegion:  body.DataRegion,
	}, r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handler) verifyOAuth(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AuthorizationCode string `json:"authorization_code"`
		RedirectURI       string `json:"redirect_uri"`
		DataRegion        string `json:"data_region"`
	}
	if !h.decodeRegionBody(w, r, &body, &body.DataRegion) {
		return
	}
	result, err := h.app.VerifyOAuth(r.Context(), identity.VerifyOAuthInput{
		Provider:          identity.Provider(r.PathValue("provider")),
		AuthorizationCode: body.AuthorizationCode,
		RedirectURI:       body.RedirectURI,
		DataRegion:        body.DataRegion,
	}, r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handler) createSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProofToken   string               `json:"proof_token"`
		DataRegion   string               `json:"data_region"`
		Registration *registrationRequest `json:"registration"`
	}
	if !h.decodeRegionBody(w, r, &body, &body.DataRegion) {
		return
	}
	var registration *identity.Registration
	if body.Registration != nil {
		registration = body.Registration.domain()
	}
	result, err := h.app.CreateSession(r.Context(), identity.CreateSessionInput{
		ProofToken:   body.ProofToken,
		DataRegion:   body.DataRegion,
		Registration: registration,
	}, r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *handler) refreshSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
		DataRegion   string `json:"data_region"`
	}
	if !h.decodeRegionBody(w, r, &body, &body.DataRegion) {
		return
	}
	result, err := h.app.RefreshSession(r.Context(), identity.RefreshSessionInput(body), r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handler) getAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	result, err := h.app.GetAccount(r.Context(), claims)
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handler) updateAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var body struct {
		UILanguage  *identity.Language `json:"ui_language"`
		DisplayName json.RawMessage    `json:"display_name"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.writeError(w, validationError())
		return
	}
	input := identity.UpdateAccountInput{UILanguage: body.UILanguage}
	if body.DisplayName != nil {
		if string(body.DisplayName) == "null" {
			input.ClearDisplayName = true
		} else if err := json.Unmarshal(body.DisplayName, &input.DisplayName); err != nil {
			h.writeError(w, validationError())
			return
		}
	}
	result, err := h.app.UpdateAccount(r.Context(), claims, input, r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *handler) bindIdentity(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var body struct {
		SourceProofToken string `json:"source_proof_token"`
		TargetProofToken string `json:"target_proof_token"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.writeError(w, validationError())
		return
	}
	result, err := h.app.BindIdentity(r.Context(), claims, identity.BindIdentityInput(body), r.Header.Get("Idempotency-Key"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

type registrationRequest struct {
	UILanguage            identity.Language  `json:"ui_language"`
	AgeStatus             identity.AgeStatus `json:"age_status"`
	TermsVersion          string             `json:"terms_version"`
	PrivacyVersion        string             `json:"privacy_version"`
	DataProcessingVersion string             `json:"data_processing_version"`
	AcceptedAt            time.Time          `json:"accepted_at"`
	AcceptanceContext     struct {
		UILanguage identity.Language `json:"ui_language"`
		Surface    string            `json:"surface"`
	} `json:"acceptance_context"`
}

func (r registrationRequest) domain() *identity.Registration {
	return &identity.Registration{
		UILanguage: r.UILanguage,
		AgeStatus:  r.AgeStatus,
		Evidence: identity.RegistrationEvidence{
			TermsVersion:          r.TermsVersion,
			PrivacyVersion:        r.PrivacyVersion,
			DataProcessingVersion: r.DataProcessingVersion,
			AcceptedAt:            r.AcceptedAt,
			Context: identity.AcceptanceContext{
				UILanguage: r.AcceptanceContext.UILanguage,
				Surface:    r.AcceptanceContext.Surface,
			},
		},
	}
}

func (h *handler) decodeRegionBody(w http.ResponseWriter, r *http.Request, target any, bodyRegion *string) bool {
	if err := decodeJSON(w, r, target); err != nil {
		h.writeError(w, validationError())
		return false
	}
	if bodyRegion == nil || *bodyRegion != h.dataRegion {
		h.writeError(w, regionMismatchError())
		return false
	}
	return true
}

func (h *handler) authenticate(w http.ResponseWriter, r *http.Request) (identity.Claims, bool) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		h.writeError(w, unauthorizedError())
		return identity.Claims{}, false
	}
	claims, err := h.app.Authenticate(parts[1])
	if err != nil {
		h.writeError(w, err)
		return identity.Claims{}, false
	}
	if claims.DataRegion != h.dataRegion {
		h.writeError(w, regionMismatchError())
		return identity.Claims{}, false
	}
	return claims, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code       identity.ErrorCode `json:"code"`
	Message    string             `json:"message"`
	Details    map[string]any     `json:"details,omitempty"`
	TraceID    string             `json:"trace_id"`
	DataRegion string             `json:"data_region"`
}

func (h *handler) writeError(w http.ResponseWriter, err error) {
	domain := identity.AsDomainError(err)
	writeJSON(w, statusFor(domain.Code), errorEnvelope{Error: errorBody{
		Code:       domain.Code,
		Message:    domain.Message,
		Details:    domain.Details,
		TraceID:    newTraceID(),
		DataRegion: h.dataRegion,
	}})
}

func statusFor(code identity.ErrorCode) int {
	switch code {
	case identity.CodeValidationFailed:
		return http.StatusBadRequest
	case identity.CodeUnauthorized, identity.CodeVerificationInvalid, identity.CodeVerificationExpired:
		return http.StatusUnauthorized
	case identity.CodeForbidden, identity.CodeRiskVerificationRequired:
		return http.StatusForbidden
	case identity.CodeNotFound:
		return http.StatusNotFound
	case identity.CodeConflict, identity.CodeIdempotencyConflict, identity.CodeIdentityConflict, identity.CodeRegionMismatch:
		return http.StatusConflict
	case identity.CodeRateLimited:
		return http.StatusTooManyRequests
	case identity.CodeProviderUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func newTraceID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "identity-trace-unavailable"
	}
	return hex.EncodeToString(value[:])
}

func validationError() error {
	return &identity.DomainError{
		Code:    identity.CodeValidationFailed,
		Message: "请求未受理，未创建或修改身份数据。请修正输入后重试；不计费且不影响评分。",
	}
}

func unauthorizedError() error {
	return &identity.DomainError{
		Code:    identity.CodeUnauthorized,
		Message: "登录凭证无效或已过期，账户数据保持不变。请重新登录后重试；不计费且不影响评分。",
	}
}

func regionMismatchError() error {
	return &identity.DomainError{
		Code:    identity.CodeRegionMismatch,
		Message: "请求数据区与服务所属区域不一致，操作已拒绝且没有跨区读取或写入。请从账户所属区域重试；不计费且不影响评分。",
	}
}
