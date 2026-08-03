// Package adminapi 提供后台内存存储（TASK-080；开发/测试；生产 PostgreSQL）。
package adminapi

import "sync"

// MemoryStore 为内存版后台存储。
type MemoryStore struct {
	mu           sync.RWMutex
	providers    map[string]ProviderInfo
	rooms        map[string][]RoomSnapshot
	regionStatus map[string]RegionOpsStatus
	audits       map[string][]AuditEntry
}

// NewMemoryStore 创建空内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		providers:    make(map[string]ProviderInfo),
		rooms:        make(map[string][]RoomSnapshot),
		regionStatus: make(map[string]RegionOpsStatus),
		audits:       make(map[string][]AuditEntry),
	}
}

// SaveProvider 保存供应商条目。
func (m *MemoryStore) SaveProvider(p ProviderInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[p.Region+"|"+p.ProviderID] = p
	return nil
}

// GetProvider 查询供应商。
func (m *MemoryStore) GetProvider(dataRegion, providerID string) (ProviderInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.providers[dataRegion+"|"+providerID]
	if !ok {
		return ProviderInfo{}, ErrNotFound
	}
	return p, nil
}

// ListProviders 列出区域供应商。
func (m *MemoryStore) ListProviders(dataRegion string) ([]ProviderInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ProviderInfo, 0)
	for _, p := range m.providers {
		if p.Region == dataRegion {
			out = append(out, p)
		}
	}
	return out, nil
}

// UpdateProvider 更新供应商状态/指标。
func (m *MemoryStore) UpdateProvider(p ProviderInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[p.Region+"|"+p.ProviderID] = p
	return nil
}

// SaveRoomSnapshot 追加匿名房间快照。
func (m *MemoryStore) SaveRoomSnapshot(r RoomSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rooms[r.Region] = append(m.rooms[r.Region], r)
	return nil
}

// ListRoomSnapshots 列出区域房间快照。
func (m *MemoryStore) ListRoomSnapshots(dataRegion string) ([]RoomSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.rooms[dataRegion]
	out := make([]RoomSnapshot, len(items))
	copy(out, items)
	return out, nil
}

// SaveRegionStatus 保存区域监控快照。
func (m *MemoryStore) SaveRegionStatus(s RegionOpsStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.regionStatus[s.DataRegion] = s
	return nil
}

// GetRegionStatus 查询区域监控快照。
func (m *MemoryStore) GetRegionStatus(dataRegion string) (RegionOpsStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.regionStatus[dataRegion]
	if !ok {
		return RegionOpsStatus{}, ErrNotFound
	}
	return s, nil
}

// AppendAudit 追加审计（只增不改；无更新/删除路径）。
func (m *MemoryStore) AppendAudit(a AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audits[a.DataRegion] = append(m.audits[a.DataRegion], a)
	return nil
}

// ListAudits 查询审计。
func (m *MemoryStore) ListAudits(dataRegion string) ([]AuditEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.audits[dataRegion]
	out := make([]AuditEntry, len(items))
	copy(out, items)
	return out, nil
}
