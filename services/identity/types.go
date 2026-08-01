// Package identity implements TASK-010 user accounts, multi-provider login,
// short-lived verification proofs, sessions, and dual-proof identity binding.
//
// Tracing: US-05, FR-027, SEC-002/012/040, ADR-0005.
package identity

import "time"

// Provider is the stable domain identifier for a login identity.
type Provider string

const (
	ProviderEmailOTP Provider = "email_otp"
	ProviderGoogle   Provider = "google"
	ProviderApple    Provider = "apple"
	ProviderWeChat   Provider = "wechat"
)

// Valid reports whether p is one of the four PRD-approved login providers.
func (p Provider) Valid() bool {
	switch p {
	case ProviderEmailOTP, ProviderGoogle, ProviderApple, ProviderWeChat:
		return true
	default:
		return false
	}
}

// AgeStatus is the account age/guardian state. Users under 16 do not get an
// account and therefore are deliberately absent from these persisted values.
type AgeStatus string

const (
	AgeAdult                 AgeStatus = "adult"
	AgeMinorGuardianVerified AgeStatus = "minor_guardian_verified"
	AgeMinorPending          AgeStatus = "minor_pending"
)

// Valid reports whether the age status may be persisted on an account.
func (a AgeStatus) Valid() bool {
	switch a {
	case AgeAdult, AgeMinorGuardianVerified, AgeMinorPending:
		return true
	default:
		return false
	}
}

// AccountStatus is the user account lifecycle state.
type AccountStatus string

const (
	AccountActive            AccountStatus = "active"
	AccountDeletionPending   AccountStatus = "deletion_pending"
	AccountDeletedAnonymized AccountStatus = "deleted_anonymized"
)

// Language is an approved launch UI language.
type Language string

const (
	LanguageZH Language = "zh-CN"
	LanguageEN Language = "en-US"
)

// Valid reports whether l is an approved launch UI language.
func (l Language) Valid() bool { return l == LanguageZH || l == LanguageEN }

// RegistrationEvidence proves acceptance of the mandatory registration
// notices. It never substitutes for the six independent ConsentGrant records
// implemented by TASK-011.
type RegistrationEvidence struct {
	TermsVersion          string            `json:"terms_version"`
	PrivacyVersion        string            `json:"privacy_version"`
	DataProcessingVersion string            `json:"data_processing_version"`
	AcceptedAt            time.Time         `json:"accepted_at"`
	Context               AcceptanceContext `json:"acceptance_context"`
}

// AcceptanceContext records the surface and language in which registration
// notices were accepted. It must not contain free-form user content.
type AcceptanceContext struct {
	UILanguage Language `json:"ui_language"`
	Surface    string   `json:"surface"`
}

// Registration is required only when a verified identity has no account.
type Registration struct {
	UILanguage Language             `json:"ui_language"`
	AgeStatus  AgeStatus            `json:"age_status"`
	Evidence   RegistrationEvidence `json:"evidence"`
}

// User is the account aggregate root. DataRegion is immutable after creation.
type User struct {
	UserID       string
	DataRegion   string
	UILanguage   Language
	AgeStatus    AgeStatus
	Status       AccountStatus
	DisplayName  *string
	Registration RegistrationEvidence
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Identity is a provider identity bound to exactly one user. ProviderSubjectHash
// is internal and must never be returned by the HTTP API or written to logs.
type Identity struct {
	IdentityID          string
	UserID              string
	Provider            Provider
	ProviderSubjectHash string
	VerifiedAt          time.Time
	DataRegion          string
	CreatedAt           time.Time
}

// IdentityBinding is the public, non-sensitive view of a bound identity.
type IdentityBinding struct {
	IdentityID string    `json:"identity_id"`
	Provider   Provider  `json:"provider"`
	VerifiedAt time.Time `json:"verified_at"`
	DataRegion string    `json:"data_region"`
}

// Account is the public account representation. Provider subjects and contact
// values are intentionally absent.
type Account struct {
	UserID      string            `json:"user_id"`
	DataRegion  string            `json:"data_region"`
	UILanguage  Language          `json:"ui_language"`
	AgeStatus   AgeStatus         `json:"age_status"`
	Status      AccountStatus     `json:"status"`
	DisplayName *string           `json:"display_name"`
	Identities  []IdentityBinding `json:"identities"`
}

// VerificationStatus is the lifecycle of an email/OAuth verification.
type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "pending"
	VerificationVerified VerificationStatus = "verified"
	VerificationConsumed VerificationStatus = "consumed"
	VerificationExpired  VerificationStatus = "expired"
	VerificationLocked   VerificationStatus = "locked"
)

// Verification stores only digests. Email addresses, OTP plaintext, OAuth
// authorization codes and proof tokens are never persisted.
type Verification struct {
	VerificationID      string
	Provider            Provider
	ProviderSubjectHash string
	CodeHash            string
	ProofHash           string
	Status              VerificationStatus
	FailedAttempts      int
	MaxAttempts         int
	RequestedAt         time.Time
	VerifiedAt          *time.Time
	ExpiresAt           time.Time
	ProofExpiresAt      *time.Time
	ConsumedAt          *time.Time
	NotificationSentAt  *time.Time
	DataRegion          string
	RequestKey          string
}

// VerificationChallenge is returned after a region-local notification has
// been accepted for delivery.
type VerificationChallenge struct {
	ChallengeID       string    `json:"challenge_id"`
	ExpiresAt         time.Time `json:"expires_at"`
	RetryAfterSeconds int       `json:"retry_after_seconds"`
	DeliveryStatus    string    `json:"delivery_status"`
	DataRegion        string    `json:"data_region"`
}

// VerificationProof is a short-lived single-use proof. ProofToken is returned
// once to the caller and must never be logged.
type VerificationProof struct {
	ProofToken string    `json:"proof_token"`
	Provider   Provider  `json:"provider"`
	ExpiresAt  time.Time `json:"expires_at"`
	DataRegion string    `json:"data_region"`
}

// SessionRecord persists token digests and lifecycle state, never token text.
type SessionRecord struct {
	SessionID        string
	UserID           string
	RefreshTokenHash string
	Status           string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	RotatedTo        string
	DataRegion       string
	CreatedAt        time.Time
	RotatedAt        *time.Time
}

const (
	SessionActive  = "active"
	SessionRotated = "rotated"
	SessionRevoked = "revoked"
	SessionExpired = "expired"
)

// Session is the token response. Tokens are deliberately omitted from all
// persisted entity types.
type Session struct {
	SessionID        string  `json:"session_id"`
	AccessToken      string  `json:"access_token"`
	RefreshToken     string  `json:"refresh_token"`
	TokenType        string  `json:"token_type"`
	ExpiresIn        int64   `json:"expires_in"`
	RefreshExpiresIn int64   `json:"refresh_expires_in"`
	Account          Account `json:"account"`
	DataRegion       string  `json:"data_region"`
}

// Claims are the verified business-token claims used by identity HTTP routes.
type Claims struct {
	UserID     string
	SessionID  string
	DataRegion string
	ExpiresAt  time.Time
}

// RecoveryCase records an identity collision without merging either account.
// ConflictingUserID and ProviderSubjectHash are internal and never exposed.
type RecoveryCase struct {
	RecoveryCaseID      string
	RequestingUserID    string
	ConflictingUserID   string
	Provider            Provider
	ProviderSubjectHash string
	Status              string
	DataRegion          string
	CreatedAt           time.Time
}
