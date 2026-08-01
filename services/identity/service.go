package identity

import (
	"context"
	"crypto/hmac"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	identityprovider "miangedan/services/identity/provider"
	"miangedan/services/notify"
	"miangedan/services/region"
)

const emailTemplateID = "identity_email_otp_v1"

// RequestEmailChallengeInput starts an email verification without persisting
// the address in plaintext.
type RequestEmailChallengeInput struct {
	Email      string `json:"email"`
	DataRegion string `json:"data_region"`
}

// VerifyEmailChallengeInput verifies one six-digit challenge.
type VerifyEmailChallengeInput struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
	DataRegion  string `json:"data_region"`
}

// VerifyOAuthInput is transient and must never be logged or persisted.
type VerifyOAuthInput struct {
	Provider          Provider `json:"provider"`
	AuthorizationCode string   `json:"-"`
	RedirectURI       string   `json:"redirect_uri,omitempty"`
	DataRegion        string   `json:"data_region"`
}

// CreateSessionInput consumes one verified proof. Registration is required only
// for a previously unseen identity.
type CreateSessionInput struct {
	ProofToken   string        `json:"-"`
	DataRegion   string        `json:"data_region"`
	Registration *Registration `json:"registration,omitempty"`
}

// RefreshSessionInput rotates a single-use refresh token.
type RefreshSessionInput struct {
	RefreshToken string `json:"-"`
	DataRegion   string `json:"data_region"`
}

// UpdateAccountInput changes account preferences only; interview language is a
// separate project field and is not touched here.
type UpdateAccountInput struct {
	UILanguage       *Language
	DisplayName      *string
	ClearDisplayName bool
}

// BindIdentityInput requires independent, fresh proofs for the current and
// target identities (US-05 scenario 4).
type BindIdentityInput struct {
	SourceProofToken string `json:"-"`
	TargetProofToken string `json:"-"`
}

