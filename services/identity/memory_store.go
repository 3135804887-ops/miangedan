package identity

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	errDuplicateRecord = errors.New("duplicate identity record")
	errMissingRecord   = errors.New("identity record not found")
)

type memoryState struct {
	verifications     map[string]Verification
	verificationByKey map[string]string
	users             map[string]User
	identities        map[string]Identity
	identityBySubject map[string]string
	sessions          map[string]SessionRecord
	recoveryCases     map[string]RecoveryCase
}

func newMemoryState() memoryState {
	return memoryState{
		verifications:     make(map[string]Verification),
		verificationByKey: make(map[string]string),
		users:             make(map[string]User),
		identities:        make(map[string]Identity),
		identityBySubject: make(map[string]string),
		sessions:          make(map[string]SessionRecord),
		recoveryCases:     make(map[string]RecoveryCase),
	}
}

// MemoryStore provides serializable, rollback-on-error transactions. It is
// deliberately deterministic and contains no network or provider behavior.
type MemoryStore struct {
	mu    sync.Mutex
	state memoryState
}

// NewMemoryStore creates an empty regional identity reference store.
func NewMemoryStore() *MemoryStore { return &MemoryStore{state: newMemoryState()} }

// Transact serializes a callback and commits a cloned state only on success.
func (s *MemoryStore) Transact(ctx context.Context, fn func(Tx) error) error {
	if ctx == nil || fn == nil {
		return errors.New("identity transaction requires context and callback")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	working := cloneMemoryState(s.state)
	if err := fn((*memoryTx)(&working)); err != nil {
		return err
	}
	s.state = working
	return nil
}

type memoryTx memoryState

func (tx *memoryTx) VerificationByID(id string) (Verification, bool, error) {
	v, ok := tx.verifications[id]
	return cloneVerification(v), ok, nil
}

func requestKey(dataRegion, key string) string { return dataRegion + "\x00" + key }

func subjectKey(dataRegion string, provider Provider, subjectHash string) string {
	return dataRegion + "\x00" + string(provider) + "\x00" + subjectHash
}

func (tx *memoryTx) VerificationByRequestKey(dataRegion, key string) (Verification, bool, error) {
	id, ok := tx.verificationByKey[requestKey(dataRegion, key)]
	if !ok {
		return Verification{}, false, nil
	}
	return tx.VerificationByID(id)
}

func (tx *memoryTx) RecentVerificationCount(dataRegion string, provider Provider, subjectHash string, since time.Time) (int, error) {
	count := 0
	for _, verification := range tx.verifications {
		if verification.DataRegion == dataRegion && verification.Provider == provider &&
			verification.ProviderSubjectHash == subjectHash && !verification.RequestedAt.Before(since) {
			count++
		}
	}
	return count, nil
}

func (tx *memoryTx) CreateVerification(v Verification) error {
	if _, exists := tx.verifications[v.VerificationID]; exists {
		return errDuplicateRecord
	}
	key := requestKey(v.DataRegion, v.RequestKey)
	if _, exists := tx.verificationByKey[key]; exists {
		return errDuplicateRecord
	}
	tx.verifications[v.VerificationID] = cloneVerification(v)
	tx.verificationByKey[key] = v.VerificationID
	return nil
}

func (tx *memoryTx) UpdateVerification(v Verification) error {
	if _, exists := tx.verifications[v.VerificationID]; !exists {
		return errMissingRecord
	}
	tx.verifications[v.VerificationID] = cloneVerification(v)
	return nil
}

func (tx *memoryTx) UserByID(userID string) (User, bool, error) {
	u, ok := tx.users[userID]
	return cloneUser(u), ok, nil
}

func (tx *memoryTx) CreateUser(u User) error {
	if _, exists := tx.users[u.UserID]; exists {
		return errDuplicateRecord
	}
	tx.users[u.UserID] = cloneUser(u)
	return nil
}

func (tx *memoryTx) UpdateUser(u User) error {
	if _, exists := tx.users[u.UserID]; !exists {
		return errMissingRecord
	}
	tx.users[u.UserID] = cloneUser(u)
	return nil
}

func (tx *memoryTx) IdentityBySubject(dataRegion string, provider Provider, hash string) (Identity, bool, error) {
	id, ok := tx.identityBySubject[subjectKey(dataRegion, provider, hash)]
	if !ok {
		return Identity{}, false, nil
	}
	identity, ok := tx.identities[id]
	return identity, ok, nil
}

func (tx *memoryTx) IdentitiesByUser(userID string) ([]Identity, error) {
	identities := make([]Identity, 0)
	for _, item := range tx.identities {
		if item.UserID == userID {
			identities = append(identities, item)
		}
	}
	sort.Slice(identities, func(i, j int) bool {
		if identities[i].Provider == identities[j].Provider {
			return identities[i].IdentityID < identities[j].IdentityID
		}
		return identities[i].Provider < identities[j].Provider
	})
	return identities, nil
}

func (tx *memoryTx) CreateIdentity(item Identity) error {
	if _, exists := tx.identities[item.IdentityID]; exists {
		return errDuplicateRecord
	}
	user, exists := tx.users[item.UserID]
	if !exists || user.DataRegion != item.DataRegion {
		return errMissingRecord
	}
	key := subjectKey(item.DataRegion, item.Provider, item.ProviderSubjectHash)
	if _, exists := tx.identityBySubject[key]; exists {
		return errDuplicateRecord
	}
	tx.identities[item.IdentityID] = item
	tx.identityBySubject[key] = item.IdentityID
	return nil
}

func (tx *memoryTx) SessionByID(sessionID string) (SessionRecord, bool, error) {
	record, ok := tx.sessions[sessionID]
	return cloneSessionRecord(record), ok, nil
}

func (tx *memoryTx) CreateSession(record SessionRecord) error {
	if _, exists := tx.sessions[record.SessionID]; exists {
		return errDuplicateRecord
	}
	user, exists := tx.users[record.UserID]
	if !exists || user.DataRegion != record.DataRegion {
		return errMissingRecord
	}
	for _, existing := range tx.sessions {
		if existing.RefreshTokenHash == record.RefreshTokenHash {
			return errDuplicateRecord
		}
	}
	tx.sessions[record.SessionID] = cloneSessionRecord(record)
	return nil
}

func (tx *memoryTx) UpdateSession(record SessionRecord) error {
	if _, exists := tx.sessions[record.SessionID]; !exists {
		return errMissingRecord
	}
	tx.sessions[record.SessionID] = cloneSessionRecord(record)
	return nil
}

func (tx *memoryTx) CreateRecoveryCase(recovery RecoveryCase) error {
	if _, exists := tx.recoveryCases[recovery.RecoveryCaseID]; exists {
		return errDuplicateRecord
	}
	tx.recoveryCases[recovery.RecoveryCaseID] = recovery
	return nil
}

// MemoryStats supports behavior tests without exposing stored sensitive values.
type MemoryStats struct {
	Users, Identities, Verifications, Sessions, RecoveryCases int
}

// Stats returns record counts only.
func (s *MemoryStore) Stats() MemoryStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return MemoryStats{
		Users:         len(s.state.users),
		Identities:    len(s.state.identities),
		Verifications: len(s.state.verifications),
		Sessions:      len(s.state.sessions),
		RecoveryCases: len(s.state.recoveryCases),
	}
}

