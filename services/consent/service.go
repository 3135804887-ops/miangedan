package consent

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	"miangedan/services/region"
)

const (
	operationGrant    = "grant"
	operationWithdraw = "withdraw"
	maxRawAVDuration  = 30 * 24 * time.Hour
)

var (
	idempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)
	versionRefPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)

// Clock is injectable so expiry and immediate-withdrawal tests do not sleep.
type Clock interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// RecordingEligibility fail-closes raw audio/video consent for minors.
// A production adapter reads the authoritative age status from identity.
type RecordingEligibility interface {
	AllowRawAV(context.Context, string, string) (bool, error)
}

// Dependencies are explicit regional persistence and identity-policy adapters.
type Dependencies struct {
	Store       Store
	Eligibility RecordingEligibility
	Clock       Clock
	IDs         IDGenerator
	Observer    Observer
}

// Service owns TASK-011 append-only consent behavior.
type Service struct {
	store       Store
	eligibility RecordingEligibility
	clock       Clock
	ids         IDGenerator
	observer    Observer
}

// NewService validates consent dependencies and installs safe defaults.
func NewService(dependencies Dependencies) (*Service, error) {
	if dependencies.Store == nil || dependencies.Eligibility == nil {
		return nil, errors.New("consent store and recording eligibility are required")
	}
	if dependencies.Clock == nil {
		dependencies.Clock = systemClock{}
	}
	if dependencies.IDs == nil {
		dependencies.IDs = CryptoIDGenerator{}
	}
	if dependencies.Observer == nil {
		dependencies.Observer = NoopObserver{}
	}
	return &Service{
		store: dependencies.Store, eligibility: dependencies.Eligibility,
		clock: dependencies.Clock, ids: dependencies.IDs, observer: dependencies.Observer,
	}, nil
}

// List returns at least one current state for each of the six categories.
func (s *Service) List(ctx context.Context, actor Actor) ([]State, error) {
	if err := validateActor(ctx, actor); err != nil {
		return nil, err
	}
	now := s.clock.Now().UTC()
	var grants []Grant
	if err := s.store.Transact(ctx, func(tx Tx) error {
		var err error
		grants, err = tx.LatestGrantsByUser(actor.UserID, actor.DataRegion)
		return err
	}); err != nil {
		return nil, internalError(err)
	}
	states := make([]State, 0, len(grants)+len(allTypes))
	seen := make(map[Type]bool)
	for _, grant := range grants {
		cloned := cloneGrant(grant)
		states = append(states, State{
			Type: grant.Type, Scope: cloneScope(grant.Scope), ScopeHash: grant.ScopeHash,
			EffectiveStatus: effectiveStatus(grant, now), Version: grant.Version,
			Grant: &cloned, DataRegion: actor.DataRegion,
		})
		seen[grant.Type] = true
	}
	emptyScope, emptyHash, err := normalizeScope(Scope{})
	if err != nil {
		return nil, internalError(err)
	}
	for _, consentType := range allTypes {
		if !seen[consentType] {
			states = append(states, State{
				Type: consentType, Scope: emptyScope, ScopeHash: emptyHash,
				EffectiveStatus: EffectiveNotGranted, Version: 0, DataRegion: actor.DataRegion,
			})
		}
	}
	sort.Slice(states, func(i, j int) bool {
		if states[i].Type == states[j].Type {
			return states[i].ScopeHash < states[j].ScopeHash
		}
		return states[i].Type < states[j].Type
	})
	return states, nil
}

// History returns every immutable version for one category.
func (s *Service) History(ctx context.Context, actor Actor, consentType Type) ([]Grant, error) {
	if err := validateActor(ctx, actor); err != nil || !consentType.Valid() {
		return nil, validationError()
	}
	var history []Grant
	if err := s.store.Transact(ctx, func(tx Tx) error {
		var err error
		history, err = tx.HistoryByType(actor.UserID, actor.DataRegion, consentType)
		return err
	}); err != nil {
		return nil, internalError(err)
	}
	return history, nil
}