// RequestEmailChallenge sends a deterministic OTP through services/notify.
// Retries with the same key reuse the challenge and never create a second one.
func (s *Service) RequestEmailChallenge(
	ctx context.Context,
	input RequestEmailChallengeInput,
	idempotencyKey string,
) (VerificationChallenge, error) {
	if err := validateContextAndKey(ctx, idempotencyKey); err != nil {
		return VerificationChallenge{}, err
	}
	if err := region.ValidateDataRegion(input.DataRegion); err != nil {
		return VerificationChallenge{}, regionMismatchError()
	}
	normalizedEmail, err := normalizeEmail(input.Email)
	if err != nil {
		return VerificationChallenge{}, validationError()
	}
	subjectHash := subjectDigest(s.secrets.SubjectKey, ProviderEmailOTP, normalizedEmail)
	if err := s.risk.Evaluate(ctx, RiskRequest{
		DataRegion: input.DataRegion, Provider: ProviderEmailOTP, ProviderSubjectHash: subjectHash,
	}); err != nil {
		return VerificationChallenge{}, domainError(
			CodeRiskVerificationRequired,
			"风险校验尚未通过，验证码未发送且账户数据未改变。请先完成页面风险验证后重试；不计费且不影响评分。",
			true,
			nil,
		)
	}
	requestHash, err := hashRequest(struct {
		SubjectHash string
		DataRegion  string
	}{subjectHash, input.DataRegion})
	if err != nil {
		return VerificationChallenge{}, internalError(err)
	}
	persistedRequestKey := verificationRequestKey("challenge", ProviderEmailOTP, idempotencyKey)
	return executeJSON(ctx, s.idempotency, "identity.email.challenge", input.DataRegion, idempotencyKey, requestHash, func() (VerificationChallenge, error) {
		now := s.clock.Now().UTC()
		var verification Verification
		if err := s.store.Transact(ctx, func(tx Tx) error {
			existing, found, storeErr := tx.VerificationByRequestKey(input.DataRegion, persistedRequestKey)
			if storeErr != nil {
				return storeErr
			}
			if found {
				verification = existing
				return nil
			}
			count, storeErr := tx.RecentVerificationCount(
				input.DataRegion,
				ProviderEmailOTP,
				subjectHash,
				now.Add(-s.config.RateWindow),
			)
			if storeErr != nil {
				return storeErr
			}
			if count >= s.config.MaxEmailChallenges {
				return domainError(
					CodeRateLimited,
					"验证码请求过于频繁，本次未发送且账户数据未改变。请稍后重试；不计费且不影响评分。",
					true,
					nil,
				)
			}
			verificationID, idErr := s.ids.NewID()
			if idErr != nil {
				return idErr
			}
			verification = Verification{
				VerificationID:      verificationID,
				Provider:            ProviderEmailOTP,
				ProviderSubjectHash: subjectHash,
				CodeHash:            otpDigest(s.secrets.OTPKey, verificationID, deriveOTP(s.secrets.OTPKey, verificationID)),
				Status:              VerificationPending,
				MaxAttempts:         s.config.MaxVerificationAttempts,
				RequestedAt:         now,
				ExpiresAt:           now.Add(s.config.OTPTTL),
				DataRegion:          input.DataRegion,
				RequestKey:          persistedRequestKey,
			}
			return tx.CreateVerification(verification)
		}); err != nil {
			return VerificationChallenge{}, mapStoreError(err)
		}
		if verification.NotificationSentAt == nil {
			message := notify.Message{
				DataRegion: input.DataRegion,
				To:         normalizedEmail,
				TemplateID: emailTemplateID,
				Variables: map[string]string{
					"otp_code":           deriveOTP(s.secrets.OTPKey, verification.VerificationID),
					"expires_in_minutes": fmt.Sprintf("%d", int(s.config.OTPTTL.Minutes())),
				},
				IdempotencyKey: "identity-otp-" + idempotencyKey,
			}
			if err := s.notifier.Send(ctx, message); err != nil {
				return VerificationChallenge{}, domainError(
					CodeProviderUnavailable,
					"邮箱通道暂时不可用，验证码挑战已安全保留但尚未确认投递。请重试发送；不计费且不影响评分。",
					true,
					nil,
				)
			}
			sentAt := s.clock.Now().UTC()
			verification.NotificationSentAt = &sentAt
			if err := s.store.Transact(ctx, func(tx Tx) error { return tx.UpdateVerification(verification) }); err != nil {
				return VerificationChallenge{}, internalError(err)
			}
		}
		return VerificationChallenge{
			ChallengeID:       verification.VerificationID,
			ExpiresAt:         verification.ExpiresAt,
			RetryAfterSeconds: int(s.config.RetryAfter.Seconds()),
			DeliveryStatus:    "accepted",
			DataRegion:        verification.DataRegion,
		}, nil
	})
}

