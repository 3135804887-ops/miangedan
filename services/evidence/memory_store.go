package evidence

import (
	"sort"
	"sync"
)

// MemoryStore 为内存版证据账本（开发/测试；生产 PostgreSQL 只读+插入角色）。
type MemoryStore struct {
	mu      sync.RWMutex
	byEvent map[string]Entry
	bySess  map[string][]Entry
}

// NewMemoryStore 创建空内存证据账本。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byEvent: make(map[string]Entry), bySess: make(map[string][]Entry)}
}

// SaveEntry 保存条目（同 event_id 覆盖等价，幂等由服务层保证）。
func (m *MemoryStore) SaveEntry(e Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := e.DataRegion + "|" + e.EventID
	if _, ok := m.byEvent[key]; !ok {
		m.bySess[e.DataRegion+"|"+e.SessionID] = append(m.bySess[e.DataRegion+"|"+e.SessionID], e)
	}
	m.byEvent[key] = e
	return nil
}

// GetByEventID 查询单条。
func (m *MemoryStore) GetByEventID(dataRegion, eventID string) (Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.byEvent[dataRegion+"|"+eventID]
	if !ok {
		return Entry{}, ErrEntryNotFound
	}
	return e, nil
}

// ListBySession 列出会话证据（recorded_at 升序）。
func (m *MemoryStore) ListBySession(dataRegion, sessionID string) ([]Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.bySess[dataRegion+"|"+sessionID]
	out := make([]Entry, len(items))
	copy(out, items)
	sort.Slice(out, func(i, j int) bool { return out[i].RecordedAt.Before(out[j].RecordedAt) })
	return out, nil
}
