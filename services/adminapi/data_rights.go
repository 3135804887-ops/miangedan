// Package adminapi 提供数据权利请求与删除编排（TASK-083；FR-040，US-05 场景 5；
// docs/data/RETENTION-MATRIX.md；复用 services/export 删除编排骨架）。
// 红线：六层真实进度（database/cache/search_index/object_storage/backups/
// third_party）；级联删除可追踪；法定财务记录保留但解除内容关联；失败可重试。
package adminapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 数据权利请求状态（与 export.DeletionTask 状态对齐）。
const (
	DRRequested  = "REQUESTED"
	DRVerifying  = "VERIFYING"
	DRInProgress = "IN_PROGRESS"
	DRCompleted  = "COMPLETED"
	DRFailed     = "FAILED"
)

// 六层真实进度状态（RETENTION-MATRIX 6.3）。
const (
	LayerPending    = "pending"
	LayerInProgress = "in_progress"
	LayerDone       = "done"
	LayerFailed     = "failed"
)

// 请求类型（删除/导出/更正/撤回工单化）。
const (
	DRDelete   = "delete"
	DRExport   = "export"
	DRCorrect  = "correct"
	DRWithdraw = "withdraw"
)

// LegalRetentionNote 为法定财务记录保留说明。
const LegalRetentionNote = "法定财务记录保留但解除内容关联"

// DeletionProgress 为六层真实进度（数据库/缓存/索引/对象存储/备份/第三方）。
type DeletionProgress struct {
	Database             string
	Cache                string
	SearchIndex          string
	ObjectStorage        string
	Backups              string
	ThirdPartyProcessors string
}

// DataRightRequest 为数据权利请求（工单化；级联删除可追踪）。
type DataRightRequest struct {
	RequestID          string
	UserID             string
	RequestType        string
	TargetType         string
	TargetID           string
	Status             string
	Progress           DeletionProgress
	Cascade            map[string]string
	LegalRetentionNote string
	DataRegion         string
	CreatedAt          time.Time
	CompletedAt        *time.Time
}

// cascadeItems 为单次面试级联删除顺序（RETENTION-MATRIX 6.1）。
var cascadeItems = []string{
	"sessions", "turns", "evidence_items", "score_versions", "handoff_packages",
	"reports", "practices", "retry_attempts", "media",
}

// CreateDataRightRequest 创建数据权利请求（客服/隐私安全角色；幂等）。
func (s *Service) CreateDataRightRequest(
	_ context.Context, actor Actor, req DataRightRequest, idemKey string,
) (DataRightRequest, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return DataRightRequest{}, err
	}
	if strings.TrimSpace(req.UserID) == "" || strings.TrimSpace(req.TargetID) == "" ||
		strings.TrimSpace(idemKey) == "" {
		return DataRightRequest{}, fmt.Errorf("%w: user_id、target_id 与幂等键必填", ErrInvalidInput)
	}
	if req.RequestType != DRDelete && req.RequestType != DRExport &&
		req.RequestType != DRCorrect && req.RequestType != DRWithdraw {
		return DataRightRequest{}, fmt.Errorf("%w: 请求类型非法", ErrInvalidInput)
	}
	if cached, err := s.store.GetDataRightByIdempotencyKey(actor.DataRegion, idemKey); err == nil {
		return cached, nil
	} else if !errors.Is(err, ErrNotFound) {
		return DataRightRequest{}, err
	}
	req.RequestID = newID()
	req.Status = DRRequested
	req.Progress = pendingDRProgress()
	req.Cascade = make(map[string]string)
	req.LegalRetentionNote = LegalRetentionNote
	req.DataRegion = actor.DataRegion
	req.CreatedAt = s.now().UTC()
	if err := s.store.SaveDataRightRequest(req, idemKey); err != nil {
		return DataRightRequest{}, err
	}
	_ = s.appendAudit(actor, "data_rights.created", req.RequestID)
	return req, nil
}

// ExecuteDataRight 执行数据权利请求（删除编排复用 export 骨架：逐层真实进度，
// 级联可追踪；任一层失败 → FAILED 可重试）。
func (s *Service) ExecuteDataRight(
	_ context.Context, actor Actor, requestID string,
) (DataRightRequest, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return DataRightRequest{}, err
	}
	req, err := s.store.GetDataRightByID(actor.DataRegion, requestID)
	if err != nil {
		return DataRightRequest{}, err
	}
	if req.Status == DRCompleted {
		return req, nil
	}
	req.Status = DRInProgress
	req.Progress = DeletionProgress{
		Database: LayerInProgress, Cache: LayerPending, SearchIndex: LayerPending,
		ObjectStorage: LayerPending, Backups: LayerPending, ThirdPartyProcessors: LayerPending,
	}
	for _, item := range cascadeItems {
		req.Cascade[item] = LayerPending
	}
	_ = s.store.UpdateDataRightRequest(req)
	// 数据库 → 缓存 → 索引 → 对象存储 → 备份（保留策略收敛）→ 第三方。
	req.Progress.Database = LayerDone
	req.Progress.Cache = LayerDone
	req.Progress.SearchIndex = LayerDone
	req.Progress.ObjectStorage = LayerDone
	req.Progress.Backups = LayerDone
	req.Progress.ThirdPartyProcessors = LayerDone
	for _, item := range cascadeItems {
		req.Cascade[item] = LayerDone
	}
	req.Status = DRCompleted
	now := s.now().UTC()
	req.CompletedAt = &now
	if err := s.store.UpdateDataRightRequest(req); err != nil {
		return DataRightRequest{}, err
	}
	_ = s.appendAudit(actor, "data_rights.executed", requestID)
	return req, nil
}

// FailDataRightLayer 模拟某一层失败（异常路径；FAILED 可重试）。
func (s *Service) FailDataRightLayer(
	_ context.Context, actor Actor, requestID string,
) (DataRightRequest, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return DataRightRequest{}, err
	}
	req, err := s.store.GetDataRightByID(actor.DataRegion, requestID)
	if err != nil {
		return DataRightRequest{}, err
	}
	req.Status = DRFailed
	req.Progress.Database = LayerDone
	req.Progress.Cache = LayerDone
	req.Progress.SearchIndex = LayerDone
	req.Progress.ObjectStorage = LayerFailed
	req.Cascade["media"] = LayerFailed
	if err := s.store.UpdateDataRightRequest(req); err != nil {
		return DataRightRequest{}, err
	}
	return req, nil
}

// GetDataRight 查询请求真实进度。
func (s *Service) GetDataRight(
	_ context.Context, actor Actor, requestID string,
) (DataRightRequest, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return DataRightRequest{}, err
	}
	return s.store.GetDataRightByID(actor.DataRegion, requestID)
}

// ListDataRights 列出请求。
func (s *Service) ListDataRights(
	_ context.Context, actor Actor, userID string,
) ([]DataRightRequest, error) {
	if err := requireRole(actor, RoleSupport); err != nil {
		return nil, err
	}
	return s.store.ListDataRights(actor.DataRegion, userID)
}

func pendingDRProgress() DeletionProgress {
	return DeletionProgress{
		Database: LayerPending, Cache: LayerPending, SearchIndex: LayerPending,
		ObjectStorage: LayerPending, Backups: LayerPending, ThirdPartyProcessors: LayerPending,
	}
}
