package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"miangedan/services/secretref"
)

var (
	// ErrUnavailable is safe for callers to map to an email fallback response.
	ErrUnavailable = errors.New("identity provider unavailable")
	// ErrInvalidCredential means the one-time provider authorization code could
	// not be verified. The credential itself must never be included in the error.
	ErrInvalidCredential = errors.New("identity provider credential invalid")
	// ErrRegionNotAllowed indicates the TASK-007 provider matrix rejected a call.
	ErrRegionNotAllowed = errors.New("identity provider not allowed in data region")
)

// AdapterConfig contains only a client identifier and a *_REF handle. Concrete
// secret values are resolved inside the regional adapter process.
type AdapterConfig struct {
	Provider        string
	DataRegion      string
	ClientID        string
	ClientSecretRef string
}

// Validate enforces the regional provider matrix and reference-only secrets.
func (c AdapterConfig) Validate() error {
	if c.Provider == Email {
		return errors.New("email OTP uses services/notify, not an OAuth adapter")
	}
	allowed, err := RegionProviders(c.DataRegion)
	if err != nil {
		return err
	}
	if !providerAllowed(allowed, c.Provider) {
		return fmt.Errorf("%w: provider %s", ErrRegionNotAllowed, c.Provider)
	}
	if strings.TrimSpace(c.ClientID) == "" {
		return errors.New("identity provider client ID is required")
	}
	if err := secretref.ValidateRefName(c.ClientSecretRef); err != nil {
		return err
	}
	return nil
}

// VerifyRequest contains the transient authorization material passed to a
// vendor-neutral adapter. Implementations must not persist or log it.
type VerifyRequest struct {
	AuthorizationCode string
	RedirectURI       string
	DataRegion        string
}

// VerifiedSubject is the minimum verified provider response used by TASK-010.
type VerifiedSubject struct {
	Subject    string
	VerifiedAt time.Time
}

// Adapter verifies one provider's authorization code.
type Adapter interface {
	Verify(context.Context, VerifyRequest) (VerifiedSubject, error)
}

// Registry routes OAuth verification through configured provider adapters.
// Region admission always runs before an adapter can see the authorization code.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry rejects unknown/email/nil adapters but permits a partial set so
// a single provider outage does not disable email or another regional provider.
func NewRegistry(adapters map[string]Adapter) (*Registry, error) {
	registry := &Registry{adapters: make(map[string]Adapter, len(adapters))}
	for name, adapter := range adapters {
		if name == Email || (name != Google && name != Apple && name != WeChat) {
			return nil, fmt.Errorf("unsupported OAuth adapter %q", name)
		}
		if adapter == nil {
			return nil, fmt.Errorf("OAuth adapter %q is nil", name)
		}
		registry.adapters[name] = adapter
	}
	return registry, nil
}

// Verify enforces region admission and returns a sanitized provider result.
func (r *Registry) Verify(ctx context.Context, providerName string, request VerifyRequest) (VerifiedSubject, error) {
	if ctx == nil {
		return VerifiedSubject{}, errors.New("provider verification context is nil")
	}
	allowed, err := RegionProviders(request.DataRegion)
	if err != nil {
		return VerifiedSubject{}, err
	}
	if providerName == Email || !providerAllowed(allowed, providerName) {
		return VerifiedSubject{}, ErrRegionNotAllowed
	}
	adapter, ok := r.adapters[providerName]
	if !ok {
		return VerifiedSubject{}, ErrUnavailable
	}
	if strings.TrimSpace(request.AuthorizationCode) == "" {
		return VerifiedSubject{}, ErrInvalidCredential
	}
	verified, err := adapter.Verify(ctx, request)
	if err != nil {
		return VerifiedSubject{}, err
	}
	if strings.TrimSpace(verified.Subject) == "" || verified.VerifiedAt.IsZero() {
		return VerifiedSubject{}, ErrInvalidCredential
	}
	return verified, nil
}

func providerAllowed(allowed []string, candidate string) bool {
	for _, item := range allowed {
		if item == candidate {
			return true
		}
	}
	return false
}
