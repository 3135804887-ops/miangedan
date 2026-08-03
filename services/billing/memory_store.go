package billing

import (
	"sort"
	"sync"
	"time"
)

// MemoryStore 为内存版存储（开发/测试；生产 PostgreSQL）。
type MemoryStore struct {
	mu               sync.RWMutex
	entitlements     map[string]Entitlement
	entitlementID    map[string]Entitlement
	quotes           map[string]Quote
	quoteID          map[string]Quote
	quotesByProj     map[string][]Quote
	freezes          map[string]Freeze
	subscriptions    map[string]ProSubscription
	ledger           map[string][]LedgerEntry
	ledgerIDem       map[string]LedgerEntry
	meters           map[string]Meter
	orders           map[string]Order
	orderIDem        map[string]Order
	ordersByUser     map[string][]Order
	paymentEvents    map[string]PaymentEvent
	incidents        map[string][]Incident
	refunds          map[string]Refund
	refundIDem       map[string]Refund
	refundsByOrd     map[string][]Refund
	refundsByUser    map[string][]Refund
	subscriptionIDem map[string]ProSubscription
	renewals         map[string]RenewalRecord
	renewalIDem      map[string]RenewalRecord
	renewalsBySub    map[string][]RenewalRecord
}

// NewMemoryStore 创建空内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		entitlements:     make(map[string]Entitlement),
		entitlementID:    make(map[string]Entitlement),
		quotes:           make(map[string]Quote),
		quoteID:          make(map[string]Quote),
		quotesByProj:     make(map[string][]Quote),
		freezes:          make(map[string]Freeze),
		subscriptions:    make(map[string]ProSubscription),
		ledger:           make(map[string][]LedgerEntry),
		ledgerIDem:       make(map[string]LedgerEntry),
		meters:           make(map[string]Meter),
		orders:           make(map[string]Order),
		orderIDem:        make(map[string]Order),
		ordersByUser:     make(map[string][]Order),
		paymentEvents:    make(map[string]PaymentEvent),
		incidents:        make(map[string][]Incident),
		refunds:          make(map[string]Refund),
		refundIDem:       make(map[string]Refund),
		refundsByOrd:     make(map[string][]Refund),
		refundsByUser:    make(map[string][]Refund),
		subscriptionIDem: make(map[string]ProSubscription),
		renewals:         make(map[string]RenewalRecord),
		renewalIDem:      make(map[string]RenewalRecord),
		renewalsBySub:    make(map[string][]RenewalRecord),
	}
}

// SaveEntitlement 保存权益。
func (m *MemoryStore) SaveEntitlement(e Entitlement, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entitlementID[e.DataRegion+"|"+e.EntitlementID] = e
	if idemKey != "" {
		m.entitlements[e.DataRegion+"|"+idemKey] = e
	}
	return nil
}

// GetEntitlementByIdempotencyKey 幂等键查询权益。
func (m *MemoryStore) GetEntitlementByIdempotencyKey(dataRegion, key string) (Entitlement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.entitlements[dataRegion+"|"+key]
	if !ok {
		return Entitlement{}, ErrNotFound
	}
	return e, nil
}

// ListEntitlements 列出用户权益。
func (m *MemoryStore) ListEntitlements(dataRegion, userID string) ([]Entitlement, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Entitlement, 0)
	for _, e := range m.entitlementID {
		if e.DataRegion == dataRegion && e.UserID == userID {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ValidFrom.Before(out[j].ValidFrom)
	})
	return out, nil
}

// UpdateEntitlement 更新权益（消耗由账本驱动，只增）。
func (m *MemoryStore) UpdateEntitlement(e Entitlement) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entitlementID[e.DataRegion+"|"+e.EntitlementID] = e
	return nil
}

// SaveQuote 保存报价。
func (m *MemoryStore) SaveQuote(q Quote, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.quoteID[q.DataRegion+"|"+q.QuoteID] = q
	if idemKey != "" {
		m.quotes[q.DataRegion+"|"+idemKey] = q
	}
	m.quotesByProj[q.DataRegion+"|"+q.ProjectID] =
		append(m.quotesByProj[q.DataRegion+"|"+q.ProjectID], q)
	return nil
}

