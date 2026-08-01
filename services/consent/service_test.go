package consent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// synthetic: true — fixed UUIDs/timestamps below are non-personal fixtures.
const (
	syntheticUserID    = "00000000-0000-4000-8000-000000000001"
	syntheticSessionID = "00000000-0000-4000-8000-000000000002"
	syntheticAssignID  = "00000000-0000-4000-8000-000000000003"
)

var syntheticNow = time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)

type fixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fixedClock) set(value time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = value
}

type sequenceIDs struct {
	mu   sync.Mutex
	next int
}

func (s *sequenceIDs) NewID() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", s.next), nil
}

type fixedEligibility struct {
	allowed bool
	err     error
}

type captureObserver struct {
	mu           sync.Mutex
	observations []Observation
	panicOnWrite bool
}

func (o *captureObserver) Record(_ context.Context, observation Observation) {
	if o.panicOnWrite {
		panic("synthetic telemetry outage")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.observations = append(o.observations, observation)
}

func (o *captureObserver) snapshot() []Observation {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]Observation(nil), o.observations...)
}

func (e fixedEligibility) AllowRawAV(context.Context, string, string) (bool, error) {
	return e.allowed, e.err
}

type mutableEligibility struct {
	mu      sync.Mutex
	allowed bool
	err     error
	calls   int
}

func (e *mutableEligibility) AllowRawAV(context.Context, string, string) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	return e.allowed, e.err
}

func (e *mutableEligibility) set(allowed bool, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.allowed = allowed
	e.err = err
}

func (e *mutableEligibility) callCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

