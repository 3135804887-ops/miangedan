package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	identityprovider "miangedan/services/identity/provider"
	"miangedan/services/notify"
)

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(duration time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(duration)
	c.mu.Unlock()
}

type testIDs struct {
	mu      sync.Mutex
	counter int
}

func (g *testIDs) NewID() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counter++
	return fmt.Sprintf("00000000-0000-4000-8000-%012d", g.counter), nil
}

type testNotifier struct {
	mu       sync.Mutex
	messages map[string]notify.Message
	fail     bool
}

func newTestNotifier() *testNotifier {
	return &testNotifier{messages: make(map[string]notify.Message)}
}

func (n *testNotifier) Send(_ context.Context, message notify.Message) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.fail {
		return errors.New("synthetic notification outage")
	}
	if err := message.Validate(message.DataRegion); err != nil {
		return err
	}
	if _, exists := n.messages[message.IdempotencyKey]; !exists {
		n.messages[message.IdempotencyKey] = message
	}
	return nil
}

func (n *testNotifier) codeFor(key string) string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.messages["identity-otp-"+key].Variables["otp_code"]
}

func (n *testNotifier) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.messages)
}

type testOAuthAdapter struct {
	mu       sync.Mutex
	subjects map[string]string
	err      error
	calls    int
}

func (a *testOAuthAdapter) Verify(_ context.Context, request identityprovider.VerifyRequest) (identityprovider.VerifiedSubject, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls++
	if a.err != nil {
		return identityprovider.VerifiedSubject{}, a.err
	}
	subject, exists := a.subjects[request.AuthorizationCode]
	if !exists {
		return identityprovider.VerifiedSubject{}, identityprovider.ErrInvalidCredential
	}
	return identityprovider.VerifiedSubject{Subject: subject, VerifiedAt: testNow}, nil
}

func (a *testOAuthAdapter) callCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls
}

type rejectRisk struct{}

func (rejectRisk) Evaluate(context.Context, RiskRequest) error {
	return errors.New("synthetic risk rejection")
}

var testNow = time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)

type testHarness struct {
	service  *Service
	store    *MemoryStore
	notifier *testNotifier
	clock    *testClock
	google   *testOAuthAdapter
	apple    *testOAuthAdapter
	wechat   *testOAuthAdapter
}

func newHarness(t *testing.T, mutate func(*Config, *Dependencies)) testHarness {
	t.Helper()
	store := NewMemoryStore()
	notifier := newTestNotifier()
	clock := &testClock{now: testNow}
	google := &testOAuthAdapter{subjects: map[string]string{
		"synthetic-google-a":       "synthetic-google-subject-a",
		"synthetic-google-b":       "synthetic-google-subject-b",
		"synthetic-google-b-again": "synthetic-google-subject-b",
	}}
	apple := &testOAuthAdapter{subjects: map[string]string{"synthetic-apple-a": "synthetic-apple-subject-a"}}
	wechat := &testOAuthAdapter{subjects: map[string]string{"synthetic-wechat-a": "synthetic-wechat-subject-a"}}
	registry, err := identityprovider.NewRegistry(map[string]identityprovider.Adapter{
		identityprovider.Google: google,
		identityprovider.Apple:  apple,
		identityprovider.WeChat: wechat,
	})
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	dependencies := Dependencies{
		Store:       store,
		Idempotency: NewMemoryIdempotency(),
		Notifier:    notifier,
		Providers:   registry,
		Clock:       clock,
		IDs:         &testIDs{},
		Secrets: SecretMaterial{
			OTPKey:     bytes.Repeat([]byte{0x11}, 32),
			SubjectKey: bytes.Repeat([]byte{0x22}, 32),
			ProofKey:   bytes.Repeat([]byte{0x33}, 32),
			SigningKey: bytes.Repeat([]byte{0x44}, 32),
			RefreshKey: bytes.Repeat([]byte{0x55}, 32),
		},
	}
	if mutate != nil {
		mutate(&config, &dependencies)
	}
	service, err := NewService(config, dependencies)
	if err != nil {
		t.Fatalf("create identity service: %v", err)
	}
	return testHarness{service: service, store: store, notifier: notifier, clock: clock, google: google, apple: apple, wechat: wechat}
}

