// Package adminapi 提供后台存储接口（TASK-080；审计仅 SELECT/INSERT）。
package adminapi

// Store 为运营后台存储（生产 PostgreSQL；审计无更新/删除路径）。
type Store interface {
	SaveProvider(ProviderInfo) error
	GetProvider(dataRegion, providerID string) (ProviderInfo, error)
	ListProviders(dataRegion string) ([]ProviderInfo, error)
	UpdateProvider(ProviderInfo) error
	SaveRoomSnapshot(RoomSnapshot) error
	ListRoomSnapshots(dataRegion string) ([]RoomSnapshot, error)
	SaveRegionStatus(RegionOpsStatus) error
	GetRegionStatus(dataRegion string) (RegionOpsStatus, error)
	AppendAudit(AuditEntry) error
	ListAudits(dataRegion string) ([]AuditEntry, error)
	// TASK-081 版本治理（注册/阶段/固定/回滚）。
	SaveVersion(ArtifactVersion) error
	GetVersion(dataRegion, versionID string) (ArtifactVersion, error)
	GetVersionByKey(dataRegion, assetType, assetKey string) (ArtifactVersion, error)
	UpdateVersion(ArtifactVersion) error
	ListVersions(dataRegion, assetType string) ([]ArtifactVersion, error)
	SavePin(VersionPin) error
	GetPin(dataRegion, projectID string) (VersionPin, error)
	UpdatePin(VersionPin) error
	HasActiveSession(dataRegion, projectID string) bool
	// TASK-082 破窗访问（仅 SELECT/INSERT；状态由追加式评审事件推导）。
	SaveBreakGlass(BreakGlass) error
	GetBreakGlass(dataRegion, glassID string) (BreakGlass, error)
	ListBreakGlassByTarget(dataRegion, targetUserID string) ([]BreakGlass, error)
	AppendBreakGlassReview(BreakGlassReview) error
	ListBreakGlassReviews(glassID string) ([]BreakGlassReview, error)
	// TASK-083 数据权利请求（删除编排；级联可追踪；失败可重试）。
	SaveDataRightRequest(DataRightRequest, string) error
	GetDataRightByID(dataRegion, requestID string) (DataRightRequest, error)
	GetDataRightByIdempotencyKey(dataRegion, key string) (DataRightRequest, error)
	UpdateDataRightRequest(DataRightRequest) error
	ListDataRights(dataRegion, userID string) ([]DataRightRequest, error)
}