// VerifyEmailChallenge validates an OTP with constant-time digest comparison
// and returns a deterministic single-use proof token.
func (s *Service) VerifyEmailChallenge(
	ctx context.Context,
	input VerifyEmailChallengeInput,
	idempotencyKey string,
) (VerificationProof, error) {
	if err := validateContextAndKey(ctx, idempotencyKey); err != nil {
		return VerificationProof{}, err
	}
	if err := region.ValidateDataRegion(input.DataRegion); err != nil || len(input.Code) != 6 || !digitsOnly(input.Code) || input.ChallengeID == "" {
		return VerificationProof{}, validationError()
	}
	requestHash := secureDigest(input.ChallengeID + "\x00" + secureDigest(input.Code) + "\x00" + input.DataRegion)
	return executeJSON(ctx, s.idempotency, "identity.email.verify", input.DataRegion, idempotencyKey, requestHash, func() (VerificationProof, error) {
		now := s.clock.Now().UTC()
		var verification Verification
		var outcomeErr error
		if err := s.store.Transact(ctx, func(tx Tx) error {
			current, found, storeErr := tx.VerificationByID(input.ChallengeID)
			if storeErr != nil {
				return storeErr
			}
			if !found || current.Provider != ProviderEmailOTP || current.DataRegion != input.DataRegion {
				outcomeErr = invalidVerificationError()
				return nil
			}
			verification = current
			if !now.Before(current.ExpiresAt) {
				current.Status = VerificationExpired
				verification = current
				outcomeErr = expiredVerificationError()
				return tx.UpdateVerification(current)
			}
			if current.Status == VerificationVerified && current.ProofExpiresAt != nil && now.Before(*current.ProofExpiresAt) {
				return nil
			}
			if current.Status != VerificationPending {
				outcomeErr = invalidVerificationError()
				return nil
			}
			if !constantDigestEqual(current.CodeHash, otpDigest(s.secrets.OTPKey, current.VerificationID, input.Code)) {
				current.FailedAttempts++
				if current.FailedAttempts >= current.MaxAttempts {
					current.Status = VerificationLocked
				}
				verification = current
				outcomeErr = invalidVerificationError()
				return tx.UpdateVerification(current)
			}
			verifiedAt := now
			proofExpiresAt := now.Add(s.config.ProofTTL)
			current.Status = VerificationVerified
			current.VerifiedAt = &verifiedAt
			current.ProofExpiresAt = &proofExpiresAt
			current.ProofHash = secureDigest(proofToken(s.secrets.ProofKey, current))
			verification = current
			return tx.UpdateVerification(current)
		}); err != nil {
			return VerificationProof{}, mapStoreError(err)
		}
		if outcomeErr != nil {
			return VerificationProof{}, outcomeErr
		}
		return proofView(s.secrets.ProofKey, verification), nil
	})
}

// VerifyOAuth verifies a provider authorization code through the TASK-007
// region-aware adapter registry and returns the same proof abstraction as OTP.
func (s *Service) VerifyOAuth(
	ctx context.Context,
	input VerifyOAuthInput,
	idempotencyKey string,
) (VerificationProof, error) {
	if err := validateContextAndKey(ctx, idempotencyKey); err != nil {
		return VerificationProof{}, err
	}
	if err := region.ValidateDataRegion(input.DataRegion); err != nil || !input.Provider.Valid() || input.Provider == ProviderEmailOTP || strings.TrimSpace(input.AuthorizationCode) == "" {
		return VerificationProof{}, validationError()
	}
	requestHash := secureDigest(string(input.Provider) + "\x00" + input.DataRegion + "\x00" + secureDigest(input.AuthorizationCode) + "\x00" + input.RedirectURI)
	persistedRequestKey := verificationRequestKey("oauth", input.Provider, idempotencyKey)
	return executeJSON(ctx, s.idempotency, "identity.oauth.verify", input.DataRegion, idempotencyKey, requestHash, func() (VerificationProof, error) {
		verified, err := s.providers.Verify(ctx, string(input.Provider), identityprovider.VerifyRequest{
			AuthorizationCode: input.AuthorizationCode,
			RedirectURI:       input.RedirectURI,
			DataRegion:        input.DataRegion,
		})
		if err != nil {
			return VerificationProof{}, mapProviderError(err)
		}
		now := s.clock.Now().UTC()
		verificationID, err := s.ids.NewID()
		if err != nil {
			return VerificationProof{}, internalError(err)
		}
		verifiedAt := verified.VerifiedAt.UTC()
		proofExpiresAt := now.Add(s.config.ProofTTL)
		verification := Verification{
			VerificationID:      verificationID,
			Provider:            input.Provider,
			ProviderSubjectHash: subjectDigest(s.secrets.SubjectKey, input.Provider, verified.Subject),
			Status:              VerificationVerified,
			MaxAttempts:         1,
			RequestedAt:         now,
			VerifiedAt:          &verifiedAt,
			ExpiresAt:           proofExpiresAt,
			ProofExpiresAt:      &proofExpiresAt,
			DataRegion:          input.DataRegion,
			RequestKey:          persistedRequestKey,
		}
		verification.ProofHash = secureDigest(proofToken(s.secrets.ProofKey, verification))
		if err := s.store.Transact(ctx, func(tx Tx) error { return tx.CreateVerification(verification) }); err != nil {
			return VerificationProof{}, mapStoreError(err)
		}
		return proofView(s.secrets.ProofKey, verification), nil
	})
}

