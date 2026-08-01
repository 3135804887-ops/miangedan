package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"miangedan/services/identity/provider"
	"miangedan/services/notify"
	"miangedan/services/secretref"
)

// Identity secret reference names are environment/KMS handles, never secret material.
const (
	OTPKeyRefName     = "EMAIL_OTP_PEPPER_REF"
	SubjectKeyRefName = "IDENTITY_SUBJECT_KEY_REF"
	ProofKeyRefName   = "IDENTITY_PROOF_SIGNING_KEY_REF"
	SigningKeyRefName = "IDENTITY_SIGNING_KEY_REF"
	RefreshKeyRefName = "IDENTITY_REFRESH_KEY_REF"
)

// SecretReferences contains names only. A resolver backed by the regional
// secret-management system supplies bytes at runtime.
type SecretReferences struct {
	OTPKeyRef     string
	SubjectKeyRef string
	ProofKeyRef   string
	SigningKeyRef string
	RefreshKeyRef string
}

// DefaultSecretReferences returns the approved *_REF environment/KMS handles.
func DefaultSecretReferences() SecretReferences {
	return SecretReferences{
		OTPKeyRef:     OTPKeyRefName,
		SubjectKeyRef: SubjectKeyRefName,
		ProofKeyRef:   ProofKeyRefName,
		SigningKeyRef: SigningKeyRefName,
		RefreshKeyRef: RefreshKeyRefName,
	}
}

// Validate rejects missing or non-*_REF secret handles.
func (r SecretReferences) Validate() error {
	for _, name := range []string{r.OTPKeyRef, r.SubjectKeyRef, r.ProofKeyRef, r.SigningKeyRef, r.RefreshKeyRef} {
		if err := secretref.ValidateRefName(name); err != nil {
			return err
		}
	}
	return nil
}

// SecretResolver resolves a reference inside the current data region. It must
// fail closed rather than falling back to another region.
type SecretResolver interface {
	Resolve(context.Context, string) ([]byte, error)
}

// ResolveSecretMaterial resolves all identity keys without exposing them in
// configuration. Returned bytes are held only by the service process.
func ResolveSecretMaterial(ctx context.Context, resolver SecretResolver, refs SecretReferences) (SecretMaterial, error) {
	if ctx == nil || resolver == nil {
		return SecretMaterial{}, errors.New("secret resolver and context are required")
	}
	if err := refs.Validate(); err != nil {
		return SecretMaterial{}, err
	}
	resolve := func(name string) ([]byte, error) {
		value, err := resolver.Resolve(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", name, err)
		}
		return value, nil
	}
	var material SecretMaterial
	var err error
	if material.OTPKey, err = resolve(refs.OTPKeyRef); err != nil {
		return SecretMaterial{}, err
	}
	if material.SubjectKey, err = resolve(refs.SubjectKeyRef); err != nil {
		return SecretMaterial{}, err
	}
	if material.ProofKey, err = resolve(refs.ProofKeyRef); err != nil {
		return SecretMaterial{}, err
	}
	if material.SigningKey, err = resolve(refs.SigningKeyRef); err != nil {
		return SecretMaterial{}, err
	}
	if material.RefreshKey, err = resolve(refs.RefreshKeyRef); err != nil {
		return SecretMaterial{}, err
	}
	if err := material.validate(); err != nil {
		return SecretMaterial{}, err
	}
	return material, nil
}

// Clock is injectable so expiry/rate-limit tests never sleep.
type Clock interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// RiskRequest intentionally contains only a subject digest, never an email or
// provider credential.
type RiskRequest struct {
	DataRegion          string
	Provider            Provider
	ProviderSubjectHash string
}

// RiskEvaluator applies abuse/risk policy before a verification is created.
type RiskEvaluator interface {
	Evaluate(context.Context, RiskRequest) error
}

type allowRiskEvaluator struct{}

