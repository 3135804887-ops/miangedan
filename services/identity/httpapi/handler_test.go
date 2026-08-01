package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miangedan/services/identity"
)

func TestEmailChallengeContractAndRegionIsolation(t *testing.T) {
	t.Parallel()
	called := 0
	app := stubApplication{
		requestEmailChallenge: func(_ context.Context, input identity.RequestEmailChallengeInput, key string) (identity.VerificationChallenge, error) {
			called++
			if input.Email != "synthetic.identity@example.com" || input.DataRegion != "intl" || key != "idem-email-0001" {
				t.Fatalf("unexpected application input: %+v key=%q", input, key)
			}
			return identity.VerificationChallenge{
				ChallengeID:    "00000000-0000-4000-8000-000000000001",
				ExpiresAt:      time.Date(2026, 8, 1, 4, 10, 0, 0, time.UTC),
				DeliveryStatus: "accepted",
				DataRegion:     "intl",
			}, nil
		},
	}
	h := mustHandler(t, app, "intl")

	response := performJSON(h, http.MethodPost, "/v1/identity/email/challenges", `{"email":"synthetic.identity@example.com","data_region":"intl"}`, map[string]string{
		"Idempotency-Key": "idem-email-0001",
	})
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("identity responses must disable caching")
	}

	crossRegion := performJSON(h, http.MethodPost, "/v1/identity/email/challenges", `{"email":"synthetic.identity@example.com","data_region":"cn"}`, map[string]string{
		"Idempotency-Key": "idem-email-0002",
	})
	assertErrorCode(t, crossRegion, http.StatusConflict, identity.CodeRegionMismatch)
	if called != 1 {
		t.Fatalf("application calls = %d, cross-region request must fail before use case", called)
	}

	unknownField := performJSON(h, http.MethodPost, "/v1/identity/email/challenges", `{"email":"synthetic.identity@example.com","data_region":"intl","raw_token":"never"}`, map[string]string{
		"Idempotency-Key": "idem-email-0003",
	})
	assertErrorCode(t, unknownField, http.StatusBadRequest, identity.CodeValidationFailed)
	if strings.Contains(unknownField.Body.String(), "never") {
		t.Fatal("validation error echoed a rejected field value")
	}
}

