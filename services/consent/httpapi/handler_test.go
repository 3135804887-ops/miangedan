package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"miangedan/services/consent"
	"miangedan/services/identity"
)

// synthetic: true — fixed UUIDs/tokens/timestamps below are non-personal fixtures.
const (
	httpUserID    = "00000000-0000-4000-8000-000000000101"
	httpSessionID = "00000000-0000-4000-8000-000000000102"
	httpAssignID  = "00000000-0000-4000-8000-000000000103"
)

var httpNow = time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)

type authFunc func(string) (identity.Claims, error)

func (fn authFunc) Authenticate(token string) (identity.Claims, error) { return fn(token) }

type stubApplication struct {
	list     func(context.Context, consent.Actor) ([]consent.State, error)
	history  func(context.Context, consent.Actor, consent.Type) ([]consent.Grant, error)
	grant    func(context.Context, consent.Actor, consent.Type, consent.GrantInput, string) (consent.Grant, error)
	withdraw func(context.Context, consent.Actor, consent.Type, consent.WithdrawalInput, string) (consent.Grant, error)
	decide   func(context.Context, consent.Actor, consent.AccessRequest) (consent.AccessDecision, error)
}

func (s stubApplication) List(ctx context.Context, actor consent.Actor) ([]consent.State, error) {
	return s.list(ctx, actor)
}

func (s stubApplication) History(ctx context.Context, actor consent.Actor, consentType consent.Type) ([]consent.Grant, error) {
	return s.history(ctx, actor, consentType)
}

func (s stubApplication) Grant(ctx context.Context, actor consent.Actor, consentType consent.Type, input consent.GrantInput, key string) (consent.Grant, error) {
	return s.grant(ctx, actor, consentType, input, key)
}

func (s stubApplication) Withdraw(ctx context.Context, actor consent.Actor, consentType consent.Type, input consent.WithdrawalInput, key string) (consent.Grant, error) {
	return s.withdraw(ctx, actor, consentType, input, key)
}

func (s stubApplication) Decide(ctx context.Context, actor consent.Actor, request consent.AccessRequest) (consent.AccessDecision, error) {
	return s.decide(ctx, actor, request)
}

func testAuthenticator(token string) (identity.Claims, error) {
	switch token {
	case "synthetic-valid-token":
		return identity.Claims{UserID: httpUserID, SessionID: httpSessionID, DataRegion: "intl", ExpiresAt: httpNow.Add(time.Hour)}, nil
	case "synthetic-cross-region-token":
		return identity.Claims{UserID: httpUserID, SessionID: httpSessionID, DataRegion: "eu", ExpiresAt: httpNow.Add(time.Hour)}, nil
	default:
		return identity.Claims{}, errors.New("synthetic invalid token")
	}
}

// TASK-010 reuse / ADR-0005: consent routes require a valid business token and
// reject a token from another data region before the application is called.
func TestIdentityAuthenticationAndRegionPinning(t *testing.T) {
	listCalls := 0
	app := stubApplication{
		list: func(_ context.Context, actor consent.Actor) ([]consent.State, error) {
			listCalls++
			if actor.UserID != httpUserID || actor.DataRegion != "intl" {
				t.Fatalf("unexpected actor: %+v", actor)
			}
			return []consent.State{}, nil
		},
	}
	handler, err := New(app, authFunc(testAuthenticator), "intl")
	if err != nil {
		t.Fatal(err)
	}
	rows := []struct {
		name       string
		authHeader string
		wantStatus int
		wantCode   string
	}{
		{"missing", "", http.StatusUnauthorized, "unauthorized"},
		{"invalid", "Bearer synthetic-invalid-token", http.StatusUnauthorized, "unauthorized"},
		{"cross region", "Bearer synthetic-cross-region-token", http.StatusForbidden, "region_mismatch"},
		{"valid", "Bearer synthetic-valid-token", http.StatusOK, ""},
		{"case insensitive scheme", "bearer synthetic-valid-token", http.StatusOK, ""},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/v1/consent/grants", nil)
			if row.authHeader != "" {
				request.Header.Set("Authorization", row.authHeader)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != row.wantStatus {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("consent response must disable caching")
			}
			if row.wantCode != "" && !strings.Contains(response.Body.String(), `"code":"`+row.wantCode+`"`) {
				t.Fatalf("missing error code %q: %s", row.wantCode, response.Body.String())
			}
			if strings.Contains(response.Body.String(), "synthetic-invalid-token") || strings.Contains(response.Body.String(), "synthetic-cross-region-token") {
				t.Fatalf("token leaked in error response: %s", response.Body.String())
			}
		})
	}
	if listCalls != 2 {
		t.Fatalf("application must run only for valid regional token, calls=%d", listCalls)
	}
}