func (allowRiskEvaluator) Evaluate(context.Context, RiskRequest) error { return nil }

// Notifier is satisfied by services/notify.Router; identity code cannot call a
// concrete email vendor directly.
type Notifier interface {
	Send(context.Context, notify.Message) error
}

// Config contains deterministic policy parameters; commercial provider choice
// and credentials are deliberately absent.
type Config struct {
	OTPTTL                  time.Duration
	ProofTTL                time.Duration
	AccessTTL               time.Duration
	RefreshTTL              time.Duration
	RateWindow              time.Duration
	MaxEmailChallenges      int
	MaxVerificationAttempts int
	RetryAfter              time.Duration
	Issuer                  string
	SupportPath             string
}

// DefaultConfig reflects the conservative TASK-010 defaults.
func DefaultConfig() Config {
	return Config{
		OTPTTL:                  10 * time.Minute,
		ProofTTL:                5 * time.Minute,
		AccessTTL:               15 * time.Minute,
		RefreshTTL:              30 * 24 * time.Hour,
		RateWindow:              15 * time.Minute,
		MaxEmailChallenges:      5,
		MaxVerificationAttempts: 5,
		RetryAfter:              time.Minute,
		Issuer:                  "miangedan-identity",
		SupportPath:             "/support/account-recovery",
	}
}

func (c Config) validate() error {
	if c.OTPTTL <= 0 || c.ProofTTL <= 0 || c.AccessTTL <= 0 || c.RefreshTTL <= 0 || c.RateWindow <= 0 || c.RetryAfter <= 0 {
		return errors.New("identity durations must all be positive")
	}
	if c.MaxEmailChallenges <= 0 || c.MaxVerificationAttempts <= 0 {
		return errors.New("identity rate and attempt limits must be positive")
	}
	if c.Issuer == "" || c.SupportPath == "" {
		return errors.New("identity issuer and support path are required")
	}
	return nil
}

// Service owns TASK-010 deterministic identity behavior.
type Service struct {
	config      Config
	store       Store
	idempotency IdempotencyExecutor
	notifier    Notifier
	providers   *provider.Registry
	risk        RiskEvaluator
	clock       Clock
	ids         IDGenerator
	secrets     SecretMaterial
	tokens      *TokenManager
}

// Dependencies are explicit to keep provider, notification and secret access
// vendor-neutral and testable.
type Dependencies struct {
	Store       Store
	Idempotency IdempotencyExecutor
	Notifier    Notifier
	Providers   *provider.Registry
	Risk        RiskEvaluator
	Clock       Clock
	IDs         IDGenerator
	Secrets     SecretMaterial
}

// NewService validates all dependencies fail-closed.
func NewService(config Config, dependencies Dependencies) (*Service, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	if dependencies.Store == nil || dependencies.Idempotency == nil || dependencies.Notifier == nil || dependencies.Providers == nil {
		return nil, errors.New("identity store, idempotency, notifier and provider registry are required")
	}
	if err := dependencies.Secrets.validate(); err != nil {
		return nil, err
	}
	if dependencies.Risk == nil {
		dependencies.Risk = allowRiskEvaluator{}
	}
	if dependencies.Clock == nil {
		dependencies.Clock = systemClock{}
	}
	if dependencies.IDs == nil {
		dependencies.IDs = CryptoIDGenerator{}
	}
	tokenManager, err := NewTokenManager(config.Issuer, config.AccessTTL, config.RefreshTTL, dependencies.Secrets, dependencies.IDs)
	if err != nil {
		return nil, err
	}
	return &Service{
		config:      config,
		store:       dependencies.Store,
		idempotency: dependencies.Idempotency,
		notifier:    dependencies.Notifier,
		providers:   dependencies.Providers,
		risk:        dependencies.Risk,
		clock:       dependencies.Clock,
		ids:         dependencies.IDs,
		secrets:     dependencies.Secrets,
		tokens:      tokenManager,
	}, nil
}