// Grant appends an explicit consent version and its audit in one transaction.
func (s *Service) Grant(
	ctx context.Context,
	actor Actor,
	consentType Type,
	input GrantInput,
	idempotencyKey string,
) (result Grant, resultErr error) {
	defer func() {
		s.observeMutation(ctx, operationGrant, actor.DataRegion, consentType, result, resultErr)
	}()
	if err := validateActor(ctx, actor); err != nil || !consentType.Valid() || !validIdempotencyKey(idempotencyKey) {
		return Grant{}, validationError()
	}
	now := s.clock.Now().UTC()
	scope, scopeHash, err := normalizeScope(input.Scope)
	if err != nil || validateScopeShape(consentType, scope) != nil {
		return Grant{}, validationError()
	}
	evidenceInput, err := normalizeEvidence(input.Evidence)
	if err != nil {
		return Grant{}, validationError()
	}
	expiresAt := cloneTime(input.ExpiresAt)
	requestHash, err := hashJSON(struct {
		Type      Type
		Scope     Scope
		ExpiresAt *time.Time
		Evidence  EvidenceInput
	}{consentType, scope, expiresAt, evidenceInput})
	if err != nil {
		return Grant{}, internalError(err)
	}
	if existing, found, replayErr := s.replayGrant(
		ctx, actor, operationGrant, idempotencyKey, requestHash,
	); replayErr != nil {
		return Grant{}, replayErr
	} else if found {
		return existing, nil
	}
	if validateEvidenceTime(evidenceInput, now) != nil || validateGrantExpiry(consentType, expiresAt, now) != nil {
		return Grant{}, validationError()
	}
	if consentType == TypeRawAVRecording {
		allowed, eligibilityErr := s.eligibility.AllowRawAV(ctx, actor.UserID, actor.DataRegion)
		if eligibilityErr != nil {
			return Grant{}, internalError(eligibilityErr)
		}
		if !allowed {
			return Grant{}, forbiddenError()
		}
	}
	evidence, err := buildEvidence(evidenceInput, operationGrant, now)
	if err != nil {
		return Grant{}, internalError(err)
	}
	if err := s.store.Transact(ctx, func(tx Tx) error {
		existing, found, lookupErr := tx.GrantByRequest(actor.UserID, actor.DataRegion, operationGrant, idempotencyKey)
		if lookupErr != nil {
			return lookupErr
		}
		if found {
			if existing.RequestHash != requestHash {
				return idempotencyConflictError()
			}
			result = existing
			return nil
		}
		latest, found, latestErr := tx.LatestGrant(actor.UserID, actor.DataRegion, consentType, scopeHash)
		if latestErr != nil {
			return latestErr
		}
		grantID, auditID, idErr := s.newRecordIDs()
		if idErr != nil {
			return idErr
		}
		version := 1
		var supersedes *string
		if found {
			version = latest.Version + 1
			supersedes = &latest.GrantID
		}
		result = Grant{
			GrantID: grantID, UserID: actor.UserID, Type: consentType,
			Scope: scope, ScopeHash: scopeHash, Status: StatusGranted,
			GrantedAt: now, ExpiresAt: expiresAt, SupersedesGrantID: supersedes,
			Evidence: evidence, Version: version, RecordedAt: now, DataRegion: actor.DataRegion,
			RequestOperation: operationGrant, RequestKey: idempotencyKey, RequestHash: requestHash,
			AuditID: auditID,
		}
		if err := tx.AppendAudit(newAudit(result, actor, "consent.granted", now)); err != nil {
			return err
		}
		return tx.AppendGrant(result)
	}); err != nil {
		return Grant{}, mapStoreError(err)
	}
	return cloneGrant(result), nil
}

