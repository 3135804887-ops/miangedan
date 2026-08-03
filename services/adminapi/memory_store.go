// Package adminapi 提供后台内存存储（TASK-080；开发/测试；生产 PostgreSQL）。
package adminapi

import "sync"

// MemoryStore 为内存版后台存储。
type MemoryStore struct {
	mu              sync.RWMutex
	providers       map[string]ProviderInfo
	rooms           map[string][]RoomSnapshot
	regionStatus    map[string]RegionOpsStatus
	audits          map[string][]AuditEntry
	versions        map[string]ArtifactVersion
	versionByKey    map[string]ArtifactVersion
	pins            map[string]VersionPin
	activeSess      map[string]bool
	breakGlass      map[string]BreakGlass
	glassReviews    map[string][]BreakGlassReview
	dataRights      map[string]DataRightRequest
	drIDem          map[string]DataRightRequest
	mfaDevices      map[string]MFADevice
	mfaChallenges   map[string]MFAChallenge
	mfaVerifs       map[string][]MFAVerification
	tickets         map[string]Ticket
	ticketIDem      map[string]Ticket
	transcriptAuths map[string]TranscriptAuthorization
	trIDem          map[string]TranscriptAuthorization
	mediaRequests   map[string]MediaAccessRequest
	mediaIDem       map[string]MediaAccessRequest
}

// NewMemoryStore 创建空内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		providers:       make(map[string]ProviderInfo),
		rooms:           make(map[string][]RoomSnapshot),
		regionStatus:    make(map[string]RegionOpsStatus),
		audits:          make(map[string][]AuditEntry),
		versions:        make(map[string]ArtifactVersion),
		versionByKey:    make(map[string]ArtifactVersion),
		pins:            make(map[string]VersionPin),
		activeSess:      make(map[string]bool),
		breakGlass:      make(map[string]BreakGlass),
		glassReviews:    make(map[string][]BreakGlassReview),
		dataRights:      make(map[string]DataRightRequest),
		drIDem:          make(map[string]DataRightRequest),
		mfaDevices:      make(map[string]MFADevice),
		mfaChallenges:   make(map[string]MFAChallenge),
		mfaVerifs:       make(map[string][]MFAVerification),
		tickets:         make(map[string]Ticket),
		ticketIDem:      make(map[string]Ticket),
		transcriptAuths: make(map[string]TranscriptAuthorization),
		trIDem:          make(map[string]TranscriptAuthorization),
		mediaRequests:   make(map[string]MediaAccessRequest),
		mediaIDem:       make(map[string]MediaAccessRequest),
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

// SaveMFADevice 登记 MFA 设备。
func (m *MemoryStore) SaveMFADevice(d MFADevice) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mfaDevices[d.DataRegion+"|"+d.DeviceID] = d
	return nil
}

// GetMFADevice 查询 MFA 设备。
func (m *MemoryStore) GetMFADevice(dataRegion, deviceID string) (MFADevice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.mfaDevices[dataRegion+"|"+deviceID]
	if !ok {
		return MFADevice{}, ErrNotFound
	}
	return d, nil
}

// ListMFADevices 列出员工设备。
func (m *MemoryStore) ListMFADevices(dataRegion, staffID string) ([]MFADevice, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]MFADevice, 0)
	for _, d := range m.mfaDevices {
		if d.DataRegion == dataRegion && d.StaffID == staffID {
			out = append(out, d)
		}
	}
	return out, nil
}

// SaveMFAChallenge 保存挑战。
func (m *MemoryStore) SaveMFAChallenge(c MFAChallenge) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mfaChallenges[c.DataRegion+"|"+c.ChallengeID] = c
	return nil
}

// GetMFAChallenge 查询挑战。
func (m *MemoryStore) GetMFAChallenge(dataRegion, challengeID string) (MFAChallenge, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.mfaChallenges[dataRegion+"|"+challengeID]
	if !ok {
		return MFAChallenge{}, ErrNotFound
	}
	return c, nil
}

// UpdateMFAChallenge 标记挑战已使用。
func (m *MemoryStore) UpdateMFAChallenge(c MFAChallenge) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mfaChallenges[c.DataRegion+"|"+c.ChallengeID] = c
	return nil
}

// SaveMFAVerification 追加验证记录。
func (m *MemoryStore) SaveMFAVerification(v MFAVerification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mfaVerifs[v.DataRegion+"|"+v.StaffID] =
		append(m.mfaVerifs[v.DataRegion+"|"+v.StaffID], v)
	return nil
}

// GetLatestMFAVerification 查询员工最近验证。
func (m *MemoryStore) GetLatestMFAVerification(dataRegion, staffID string) (MFAVerification, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.mfaVerifs[dataRegion+"|"+staffID]
	if len(items) == 0 {
		return MFAVerification{}, ErrNotFound
	}
	return items[len(items)-1], nil
}