func syntheticRegistration(now time.Time) *Registration {
	return &Registration{
		UILanguage: LanguageZH,
		AgeStatus:  AgeAdult,
		Evidence: RegistrationEvidence{
			TermsVersion:          "terms-synthetic-v1",
			PrivacyVersion:        "privacy-synthetic-v1",
			DataProcessingVersion: "processing-synthetic-v1",
			AcceptedAt:            now,
			Context:               AcceptanceContext{UILanguage: LanguageZH, Surface: "web"},
		},
	}
}

func verifiedEmailProof(t *testing.T, harness testHarness, email, regionCode, keyPrefix string) VerificationProof {
	t.Helper()
	challengeKey := keyPrefix + "-challenge"
	challenge, err := harness.service.RequestEmailChallenge(context.Background(), RequestEmailChallengeInput{
		Email: email, DataRegion: regionCode,
	}, challengeKey)
	if err != nil {
		t.Fatalf("request email challenge: %v", err)
	}
	proof, err := harness.service.VerifyEmailChallenge(context.Background(), VerifyEmailChallengeInput{
		ChallengeID: challenge.ChallengeID,
		Code:        harness.notifier.codeFor(challengeKey),
		DataRegion:  regionCode,
	}, keyPrefix+"-verify")
	if err != nil {
		t.Fatalf("verify email challenge: %v", err)
	}
	return proof
}

func loginWithEmail(t *testing.T, harness testHarness, email, regionCode, keyPrefix string) Session {
	t.Helper()
	proof := verifiedEmailProof(t, harness, email, regionCode, keyPrefix)
	session, err := harness.service.CreateSession(context.Background(), CreateSessionInput{
		ProofToken: proof.ProofToken, DataRegion: regionCode, Registration: syntheticRegistration(harness.clock.Now()),
	}, keyPrefix+"-session")
	if err != nil {
		t.Fatalf("create email session: %v", err)
	}
	return session
}

// TC-FR-027-N01 / TC-NFR-006-N01: email login is complete and every write is idempotent.
func TestEmailLoginAndIdempotency(t *testing.T) {
	harness := newHarness(t, nil)
	ctx := context.Background()
	input := RequestEmailChallengeInput{Email: "synthetic.user@example.com", DataRegion: "intl"}
	firstChallenge, err := harness.service.RequestEmailChallenge(ctx, input, "email-01-challenge")
	if err != nil {
		t.Fatal(err)
	}
	secondChallenge, err := harness.service.RequestEmailChallenge(ctx, input, "email-01-challenge")
	if err != nil {
		t.Fatal(err)
	}
	if firstChallenge != secondChallenge || harness.notifier.count() != 1 || harness.store.Stats().Verifications != 1 {
		t.Fatalf("challenge retry produced duplicate side effect: first=%+v second=%+v stats=%+v notifications=%d", firstChallenge, secondChallenge, harness.store.Stats(), harness.notifier.count())
	}
	verifyInput := VerifyEmailChallengeInput{
		ChallengeID: firstChallenge.ChallengeID,
		Code:        harness.notifier.codeFor("email-01-challenge"),
		DataRegion:  "intl",
	}
	firstProof, err := harness.service.VerifyEmailChallenge(ctx, verifyInput, "email-01-verify")
	if err != nil {
		t.Fatal(err)
	}
	secondProof, err := harness.service.VerifyEmailChallenge(ctx, verifyInput, "email-01-verify")
	if err != nil || firstProof != secondProof {
		t.Fatalf("verification retry must return first result: %v %+v %+v", err, firstProof, secondProof)
	}
	createInput := CreateSessionInput{ProofToken: firstProof.ProofToken, DataRegion: "intl", Registration: syntheticRegistration(harness.clock.Now())}
	firstSession, err := harness.service.CreateSession(ctx, createInput, "email-01-session")
	if err != nil {
		t.Fatal(err)
	}
	secondSession, err := harness.service.CreateSession(ctx, createInput, "email-01-session")
	if err != nil || firstSession.SessionID != secondSession.SessionID || firstSession.RefreshToken != secondSession.RefreshToken {
		t.Fatalf("session retry must return first result: %v", err)
	}
	stats := harness.store.Stats()
	if stats.Users != 1 || stats.Identities != 1 || stats.Sessions != 1 {
		t.Fatalf("unexpected duplicate account/session: %+v", stats)
	}
	claims, err := harness.service.Authenticate(firstSession.AccessToken)
	if err != nil || claims.UserID != firstSession.Account.UserID || claims.DataRegion != "intl" {
		t.Fatalf("business token claims mismatch: %+v %v", claims, err)
	}
	encoded, err := json.Marshal(firstSession.Account)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("synthetic.user@example.com")) || bytes.Contains(encoded, []byte("provider_subject")) {
		t.Fatalf("public account leaked identity subject: %s", encoded)
	}
}

