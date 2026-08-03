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
	// TASK-062 支付订单与回调去重（幂等键唯一；同一订单只记一次权益和一次扣款）。
	SaveOrder(Order, string) error
	GetOrderByIdempotencyKey(dataRegion, key string) (Order, error)
	GetOrderByID(dataRegion, orderID string) (Order, error)
	ListOrdersByUser(dataRegion, userID string) ([]Order, error)
	UpdateOrder(Order) error
	SavePaymentEvent(PaymentEvent) error
	GetPaymentEvent(provider, eventID string) (PaymentEvent, error)
	SaveIncident(Incident) error
	ListIncidents(dataRegion string) ([]Incident, error)
	SaveRefund(Refund, string) error
	GetRefundByIdempotencyKey(dataRegion, key string) (Refund, error)
	GetRefundByID(dataRegion, refundID string) (Refund, error)
	ListRefundsByOrder(dataRegion, orderID string) ([]Refund, error)
	ListRefundsByUser(dataRegion, userID string) ([]Refund, error)
	UpdateRefund(Refund) error
}
