package billing

// Store 为权益/报价/冻结存储（生产 PostgreSQL；余额扣减由 TASK-061 账本驱动）。
type Store interface {
	SaveEntitlement(Entitlement, string) error
	GetEntitlementByIdempotencyKey(dataRegion, key string) (Entitlement, error)
	ListEntitlements(dataRegion, userID string) ([]Entitlement, error)
	UpdateEntitlement(Entitlement) error
	SaveQuote(Quote, string) error
	GetQuoteByIdempotencyKey(dataRegion, key string) (Quote, error)
	GetQuote(dataRegion, quoteID string) (Quote, error)
	ListQuotes(dataRegion, projectID string) ([]Quote, error)
	UpdateQuote(Quote) error
	SaveFreeze(Freeze, string) error
	GetFreeze(dataRegion, projectID string) (Freeze, error)
	SaveSubscription(ProSubscription, string) error
	GetSubscription(dataRegion, userID string) (ProSubscription, error)
	// TASK-061 秒级账本（追加式；幂等键唯一）。
	AppendLedger(LedgerEntry) error
	GetLedgerByIdempotencyKey(dataRegion, key string) (LedgerEntry, error)
	GetLedgerByProject(dataRegion, projectID string) ([]LedgerEntry, error)
	SaveMeter(Meter) error
	GetMeter(dataRegion, sessionID string) (Meter, error)
}