// CreateSession consumes a proof and resolves an existing account or creates a
// new one with mandatory registration evidence.
func (s *Service) CreateSession(
	ctx context.Context,
	input CreateSessionInput,
	idempotencyKey string,
) (Session, error) {
	if err := validateContextAndKey(ctx, idempotencyKey); err != nil || strings.TrimSpace(input.ProofToken) == "" {
		return Session{}, validationError()
	}
	if err := region.ValidateDataRegion(input.DataRegion); err != nil {
		return Session{}, regionMismatchError()
	}
	requestHash := secureDigest(secureDigest(input.ProofToken) + "\x00" + input.DataRegion + "\x00" + registrationHash(input.Registration))
	return executeJSON(ctx, s.idempotency, "identity.session.create", input.DataRegion, idempotencyKey, requestHash, func() (Session, error) {
		now := s.clock.Now().UTC()
		var response Session
		var outcomeErr error
		if err := s.store.Transact(ctx, func(tx Tx) error {
			verification, proofErr := s.usableProof(tx, input.ProofToken, input.DataRegion, now)
			if proofErr != nil {
				outcomeErr = proofErr
				return nil
			}
			boundIdentity, found, storeErr := tx.IdentityBySubject(input.DataRegion, verification.Provider, verification.ProviderSubjectHash)
			if storeErr != nil {
				return storeErr
			}
			var user User
			if found {
				user, found, storeErr = tx.UserByID(boundIdentity.UserID)
				if storeErr != nil {
					return storeErr
				}
				if !found || user.DataRegion != input.DataRegion || user.Status != AccountActive {
					outcomeErr = domainError(CodeForbidden, "账户当前不可登录，数据保持不变。请使用账户恢复入口；不计费且不影响评分。", false, nil)
					return nil
				}
			} else {
				if err := validateRegistration(input.Registration, now); err != nil {
					outcomeErr = err
					return nil
				}
				userID, idErr := s.ids.NewID()
				if idErr != nil {
					return idErr
				}
				identityID, idErr := s.ids.NewID()
				if idErr != nil {
					return idErr
				}
				user = User{
					UserID:       userID,
					DataRegion:   input.DataRegion,
					UILanguage:   input.Registration.UILanguage,
					AgeStatus:    input.Registration.AgeStatus,
					Status:       AccountActive,
					Registration: input.Registration.Evidence,
					CreatedAt:    now,
					UpdatedAt:    now,
				}
				if err := tx.CreateUser(user); err != nil {
					return err
				}
				boundIdentity = Identity{
					IdentityID:          identityID,
					UserID:              userID,
					Provider:            verification.Provider,
					ProviderSubjectHash: verification.ProviderSubjectHash,
					VerifiedAt:          dereferenceTime(verification.VerifiedAt, now),
					DataRegion:          input.DataRegion,
					CreatedAt:           now,
				}
				if err := tx.CreateIdentity(boundIdentity); err != nil {
					return err
				}
			}
			account, accountErr := accountFromTx(tx, user)
			if accountErr != nil {
				return accountErr
			}
			record, issued, issueErr := s.tokens.newSession(account, now)
			if issueErr != nil {
				return issueErr
			}
			if err := tx.CreateSession(record); err != nil {
				return err
			}
			consumeVerification(&verification, now)
			if err := tx.UpdateVerification(verification); err != nil {
				return err
			}
			response = issued
			return nil
		}); err != nil {
			return Session{}, mapStoreError(err)
		}
		if outcomeErr != nil {
			return Session{}, outcomeErr
		}
		return response, nil
	})
}

