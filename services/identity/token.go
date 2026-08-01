package identity

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"miangedan/services/region"
)

// TokenManager issues and validates provider-neutral business JWTs and
// deterministic rotating refresh tokens. Signing material is injected from
// *_REF-backed secret resolution.
type TokenManager struct {
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	signingKey []byte
	refreshKey []byte
	ids        IDGenerator
}

// NewTokenManager validates token policy and key material.
func NewTokenManager(
	issuer string,
	accessTTL, refreshTTL time.Duration,
	secrets SecretMaterial,
	ids IDGenerator,
) (*TokenManager, error) {
	if issuer == "" || accessTTL <= 0 || refreshTTL <= 0 || ids == nil {
		return nil, errors.New("token issuer, positive TTLs and ID generator are required")
	}
	if len(secrets.SigningKey) < 32 || len(secrets.RefreshKey) < 32 {
		return nil, errors.New("identity token keys must be at least 32 bytes")
	}
	return &TokenManager{
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		signingKey: append([]byte(nil), secrets.SigningKey...),
		refreshKey: append([]byte(nil), secrets.RefreshKey...),
		ids:        ids,
	}, nil
}

type tokenHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type tokenPayload struct {
	Issuer     string `json:"iss"`
	Subject    string `json:"sub"`
	SessionID  string `json:"sid"`
	DataRegion string `json:"data_region"`
	IssuedAt   int64  `json:"iat"`
	ExpiresAt  int64  `json:"exp"`
}

func (m *TokenManager) newSession(account Account, now time.Time) (SessionRecord, Session, error) {
	sessionID, err := m.ids.NewID()
	if err != nil {
		return SessionRecord{}, Session{}, err
	}
	now = now.UTC()
	record := SessionRecord{
		SessionID:        sessionID,
		UserID:           account.UserID,
		Status:           SessionActive,
		AccessExpiresAt:  now.Add(m.accessTTL),
		RefreshExpiresAt: now.Add(m.refreshTTL),
		DataRegion:       account.DataRegion,
		CreatedAt:        now,
	}
	accessToken, err := m.accessToken(record)
	if err != nil {
		return SessionRecord{}, Session{}, err
	}
	refreshToken := m.refreshToken(record)
	record.RefreshTokenHash = secureDigest(refreshToken)
	return record, Session{
		SessionID:        sessionID,
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        int64(m.accessTTL.Seconds()),
		RefreshExpiresIn: int64(m.refreshTTL.Seconds()),
		Account:          account,
		DataRegion:       account.DataRegion,
	}, nil
}

func (m *TokenManager) accessToken(record SessionRecord) (string, error) {
	headerJSON, err := json.Marshal(tokenHeader{Algorithm: "HS256", Type: "JWT"})
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(tokenPayload{
		Issuer:     m.issuer,
		Subject:    record.UserID,
		SessionID:  record.SessionID,
		DataRegion: record.DataRegion,
		IssuedAt:   record.CreatedAt.Unix(),
		ExpiresAt:  record.AccessExpiresAt.Unix(),
	})
	if err != nil {
		return "", err
	}
	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := header + "." + payload
	signature := base64.RawURLEncoding.EncodeToString(hmacBytes(m.signingKey, unsigned))
	return unsigned + "." + signature, nil
}

func (m *TokenManager) refreshToken(record SessionRecord) string {
	payload := record.SessionID + "|" + record.UserID + "|" + record.DataRegion + "|" +
		record.RefreshExpiresAt.UTC().Format(time.RFC3339Nano)
	signature := base64.RawURLEncoding.EncodeToString(hmacBytes(m.refreshKey, "identity-refresh-v1\x00"+payload))
	return record.SessionID + "." + signature
}

// VerifyRefresh validates token shape, session binding, lifecycle and expiry.
func (m *TokenManager) VerifyRefresh(record SessionRecord, token string, now time.Time) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] != record.SessionID || record.Status != SessionActive || !now.Before(record.RefreshExpiresAt) {
		return false
	}
	expected := m.refreshToken(record)
	return hmac.Equal([]byte(expected), []byte(token)) &&
		hmac.Equal([]byte(record.RefreshTokenHash), []byte(secureDigest(token)))
}

// Authenticate validates a business JWT without accepting algorithm changes.
func (m *TokenManager) Authenticate(token string, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, unauthorizedTokenError()
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, unauthorizedTokenError()
	}
	var header tokenHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != "HS256" || header.Type != "JWT" {
		return Claims{}, unauthorizedTokenError()
	}
	unsigned := parts[0] + "." + parts[1]
	expected := base64.RawURLEncoding.EncodeToString(hmacBytes(m.signingKey, unsigned))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return Claims{}, unauthorizedTokenError()
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, unauthorizedTokenError()
	}
	var payload tokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return Claims{}, unauthorizedTokenError()
	}
	now = now.UTC()
	if payload.Issuer != m.issuer || payload.Subject == "" || payload.SessionID == "" ||
		!region.Valid(payload.DataRegion) || payload.ExpiresAt <= now.Unix() || payload.IssuedAt > now.Add(time.Minute).Unix() {
		return Claims{}, unauthorizedTokenError()
	}
	return Claims{
		UserID:     payload.Subject,
		SessionID:  payload.SessionID,
		DataRegion: payload.DataRegion,
		ExpiresAt:  time.Unix(payload.ExpiresAt, 0).UTC(),
	}, nil
}

func unauthorizedTokenError() *DomainError {
	return domainError(
		CodeUnauthorized,
		"身份令牌无效或已过期，账户数据保持不变。请重新登录；不计费且不影响评分。",
		false,
		nil,
	)
}