// TC-FR-027-A01: expiry, bad codes, lockout, rate limiting and risk checks fail closed.
func TestEmailVerificationFailurePaths(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		harness := newHarness(t, nil)
		challenge, err := harness.service.RequestEmailChallenge(context.Background(), RequestEmailChallengeInput{
			Email: "synthetic.expired@example.com", DataRegion: "eu",
		}, "expired-challenge")
		if err != nil {
			t.Fatal(err)
		}
		harness.clock.Advance(11 * time.Minute)
		_, err = harness.service.VerifyEmailChallenge(context.Background(), VerifyEmailChallengeInput{
			ChallengeID: challenge.ChallengeID,
			Code:        harness.notifier.codeFor("expired-challenge"),
			DataRegion:  "eu",
		}, "expired-verify")
		if ErrorCodeOf(err) != CodeVerificationExpired {
			t.Fatalf("expected expiry, got %v", err)
		}
	})

	t.Run("attempt lockout", func(t *testing.T) {
		harness := newHarness(t, func(config *Config, _ *Dependencies) { config.MaxVerificationAttempts = 2 })
		challenge, err := harness.service.RequestEmailChallenge(context.Background(), RequestEmailChallengeInput{
			Email: "synthetic.locked@example.com", DataRegion: "cn",
		}, "locked-challenge")
		if err != nil {
			t.Fatal(err)
		}
		invalidCode := "000000"
		if invalidCode == harness.notifier.codeFor("locked-challenge") {
			invalidCode = "999999"
		}
		for index := 0; index < 2; index++ {
			_, err = harness.service.VerifyEmailChallenge(context.Background(), VerifyEmailChallengeInput{
				ChallengeID: challenge.ChallengeID, Code: invalidCode, DataRegion: "cn",
			}, fmt.Sprintf("locked-verify-%d", index))
			if ErrorCodeOf(err) != CodeVerificationInvalid {
				t.Fatalf("expected invalid verification, got %v", err)
			}
		}
		_, err = harness.service.VerifyEmailChallenge(context.Background(), VerifyEmailChallengeInput{
			ChallengeID: challenge.ChallengeID,
			Code:        harness.notifier.codeFor("locked-challenge"),
			DataRegion:  "cn",
		}, "locked-verify-correct")
		if ErrorCodeOf(err) != CodeVerificationInvalid {
			t.Fatalf("locked challenge must reject correct code, got %v", err)
		}
	})

	t.Run("rate limit", func(t *testing.T) {
		harness := newHarness(t, func(config *Config, _ *Dependencies) { config.MaxEmailChallenges = 1 })
		input := RequestEmailChallengeInput{Email: "synthetic.rate@example.com", DataRegion: "intl"}
		if _, err := harness.service.RequestEmailChallenge(context.Background(), input, "rate-challenge-1"); err != nil {
			t.Fatal(err)
		}
		_, err := harness.service.RequestEmailChallenge(context.Background(), input, "rate-challenge-2")
		if ErrorCodeOf(err) != CodeRateLimited || harness.notifier.count() != 1 {
			t.Fatalf("rate limit must prevent second send: %v", err)
		}
	})

	t.Run("risk rejection", func(t *testing.T) {
		harness := newHarness(t, func(_ *Config, dependencies *Dependencies) { dependencies.Risk = rejectRisk{} })
		_, err := harness.service.RequestEmailChallenge(context.Background(), RequestEmailChallengeInput{
			Email: "synthetic.risk@example.com", DataRegion: "intl",
		}, "risk-challenge")
		if ErrorCodeOf(err) != CodeRiskVerificationRequired || harness.notifier.count() != 0 || harness.store.Stats().Verifications != 0 {
			t.Fatalf("risk rejection must have zero side effects: %v %+v", err, harness.store.Stats())
		}
	})
}