func TestSessionRegistrationMapping(t *testing.T) {
	t.Parallel()
	acceptedAt := "2026-08-01T04:00:00Z"
	app := stubApplication{
		createSession: func(_ context.Context, input identity.CreateSessionInput, key string) (identity.Session, error) {
			if key != "idem-session-0001" || input.DataRegion != "intl" || input.ProofToken != "synthetic-proof-token-with-more-than-32-characters" {
				t.Fatalf("unexpected session input: %+v key=%q", input, key)
			}
			if input.Registration == nil || input.Registration.Evidence.TermsVersion != "terms-v1" ||
				input.Registration.Evidence.Context.Surface != "pwa" || input.Registration.Evidence.AcceptedAt.Format(time.RFC3339) != acceptedAt {
				t.Fatalf("registration mapping mismatch: %+v", input.Registration)
			}
			return identity.Session{SessionID: "00000000-0000-4000-8000-000000000010", TokenType: "Bearer", DataRegion: "intl"}, nil
		},
	}
	h := mustHandler(t, app, "intl")
	body := `{
		"proof_token":"synthetic-proof-token-with-more-than-32-characters",
		"data_region":"intl",
		"registration":{
			"ui_language":"zh-CN",
			"age_status":"adult",
			"terms_version":"terms-v1",
			"privacy_version":"privacy-v1",
			"data_processing_version":"processing-v1",
			"accepted_at":"2026-08-01T04:00:00Z",
			"acceptance_context":{"ui_language":"zh-CN","surface":"pwa"}
		}
	}`
	response := performJSON(h, http.MethodPost, "/v1/identity/sessions", body, map[string]string{
		"Idempotency-Key": "idem-session-0001",
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestBindingRequiresBearerAndDualProof(t *testing.T) {
	t.Parallel()
	bindCalls := 0
	app := stubApplication{
		authenticate: func(token string) (identity.Claims, error) {
			if token == "cross-region-token" {
				return identity.Claims{UserID: "synthetic-user", SessionID: "session", DataRegion: "cn"}, nil
			}
			if token != "synthetic-access-token" {
				return identity.Claims{}, errors.New("invalid access token")
			}
			return identity.Claims{UserID: "synthetic-user", SessionID: "session", DataRegion: "intl"}, nil
		},
		bindIdentity: func(_ context.Context, claims identity.Claims, input identity.BindIdentityInput, key string) (identity.Binding, error) {
			bindCalls++
			if claims.UserID != "synthetic-user" || key != "idem-binding-0001" ||
				input.SourceProofToken != "synthetic-source-proof" || input.TargetProofToken != "synthetic-target-proof" {
				t.Fatalf("unexpected binding input: claims=%+v input=%+v key=%q", claims, input, key)
			}
			return identity.Binding{
				IdentityID: "00000000-0000-4000-8000-000000000020",
				Provider:   identity.ProviderApple,
				VerifiedAt: time.Date(2026, 8, 1, 4, 0, 0, 0, time.UTC),
				DataRegion: "intl",
			}, nil
		},
	}
	h := mustHandler(t, app, "intl")
	body := `{"source_proof_token":"synthetic-source-proof","target_proof_token":"synthetic-target-proof"}`

	unauthorized := performJSON(h, http.MethodPost, "/v1/identity/bindings", body, map[string]string{
		"Idempotency-Key": "idem-binding-0001",
	})
	assertErrorCode(t, unauthorized, http.StatusUnauthorized, identity.CodeUnauthorized)

	success := performJSON(h, http.MethodPost, "/v1/identity/bindings", body, map[string]string{
		"Authorization":   "Bearer synthetic-access-token",
		"Idempotency-Key": "idem-binding-0001",
	})
	if success.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", success.Code, success.Body.String())
	}

	crossRegion := performJSON(h, http.MethodPost, "/v1/identity/bindings", body, map[string]string{
		"Authorization":   "Bearer cross-region-token",
		"Idempotency-Key": "idem-binding-0002",
	})
	assertErrorCode(t, crossRegion, http.StatusConflict, identity.CodeRegionMismatch)
	if bindCalls != 1 {
		t.Fatalf("binding calls = %d, unauthorized/cross-region requests must not reach use case", bindCalls)
	}
}

func TestProviderFailureDoesNotEchoAuthorizationCode(t *testing.T) {
	t.Parallel()
	const oneTimeCode = "synthetic-oauth-code-that-must-never-be-echoed"
	app := stubApplication{
		verifyOAuth: func(_ context.Context, input identity.VerifyOAuthInput, _ string) (identity.VerificationProof, error) {
			if input.AuthorizationCode != oneTimeCode {
				t.Fatalf("authorization code not delivered transiently to adapter use case")
			}
			return identity.VerificationProof{}, &identity.DomainError{
				Code:    identity.CodeProviderUnavailable,
				Message: "第三方登录暂时不可用，未创建或修改账户。请稍后重试或改用邮箱验证码；不计费且不影响评分。",
				Details: map[string]any{"email_fallback_available": true},
			}
		},
	}
	h := mustHandler(t, app, "eu")
	response := performJSON(h, http.MethodPost, "/v1/identity/oauth/google/verify", `{"authorization_code":"`+oneTimeCode+`","data_region":"eu"}`, map[string]string{
		"Idempotency-Key": "idem-oauth-0001",
	})
	assertErrorCode(t, response, http.StatusServiceUnavailable, identity.CodeProviderUnavailable)
	if strings.Contains(response.Body.String(), oneTimeCode) {
		t.Fatal("provider error echoed the transient authorization code")
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if fallback, ok := envelope.Error.Details["email_fallback_available"].(bool); !ok || !fallback {
		t.Fatalf("email fallback detail missing: %+v", envelope.Error.Details)
	}
}

type stubApplication struct {
	requestEmailChallenge func(context.Context, identity.RequestEmailChallengeInput, string) (identity.VerificationChallenge, error)
	verifyEmailChallenge  func(context.Context, identity.VerifyEmailChallengeInput, string) (identity.VerificationProof, error)
	verifyOAuth           func(context.Context, identity.VerifyOAuthInput, string) (identity.VerificationProof, error)
	createSession         func(context.Context, identity.CreateSessionInput, string) (identity.Session, error)
	refreshSession        func(context.Context, identity.RefreshSessionInput, string) (identity.Session, error)
	authenticate          func(string) (identity.Claims, error)
	getAccount            func(context.Context, identity.Claims) (identity.Account, error)
	updateAccount         func(context.Context, identity.Claims, identity.UpdateAccountInput, string) (identity.Account, error)
	bindIdentity          func(context.Context, identity.Claims, identity.BindIdentityInput, string) (identity.Binding, error)
}

func (s stubApplication) RequestEmailChallenge(ctx context.Context, input identity.RequestEmailChallengeInput, key string) (identity.VerificationChallenge, error) {
	return s.requestEmailChallenge(ctx, input, key)
}

func (s stubApplication) VerifyEmailChallenge(ctx context.Context, input identity.VerifyEmailChallengeInput, key string) (identity.VerificationProof, error) {
	return s.verifyEmailChallenge(ctx, input, key)
}

func (s stubApplication) VerifyOAuth(ctx context.Context, input identity.VerifyOAuthInput, key string) (identity.VerificationProof, error) {
	return s.verifyOAuth(ctx, input, key)
}

func (s stubApplication) CreateSession(ctx context.Context, input identity.CreateSessionInput, key string) (identity.Session, error) {
	return s.createSession(ctx, input, key)
}

func (s stubApplication) RefreshSession(ctx context.Context, input identity.RefreshSessionInput, key string) (identity.Session, error) {
	return s.refreshSession(ctx, input, key)
}

func (s stubApplication) Authenticate(token string) (identity.Claims, error) {
	return s.authenticate(token)
}

func (s stubApplication) GetAccount(ctx context.Context, claims identity.Claims) (identity.Account, error) {
	return s.getAccount(ctx, claims)
}

func (s stubApplication) UpdateAccount(ctx context.Context, claims identity.Claims, input identity.UpdateAccountInput, key string) (identity.Account, error) {
	return s.updateAccount(ctx, claims, input, key)
}

func (s stubApplication) BindIdentity(ctx context.Context, claims identity.Claims, input identity.BindIdentityInput, key string) (identity.Binding, error) {
	return s.bindIdentity(ctx, claims, input, key)
}

func mustHandler(t *testing.T, app Application, dataRegion string) http.Handler {
	t.Helper()
	h, err := New(app, dataRegion)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func performJSON(h http.Handler, method, target, body string, headers map[string]string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	return response
}

func assertErrorCode(t *testing.T, response *httptest.ResponseRecorder, status int, code identity.ErrorCode) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, status, response.Body.String())
	}
	var envelope errorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.Code != code || envelope.Error.TraceID == "" || envelope.Error.DataRegion == "" {
		t.Fatalf("unexpected error response: %+v", envelope.Error)
	}
}
