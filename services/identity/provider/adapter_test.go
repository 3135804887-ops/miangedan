package provider

import (
	"context"
	"errors"
	"testing"
	"time"
)

type syntheticAdapter struct {
	calls int
	err   error
}

func (a *syntheticAdapter) Verify(context.Context, VerifyRequest) (VerifiedSubject, error) {
	a.calls++
	if a.err != nil {
		return VerifiedSubject{}, a.err
	}
	return VerifiedSubject{Subject: "synthetic-provider-subject", VerifiedAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)}, nil
}

// TASK-010 / SEC-012: OAuth configuration uses region admission and *_REF only.
func TestAdapterConfigValidation(t *testing.T) {
	//nolint:gosec // Synthetic *_REF handles exercise reference validation; they are not credentials.
	valid := AdapterConfig{
		Provider: Google, DataRegion: "eu", ClientID: "synthetic-client-id",
		ClientSecretRef: "OAUTH_GOOGLE_CLIENT_SECRET_REF",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid adapter config rejected: %v", err)
	}
	for name, config := range map[string]AdapterConfig{
		//nolint:gosec // Synthetic *_REF handles exercise reference validation; they are not credentials.
		"cross region": {Provider: Google, DataRegion: "cn", ClientID: "synthetic", ClientSecretRef: "OAUTH_GOOGLE_CLIENT_SECRET_REF"},
		//nolint:gosec // This deliberately invalid synthetic name verifies raw-secret rejection.
		"raw secret name": {Provider: Google, DataRegion: "eu", ClientID: "synthetic", ClientSecretRef: "OAUTH_GOOGLE_CLIENT_SECRET"},
		//nolint:gosec // Synthetic *_REF handles exercise reference validation; they are not credentials.
		"email adapter": {Provider: Email, DataRegion: "eu", ClientID: "synthetic", ClientSecretRef: "EMAIL_SECRET_REF"},
	} {
		if err := config.Validate(); err == nil {
			t.Fatalf("%s config must be rejected", name)
		}
	}
}

// TASK-010 / FR-027: region rejection occurs before a provider sees credentials.
func TestRegistryRegionAdmissionAndUnavailable(t *testing.T) {
	google := &syntheticAdapter{}
	registry, err := NewRegistry(map[string]Adapter{Google: google})
	if err != nil {
		t.Fatal(err)
	}
	verified, err := registry.Verify(context.Background(), Google, VerifyRequest{
		AuthorizationCode: "synthetic-code", DataRegion: "intl",
	})
	if err != nil || verified.Subject == "" || google.calls != 1 {
		t.Fatalf("valid verification failed: %+v %v", verified, err)
	}
	_, err = registry.Verify(context.Background(), Google, VerifyRequest{
		AuthorizationCode: "synthetic-code", DataRegion: "cn",
	})
	if !errors.Is(err, ErrRegionNotAllowed) || google.calls != 1 {
		t.Fatalf("region rejection must occur before adapter: calls=%d err=%v", google.calls, err)
	}
	_, err = registry.Verify(context.Background(), Apple, VerifyRequest{
		AuthorizationCode: "synthetic-code", DataRegion: "eu",
	})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing regional adapter must be unavailable: %v", err)
	}
}