func newTestService(t *testing.T, recordingAllowed bool) (*Service, *MemoryStore, *fixedClock) {
	t.Helper()
	store := NewMemoryStore()
	clock := &fixedClock{now: syntheticNow}
	service, err := NewService(Dependencies{
		Store: store, Eligibility: fixedEligibility{allowed: recordingAllowed},
		Clock: clock, IDs: &sequenceIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, store, clock
}

func syntheticActor() Actor {
	return Actor{UserID: syntheticUserID, SessionID: syntheticSessionID, DataRegion: "intl"}
}

func syntheticEvidence(actionIndex int) EvidenceInput {
	return EvidenceInput{
		CopyVersion:          fmt.Sprintf("consent-copy-v%d", actionIndex),
		PrivacyPolicyVersion: "privacy-v1",
		PresentedAt:          syntheticNow.Add(-time.Minute),
		UIContext: UIContext{
			Surface: "web", Flow: "consent_center", UILanguage: "zh-CN",
		},
	}
}

func pointer[T any](value T) *T { return &value }

// TASK-011 DoD: key paths emit content-free low-cardinality telemetry, and a
// telemetry outage cannot change a committed authorization result.
func TestConsentObservationsAreContentFreeAndFailOpen(t *testing.T) {
	observer := &captureObserver{}
	store := NewMemoryStore()
	service, err := NewService(Dependencies{
		Store: store, Eligibility: fixedEligibility{allowed: true},
		Clock: &fixedClock{now: syntheticNow}, IDs: &sequenceIDs{}, Observer: observer,
	})
	if err != nil {
		t.Fatal(err)
	}
	actor := syntheticActor()
	ctx := context.Background()
	if _, err := service.Grant(ctx, actor, TypeModelTraining, GrantInput{
		Scope: Scope{}, Evidence: syntheticEvidence(1),
	}, "idem-observe-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Decide(ctx, actor, AccessRequest{Type: TypeModelTraining, Scope: Scope{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Withdraw(ctx, actor, TypeModelTraining, WithdrawalInput{
		Scope: Scope{}, Evidence: syntheticEvidence(2),
	}, "idem-observe-0002"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Decide(ctx, actor, AccessRequest{Type: TypeModelTraining, Scope: Scope{}}); err != nil {
		t.Fatal(err)
	}
	observations := observer.snapshot()
	if len(observations) != 4 || observations[0].Operation != operationGrant ||
		observations[1].Outcome != "allowed" || observations[2].Operation != operationWithdraw ||
		observations[3].Outcome != "denied" || observations[3].EffectiveStatus != EffectiveWithdrawn {
		t.Fatalf("unexpected observations: %+v", observations)
	}
	serialized := fmt.Sprint(observations)
	for _, forbidden := range []string{syntheticUserID, syntheticAssignID, "idem-observe", "consent-copy"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("observation leaked forbidden value %q: %s", forbidden, serialized)
		}
	}

	panicObserver := &captureObserver{panicOnWrite: true}
	panicService, panicStore, _ := newTestService(t, true)
	panicService.observer = panicObserver
	if _, err := panicService.Grant(ctx, actor, TypeProductAnalytics, GrantInput{
		Scope: Scope{}, Evidence: syntheticEvidence(3),
	}, "idem-observe-0003"); err != nil {
		t.Fatalf("telemetry outage affected grant: %v", err)
	}
	if stats := panicStore.Stats(); stats.Grants != 1 || stats.Audits != 1 {
		t.Fatalf("grant was not committed during telemetry outage: %+v", stats)
	}
}

// TASK-011 privacy rules: model training is off by default and a refusal does
// not alter the independent core-service grant.
func TestModelTrainingDefaultOffAndIndependentFromCore(t *testing.T) {
	service, store, _ := newTestService(t, true)
	ctx := context.Background()
	actor := syntheticActor()

	states, err := service.List(ctx, actor)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 6 {
		t.Fatalf("expected six default states, got %d", len(states))
	}
	for _, state := range states {
		if state.EffectiveStatus != EffectiveNotGranted || state.Version != 0 || state.Grant != nil {
			t.Fatalf("unexpected default state: %+v", state)
		}
	}
	modelDecision, err := service.Decide(ctx, actor, AccessRequest{Type: TypeModelTraining, Scope: Scope{}})
	if err != nil || modelDecision.Allowed || modelDecision.EffectiveStatus != EffectiveNotGranted {
		t.Fatalf("model training must default off: %+v %v", modelDecision, err)
	}

	if _, err := service.Grant(ctx, actor, TypeCoreService, GrantInput{
		Scope: Scope{}, Evidence: syntheticEvidence(1),
	}, "idem-core-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Grant(ctx, actor, TypeModelTraining, GrantInput{
		Scope: Scope{}, Evidence: syntheticEvidence(2),
	}, "idem-model-0001"); err != nil {
		t.Fatal(err)
	}
	withdrawn, err := service.Withdraw(ctx, actor, TypeModelTraining, WithdrawalInput{
		Scope: Scope{}, Evidence: syntheticEvidence(3),
	}, "idem-model-0002")
	if err != nil || withdrawn.Version != 2 || withdrawn.Status != StatusWithdrawn {
		t.Fatalf("model withdrawal failed: %+v %v", withdrawn, err)
	}
	coreDecision, err := service.Decide(ctx, actor, AccessRequest{Type: TypeCoreService, Scope: Scope{}})
	if err != nil || !coreDecision.Allowed || coreDecision.EffectiveStatus != EffectiveGranted {
		t.Fatalf("model refusal must not affect core: %+v %v", coreDecision, err)
	}
	modelDecision, err = service.Decide(ctx, actor, AccessRequest{Type: TypeModelTraining, Scope: Scope{}})
	if err != nil || modelDecision.Allowed || modelDecision.EffectiveStatus != EffectiveWithdrawn {
		t.Fatalf("withdrawn model consent must deny: %+v %v", modelDecision, err)
	}
	if stats := store.Stats(); stats.Grants != 3 || stats.Audits != 3 {
		t.Fatalf("unexpected append-only counts: %+v", stats)
	}
}