// ListAuditsPaged 分页查询审计。
func (m *MemoryStore) ListAuditsPaged(dataRegion string, limit, offset int) ([]AuditEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.audits[dataRegion]
	start := offset
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	out := make([]AuditEntry, end-start)
	copy(out, items[start:end])
	return out, nil
}

// SaveTicket 保存工单。
func (m *MemoryStore) SaveTicket(t Ticket, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tickets[t.DataRegion+"|"+t.TicketID] = t
	if idemKey != "" {
		m.ticketIDem[t.DataRegion+"|"+idemKey] = t
	}
	return nil
}

// GetTicketByID 查询工单。
func (m *MemoryStore) GetTicketByID(dataRegion, ticketID string) (Ticket, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tickets[dataRegion+"|"+ticketID]
	if !ok {
		return Ticket{}, ErrNotFound
	}
	return t, nil
}

// GetTicketByIdempotencyKey 幂等键查询工单。
func (m *MemoryStore) GetTicketByIdempotencyKey(dataRegion, key string) (Ticket, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.ticketIDem[dataRegion+"|"+key]
	if !ok {
		return Ticket{}, ErrNotFound
	}
	return t, nil
}

// UpdateTicket 更新工单状态。
func (m *MemoryStore) UpdateTicket(t Ticket) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tickets[t.DataRegion+"|"+t.TicketID] = t
	return nil
}

// ListTickets 列出区域工单。
func (m *MemoryStore) ListTickets(dataRegion string) ([]Ticket, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Ticket, 0)
	for _, t := range m.tickets {
		if t.DataRegion == dataRegion {
			out = append(out, t)
		}
	}
	return out, nil
}

// SaveTranscriptAuthorization 保存逐字稿授权。
func (m *MemoryStore) SaveTranscriptAuthorization(a TranscriptAuthorization, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transcriptAuths[a.DataRegion+"|"+a.AuthID] = a
	if idemKey != "" {
		m.trIDem[a.DataRegion+"|"+idemKey] = a
	}
	return nil
}

// GetTranscriptAuthByIdempotencyKey 幂等键查询逐字稿授权。
func (m *MemoryStore) GetTranscriptAuthByIdempotencyKey(dataRegion, key string) (TranscriptAuthorization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.trIDem[dataRegion+"|"+key]
	if !ok {
		return TranscriptAuthorization{}, ErrNotFound
	}
	return a, nil
}

// ListTranscriptAuths 列出会话逐字稿授权。
func (m *MemoryStore) ListTranscriptAuths(dataRegion, ticketID, sessionID string) ([]TranscriptAuthorization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]TranscriptAuthorization, 0)
	for _, a := range m.transcriptAuths {
		if a.DataRegion == dataRegion && a.TicketID == ticketID && a.SessionID == sessionID {
			out = append(out, a)
		}
	}
	return out, nil
}

// SaveMediaRequest 保存媒体访问申请。
func (m *MemoryStore) SaveMediaRequest(req MediaAccessRequest, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mediaRequests[req.DataRegion+"|"+req.AccessRequestID] = req
	if idemKey != "" {
		m.mediaIDem[req.DataRegion+"|"+idemKey] = req
	}
	return nil
}

// GetMediaRequestByID 查询媒体申请。
func (m *MemoryStore) GetMediaRequestByID(dataRegion, requestID string) (MediaAccessRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	req, ok := m.mediaRequests[dataRegion+"|"+requestID]
	if !ok {
		return MediaAccessRequest{}, ErrNotFound
	}
	return req, nil
}

// GetMediaRequestByIdempotencyKey 幂等键查询媒体申请。
func (m *MemoryStore) GetMediaRequestByIdempotencyKey(dataRegion, key string) (MediaAccessRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	req, ok := m.mediaIDem[dataRegion+"|"+key]
	if !ok {
		return MediaAccessRequest{}, ErrNotFound
	}
	return req, nil
}

// UpdateMediaRequest 更新媒体申请状态。
func (m *MemoryStore) UpdateMediaRequest(req MediaAccessRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mediaRequests[req.DataRegion+"|"+req.AccessRequestID] = req
	return nil
}

// AppendMediaApproval 原子追加媒体审批人（同一审批人去重）。
func (m *MemoryStore) AppendMediaApproval(dataRegion, requestID, approverID string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	req, ok := m.mediaRequests[dataRegion+"|"+requestID]
	if !ok {
		return nil, ErrNotFound
	}
	for _, existing := range req.ApproverPair {
		if existing == approverID {
			return append([]string(nil), req.ApproverPair...), nil
		}
	}
	req.ApproverPair = append(req.ApproverPair, approverID)
	m.mediaRequests[dataRegion+"|"+requestID] = req
	return append([]string(nil), req.ApproverPair...), nil
}