// RefreshSession rotates the refresh token; the old token becomes unusable in
// the same transaction that creates the replacement session.
func (s *Service) RefreshSession(
	ctx context.Context,
	input RefreshSessionInput,
	idempotencyKey string,
) (Session, error) {
	if err := validateContextAndKey(ctx, idempotencyKey); err != nil || strings.TrimSpace(input.RefreshToken) == "" {
		return Session{}, validationError()
	}
	if err := region.ValidateDataRegion(input.DataRegion); err != nil {
		return Session{}, regionMismatchError()
	}
	sessionID, err := tokenPrefix(input.RefreshToken)
	if err != nil {
		return Session{}, unauthorizedTokenError()
	}
	requestHash := secureDigest(secureDigest(input.RefreshToken) + "\x00" + input.DataRegion)
	return executeJSON(ctx, s.idempotency, "identity.session.refresh", input.DataRegion, idempotencyKey, requestHash, func() (Session, error) {
		now := s.clock.Now().UTC()
		var response Session
		var outcomeErr error
		if err := s.store.Transact(ctx, func(tx Tx) error {
			current, found, storeErr := tx.SessionByID(sessionID)
			if storeErr != nil {
				return storeErr
			}
			if !found || current.DataRegion != input.DataRegion || !s.tokens.VerifyRefresh(current, input.RefreshToken, now) {
				outcomeErr = unauthorizedTokenError()
				return nil
			}
			user, found, storeErr := tx.UserByID(current.UserID)
			if storeErr != nil {
				return storeErr
			}
			if !found || user.Status != AccountActive {
				outcomeErr = unauthorizedTokenError()
				return nil
			}
			account, accountErr := accountFromTx(tx, user)
			if accountErr != nil {
				return accountErr
			}
			replacement, issued, issueErr := s.tokens.newSession(account, now)
			if issueErr != nil {
				return issueErr
			}
			current.Status = SessionRotated
			current.RotatedTo = replacement.SessionID
			current.RotatedAt = &now
			if err := tx.UpdateSession(current); err != nil {
				return err
			}
			if err := tx.CreateSession(replacement); err != nil {
				return err
			}
			response = issued
			return nil
		}); err != nil {
			return Session{}, mapStoreError(err)
		}
		if outcomeErr != nil {
			return Session{}, outcomeErr
		}
		return response, nil
	})
}

// Authenticate validates an access token and returns only safe claims.
func (s *Service) Authenticate(accessToken string) (Claims, error) {
	return s.tokens.Authenticate(accessToken, s.clock.Now().UTC())
}

// GetAccount returns only the public account view.
func (s *Service) GetAccount(ctx context.Context, claims Claims) (Account, error) {
	if ctx == nil || claims.UserID == "" || !region.Valid(claims.DataRegion) {
		return Account{}, unauthorizedTokenError()
	}
	var account Account
	if err := s.store.Transact(ctx, func(tx Tx) error {
		user, found, err := tx.UserByID(claims.UserID)
		if err != nil {
			return err
		}
		if !found || user.DataRegion != claims.DataRegion {
			return regionMismatchError()
		}
		account, err = accountFromTx(tx, user)
		return err
	}); err != nil {
		return Account{}, mapStoreError(err)
	}
	return account, nil
}

