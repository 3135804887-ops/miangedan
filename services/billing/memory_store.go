package billing

import (
	"sort"
	"sync"
)

// MemoryStore 为内存版存储（开发/测试；生产 PostgreSQL）。
type MemoryStore struct {
	mu            sync.RWMutex
	entitlements  map[string]Entitlement
	entitlementID map[string]Entitlement
	quotes        map[string]Quote
	quoteID       map[string]Quote
	quotesByProj  map[string][]Quote
	freezes       map[string]Freeze
	subscriptions map[string]ProSubscription
	ledger        map[string][]LedgerEntry
	ledgerIDem    map[string]LedgerEntry
	meters        map[string]Meter
}

// NewMemoryStore 创建空内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		entitlements:  make(map[string]Entitlement),
		entitlementID: make(map[string]Entitlement),
		quotes:        make(map[string]Quote),
		quoteID:       make(map[string]Quote),
		quotesByProj:  make(map[string][]Quote),
		freezes:       make(map[string]Freeze),
		subscriptions: make(map[string]ProSubscription),
		ledger:        make(map[string][]LedgerEntry),
		ledgerIDem:    make(map[string]LedgerEntry),
		meters:        make(map[string]Meter),
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
func (m *MemoryStore) SaveSubscription(s ProSubscription, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriptions[s.DataRegion+"|"+s.UserID] = s
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