// GetQuoteByIdempotencyKey 幂等键查询报价。
func (m *MemoryStore) GetQuoteByIdempotencyKey(dataRegion, key string) (Quote, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	q, ok := m.quotes[dataRegion+"|"+key]
	if !ok {
		return Quote{}, ErrNotFound
	}
	return q, nil
}

// GetQuote 按报价 ID 查询。
func (m *MemoryStore) GetQuote(dataRegion, quoteID string) (Quote, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	q, ok := m.quoteID[dataRegion+"|"+quoteID]
	if !ok {
		return Quote{}, ErrNotFound
	}
	return q, nil
}

// ListQuotes 列出项目报价（版本升序）。
func (m *MemoryStore) ListQuotes(dataRegion, projectID string) ([]Quote, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.quotesByProj[dataRegion+"|"+projectID]
	out := make([]Quote, len(items))
	copy(out, items)
	sort.Slice(out, func(i, j int) bool { return out[i].PlanVersion < out[j].PlanVersion })
	return out, nil
}

// UpdateQuote 更新报价状态。
func (m *MemoryStore) UpdateQuote(q Quote) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.quoteID[q.DataRegion+"|"+q.QuoteID] = q
	return nil
}

// SaveFreeze 保存计费版本冻结。
func (m *MemoryStore) SaveFreeze(f Freeze, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.freezes[f.DataRegion+"|"+f.ProjectID] = f
	return nil
}

// GetFreeze 查询计费版本冻结。
func (m *MemoryStore) GetFreeze(dataRegion, projectID string) (Freeze, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.freezes[dataRegion+"|"+projectID]
	if !ok {
		return Freeze{}, ErrNotFound
	}
	return f, nil
}

// SaveSubscription 保存订阅。
func (m *MemoryStore) SaveSubscription(s ProSubscription, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriptions[s.DataRegion+"|"+s.UserID] = s
	if idemKey != "" {
		m.subscriptionIDem[s.DataRegion+"|"+idemKey] = s
	}
	return nil
}

// GetSubscription 查询用户订阅。
func (m *MemoryStore) GetSubscription(dataRegion, userID string) (ProSubscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.subscriptions[dataRegion+"|"+userID]
	if !ok {
		return ProSubscription{}, ErrNotFound
	}
	return s, nil
}

// UpdateSubscription 更新订阅（自动续费条款/状态）。
func (m *MemoryStore) UpdateSubscription(s ProSubscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriptions[s.DataRegion+"|"+s.UserID] = s
	if s.IdempotencyKey != "" {
		m.subscriptionIDem[s.DataRegion+"|"+s.IdempotencyKey] = s
	}
	return nil
}

// GetSubscriptionByIdempotencyKey 幂等键查询订阅操作结果。
func (m *MemoryStore) GetSubscriptionByIdempotencyKey(dataRegion, key string) (ProSubscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.subscriptionIDem[dataRegion+"|"+key]
	if !ok {
		return ProSubscription{}, ErrNotFound
	}
	return s, nil
}

// ListSubscriptions 列出区域全部订阅（到期扫描任务使用）。
func (m *MemoryStore) ListSubscriptions(dataRegion string) ([]ProSubscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ProSubscription, 0)
	for _, s := range m.subscriptions {
		if s.DataRegion == dataRegion {
			out = append(out, s)
		}
	}
	return out, nil
}

// SaveRenewalRecord 保存续费事件。
func (m *MemoryStore) SaveRenewalRecord(r RenewalRecord, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.renewals[r.DataRegion+"|"+r.RenewalID] = r
	if idemKey != "" {
		m.renewalIDem[r.DataRegion+"|"+idemKey] = r
	}
	m.renewalsBySub[r.DataRegion+"|"+r.SubscriptionID] =
		append(m.renewalsBySub[r.DataRegion+"|"+r.SubscriptionID], r)
	return nil
}