// UpdateAccount updates UI preferences idempotently without changing region,
// identities, interview language, scoring, consent or billing state.
func (s *Service) UpdateAccount(
	ctx context.Context,
	claims Claims,
	input UpdateAccountInput,
	idempotencyKey string,
) (Account, error) {
	if err := validateContextAndKey(ctx, idempotencyKey); err != nil || claims.UserID == "" || !region.Valid(claims.DataRegion) {
		return Account{}, validationError()
	}
	if input.UILanguage == nil && input.DisplayName == nil && !input.ClearDisplayName {
		return Account{}, validationError()
	}
	if input.UILanguage != nil && !input.UILanguage.Valid() {
		return Account{}, validationError()
	}
	if input.DisplayName != nil {
		trimmed := strings.TrimSpace(*input.DisplayName)
		if trimmed == "" || len([]rune(trimmed)) > 120 {
			return Account{}, validationError()
		}
		input.DisplayName = &trimmed
	}
	requestHash, err := hashRequest(input)
	if err != nil {
		return Account{}, internalError(err)
	}
	return executeJSON(ctx, s.idempotency, "identity.account.update", claims.DataRegion, idempotencyKey, requestHash, func() (Account, error) {
		var account Account
		if err := s.store.Transact(ctx, func(tx Tx) error {
			user, found, storeErr := tx.UserByID(claims.UserID)
			if storeErr != nil {
				return storeErr
			}
			if !found || user.DataRegion != claims.DataRegion {
				return regionMismatchError()
			}
			if input.UILanguage != nil {
				user.UILanguage = *input.UILanguage
			}
			if input.ClearDisplayName {
				user.DisplayName = nil
			} else if input.DisplayName != nil {
				user.DisplayName = input.DisplayName
			}
			user.UpdatedAt = s.clock.Now().UTC()
			if err := tx.UpdateUser(user); err != nil {
				return err
			}
			account, storeErr = accountFromTx(tx, user)
			return storeErr
		}); err != nil {
			return Account{}, mapStoreError(err)
		}
		return account, nil
	})
}

// BindIdentity implements dual-sided verification and the hard no-auto-merge
// rule. A collision commits only a recovery case and proof consumption.
func (s *Service) BindIdentity(
	ctx context.Context,
	claims Claims,
	input BindIdentityInput,
	idempotencyKey string,
) (Binding, error) {
	if err := validateContextAndKey(ctx, idempotencyKey); err != nil || claims.UserID == "" || !region.Valid(claims.DataRegion) ||
		strings.TrimSpace(input.SourceProofToken) == "" || strings.TrimSpace(input.TargetProofToken) == "" {
		return Binding{}, validationError()
	}
	requestHash := secureDigest(secureDigest(input.SourceProofToken) + "\x00" + secureDigest(input.TargetProofToken))
	return executeJSON(ctx, s.idempotency, "identity.binding.create", claims.DataRegion, idempotencyKey, requestHash, func() (Binding, error) {
		now := s.clock.Now().UTC()
		var binding Binding
		var outcomeErr error
		if err := s.store.Transact(ctx, func(tx Tx) error {
			source, sourceErr := s.usableProof(tx, input.SourceProofToken, claims.DataRegion, now)
			if sourceErr != nil {
				outcomeErr = sourceErr
				return nil
			}
			target, targetErr := s.usableProof(tx, input.TargetProofToken, claims.DataRegion, now)
			if targetErr != nil {
				outcomeErr = targetErr
				return nil
			}
			if source.VerificationID == target.VerificationID ||
				(source.Provider == target.Provider && source.ProviderSubjectHash == target.ProviderSubjectHash) {
				outcomeErr = domainError(CodeConflict, "当前身份与目标身份必须分别验证；本次未绑定且账户数据保持不变。请重新验证两侧后重试；不计费且不影响评分。", false, nil)
				return nil
			}
			sourceIdentity, found, storeErr := tx.IdentityBySubject(claims.DataRegion, source.Provider, source.ProviderSubjectHash)
			if storeErr != nil {
				return storeErr
			}
			if !found || sourceIdentity.UserID != claims.UserID {
				outcomeErr = domainError(CodeForbidden, "当前侧身份验证不属于此账户，本次未绑定且账户数据保持不变。请重新验证当前身份；不计费且不影响评分。", false, nil)
				return nil
			}
			targetIdentity, targetFound, storeErr := tx.IdentityBySubject(claims.DataRegion, target.Provider, target.ProviderSubjectHash)
			if storeErr != nil {
				return storeErr
			}
			if targetFound && targetIdentity.UserID != claims.UserID {
				recoveryID, idErr := s.ids.NewID()
				if idErr != nil {
					return idErr
				}
				if err := tx.CreateRecoveryCase(RecoveryCase{
					RecoveryCaseID:      recoveryID,
					RequestingUserID:    claims.UserID,
					ConflictingUserID:   targetIdentity.UserID,
					Provider:            target.Provider,
					ProviderSubjectHash: target.ProviderSubjectHash,
					Status:              "open",
					DataRegion:          claims.DataRegion,
					CreatedAt:           now,
				}); err != nil {
					return err
				}
				consumeVerification(&source, now)
				consumeVerification(&target, now)
				if err := tx.UpdateVerification(source); err != nil {
					return err
				}
				if err := tx.UpdateVerification(target); err != nil {
					return err
				}
				conflict := domainError(
					CodeIdentityConflict,
					"目标身份已属于另一账户，系统未执行合并且两边数据保持隔离。请使用账户恢复或人工支持路径；不计费且不影响评分。",
					false,
					nil,
				)
				conflict.Details = map[string]any{
					"recovery_case_id": recoveryID,
					"support_path":     s.config.SupportPath,
					"accounts_merged":  false,
				}
				outcomeErr = conflict
				return nil
			}
			if !targetFound {
				identityID, idErr := s.ids.NewID()
				if idErr != nil {
					return idErr
				}
				targetIdentity = Identity{
					IdentityID:          identityID,
					UserID:              claims.UserID,
					Provider:            target.Provider,
					ProviderSubjectHash: target.ProviderSubjectHash,
					VerifiedAt:          dereferenceTime(target.VerifiedAt, now),
					DataRegion:          claims.DataRegion,
					CreatedAt:           now,
				}
				if err := tx.CreateIdentity(targetIdentity); err != nil {
					return err
				}
			}
			consumeVerification(&source, now)
			consumeVerification(&target, now)
			if err := tx.UpdateVerification(source); err != nil {
				return err
			}
			if err := tx.UpdateVerification(target); err != nil {
				return err
			}
			binding = publicBinding(targetIdentity)
			return nil
		}); err != nil {
			return Binding{}, mapStoreError(err)
		}
		if outcomeErr != nil {
			return Binding{}, outcomeErr
		}
		return binding, nil
	})
}