// Contract path: grant, withdrawal, history and online decision all remain
// under /v1/consent and carry only typed scope/evidence data.
func TestConsentRouteContractAndStrictJSON(t *testing.T) {
	grantCalls := 0
	withdrawCalls := 0
	decisionCalls := 0
	grantRecord := syntheticHTTPGrant(consent.StatusGranted, 1)
	withdrawRecord := syntheticHTTPGrant(consent.StatusWithdrawn, 2)
	withdrawRecord.GrantID = "00000000-0000-4000-8000-000000000106"
	withdrawRecord.WithdrawnAt = pointerHTTP(httpNow)
	withdrawRecord.SupersedesGrantID = &grantRecord.GrantID
	app := stubApplication{
		list: func(context.Context, consent.Actor) ([]consent.State, error) { return nil, nil },
		history: func(_ context.Context, actor consent.Actor, consentType consent.Type) ([]consent.Grant, error) {
			if actor.DataRegion != "intl" || consentType != consent.TypeOrgSharing {
				t.Fatalf("unexpected history request: %+v %s", actor, consentType)
			}
			return []consent.Grant{grantRecord, withdrawRecord}, nil
		},
		grant: func(_ context.Context, actor consent.Actor, consentType consent.Type, input consent.GrantInput, key string) (consent.Grant, error) {
			grantCalls++
			if actor.UserID != httpUserID || consentType != consent.TypeOrgSharing || key != "idem-http-0001" ||
				input.Scope.AssignmentID == nil || *input.Scope.AssignmentID != httpAssignID || len(input.Scope.DataCategories) != 1 ||
				input.Evidence.CopyVersion != "share-copy-v1" || input.ExpiresAt == nil {
				t.Fatalf("unexpected grant request: actor=%+v type=%s input=%+v key=%q", actor, consentType, input, key)
			}
			return grantRecord, nil
		},
		withdraw: func(_ context.Context, actor consent.Actor, consentType consent.Type, input consent.WithdrawalInput, key string) (consent.Grant, error) {
			withdrawCalls++
			if actor.UserID != httpUserID || consentType != consent.TypeOrgSharing || key != "idem-http-0002" ||
				input.Scope.AssignmentID == nil || input.Evidence.CopyVersion != "share-withdraw-v1" {
				t.Fatalf("unexpected withdrawal request: actor=%+v type=%s input=%+v key=%q", actor, consentType, input, key)
			}
			return withdrawRecord, nil
		},
		decide: func(_ context.Context, actor consent.Actor, request consent.AccessRequest) (consent.AccessDecision, error) {
			decisionCalls++
			if actor.UserID != httpUserID || request.Type != consent.TypeOrgSharing || request.Scope.AssignmentID == nil {
				t.Fatalf("unexpected decision request: actor=%+v request=%+v", actor, request)
			}
			return consent.AccessDecision{
				Allowed: false, Type: consent.TypeOrgSharing, ScopeHash: strings.Repeat("b", 64),
				EffectiveStatus: consent.EffectiveWithdrawn, GrantID: &withdrawRecord.GrantID,
				DecidedAt: httpNow, DataRegion: "intl",
			}, nil
		},
	}
	handler, err := New(app, authFunc(testAuthenticator), "intl")
	if err != nil {
		t.Fatal(err)
	}

	invalid := httptest.NewRequest(http.MethodPut, "/v1/consent/grants/org_sharing", strings.NewReader(`{"scope":{},"evidence":{},"unexpected":true}`))
	invalid.Header.Set("Authorization", "Bearer synthetic-valid-token")
	invalid.Header.Set("Idempotency-Key", "idem-http-bad1")
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusUnprocessableEntity || grantCalls != 0 {
		t.Fatalf("unknown JSON field must fail before application: status=%d calls=%d body=%s", invalidResponse.Code, grantCalls, invalidResponse.Body.String())
	}
	missingScope := httptest.NewRequest(http.MethodPut, "/v1/consent/grants/core_service", strings.NewReader(`{"evidence":{"copy_version":"v1","privacy_policy_version":"v1","presented_at":"2026-08-01T08:59:00Z","ui_context":{"surface":"web","flow":"consent_center","ui_language":"zh-CN"}}}`))
	missingScope.Header.Set("Authorization", "Bearer synthetic-valid-token")
	missingScope.Header.Set("Idempotency-Key", "idem-http-bad2")
	missingScopeResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingScopeResponse, missingScope)
	if missingScopeResponse.Code != http.StatusUnprocessableEntity || grantCalls != 0 {
		t.Fatalf("required scope was not enforced: status=%d calls=%d body=%s", missingScopeResponse.Code, grantCalls, missingScopeResponse.Body.String())
	}

	grantBody := `{"scope":{"assignment_id":"` + httpAssignID + `","data_categories":["radar"]},"expires_at":"2026-08-20T09:00:00Z","evidence":{"copy_version":"share-copy-v1","privacy_policy_version":"privacy-v1","presented_at":"2026-08-01T08:59:00Z","ui_context":{"surface":"web","flow":"assignment_share","ui_language":"zh-CN"}}}`
	grantRequest := httptest.NewRequest(http.MethodPut, "/v1/consent/grants/org_sharing", strings.NewReader(grantBody))
	grantRequest.Header.Set("Authorization", "Bearer synthetic-valid-token")
	grantRequest.Header.Set("Idempotency-Key", "idem-http-0001")
	grantResponse := httptest.NewRecorder()
	handler.ServeHTTP(grantResponse, grantRequest)
	if grantResponse.Code != http.StatusCreated || grantCalls != 1 {
		t.Fatalf("grant route failed: status=%d body=%s", grantResponse.Code, grantResponse.Body.String())
	}
	if strings.Contains(grantResponse.Body.String(), "user_id") || strings.Contains(grantResponse.Body.String(), "request_key") {
		t.Fatalf("internal identity/idempotency data leaked: %s", grantResponse.Body.String())
	}

	withdrawBody := `{"scope":{"assignment_id":"` + httpAssignID + `","data_categories":["radar"]},"evidence":{"copy_version":"share-withdraw-v1","privacy_policy_version":"privacy-v1","presented_at":"2026-08-01T08:59:00Z","ui_context":{"surface":"web","flow":"assignment_share","ui_language":"zh-CN"}}}`
	withdrawRequest := httptest.NewRequest(http.MethodPost, "/v1/consent/grants/org_sharing/withdrawals", strings.NewReader(withdrawBody))
	withdrawRequest.Header.Set("Authorization", "Bearer synthetic-valid-token")
	withdrawRequest.Header.Set("Idempotency-Key", "idem-http-0002")
	withdrawResponse := httptest.NewRecorder()
	handler.ServeHTTP(withdrawResponse, withdrawRequest)
	if withdrawResponse.Code != http.StatusCreated || withdrawCalls != 1 {
		t.Fatalf("withdraw route failed: status=%d body=%s", withdrawResponse.Code, withdrawResponse.Body.String())
	}

	decisionBody := `{"consent_type":"org_sharing","scope":{"assignment_id":"` + httpAssignID + `","data_categories":["radar"]}}`
	decisionRequest := httptest.NewRequest(http.MethodPost, "/v1/consent/access-decisions", strings.NewReader(decisionBody))
	decisionRequest.Header.Set("Authorization", "Bearer synthetic-valid-token")
	decisionResponse := httptest.NewRecorder()
	handler.ServeHTTP(decisionResponse, decisionRequest)
	if decisionResponse.Code != http.StatusOK || decisionCalls != 1 || !strings.Contains(decisionResponse.Body.String(), `"allowed":false`) {
		t.Fatalf("decision route failed: status=%d body=%s", decisionResponse.Code, decisionResponse.Body.String())
	}

	historyRequest := httptest.NewRequest(http.MethodGet, "/v1/consent/grants/org_sharing/history", nil)
	historyRequest.Header.Set("Authorization", "Bearer synthetic-valid-token")
	historyResponse := httptest.NewRecorder()
	handler.ServeHTTP(historyResponse, historyRequest)
	if historyResponse.Code != http.StatusOK || !strings.Contains(historyResponse.Body.String(), `"version":2`) {
		t.Fatalf("history route failed: status=%d body=%s", historyResponse.Code, historyResponse.Body.String())
	}
}