func cloneMemoryState(source memoryState) memoryState {
	destination := newMemoryState()
	for key, value := range source.verifications {
		destination.verifications[key] = cloneVerification(value)
	}
	for key, value := range source.verificationByKey {
		destination.verificationByKey[key] = value
	}
	for key, value := range source.users {
		destination.users[key] = cloneUser(value)
	}
	for key, value := range source.identities {
		destination.identities[key] = value
	}
	for key, value := range source.identityBySubject {
		destination.identityBySubject[key] = value
	}
	for key, value := range source.sessions {
		destination.sessions[key] = cloneSessionRecord(value)
	}
	for key, value := range source.recoveryCases {
		destination.recoveryCases[key] = value
	}
	return destination
}

func cloneUser(source User) User {
	cloned := source
	if source.DisplayName != nil {
		value := *source.DisplayName
		cloned.DisplayName = &value
	}
	return cloned
}

func cloneVerification(source Verification) Verification {
	cloned := source
	cloned.VerifiedAt = cloneTimePointer(source.VerifiedAt)
	cloned.ProofExpiresAt = cloneTimePointer(source.ProofExpiresAt)
	cloned.ConsumedAt = cloneTimePointer(source.ConsumedAt)
	cloned.NotificationSentAt = cloneTimePointer(source.NotificationSentAt)
	return cloned
}

func cloneSessionRecord(source SessionRecord) SessionRecord {
	cloned := source
	cloned.RotatedAt = cloneTimePointer(source.RotatedAt)
	return cloned
}

func cloneTimePointer(source *time.Time) *time.Time {
	if source == nil {
		return nil
	}
	cloned := *source
	return &cloned
}