func (s *Service) usableProof(tx Tx, token, dataRegion string, now time.Time) (Verification, error) {
	id, err := proofID(token)
	if err != nil {
		return Verification{}, invalidVerificationError()
	}
	verification, found, err := tx.VerificationByID(id)
	if err != nil {
		return Verification{}, err
	}
	if !found || verification.DataRegion != dataRegion || verification.Status != VerificationVerified ||
		verification.ProofExpiresAt == nil || !now.Before(*verification.ProofExpiresAt) ||
		!verifyProofToken(s.secrets.ProofKey, token, verification) {
		return Verification{}, invalidVerificationError()
	}
	return verification, nil
}

func proofView(key []byte, verification Verification) VerificationProof {
	return VerificationProof{
		ProofToken: proofToken(key, verification),
		Provider:   verification.Provider,
		ExpiresAt:  *verification.ProofExpiresAt,
		DataRegion: verification.DataRegion,
	}
}

func consumeVerification(verification *Verification, now time.Time) {
	verification.Status = VerificationConsumed
	verification.ConsumedAt = &now
}

func accountFromTx(tx Tx, user User) (Account, error) {
	identities, err := tx.IdentitiesByUser(user.UserID)
	if err != nil {
		return Account{}, err
	}
	bindings := make([]Binding, 0, len(identities))
	for _, identity := range identities {
		bindings = append(bindings, publicBinding(identity))
	}
	return Account{
		UserID:      user.UserID,
		DataRegion:  user.DataRegion,
		UILanguage:  user.UILanguage,
		AgeStatus:   user.AgeStatus,
		Status:      user.Status,
		DisplayName: user.DisplayName,
		Identities:  bindings,
	}, nil
}

