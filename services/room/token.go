package room

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// tokenVersion 为媒体令牌版本前缀（不含点号，保证 token 可安全按 . 拆三段）。
const tokenVersion = "mgdtv1"

// MediaTokenClaims 为短期房间令牌声明（SEC-003：与业务令牌隔离，仅媒体面用途）。
type MediaTokenClaims struct {
	Typ        string `json:"typ"`
	SessionID  string `json:"session_id"`
	DeviceID   string `json:"device_id"`
	DataRegion string `json:"data_region"`
	Nonce      string `json:"nonce"`
	Exp        int64  `json:"exp"`
}

// MediaTokenStore 为一性/吊销登记（短期状态，生产放 Redis；Redis 丢失不影响业务证据）。
// 吊销按 nonce 粒度：RevokeSession 吊销该会话已签发的全部令牌，之后新签发的令牌不受影响
// （重连/设备转移需要先吊销旧令牌再签发新令牌）。
type MediaTokenStore interface {
	RecordNonce(sessionID, deviceID, nonce string) error
	ConsumeNonce(nonce string) bool
	RevokeSession(sessionID string) error
	IsNonceRevoked(nonce string) bool
}

// TokenConfig 为媒体令牌配置（密钥经 *_REF 注入，SEC-003 与业务令牌密钥隔离）。
type TokenConfig struct {
	SigningKey string
	TTL        time.Duration
}

// MediaTokenManager 签发与校验短期媒体令牌（HMAC-SHA256）。
type MediaTokenManager struct {
	key   []byte
	ttl   time.Duration
	store MediaTokenStore
}

// NewMediaTokenManager 创建令牌管理器。
func NewMediaTokenManager(cfg TokenConfig, store MediaTokenStore) (*MediaTokenManager, error) {
	if len(cfg.SigningKey) < 32 {
		return nil, fmt.Errorf("%w: 媒体令牌签名密钥至少 32 字符（与业务令牌隔离，SEC-003）", ErrInvalidInput)
	}
	if cfg.TTL <= 0 {
		cfg.TTL = TokenTTLDefault
	}
	if store == nil {
		return nil, fmt.Errorf("%w: 缺少令牌存储", ErrInvalidInput)
	}
	return &MediaTokenManager{key: []byte(cfg.SigningKey), ttl: cfg.TTL, store: store}, nil
}

// Issue 签发一次性短期媒体令牌。
func (m *MediaTokenManager) Issue(sessionID, deviceID, dataRegion string, now time.Time) (string, time.Time, error) {
	nonce, err := newNonce()
	if err != nil {
		return "", time.Time{}, err
	}
	if err := m.store.RecordNonce(sessionID, deviceID, nonce); err != nil {
		return "", time.Time{}, err
	}
	exp := now.Add(m.ttl)
	claims := MediaTokenClaims{
		Typ:        "media",
		SessionID:  sessionID,
		DeviceID:   deviceID,
		DataRegion: dataRegion,
		Nonce:      nonce,
		Exp:        exp.Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)
	sig := m.sign(payloadB64)
	return tokenVersion + "." + payloadB64 + "." + sig, exp, nil
}

func (m *MediaTokenManager) sign(payloadB64 string) string {
	mac := hmac.New(sha256.New, m.key)
	_, _ = mac.Write([]byte(payloadB64))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Verify 校验令牌：格式、签名、有效期、未吊销，并一次性消费 nonce。
func (m *MediaTokenManager) Verify(token string, now time.Time) (MediaTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != tokenVersion {
		return MediaTokenClaims{}, ErrTokenInvalid
	}
	payloadB64, sig := parts[1], parts[2]
	if !hmac.Equal([]byte(sig), []byte(m.sign(payloadB64))) {
		return MediaTokenClaims{}, ErrTokenInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return MediaTokenClaims{}, ErrTokenInvalid
	}
	var claims MediaTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return MediaTokenClaims{}, ErrTokenInvalid
	}
	if claims.Typ != "media" || claims.SessionID == "" || claims.DeviceID == "" {
		return MediaTokenClaims{}, ErrTokenInvalid
	}
	if now.Unix() >= claims.Exp {
		return MediaTokenClaims{}, ErrTokenInvalid
	}
	if m.store.IsNonceRevoked(claims.Nonce) {
		return MediaTokenClaims{}, ErrTokenRevoked
	}
	if !m.store.ConsumeNonce(claims.Nonce) {
		return MediaTokenClaims{}, ErrTokenUsed
	}
	return claims, nil
}

func newNonce() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