// TC-FR-027-N01/A01: provider matrix is enforced and outages advertise email fallback.
func TestOAuthRegionMatrixAndFallback(t *testing.T) {
	harness := newHarness(t, nil)
	if _, err := harness.service.RequestEmailChallenge(context.Background(), RequestEmailChallengeInput{
		Email: "synthetic.shared-key@example.com", DataRegion: "intl",
	}, "shared-provider-key"); err != nil {
		t.Fatal(err)
	}
	proof, err := harness.service.VerifyOAuth(context.Background(), VerifyOAuthInput{
		Provider: ProviderGoogle, AuthorizationCode: "synthetic-google-a", DataRegion: "intl",
	}, "shared-provider-key")
	if err != nil || proof.Provider != ProviderGoogle {
		t.Fatalf("Google intl verification should succeed: %+v %v", proof, err)
	}
	before := harness.google.callCount()
	_, err = harness.service.VerifyOAuth(context.Background(), VerifyOAuthInput{
		Provider: ProviderGoogle, AuthorizationCode: "synthetic-google-a", DataRegion: "cn",
	}, "oauth-google-cn")
	if ErrorCodeOf(err) != CodeValidationFailed || harness.google.callCount() != before {
		t.Fatalf("cn Google must be rejected before adapter: %v", err)
	}
	harness.apple.err = identityprovider.ErrUnavailable
	_, err = harness.service.VerifyOAuth(context.Background(), VerifyOAuthInput{
		Provider: ProviderApple, AuthorizationCode: "synthetic-apple-a", DataRegion: "eu",
	}, "oauth-apple-outage")
	domain := AsDomainError(err)
	if domain.Code != CodeProviderUnavailable || domain.Details["email_fallback_available"] != true {
		t.Fatalf("outage must offer email fallback: %+v", domain)
	}
}