// GetRenewalByIdempotencyKey 幂等键查询续费事件。
func (m *MemoryStore) GetRenewalByIdempotencyKey(dataRegion, key string) (RenewalRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.renewalIDem[dataRegion+"|"+key]
	if !ok {
		return RenewalRecord{}, ErrNotFound
	}
	return r, nil
}

// GetRenewalByID 按续费事件 ID 查询。
func (m *MemoryStore) GetRenewalByID(dataRegion, renewalID string) (RenewalRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.renewals[dataRegion+"|"+renewalID]
	if !ok {
		return RenewalRecord{}, ErrNotFound
	}
	return r, nil
}

// UpdateRenewalRecord 更新续费事件状态。
func (m *MemoryStore) UpdateRenewalRecord(r RenewalRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.renewals[r.DataRegion+"|"+r.RenewalID] = r
	return nil
}

// ListRenewalsBySubscription 列出订阅的全部续费事件。
func (m *MemoryStore) ListRenewalsBySubscription(dataRegion, subscriptionID string) ([]RenewalRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.renewalsBySub[dataRegion+"|"+subscriptionID]
	out := make([]RenewalRecord, len(items))
	copy(out, items)
	return out, nil
}

// AppendLedger 追加账本条目（幂等键唯一由服务层保证）。
func (m *MemoryStore) AppendLedger(e LedgerEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ledger[e.DataRegion+"|"+e.ProjectID] =
		append(m.ledger[e.DataRegion+"|"+e.ProjectID], e)
	m.ledgerIDem[e.DataRegion+"|"+e.IdempotencyKey] = e
	return nil
}

// GetLedgerByIdempotencyKey 幂等键查询账本条目。
func (m *MemoryStore) GetLedgerByIdempotencyKey(dataRegion, key string) (LedgerEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.ledgerIDem[dataRegion+"|"+key]
	if !ok {
		return LedgerEntry{}, ErrNotFound
	}
	return e, nil
}

// GetLedgerByProject 列出项目账本（创建时间升序）。
func (m *MemoryStore) GetLedgerByProject(dataRegion, projectID string) ([]LedgerEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.ledger[dataRegion+"|"+projectID]
	out := make([]LedgerEntry, len(items))
	copy(out, items)
	return out, nil
}

// SaveMeter 保存计量状态。
func (m *MemoryStore) SaveMeter(meter Meter) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.meters[meter.DataRegion+"|"+meter.SessionID] = meter
	return nil
}

// GetMeter 查询计量状态。
func (m *MemoryStore) GetMeter(dataRegion, sessionID string) (Meter, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	meter, ok := m.meters[dataRegion+"|"+sessionID]
	if !ok {
		return Meter{}, ErrNotFound
	}
	return meter, nil
}

// SaveOrder 保存订单。
func (m *MemoryStore) SaveOrder(o Order, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orders[o.DataRegion+"|"+o.OrderID] = o
	if idemKey != "" {
		m.orderIDem[o.DataRegion+"|"+idemKey] = o
	}
	m.ordersByUser[o.DataRegion+"|"+o.UserID] =
		append(m.ordersByUser[o.DataRegion+"|"+o.UserID], o)
	return nil
}

// GetOrderByIdempotencyKey 幂等键查询订单。
func (m *MemoryStore) GetOrderByIdempotencyKey(dataRegion, key string) (Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.orderIDem[dataRegion+"|"+key]
	if !ok {
		return Order{}, ErrNotFound
	}
	return o, nil
}

// GetOrderByID 按订单 ID 查询。
func (m *MemoryStore) GetOrderByID(dataRegion, orderID string) (Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	o, ok := m.orders[dataRegion+"|"+orderID]
	if !ok {
		return Order{}, ErrNotFound
	}
	return o, nil
}

// ListOrdersByUser 列出用户订单。
func (m *MemoryStore) ListOrdersByUser(dataRegion, userID string) ([]Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.ordersByUser[dataRegion+"|"+userID]
	out := make([]Order, len(items))
	copy(out, items)
	return out, nil
}

// UpdateOrder 更新订单状态。
func (m *MemoryStore) UpdateOrder(o Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orders[o.DataRegion+"|"+o.OrderID] = o
	return nil
}

