package identity

import (
	"context"
	"time"
)

// Store provides serializable transactions for identity state. PostgreSQL is
// the production persistence defined by migration 0010; MemoryStore is the
// deterministic reference implementation used by unit and contract tests.
type Store interface {
	Transact(context.Context, func(Tx) error) error
}

// Tx exposes only TASK-010-owned records. Implementations must enforce region
// equality and the unique (region, provider, provider_subject_hash) constraint.
type Tx interface {
	VerificationByID(id string) (Verification, bool, error)
	VerificationByRequestKey(dataRegion, requestKey string) (Verification, bool, error)
	RecentVerificationCount(dataRegion string, provider Provider, subjectHash string, since time.Time) (int, error)
	CreateVerification(Verification) error
	UpdateVerification(Verification) error

	UserByID(userID string) (User, bool, error)
	CreateUser(User) error
	UpdateUser(User) error

	IdentityBySubject(dataRegion string, provider Provider, subjectHash string) (Identity, bool, error)
	IdentitiesByUser(userID string) ([]Identity, error)
	CreateIdentity(Identity) error

	SessionByID(sessionID string) (SessionRecord, bool, error)
	CreateSession(SessionRecord) error
	UpdateSession(SessionRecord) error

	CreateRecoveryCase(RecoveryCase) error
}
