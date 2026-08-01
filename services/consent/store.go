package consent

import "context"

// Store provides serializable regional consent transactions.
type Store interface {
	Transact(context.Context, func(Tx) error) error
}

// Tx is the minimal append-only persistence contract used by Service.
type Tx interface {
	GrantByRequest(userID, dataRegion, operation, requestKey string) (Grant, bool, error)
	LatestGrant(userID, dataRegion string, consentType Type, scopeHash string) (Grant, bool, error)
	LatestGrantsByUser(userID, dataRegion string) ([]Grant, error)
	HistoryByType(userID, dataRegion string, consentType Type) ([]Grant, error)
	AppendGrant(Grant) error
	AppendAudit(AuditEvent) error
}
