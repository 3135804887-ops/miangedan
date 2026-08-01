package consent

import "time"

// Type identifies one of the six independent consent categories.
type Type string

// Supported consent category values.
const (
	TypeCoreService      Type = "core_service"
	TypeRawAVRecording   Type = "raw_av_recording"
	TypeOrgSharing       Type = "org_sharing"
	TypeProductAnalytics Type = "product_analytics"
	TypeModelTraining    Type = "model_training"
	TypeMarketing        Type = "marketing"
)

var allTypes = []Type{
	TypeCoreService,
	TypeRawAVRecording,
	TypeOrgSharing,
	TypeProductAnalytics,
	TypeModelTraining,
	TypeMarketing,
}

// Valid reports whether the category is one of the six PRD-approved values.
func (t Type) Valid() bool {
	switch t {
	case TypeCoreService, TypeRawAVRecording, TypeOrgSharing,
		TypeProductAnalytics, TypeModelTraining, TypeMarketing:
		return true
	default:
		return false
	}
}

// AllTypes returns the six categories in stable display order.
func AllTypes() []Type { return append([]Type(nil), allTypes...) }

// Status is the immutable state recorded by a grant version.
type Status string

// Persisted grant status values.
const (
	StatusGranted   Status = "granted"
	StatusWithdrawn Status = "withdrawn"
	StatusExpired   Status = "expired"
)

// EffectiveStatus is the online authorization state, including absence.
type EffectiveStatus string

// Effective authorization status values.
const (
	EffectiveNotGranted EffectiveStatus = "not_granted"
	EffectiveGranted    EffectiveStatus = "granted"
	EffectiveWithdrawn  EffectiveStatus = "withdrawn"
	EffectiveExpired    EffectiveStatus = "expired"
)

// DataCategory is an institution-shareable result category.
type DataCategory string

// Supported institution-sharing categories.
const (
	DataTotalScore   DataCategory = "total_score"
	DataRadar        DataCategory = "radar"
	DataRoundResults DataCategory = "round_results"
	DataFullReport   DataCategory = "full_report"
	DataTranscript   DataCategory = "transcript"
	DataMedia        DataCategory = "media"
)

// MediaCategory is a raw recording category.
type MediaCategory string

// Supported raw recording categories.
const (
	MediaAudio MediaCategory = "audio"
	MediaVideo MediaCategory = "video"
)

// Channel is an optional marketing notification channel.
type Channel string

// Supported marketing channels; phone and SMS are deliberately absent.
const (
	ChannelEmail Channel = "email"
	ChannelInApp Channel = "in_app"
	ChannelPush  Channel = "push"
)

// Scope is a closed, typed consent scope. It cannot carry free-form content.
type Scope struct {
	AssignmentID    *string         `json:"assignment_id,omitempty"`
	DataCategories  []DataCategory  `json:"data_categories,omitempty"`
	MediaCategories []MediaCategory `json:"media_categories,omitempty"`
	Channels        []Channel       `json:"channels,omitempty"`
}

// UIContext records the exact product surface where a choice was made.
type UIContext struct {
	Surface    string `json:"surface"`
	Flow       string `json:"flow"`
	UILanguage string `json:"ui_language"`
}

// EvidenceInput contains versioned, non-content proof supplied by the UI.
type EvidenceInput struct {
	CopyVersion          string    `json:"copy_version"`
	PrivacyPolicyVersion string    `json:"privacy_policy_version"`
	PresentedAt          time.Time `json:"presented_at"`
	UIContext            UIContext `json:"ui_context"`
}

// Evidence is the immutable server-stamped proof attached to a grant version.
type Evidence struct {
	CopyVersion          string    `json:"copy_version"`
	PrivacyPolicyVersion string    `json:"privacy_policy_version"`
	PresentedAt          time.Time `json:"presented_at"`
	UIContext            UIContext `json:"ui_context"`
	Action               string    `json:"action"`
	RecordedAt           time.Time `json:"recorded_at"`
	EvidenceHash         string    `json:"evidence_hash"`
}

// Grant is one append-only consent version. Internal request and audit fields
// are excluded from transport responses.
type Grant struct {
	GrantID           string     `json:"grant_id"`
	UserID            string     `json:"-"`
	Type              Type       `json:"consent_type"`
	Scope             Scope      `json:"scope"`
	ScopeHash         string     `json:"scope_hash"`
	Status            Status     `json:"status"`
	GrantedAt         time.Time  `json:"granted_at"`
	ExpiresAt         *time.Time `json:"expires_at"`
	WithdrawnAt       *time.Time `json:"withdrawn_at"`
	SupersedesGrantID *string    `json:"supersedes_grant_id"`
	Evidence          Evidence   `json:"evidence"`
	Version           int        `json:"version"`
	RecordedAt        time.Time  `json:"recorded_at"`
	DataRegion        string     `json:"data_region"`
	RequestOperation  string     `json:"-"`
	RequestKey        string     `json:"-"`
	RequestHash       string     `json:"-"`
	AuditID           string     `json:"-"`
}

// State is one current consent-center item. Grant is nil when no record exists.
type State struct {
	Type            Type            `json:"consent_type"`
	Scope           Scope           `json:"scope"`
	ScopeHash       string          `json:"scope_hash"`
	EffectiveStatus EffectiveStatus `json:"effective_status"`
	Version         int             `json:"version"`
	Grant           *Grant          `json:"grant"`
	DataRegion      string          `json:"data_region"`
}

// Actor is the authenticated account requesting or evaluating consent.
type Actor struct {
	UserID     string
	SessionID  string
	DataRegion string
}

// GrantInput describes an explicit grant choice.
type GrantInput struct {
	Scope     Scope
	ExpiresAt *time.Time
	Evidence  EvidenceInput
}

// WithdrawalInput identifies an exact scope and withdrawal evidence.
type WithdrawalInput struct {
	Scope    Scope
	Evidence EvidenceInput
}

// AccessRequest asks for a synchronous decision on one exact category/scope.
type AccessRequest struct {
	Type  Type  `json:"consent_type"`
	Scope Scope `json:"scope"`
}

// AccessDecision is a fail-closed snapshot of the latest grant version.
type AccessDecision struct {
	Allowed         bool            `json:"allowed"`
	Type            Type            `json:"consent_type"`
	ScopeHash       string          `json:"scope_hash"`
	EffectiveStatus EffectiveStatus `json:"effective_status"`
	GrantID         *string         `json:"grant_id"`
	ExpiresAt       *time.Time      `json:"expires_at"`
	DecidedAt       time.Time       `json:"decided_at"`
	DataRegion      string          `json:"data_region"`
}

// AuditEvent is an append-only, content-free access audit record.
type AuditEvent struct {
	AuditID      string
	SubjectType  string
	SubjectID    string
	ActorID      string
	ActorRole    string
	Action       string
	ResourceType string
	ResourceID   string
	LegalBasis   string
	DataRegion   string
	CreatedAt    time.Time
}