// Normal path: every PRD category accepts only its own typed scope and remains
// independently visible in the consent center.
func TestGrantSixIndependentTypes(t *testing.T) {
	service, store, _ := newTestService(t, true)
	actor := syntheticActor()
	rows := []struct {
		consentType Type
		scope       Scope
		expiresAt   *time.Time
	}{
		{TypeCoreService, Scope{}, nil},
		{TypeRawAVRecording, Scope{MediaCategories: []MediaCategory{MediaVideo, MediaAudio}}, pointer(syntheticNow.Add(30 * 24 * time.Hour))},
		{TypeOrgSharing, Scope{AssignmentID: pointer(syntheticAssignID), DataCategories: []DataCategory{DataRadar, DataTotalScore}}, pointer(syntheticNow.Add(20 * 24 * time.Hour))},
		{TypeProductAnalytics, Scope{}, nil},
		{TypeModelTraining, Scope{}, nil},
		{TypeMarketing, Scope{Channels: []Channel{ChannelPush, ChannelEmail}}, nil},
	}
	for index, row := range rows {
		grant, err := service.Grant(context.Background(), actor, row.consentType, GrantInput{
			Scope: row.scope, ExpiresAt: row.expiresAt, Evidence: syntheticEvidence(index + 1),
		}, fmt.Sprintf("idem-six-%04d", index))
		if err != nil {
			t.Fatalf("grant %s failed: %v", row.consentType, err)
		}
		if grant.Type != row.consentType || grant.Version != 1 || grant.Status != StatusGranted || len(grant.Evidence.EvidenceHash) != 64 {
			t.Fatalf("unexpected grant: %+v", grant)
		}
	}
	states, err := service.List(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 6 {
		t.Fatalf("expected six current states, got %d", len(states))
	}
	for _, state := range states {
		if state.EffectiveStatus != EffectiveGranted || state.Grant == nil {
			t.Fatalf("category is not independently granted: %+v", state)
		}
	}
	if stats := store.Stats(); stats.Grants != 6 || stats.Audits != 6 {
		t.Fatalf("unexpected append-only counts: %+v", stats)
	}
}

// Required acceptance path: a completed withdrawal is immediately visible to
// online authorization and both evidence versions plus the audit remain append-only.
func TestWithdrawalImmediatelyDeniesAndWritesAudit(t *testing.T) {
	service, store, _ := newTestService(t, true)
	actor := syntheticActor()
	scope := Scope{AssignmentID: pointer(syntheticAssignID), DataCategories: []DataCategory{DataRadar}}
	expiresAt := syntheticNow.Add(30 * 24 * time.Hour)
	granted, err := service.Grant(context.Background(), actor, TypeOrgSharing, GrantInput{
		Scope: scope, ExpiresAt: &expiresAt, Evidence: syntheticEvidence(1),
	}, "idem-share-0001")
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.Decide(context.Background(), actor, AccessRequest{Type: TypeOrgSharing, Scope: scope})
	if err != nil || !before.Allowed {
		t.Fatalf("granted access must be allowed: %+v %v", before, err)
	}

	withdrawn, err := service.Withdraw(context.Background(), actor, TypeOrgSharing, WithdrawalInput{
		Scope: scope, Evidence: syntheticEvidence(2),
	}, "idem-share-0002")
	if err != nil {
		t.Fatal(err)
	}
	if withdrawn.Version != 2 || withdrawn.SupersedesGrantID == nil || *withdrawn.SupersedesGrantID != granted.GrantID {
		t.Fatalf("withdrawal version chain invalid: %+v", withdrawn)
	}
	after, err := service.Decide(context.Background(), actor, AccessRequest{Type: TypeOrgSharing, Scope: scope})
	if err != nil || after.Allowed || after.EffectiveStatus != EffectiveWithdrawn || after.GrantID == nil || *after.GrantID != withdrawn.GrantID {
		t.Fatalf("withdrawal must deny immediately: %+v %v", after, err)
	}
	history, err := service.History(context.Background(), actor, TypeOrgSharing)
	if err != nil || len(history) != 2 || history[0].Status != StatusGranted || history[1].Status != StatusWithdrawn {
		t.Fatalf("append-only history invalid: %+v %v", history, err)
	}
	if history[0].GrantID != granted.GrantID || history[0].WithdrawnAt != nil {
		t.Fatalf("historical grant was modified: %+v", history[0])
	}
	audits := store.AuditsBySubject(actor.UserID)
	if len(audits) != 2 || audits[1].Action != "consent.withdrawn" || audits[1].ResourceID != withdrawn.GrantID {
		t.Fatalf("withdrawal audit missing: %+v", audits)
	}
}

// Retry path: an audit write failure rolls back the withdrawal version; the
// same idempotency key can retry once and commits exactly one version/audit.
func TestWithdrawalAuditFailureRollsBackAndRetries(t *testing.T) {
	baseStore := NewMemoryStore()
	store := &failingAuditStore{Store: baseStore}
	clock := &fixedClock{now: syntheticNow}
	service, err := NewService(Dependencies{
		Store: store, Eligibility: fixedEligibility{allowed: true}, Clock: clock, IDs: &sequenceIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	actor := syntheticActor()
	if _, err := service.Grant(context.Background(), actor, TypeCoreService, GrantInput{
		Scope: Scope{}, Evidence: syntheticEvidence(1),
	}, "idem-retry-0001"); err != nil {
		t.Fatal(err)
	}
	store.failNextAudit()
	_, err = service.Withdraw(context.Background(), actor, TypeCoreService, WithdrawalInput{
		Scope: Scope{}, Evidence: syntheticEvidence(2),
	}, "idem-retry-0002")
	domain := AsDomainError(err)
	if domain.Code != CodeInternal || !domain.Retryable {
		t.Fatalf("audit failure must be retryable internal error: %+v", domain)
	}
	decision, err := service.Decide(context.Background(), actor, AccessRequest{Type: TypeCoreService, Scope: Scope{}})
	if err != nil || !decision.Allowed {
		t.Fatalf("failed transaction must retain prior grant: %+v %v", decision, err)
	}
	if stats := baseStore.Stats(); stats.Grants != 1 || stats.Audits != 1 {
		t.Fatalf("failed transaction leaked a side effect: %+v", stats)
	}
	if _, err := service.Withdraw(context.Background(), actor, TypeCoreService, WithdrawalInput{
		Scope: Scope{}, Evidence: syntheticEvidence(2),
	}, "idem-retry-0002"); err != nil {
		t.Fatalf("same-key retry failed: %v", err)
	}
	decision, err = service.Decide(context.Background(), actor, AccessRequest{Type: TypeCoreService, Scope: Scope{}})
	if err != nil || decision.Allowed {
		t.Fatalf("successful retry must deny access: %+v %v", decision, err)
	}
	if stats := baseStore.Stats(); stats.Grants != 2 || stats.Audits != 2 {
		t.Fatalf("retry duplicated side effects: %+v", stats)
	}
}

// Idempotency/concurrency: all concurrent copies return one stable result and
// a changed payload under the same key is rejected without another side effect.
func TestConcurrentGrantIdempotencyAndConflict(t *testing.T) {
	service, store, _ := newTestService(t, true)
	actor := syntheticActor()
	input := GrantInput{Scope: Scope{}, Evidence: syntheticEvidence(1)}
	const workers = 16
	results := make(chan Grant, workers)
	errorsCh := make(chan error, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			grant, err := service.Grant(context.Background(), actor, TypeCoreService, input, "idem-concurrent-0001")
			results <- grant
			errorsCh <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errorsCh)
	var firstID string
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent grant failed: %v", err)
		}
	}
	for grant := range results {
		if firstID == "" {
			firstID = grant.GrantID
		}
		if grant.GrantID != firstID {
			t.Fatalf("idempotent results differ: %q != %q", grant.GrantID, firstID)
		}
	}
	if stats := store.Stats(); stats.Grants != 1 || stats.Audits != 1 {
		t.Fatalf("concurrent side effects duplicated: %+v", stats)
	}
	changed := input
	changed.Evidence = syntheticEvidence(2)
	_, err := service.Grant(context.Background(), actor, TypeCoreService, changed, "idem-concurrent-0001")
	if ErrorCodeOf(err) != CodeIdempotencyConflict {
		t.Fatalf("same key with changed request must conflict: %v", err)
	}
	if stats := store.Stats(); stats.Grants != 1 || stats.Audits != 1 {
		t.Fatalf("conflict produced a side effect: %+v", stats)
	}
}

// Idempotent retries return the committed raw-recording result before mutable
// expiry/identity policy checks, while every live access decision rechecks the
// current adult status and fails closed.
func TestRawRecordingRetryAndLiveEligibility(t *testing.T) {
	store := NewMemoryStore()
	clock := &fixedClock{now: syntheticNow}
	eligibility := &mutableEligibility{allowed: true}
	service, err := NewService(Dependencies{
		Store: store, Eligibility: eligibility, Clock: clock, IDs: &sequenceIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	actor := syntheticActor()
	expiresAt := syntheticNow.Add(time.Hour)
	input := GrantInput{
		Scope:     Scope{MediaCategories: []MediaCategory{MediaAudio}},
		ExpiresAt: &expiresAt, Evidence: syntheticEvidence(1),
	}
	first, err := service.Grant(context.Background(), actor, TypeRawAVRecording, input, "idem-raw-retry-0001")
	if err != nil || eligibility.callCount() != 1 {
		t.Fatalf("initial raw grant failed: %+v %v calls=%d", first, err, eligibility.callCount())
	}
	eligibility.set(false, nil)
	if _, err := service.Decide(context.Background(), actor, AccessRequest{
		Type: TypeRawAVRecording, Scope: input.Scope,
	}); ErrorCodeOf(err) != CodeForbidden {
		t.Fatalf("live minor policy must deny raw access: %v", err)
	}
	if eligibility.callCount() != 2 {
		t.Fatalf("live decision did not recheck identity age: %d", eligibility.callCount())
	}

	clock.set(syntheticNow.Add(2 * time.Hour))
	eligibility.set(false, errors.New("synthetic identity outage"))
	replayed, err := service.Grant(context.Background(), actor, TypeRawAVRecording, input, "idem-raw-retry-0001")
	if err != nil || replayed.GrantID != first.GrantID {
		t.Fatalf("same-key retry did not return committed result: %+v %v", replayed, err)
	}
	if eligibility.callCount() != 2 {
		t.Fatalf("idempotent replay called mutable eligibility: %d", eligibility.callCount())
	}
	if stats := store.Stats(); stats.Grants != 1 || stats.Audits != 1 {
		t.Fatalf("idempotent raw retry duplicated side effects: %+v", stats)
	}
}

// Abnormal paths fail closed before persistence, including the PRD ban on raw
// recording for minors.
func TestInvalidScopesEvidenceExpiryAndMinorRecording(t *testing.T) {
	service, store, _ := newTestService(t, true)
	actor := syntheticActor()
	rows := []struct {
		name        string
		consentType Type
		input       GrantInput
		key         string
	}{
		{"invalid type", Type("unknown"), GrantInput{Scope: Scope{}, Evidence: syntheticEvidence(1)}, "idem-invalid-0001"},
		{"core with scope", TypeCoreService, GrantInput{Scope: Scope{Channels: []Channel{ChannelEmail}}, Evidence: syntheticEvidence(1)}, "idem-invalid-0002"},
		{"raw without expiry", TypeRawAVRecording, GrantInput{Scope: Scope{MediaCategories: []MediaCategory{MediaAudio}}, Evidence: syntheticEvidence(1)}, "idem-invalid-0003"},
		{"org without assignment", TypeOrgSharing, GrantInput{Scope: Scope{DataCategories: []DataCategory{DataRadar}}, ExpiresAt: pointer(syntheticNow.Add(time.Hour)), Evidence: syntheticEvidence(1)}, "idem-invalid-0004"},
		{"marketing without channel", TypeMarketing, GrantInput{Scope: Scope{}, Evidence: syntheticEvidence(1)}, "idem-invalid-0005"},
		{"past expiry", TypeProductAnalytics, GrantInput{Scope: Scope{}, ExpiresAt: pointer(syntheticNow.Add(-time.Second)), Evidence: syntheticEvidence(1)}, "idem-invalid-0006"},
		{"bad ui context", TypeProductAnalytics, GrantInput{Scope: Scope{}, Evidence: EvidenceInput{CopyVersion: "v1", PrivacyPolicyVersion: "v1", PresentedAt: syntheticNow, UIContext: UIContext{Surface: "unknown", Flow: "consent_center", UILanguage: "zh-CN"}}}, "idem-invalid-0007"},
		{"free text evidence version", TypeProductAnalytics, GrantInput{Scope: Scope{}, Evidence: EvidenceInput{CopyVersion: "unsafe version", PrivacyPolicyVersion: "v1", PresentedAt: syntheticNow, UIContext: UIContext{Surface: "web", Flow: "consent_center", UILanguage: "zh-CN"}}}, "idem-invalid-0008"},
		{"future evidence", TypeProductAnalytics, GrantInput{Scope: Scope{}, Evidence: EvidenceInput{CopyVersion: "v1", PrivacyPolicyVersion: "v1", PresentedAt: syntheticNow.Add(6 * time.Minute), UIContext: UIContext{Surface: "web", Flow: "consent_center", UILanguage: "zh-CN"}}}, "idem-invalid-0009"},
		{"short key", TypeProductAnalytics, GrantInput{Scope: Scope{}, Evidence: syntheticEvidence(1)}, "short"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			if _, err := service.Grant(context.Background(), actor, row.consentType, row.input, row.key); ErrorCodeOf(err) != CodeValidationFailed {
				t.Fatalf("expected validation failure, got %v", err)
			}
		})
	}
	if stats := store.Stats(); stats.Grants != 0 || stats.Audits != 0 {
		t.Fatalf("invalid requests were persisted: %+v", stats)
	}

	minorService, minorStore, _ := newTestService(t, false)
	_, err := minorService.Grant(context.Background(), actor, TypeRawAVRecording, GrantInput{
		Scope:     Scope{MediaCategories: []MediaCategory{MediaAudio}},
		ExpiresAt: pointer(syntheticNow.Add(24 * time.Hour)), Evidence: syntheticEvidence(1),
	}, "idem-minor-0001")
	if ErrorCodeOf(err) != CodeForbidden {
		t.Fatalf("minor recording must be forbidden: %v", err)
	}
	if stats := minorStore.Stats(); stats.Grants != 0 || stats.Audits != 0 {
		t.Fatalf("minor denial persisted data: %+v", stats)
	}
	if _, err := service.Withdraw(context.Background(), actor, TypeCoreService, WithdrawalInput{
		Scope: Scope{}, Evidence: syntheticEvidence(1),
	}, "idem-missing-0001"); ErrorCodeOf(err) != CodeNotFound {
		t.Fatalf("missing withdrawal must be not_found: %v", err)
	}
}

func TestExpiredGrantDeniesOnlineAccess(t *testing.T) {
	service, _, clock := newTestService(t, true)
	actor := syntheticActor()
	expiresAt := syntheticNow.Add(time.Hour)
	if _, err := service.Grant(context.Background(), actor, TypeProductAnalytics, GrantInput{
		Scope: Scope{}, ExpiresAt: &expiresAt, Evidence: syntheticEvidence(1),
	}, "idem-expiry-0001"); err != nil {
		t.Fatal(err)
	}
	clock.set(syntheticNow.Add(2 * time.Hour))
	decision, err := service.Decide(context.Background(), actor, AccessRequest{Type: TypeProductAnalytics, Scope: Scope{}})
	if err != nil || decision.Allowed || decision.EffectiveStatus != EffectiveExpired {
		t.Fatalf("expired grant must deny: %+v %v", decision, err)
	}
}

type failingAuditStore struct {
	Store
	mu       sync.Mutex
	failures int
}

func (s *failingAuditStore) failNextAudit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures++
}

func (s *failingAuditStore) Transact(ctx context.Context, fn func(Tx) error) error {
	return s.Store.Transact(ctx, func(tx Tx) error {
		return fn(&failingAuditTx{Tx: tx, parent: s})
	})
}

type failingAuditTx struct {
	Tx
	parent *failingAuditStore
}

func (tx *failingAuditTx) AppendAudit(event AuditEvent) error {
	tx.parent.mu.Lock()
	defer tx.parent.mu.Unlock()
	if tx.parent.failures > 0 {
		tx.parent.failures--
		return errors.New("synthetic audit outage")
	}
	return tx.Tx.AppendAudit(event)
}