// SavePaymentEvent 保存支付事件（provider + event_id 去重由服务层保证）。
func (m *MemoryStore) SavePaymentEvent(e PaymentEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.paymentEvents[e.Provider+"|"+e.PaymentEventID] = e
	return nil
}

// GetPaymentEvent 查询支付事件。
func (m *MemoryStore) GetPaymentEvent(provider, eventID string) (PaymentEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.paymentEvents[provider+"|"+eventID]
	if !ok {
		return PaymentEvent{}, ErrNotFound
	}
	return e, nil
}

// SaveIncident 追加事故记录。
func (m *MemoryStore) SaveIncident(i Incident) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.incidents[i.DataRegion] = append(m.incidents[i.DataRegion], i)
	return nil
}

// ListIncidents 列出区域事故。
func (m *MemoryStore) ListIncidents(dataRegion string) ([]Incident, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.incidents[dataRegion]
	out := make([]Incident, len(items))
	copy(out, items)
	return out, nil
}

// SaveRefund 保存退款。
func (m *MemoryStore) SaveRefund(r Refund, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refunds[r.DataRegion+"|"+r.RefundID] = r
	if idemKey != "" {
		m.refundIDem[r.DataRegion+"|"+idemKey] = r
	}
	m.refundsByOrd[r.DataRegion+"|"+r.OrderID] =
		append(m.refundsByOrd[r.DataRegion+"|"+r.OrderID], r)
	m.refundsByUser[r.DataRegion+"|"+r.UserID] =
		append(m.refundsByUser[r.DataRegion+"|"+r.UserID], r)
	return nil
}

// GetRefundByIdempotencyKey 幂等键查询退款。
func (m *MemoryStore) GetRefundByIdempotencyKey(dataRegion, key string) (Refund, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.refundIDem[dataRegion+"|"+key]
	if !ok {
		return Refund{}, ErrNotFound
	}
	return r, nil
}

// GetRefundByID 按退款 ID 查询。
func (m *MemoryStore) GetRefundByID(dataRegion, refundID string) (Refund, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.refunds[dataRegion+"|"+refundID]
	if !ok {
		return Refund{}, ErrNotFound
	}
	return r, nil
}

// ListRefundsByOrder 列出订单退款。
func (m *MemoryStore) ListRefundsByOrder(dataRegion, orderID string) ([]Refund, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.refundsByOrd[dataRegion+"|"+orderID]
	out := make([]Refund, len(items))
	copy(out, items)
	return out, nil
}

// ListRefundsByUser 列出用户退款。
func (m *MemoryStore) ListRefundsByUser(dataRegion, userID string) ([]Refund, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.refundsByUser[dataRegion+"|"+userID]
	out := make([]Refund, len(items))
	copy(out, items)
	return out, nil
}

// UpdateRefund 更新退款状态。
func (m *MemoryStore) UpdateRefund(r Refund) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refunds[r.DataRegion+"|"+r.RefundID] = r
	return nil
}

// AppendRefundApproval 原子追加审批人（同一审批人去重；返回当前审批对）。
func (m *MemoryStore) AppendRefundApproval(dataRegion, refundID, approverID string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.refunds[dataRegion+"|"+refundID]
	if !ok {
		return nil, ErrNotFound
	}
	for _, existing := range r.ApproverPair {
		if existing == approverID {
			return append([]string(nil), r.ApproverPair...), nil
		}
	}
	r.ApproverPair = append(r.ApproverPair, approverID)
	m.refunds[dataRegion+"|"+refundID] = r
	return append([]string(nil), r.ApproverPair...), nil
}

// MarkRefundExecuted 原子标记退款已执行（幂等：已执行返回 fresh=false）。
func (m *MemoryStore) MarkRefundExecuted(
	dataRegion, refundID string, at time.Time,
) (Refund, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.refunds[dataRegion+"|"+refundID]
	if !ok {
		return Refund{}, false, ErrNotFound
	}
	if r.Status == Refunded {
		return r, false, nil
	}
	r.Status = Refunded
	r.RefundedAt = &at
	m.refunds[dataRegion+"|"+refundID] = r
	return r, true, nil
}
