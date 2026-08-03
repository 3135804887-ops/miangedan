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
	versions     map[string]ArtifactVersion
	versionByKey map[string]ArtifactVersion
	pins         map[string]VersionPin
	activeSess   map[string]bool
	breakGlass   map[string]BreakGlass
	glassReviews map[string][]BreakGlassReview
	dataRights   map[string]DataRightRequest
	drIDem       map[string]DataRightRequest
}

// NewMemoryStore 创建空内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		providers:    make(map[string]ProviderInfo),
		rooms:        make(map[string][]RoomSnapshot),
		regionStatus: make(map[string]RegionOpsStatus),
		audits:       make(map[string][]AuditEntry),
		versions:     make(map[string]ArtifactVersion),
		versionByKey: make(map[string]ArtifactVersion),
		pins:         make(map[string]VersionPin),
		activeSess:   make(map[string]bool),
		breakGlass:   make(map[string]BreakGlass),
		glassReviews: make(map[string][]BreakGlassReview),
		dataRights:   make(map[string]DataRightRequest),
		drIDem:       make(map[string]DataRightRequest),
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

// SaveVersion 保存版本注册条目。
func (m *MemoryStore) SaveVersion(v ArtifactVersion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.versions[v.DataRegion+"|"+v.VersionID] = v
	m.versionByKey[v.DataRegion+"|"+v.AssetType+"|"+v.AssetKey] = v
	return nil
}

// GetVersion 按版本 ID 查询。
func (m *MemoryStore) GetVersion(dataRegion, versionID string) (ArtifactVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.versions[dataRegion+"|"+versionID]
	if !ok {
		return ArtifactVersion{}, ErrNotFound
	}
	return v, nil
}

// GetVersionByKey 按资产键查询（量表停用入口）。
func (m *MemoryStore) GetVersionByKey(dataRegion, assetType, assetKey string) (ArtifactVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.versionByKey[dataRegion+"|"+assetType+"|"+assetKey]
	if !ok {
		return ArtifactVersion{}, ErrNotFound
	}
	return v, nil
}

// UpdateVersion 更新版本阶段/废弃标记。
func (m *MemoryStore) UpdateVersion(v ArtifactVersion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.versions[v.DataRegion+"|"+v.VersionID] = v
	m.versionByKey[v.DataRegion+"|"+v.AssetType+"|"+v.AssetKey] = v
	return nil
}

// ListVersions 列出资产类型版本。
func (m *MemoryStore) ListVersions(dataRegion, assetType string) ([]ArtifactVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ArtifactVersion, 0)
	for _, v := range m.versions {
		if v.DataRegion == dataRegion && (assetType == "" || v.AssetType == assetType) {
			out = append(out, v)
		}
	}
	return out, nil
}

// SavePin 保存项目版本固定。
func (m *MemoryStore) SavePin(pin VersionPin) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pins[pin.DataRegion+"|"+pin.ProjectID] = pin
	return nil
}

// GetPin 查询项目版本固定。
func (m *MemoryStore) GetPin(dataRegion, projectID string) (VersionPin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pin, ok := m.pins[dataRegion+"|"+projectID]
	if !ok {
		return VersionPin{}, ErrNotFound
	}
	return pin, nil
}

// UpdatePin 更新固定版本（回滚）。
func (m *MemoryStore) UpdatePin(pin VersionPin) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pins[pin.DataRegion+"|"+pin.ProjectID] = pin
	return nil
}

// HasActiveSession 判断项目是否存在进行中的正式会话（回滚门禁）。
func (m *MemoryStore) HasActiveSession(dataRegion, projectID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeSess[dataRegion+"|"+projectID]
}

// SaveBreakGlass 插入破窗访问记录（INSERT only）。
func (m *MemoryStore) SaveBreakGlass(g BreakGlass) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.breakGlass[g.DataRegion+"|"+g.GlassID] = g
	return nil
}

// GetBreakGlass 查询破窗访问。
func (m *MemoryStore) GetBreakGlass(dataRegion, glassID string) (BreakGlass, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	g, ok := m.breakGlass[dataRegion+"|"+glassID]
	if !ok {
		return BreakGlass{}, ErrNotFound
	}
	return g, nil
}

// ListBreakGlassByTarget 列出目标用户破窗记录（敏感访问通知用）。
func (m *MemoryStore) ListBreakGlassByTarget(dataRegion, targetUserID string) ([]BreakGlass, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]BreakGlass, 0)
	for _, g := range m.breakGlass {
		if g.DataRegion == dataRegion && g.TargetUserID == targetUserID {
			out = append(out, g)
		}
	}
	return out, nil
}

// AppendBreakGlassReview 追加破窗复核（INSERT only）。
func (m *MemoryStore) AppendBreakGlassReview(r BreakGlassReview) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.glassReviews[r.GlassID] = append(m.glassReviews[r.GlassID], r)
	return nil
}

// ListBreakGlassReviews 查询破窗复核。
func (m *MemoryStore) ListBreakGlassReviews(glassID string) ([]BreakGlassReview, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.glassReviews[glassID]
	out := make([]BreakGlassReview, len(items))
	copy(out, items)
	return out, nil
}

// SaveDataRightRequest 保存数据权利请求。
func (m *MemoryStore) SaveDataRightRequest(req DataRightRequest, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dataRights[req.DataRegion+"|"+req.RequestID] = req
	if idemKey != "" {
		m.drIDem[req.DataRegion+"|"+idemKey] = req
	}
	return nil
}

// GetDataRightByID 按请求 ID 查询。
func (m *MemoryStore) GetDataRightByID(dataRegion, requestID string) (DataRightRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	req, ok := m.dataRights[dataRegion+"|"+requestID]
	if !ok {
		return DataRightRequest{}, ErrNotFound
	}
	return req, nil
}

// GetDataRightByIdempotencyKey 幂等键查询。
func (m *MemoryStore) GetDataRightByIdempotencyKey(dataRegion, key string) (DataRightRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	req, ok := m.drIDem[dataRegion+"|"+key]
	if !ok {
		return DataRightRequest{}, ErrNotFound
	}
	return req, nil
}

// UpdateDataRightRequest 更新请求状态/进度。
func (m *MemoryStore) UpdateDataRightRequest(req DataRightRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dataRights[req.DataRegion+"|"+req.RequestID] = req
	return nil
}

// ListDataRights 列出用户请求。
func (m *MemoryStore) ListDataRights(dataRegion, userID string) ([]DataRightRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]DataRightRequest, 0)
	for _, req := range m.dataRights {
		if req.DataRegion == dataRegion && req.UserID == userID {
			out = append(out, req)
		}
	}
	return out, nil
}