// Withdraw appends a withdrawn version and audit atomically. Once this method
// returns, Decide observes the withdrawal for the exact scope.
func (s *Service) Withdraw(
	ctx context.Context,
	actor Actor,
	consentType Type,
	input WithdrawalInput,
	idempotencyKey string,
) (result Grant, resultErr error) {
	defer func() {
		s.observeMutation(ctx, operationWithdraw, actor.DataRegion, consentType, result, resultErr)
	}()
	if err := validateActor(ctx, actor); err != nil || !consentType.Valid() || !validIdempotencyKey(idempotencyKey) {
		return Grant{}, validationError()
	}
	now := s.clock.Now().UTC()
	scope, scopeHash, err := normalizeScope(input.Scope)
	if err != nil || validateScopeShape(consentType, scope) != nil {
		return Grant{}, validationError()
	}
	evidenceInput, err := normalizeEvidence(input.Evidence)
	if err != nil {
		return Grant{}, validationError()
	}
	requestHash, err := hashJSON(struct {
		Type     Type
		Scope    Scope
		Evidence EvidenceInput
	}{consentType, scope, evidenceInput})
	if err != nil {
		return Grant{}, internalError(err)
	}
	if existing, found, replayErr := s.replayGrant(
		ctx, actor, operationWithdraw, idempotencyKey, requestHash,
	); replayErr != nil {
		return Grant{}, replayErr
	} else if found {
		return existing, nil
	}
	if validateEvidenceTime(evidenceInput, now) != nil {
		return Grant{}, validationError()
	}
	evidence, err := buildEvidence(evidenceInput, operationWithdraw, now)
	if err != nil {
		return Grant{}, internalError(err)
	}
	if err := s.store.Transact(ctx, func(tx Tx) error {
		existing, found, lookupErr := tx.GrantByRequest(actor.UserID, actor.DataRegion, operationWithdraw, idempotencyKey)
		if lookupErr != nil {
			return lookupErr
		}
		if found {
			if existing.RequestHash != requestHash {
				return idempotencyConflictError()
			}
			result = existing
			return nil
		}
		latest, found, latestErr := tx.LatestGrant(actor.UserID, actor.DataRegion, consentType, scopeHash)
		if latestErr != nil {
			return latestErr
		}
		if !found {
			return notFoundError()
		}
		if latest.Status != StatusGranted {
			return conflictError()
		}
		grantID, auditID, idErr := s.newRecordIDs()
		if idErr != nil {
			return idErr
		}
		supersedes := latest.GrantID
		withdrawnAt := now
		result = Grant{
			GrantID: grantID, UserID: actor.UserID, Type: consentType,
			Scope: scope, ScopeHash: scopeHash, Status: StatusWithdrawn,
			GrantedAt: latest.GrantedAt, ExpiresAt: cloneTime(latest.ExpiresAt), WithdrawnAt: &withdrawnAt,
			SupersedesGrantID: &supersedes, Evidence: evidence, Version: latest.Version + 1,
			RecordedAt: now, DataRegion: actor.DataRegion,
			RequestOperation: operationWithdraw, RequestKey: idempotencyKey, RequestHash: requestHash,
			AuditID: auditID,
		}
		if err := tx.AppendAudit(newAudit(result, actor, "consent.withdrawn", now)); err != nil {
			return err
		}
		return tx.AppendGrant(result)
	}); err != nil {
		return Grant{}, mapStoreError(err)
	}
	return cloneGrant(result), nil
}

// Decide returns a synchronous, linearizable authorization decision.
func (s *Service) Decide(
	ctx context.Context,
	actor Actor,
	request AccessRequest,
) (result AccessDecision, resultErr error) {
	defer func() {
		s.observeDecision(ctx, actor.DataRegion, request.Type, result, resultErr)
	}()
	if err := validateActor(ctx, actor); err != nil || !request.Type.Valid() {
		return AccessDecision{}, validationError()
	}
	scope, scopeHash, err := normalizeScope(request.Scope)
	if err != nil || validateScopeShape(request.Type, scope) != nil {
		return AccessDecision{}, validationError()
	}
	now := s.clock.Now().UTC()
	decision := AccessDecision{
		Type: request.Type, ScopeHash: scopeHash, EffectiveStatus: EffectiveNotGranted,
		DecidedAt: now, DataRegion: actor.DataRegion,
	}
	if err := s.store.Transact(ctx, func(tx Tx) error {
		latest, found, lookupErr := tx.LatestGrant(actor.UserID, actor.DataRegion, request.Type, scopeHash)
		if lookupErr != nil {
			return lookupErr
		}
		if !found {
			return nil
		}
		status := effectiveStatus(latest, now)
		decision.EffectiveStatus = status
		decision.Allowed = status == EffectiveGranted
		grantID := latest.GrantID
		decision.GrantID = &grantID
		decision.ExpiresAt = cloneTime(latest.ExpiresAt)
		return nil
	}); err != nil {
		return AccessDecision{}, internalError(err)
	}
	if decision.Allowed && request.Type == TypeRawAVRecording {
		allowed, eligibilityErr := s.eligibility.AllowRawAV(ctx, actor.UserID, actor.DataRegion)
		if eligibilityErr != nil {
			return AccessDecision{}, internalError(eligibilityErr)
		}
		if !allowed {
			return AccessDecision{}, forbiddenError()
		}
	}
	return decision, nil
}

func (s *Service) replayGrant(
	ctx context.Context,
	actor Actor,
	operation string,
	idempotencyKey string,
	requestHash string,
) (Grant, bool, error) {
	var existing Grant
	var found bool
	if err := s.store.Transact(ctx, func(tx Tx) error {
		var lookupErr error
		existing, found, lookupErr = tx.GrantByRequest(
			actor.UserID, actor.DataRegion, operation, idempotencyKey,
		)
		return lookupErr
	}); err != nil {
		return Grant{}, false, internalError(err)
	}
	if !found {
		return Grant{}, false, nil
	}
	if existing.RequestHash != requestHash {
		return Grant{}, false, idempotencyConflictError()
	}
	return cloneGrant(existing), true, nil
}

