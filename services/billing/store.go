package billing

import "time"

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
	// TASK-064 订阅生命周期：同意条款、幂等、到期扫描、续费事件。
	UpdateSubscription(ProSubscription) error
	GetSubscriptionByIdempotencyKey(dataRegion, key string) (ProSubscription, error)
	ListSubscriptions(dataRegion string) ([]ProSubscription, error)
	SaveRenewalRecord(RenewalRecord, string) error
	GetRenewalByIdempotencyKey(dataRegion, key string) (RenewalRecord, error)
	GetRenewalByID(dataRegion, renewalID string) (RenewalRecord, error)
	UpdateRenewalRecord(RenewalRecord) error
	ListRenewalsBySubscription(dataRegion, subscriptionID string) ([]RenewalRecord, error)
	// TASK-065 发票/收据（区域定价配置；同一订单一份；幂等）。
	SaveInvoice(Invoice, string) error
	GetInvoiceByIdempotencyKey(dataRegion, key string) (Invoice, error)
	GetInvoiceByOrder(dataRegion, orderID string) (Invoice, error)
	ListInvoicesByUser(dataRegion, userID string) ([]Invoice, error)
	UpdateInvoice(Invoice) error
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
	// TASK-063 退款审批与执行的原子操作（双人审批去重；执行幂等）。
	AppendRefundApproval(dataRegion, refundID, approverID string) ([]string, error)
	MarkRefundExecuted(dataRegion, refundID string, at time.Time) (Refund, bool, error)
}
