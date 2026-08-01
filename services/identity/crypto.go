package identity

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// IDGenerator creates opaque UUID identifiers without provider semantics.
type IDGenerator interface {
	NewID() (string, error)
}

// CryptoIDGenerator emits RFC 4122 version-4 UUIDs using crypto/rand.
type CryptoIDGenerator struct{}

func (CryptoIDGenerator) NewID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(raw[0:4]),
		binary.BigEndian.Uint16(raw[4:6]),
		binary.BigEndian.Uint16(raw[6:8]),
		binary.BigEndian.Uint16(raw[8:10]),
		raw[10:16],
	), nil
}

// SecretMaterial is resolved at runtime from *_REF names. Raw bytes must never
// be read from ordinary configuration, logged, serialized or returned by APIs.
type SecretMaterial struct {
	OTPKey     []byte
	SubjectKey []byte
	ProofKey   []byte
	SigningKey []byte
	RefreshKey []byte
}

func (s SecretMaterial) validate() error {
	for name, value := range map[string][]byte{
		"EMAIL_OTP_PEPPER_REF":           s.OTPKey,
		"IDENTITY_SUBJECT_KEY_REF":       s.SubjectKey,
		"IDENTITY_PROOF_SIGNING_KEY_REF": s.ProofKey,
		"IDENTITY_SIGNING_KEY_REF":       s.SigningKey,
		"IDENTITY_REFRESH_KEY_REF":       s.RefreshKey,
	} {
		if len(value) < 32 {
			return fmt.Errorf("resolved secret for %s is shorter than 32 bytes", name)
		}
	}
	return nil
}

func keyedHex(key []byte, parts ...string) string {
	mac := hmac.New(sha256.New, key)
	for index, part := range parts {
		if index > 0 {
			_, _ = mac.Write([]byte{0})
		}
		_, _ = mac.Write([]byte(part))
	}
	return hex.EncodeToString(mac.Sum(nil))
}

func subjectDigest(key []byte, provider Provider, subject string) string {
	return keyedHex(key, "identity-subject-v1", string(provider), subject)
}

func deriveOTP(key []byte, verificationID string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("identity-otp-v1\x00" + verificationID))
	value := binary.BigEndian.Uint64(mac.Sum(nil)[:8]) % 1_000_000
	return fmt.Sprintf("%06d", value)
}

func otpDigest(key []byte, verificationID, code string) string {
	return keyedHex(key, "identity-otp-digest-v1", verificationID, code)
}

func secureDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func proofToken(key []byte, verification Verification) string {
	expires := verification.ProofExpiresAt.Unix()
	payload := verification.VerificationID + "|" + string(verification.Provider) + "|" +
		verification.ProviderSubjectHash + "|" + strconv.FormatInt(expires, 10)
	signature := hmacBytes(key, "identity-proof-v1\x00"+payload)
	return verification.VerificationID + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func proofID(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", errors.New("malformed proof token")
	}
	if _, err := base64.RawURLEncoding.DecodeString(parts[1]); err != nil {
		return "", errors.New("malformed proof signature")
	}
	return parts[0], nil
}

func verifyProofToken(key []byte, token string, verification Verification) bool {
	if verification.ProofExpiresAt == nil {
		return false
	}
	expected := proofToken(key, verification)
	return hmac.Equal([]byte(expected), []byte(token)) &&
		hmac.Equal([]byte(verification.ProofHash), []byte(secureDigest(token)))
}

func hmacBytes(key []byte, text string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(text))
	return mac.Sum(nil)
}