func (s *Service) newRecordIDs() (string, string, error) {
	grantID, err := s.ids.NewID()
	if err != nil {
		return "", "", err
	}
	auditID, err := s.ids.NewID()
	if err != nil {
		return "", "", err
	}
	return grantID, auditID, nil
}

func validateActor(ctx context.Context, actor Actor) error {
	if ctx == nil || !validUUID(actor.UserID) || actor.SessionID == "" || region.ValidateDataRegion(actor.DataRegion) != nil {
		return validationError()
	}
	return nil
}

func validIdempotencyKey(value string) bool { return idempotencyKeyPattern.MatchString(value) }

func normalizeScope(input Scope) (Scope, string, error) {
	normalized := cloneScope(input)
	if normalized.AssignmentID != nil {
		value := strings.TrimSpace(*normalized.AssignmentID)
		if !validUUID(value) {
			return Scope{}, "", errors.New("invalid assignment id")
		}
		normalized.AssignmentID = &value
	}
	var err error
	if normalized.DataCategories, err = normalizeDataCategories(normalized.DataCategories); err != nil {
		return Scope{}, "", err
	}
	if normalized.MediaCategories, err = normalizeMediaCategories(normalized.MediaCategories); err != nil {
		return Scope{}, "", err
	}
	if normalized.Channels, err = normalizeChannels(normalized.Channels); err != nil {
		return Scope{}, "", err
	}
	scopeHash, err := hashJSON(normalized)
	if err != nil {
		return Scope{}, "", err
	}
	return normalized, scopeHash, nil
}

func normalizeDataCategories(values []DataCategory) ([]DataCategory, error) {
	allowed := map[DataCategory]bool{
		DataTotalScore: true, DataRadar: true, DataRoundResults: true,
		DataFullReport: true, DataTranscript: true, DataMedia: true,
	}
	return normalizeEnum(values, allowed)
}

func normalizeMediaCategories(values []MediaCategory) ([]MediaCategory, error) {
	return normalizeEnum(values, map[MediaCategory]bool{MediaAudio: true, MediaVideo: true})
}

func normalizeChannels(values []Channel) ([]Channel, error) {
	return normalizeEnum(values, map[Channel]bool{ChannelEmail: true, ChannelInApp: true, ChannelPush: true})
}