func publicBinding(identity Identity) Binding {
	return Binding{
		IdentityID: identity.IdentityID,
		Provider:   identity.Provider,
		VerifiedAt: identity.VerifiedAt,
		DataRegion: identity.DataRegion,
	}
}

func validateRegistration(registration *Registration, now time.Time) error {
	if registration == nil || !registration.UILanguage.Valid() || !registration.AgeStatus.Valid() {
		return domainError(CodeConflict, "首次注册资料不完整，账户尚未创建。请确认年龄状态及必需告知后重试；不计费且不影响评分。", false, nil)
	}
	evidence := registration.Evidence
	if strings.TrimSpace(evidence.TermsVersion) == "" || strings.TrimSpace(evidence.PrivacyVersion) == "" ||
		strings.TrimSpace(evidence.DataProcessingVersion) == "" || evidence.AcceptedAt.IsZero() ||
		evidence.AcceptedAt.After(now.Add(5*time.Minute)) || !evidence.Context.UILanguage.Valid() ||
		(evidence.Context.Surface != "web" && evidence.Context.Surface != "pwa" && evidence.Context.Surface != "mobile_web") {
		return domainError(CodeConflict, "首次注册的条款、隐私政策或数据处理说明证据不完整，账户尚未创建。请重新确认后重试；不计费且不影响评分。", false, nil)
	}
	return nil
}

func registrationHash(registration *Registration) string {
	if registration == nil {
		return "none"
	}
	hash, err := hashRequest(registration)
	if err != nil {
		return "invalid"
	}
	return hash
}

func normalizeEmail(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	parsed, err := mail.ParseAddress(trimmed)
	if err != nil || !strings.EqualFold(parsed.Address, trimmed) || !strings.Contains(parsed.Address, "@") {
		return "", errors.New("invalid email")
	}
	return strings.ToLower(parsed.Address), nil
}

func digitsOnly(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func constantDigestEqual(left, right string) bool {
	return hmac.Equal([]byte(left), []byte(right))
}

func verificationRequestKey(operation string, provider Provider, idempotencyKey string) string {
	return operation + "\x00" + string(provider) + "\x00" + idempotencyKey
}

func validateContextAndKey(ctx context.Context, key string) error {
	if ctx == nil || len(key) < 8 || len(key) > 128 || strings.TrimSpace(key) != key {
		return validationError()
	}
	return nil
}

func tokenPrefix(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", errors.New("malformed token")
	}
	return parts[0], nil
}

func dereferenceTime(value *time.Time, fallback time.Time) time.Time {
	if value == nil {
		return fallback
	}
	return *value
}

func invalidVerificationError() *DomainError {
	return domainError(CodeVerificationInvalid, "验证信息无效、已消费或尝试次数已用尽，账户数据保持不变。请重新发起验证；不计费且不影响评分。", false, nil)
}

func expiredVerificationError() *DomainError {
	return domainError(CodeVerificationExpired, "验证信息已过期，账户数据保持不变。请重新发起验证；不计费且不影响评分。", false, nil)
}

func regionMismatchError() *DomainError {
	return domainError(CodeRegionMismatch, "请求数据区与身份所属区域不一致，操作已拒绝且没有跨区读取或写入。请从账户所属区域重试；不计费且不影响评分。", false, nil)
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

func mapProviderError(err error) error {
	switch {
	case errors.Is(err, identityprovider.ErrRegionNotAllowed):
		return validationError()
	case errors.Is(err, identityprovider.ErrInvalidCredential):
		return invalidVerificationError()
	default:
		providerErr := domainError(CodeProviderUnavailable, "第三方登录暂时不可用，未创建或修改账户。请稍后重试或改用邮箱验证码；不计费且不影响评分。", true, nil)
		providerErr.Details = map[string]any{"email_fallback_available": true}
		return providerErr
	}
}