// TC-US-05-A02 / scenario 4: both sides are independently verified before binding.
func TestDualProofBindingSuccessAndIdempotency(t *testing.T) {
	harness := newHarness(t, nil)
	session := loginWithEmail(t, harness, "synthetic.bind@example.com", "intl", "bind-login")
	claims, err := harness.service.Authenticate(session.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	source := verifiedEmailProof(t, harness, "synthetic.bind@example.com", "intl", "bind-source")
	target, err := harness.service.VerifyOAuth(context.Background(), VerifyOAuthInput{
		Provider: ProviderGoogle, AuthorizationCode: "synthetic-google-a", DataRegion: "intl",
	}, "bind-target-oauth")
	if err != nil {
		t.Fatal(err)
	}
	input := BindIdentityInput{SourceProofToken: source.ProofToken, TargetProofToken: target.ProofToken}
	first, err := harness.service.BindIdentity(context.Background(), claims, input, "bind-operation")
	if err != nil {
		t.Fatal(err)
	}
	second, err := harness.service.BindIdentity(context.Background(), claims, input, "bind-operation")
	if err != nil || first != second || harness.store.Stats().Identities != 2 {
		t.Fatalf("binding retry must not duplicate identity: first=%+v second=%+v err=%v stats=%+v", first, second, err, harness.store.Stats())
	}
	_, err = harness.service.BindIdentity(context.Background(), claims, BindIdentityInput{
		SourceProofToken: source.ProofToken,
	}, "bind-missing-target")
	if ErrorCodeOf(err) != CodeValidationFailed {
		t.Fatalf("one-sided proof must be rejected: %v", err)
	}
}

// TC-US-05-A02: a target identity owned by another account never triggers merge.
func TestIdentityConflictNeverMergesAccounts(t *testing.T) {
	harness := newHarness(t, nil)
	accountA := loginWithEmail(t, harness, "synthetic.account-a@example.com", "intl", "conflict-a")
	proofB, err := harness.service.VerifyOAuth(context.Background(), VerifyOAuthInput{
		Provider: ProviderGoogle, AuthorizationCode: "synthetic-google-b", DataRegion: "intl",
	}, "conflict-b-oauth-login")
	if err != nil {
		t.Fatal(err)
	}
	accountB, err := harness.service.CreateSession(context.Background(), CreateSessionInput{
		ProofToken: proofB.ProofToken, DataRegion: "intl", Registration: syntheticRegistration(harness.clock.Now()),
	}, "conflict-b-session")
	if err != nil {
		t.Fatal(err)
	}
	claimsA, err := harness.service.Authenticate(accountA.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	sourceA := verifiedEmailProof(t, harness, "synthetic.account-a@example.com", "intl", "conflict-a-source")
	targetB, err := harness.service.VerifyOAuth(context.Background(), VerifyOAuthInput{
		Provider: ProviderGoogle, AuthorizationCode: "synthetic-google-b-again", DataRegion: "intl",
	}, "conflict-b-target")
	if err != nil {
		t.Fatal(err)
	}
	_, err = harness.service.BindIdentity(context.Background(), claimsA, BindIdentityInput{
		SourceProofToken: sourceA.ProofToken,
		TargetProofToken: targetB.ProofToken,
	}, "conflict-bind")
	domain := AsDomainError(err)
	if domain.Code != CodeIdentityConflict || domain.Details["accounts_merged"] != false {
		t.Fatalf("expected no-merge conflict, got %+v", domain)
	}
	stats := harness.store.Stats()
	if stats.Users != 2 || stats.Identities != 2 || stats.RecoveryCases != 1 {
		t.Fatalf("conflict changed account ownership: %+v", stats)
	}
	accountAView, err := harness.service.GetAccount(context.Background(), claimsA)
	if err != nil {
		t.Fatal(err)
	}
	claimsB, err := harness.service.Authenticate(accountB.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	accountBView, err := harness.service.GetAccount(context.Background(), claimsB)
	if err != nil {
		t.Fatal(err)
	}
	if accountAView.UserID == accountBView.UserID || len(accountAView.Identities) != 1 || len(accountBView.Identities) != 1 {
		t.Fatalf("accounts were merged or identities moved: A=%+v B=%+v", accountAView, accountBView)
	}
}

// US-05 rule 10 / NFR-006: refresh rotates once and duplicate retries are stable.
func TestRefreshRotationAndIdempotency(t *testing.T) {
	harness := newHarness(t, nil)
	session := loginWithEmail(t, harness, "synthetic.refresh@example.com", "eu", "refresh-login")
	input := RefreshSessionInput{RefreshToken: session.RefreshToken, DataRegion: "eu"}
	first, err := harness.service.RefreshSession(context.Background(), input, "refresh-operation")
	if err != nil {
		t.Fatal(err)
	}
	second, err := harness.service.RefreshSession(context.Background(), input, "refresh-operation")
	if err != nil || first.SessionID != second.SessionID || first.RefreshToken != second.RefreshToken {
		t.Fatalf("refresh retry must return first rotation: %v", err)
	}
	_, err = harness.service.RefreshSession(context.Background(), input, "refresh-old-token-again")
	if ErrorCodeOf(err) != CodeUnauthorized {
		t.Fatalf("old refresh token must be invalid: %v", err)
	}
	if harness.store.Stats().Sessions != 2 {
		t.Fatalf("expected original + replacement sessions: %+v", harness.store.Stats())
	}
}

// TC-NFR-006-A01: concurrent retries have one notification and one challenge.
func TestConcurrentChallengeIdempotency(t *testing.T) {
	harness := newHarness(t, nil)
	input := RequestEmailChallengeInput{Email: "synthetic.concurrent@example.com", DataRegion: "cn"}
	const workers = 12
	results := make(chan VerificationChallenge, workers)
	errorsChannel := make(chan error, workers)
	var wait sync.WaitGroup
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := harness.service.RequestEmailChallenge(context.Background(), input, "concurrent-challenge")
			results <- result
			errorsChannel <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("concurrent retry failed: %v", err)
		}
	}
	var challengeID string
	for result := range results {
		if challengeID == "" {
			challengeID = result.ChallengeID
		}
		if result.ChallengeID != challengeID {
			t.Fatalf("concurrent retries returned different challenges: %s != %s", result.ChallengeID, challengeID)
		}
	}
	if harness.notifier.count() != 1 || harness.store.Stats().Verifications != 1 {
		t.Fatalf("concurrent retries duplicated side effects: notifications=%d stats=%+v", harness.notifier.count(), harness.store.Stats())
	}
}