func normalizeEnum[T ~string](values []T, allowed map[T]bool) ([]T, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := make(map[T]bool, len(values))
	normalized := make([]T, 0, len(values))
	for _, value := range values {
		if !allowed[value] || seen[value] {
			return nil, errors.New("invalid or duplicate scope value")
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return normalized, nil
}

func validateScopeShape(consentType Type, scope Scope) error {
	hasAssignment := scope.AssignmentID != nil
	hasData := len(scope.DataCategories) > 0
	hasMedia := len(scope.MediaCategories) > 0
	hasChannels := len(scope.Channels) > 0
	switch consentType {
	case TypeCoreService, TypeProductAnalytics, TypeModelTraining:
		if hasAssignment || hasData || hasMedia || hasChannels {
			return errors.New("category requires empty scope")
		}
	case TypeRawAVRecording:
		if hasAssignment || hasData || !hasMedia || hasChannels {
			return errors.New("raw recording requires media categories only")
		}
	case TypeOrgSharing:
		if !hasAssignment || !hasData || hasMedia || hasChannels {
			return errors.New("organization sharing requires assignment and data categories")
		}
	case TypeMarketing:
		if hasAssignment || hasData || hasMedia || !hasChannels {
			return errors.New("marketing requires channels only")
		}
	default:
		return errors.New("invalid consent type")
	}
	return nil
}

func validateGrantExpiry(consentType Type, expiresAt *time.Time, now time.Time) error {
	if expiresAt != nil && !expiresAt.After(now) {
		return errors.New("expiry must be in the future")
	}
	switch consentType {
	case TypeCoreService:
		if expiresAt != nil {
			return errors.New("core service grant must not expire")
		}
	case TypeRawAVRecording:
		if expiresAt == nil || expiresAt.After(now.Add(maxRawAVDuration)) {
			return errors.New("raw recording grant must expire within 30 days")
		}
	case TypeOrgSharing:
		if expiresAt == nil {
			return errors.New("organization sharing requires expiry")
		}
	}
	return nil
}

func normalizeEvidence(input EvidenceInput) (EvidenceInput, error) {
	normalized := input
	normalized.CopyVersion = strings.TrimSpace(input.CopyVersion)
	normalized.PrivacyPolicyVersion = strings.TrimSpace(input.PrivacyPolicyVersion)
	normalized.PresentedAt = input.PresentedAt.UTC()
	if !versionRefPattern.MatchString(normalized.CopyVersion) ||
		!versionRefPattern.MatchString(normalized.PrivacyPolicyVersion) ||
		normalized.PresentedAt.IsZero() {
		return EvidenceInput{}, errors.New("invalid consent evidence")
	}
	if !validUIContext(normalized.UIContext) {
		return EvidenceInput{}, errors.New("invalid consent ui context")
	}
	return normalized, nil
}

func validateEvidenceTime(input EvidenceInput, now time.Time) error {
	if input.PresentedAt.After(now.Add(5 * time.Minute)) {
		return errors.New("consent evidence presentation time is in the future")
	}
	return nil
}

func validUIContext(context UIContext) bool {
	validSurface := context.Surface == "web" || context.Surface == "ios" || context.Surface == "android"
	validFlow := context.Flow == "registration" || context.Flow == "consent_center" ||
		context.Flow == "interview_room" || context.Flow == "assignment_share"
	validLanguage := context.UILanguage == "zh-CN" || context.UILanguage == "en-US"
	return validSurface && validFlow && validLanguage
}

func buildEvidence(input EvidenceInput, action string, recordedAt time.Time) (Evidence, error) {
	evidence := Evidence{
		CopyVersion: input.CopyVersion, PrivacyPolicyVersion: input.PrivacyPolicyVersion,
		PresentedAt: input.PresentedAt, UIContext: input.UIContext,
		Action: action, RecordedAt: recordedAt,
	}
	hash, err := hashJSON(evidence)
	if err != nil {
		return Evidence{}, err
	}
	evidence.EvidenceHash = hash
	return evidence, nil
}

func effectiveStatus(grant Grant, now time.Time) EffectiveStatus {
	switch grant.Status {
	case StatusWithdrawn:
		return EffectiveWithdrawn
	case StatusExpired:
		return EffectiveExpired
	case StatusGranted:
		if grant.ExpiresAt != nil && !now.Before(*grant.ExpiresAt) {
			return EffectiveExpired
		}
		return EffectiveGranted
	default:
		return EffectiveNotGranted
	}
}

func newAudit(grant Grant, actor Actor, action string, createdAt time.Time) AuditEvent {
	return AuditEvent{
		AuditID: grant.AuditID, SubjectType: "user", SubjectID: actor.UserID,
		ActorID: actor.UserID, ActorRole: "user", Action: action,
		ResourceType: "consent_grant", ResourceID: grant.GrantID,
		LegalBasis: "consent", DataRegion: actor.DataRegion, CreatedAt: createdAt,
	}
}

func mapStoreError(err error) error {
	if err == nil {
		return nil
	}
	var domain *DomainError
	if errors.As(err, &domain) {
		return domain
	}
	return internalError(err)
}

func (s *Service) observeMutation(
	ctx context.Context,
	operation string,
	dataRegion string,
	consentType Type,
	grant Grant,
	err error,
) {
	observation := Observation{
		Operation: operation, Outcome: "success",
		ConsentType: safeObservationType(consentType), DataRegion: safeObservationRegion(dataRegion),
	}
	if err != nil {
		observation.Outcome = "error"
		observation.ErrorCode = ErrorCodeOf(err)
	} else {
		observation.EffectiveStatus = effectiveStatus(grant, s.clock.Now().UTC())
	}
	s.recordObservation(ctx, observation)
}

func (s *Service) observeDecision(
	ctx context.Context,
	dataRegion string,
	consentType Type,
	decision AccessDecision,
	err error,
) {
	observation := Observation{
		Operation: "access_decision", Outcome: "denied",
		ConsentType: safeObservationType(consentType), DataRegion: safeObservationRegion(dataRegion),
		EffectiveStatus: decision.EffectiveStatus,
	}
	if err != nil {
		observation.Outcome = "error"
		observation.ErrorCode = ErrorCodeOf(err)
	} else if decision.Allowed {
		observation.Outcome = "allowed"
	}
	s.recordObservation(ctx, observation)
}

func (s *Service) recordObservation(ctx context.Context, observation Observation) {
	if ctx == nil {
		ctx = context.Background()
	}
	defer func() {
		// Authorization must remain available and deterministic when telemetry fails.
		_ = recover()
	}()
	s.observer.Record(ctx, observation)
}

func safeObservationType(consentType Type) Type {
	if consentType.Valid() {
		return consentType
	}
	return ""
}

func safeObservationRegion(dataRegion string) string {
	if region.ValidateDataRegion(dataRegion) == nil {
		return dataRegion
	}
	return "unknown"
}