func syntheticHTTPGrant(status consent.Status, version int) consent.Grant {
	expiresAt := httpNow.Add(30 * 24 * time.Hour)
	assignmentID := httpAssignID
	return consent.Grant{
		GrantID: "00000000-0000-4000-8000-000000000105", UserID: httpUserID,
		Type:      consent.TypeOrgSharing,
		Scope:     consent.Scope{AssignmentID: &assignmentID, DataCategories: []consent.DataCategory{consent.DataRadar}},
		ScopeHash: strings.Repeat("b", 64), Status: status, GrantedAt: httpNow,
		ExpiresAt: &expiresAt,
		Evidence: consent.Evidence{
			CopyVersion: "share-copy-v1", PrivacyPolicyVersion: "privacy-v1", PresentedAt: httpNow.Add(-time.Minute),
			UIContext: consent.UIContext{Surface: "web", Flow: "assignment_share", UILanguage: "zh-CN"},
			Action:    "grant", RecordedAt: httpNow, EvidenceHash: strings.Repeat("a", 64),
		},
		Version: version, RecordedAt: httpNow, DataRegion: "intl",
		RequestOperation: "grant", RequestKey: "synthetic-internal-key", RequestHash: strings.Repeat("c", 64),
	}
}

func pointerHTTP[T any](value T) *T { return &value }
