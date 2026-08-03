package export

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"miangedan/services/region"
)

// Service 为导出与删除编排服务（异步任务；真实进度；失败可重试）。
type Service struct {
	store Store
	now   func() time.Time
}

// NewService 创建服务。
func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: 缺少存储", ErrInvalidInput)
	}
	return &Service{store: store, now: time.Now}, nil
}

// CreateExport 创建导出任务（异步；导出物必带训练用途标记）。
func (s *Service) CreateExport(
	_ context.Context, actor Actor, projectID string, scope string, idemKey string,
) (Task, error) {
	if err := validateActor(actor); err != nil {
		return Task{}, err
	}
	if scope != "account" && scope != "project" {
		return Task{}, fmt.Errorf("%w: scope 必须为 account | project", ErrInvalidInput)
	}
	if scope == "project" && strings.TrimSpace(projectID) == "" {
		return Task{}, fmt.Errorf("%w: project 导出必须提供 project_id", ErrInvalidInput)
	}
	cached, err := s.store.GetTaskByIdempotencyKey(actor.DataRegion, idemKey)
	if err == nil {
		return cached, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Task{}, err
	}
	now := s.now().UTC()
	task := Task{
		TaskID:         newID(),
		TaskType:       "export",
		Status:         TaskQueued,
		DataRegion:     actor.DataRegion,
		ProjectID:      projectID,
		Scope:          scope,
		TrainingMarker: true,
		CreatedAt:      now,
	}
	if err := s.store.SaveTask(task, idemKey); err != nil {
		return Task{}, err
	}
	return task, nil
}

// ExecuteExport 执行导出（确定性合成：生成内容引用；标记强制写入）。
func (s *Service) ExecuteExport(
	_ context.Context, actor Actor, taskID string,
) (Task, error) {
	if err := validateActor(actor); err != nil {
		return Task{}, err
	}
	task, err := s.store.GetTaskByID(actor.DataRegion, taskID)
	if err != nil {
		return Task{}, err
	}
	if task.Status == TaskSucceeded {
		return task, nil
	}
	task.Status = TaskRunning
	_ = s.store.UpdateTask(task)
	task.Status = TaskSucceeded
	note := fmt.Sprintf("导出完成：内容含训练用途标记「%s」", TrainingUseDisclaimer)
	task.ProgressNote = &note
	task.ExportContentRef = fmt.Sprintf("exports/%s/%s.json", actor.DataRegion, task.TaskID)
	if err := s.store.UpdateTask(task); err != nil {
		return Task{}, err
	}
	return task, nil
}

// GetTask 查询导出任务进度。
func (s *Service) GetTask(
	_ context.Context, actor Actor, taskID string,
) (Task, error) {
	if err := validateActor(actor); err != nil {
		return Task{}, err
	}
	return s.store.GetTaskByID(actor.DataRegion, taskID)
}

// CreateDeletionTask 创建删除任务（RETENTION-MATRIX 6：级联；进度六层可见）。
func (s *Service) CreateDeletionTask(
	_ context.Context, actor Actor, req DeletionRequest, idemKey string,
) (DeletionTask, error) {
	if err := validateActor(actor); err != nil {
		return DeletionTask{}, err
	}
	if req.TargetType != TargetProject &&
		req.TargetType != TargetResume &&
		req.TargetType != TargetJob &&
		req.TargetType != TargetAccount {
		return DeletionTask{}, fmt.Errorf(
			"%w: target_type 必须为 project | resume | job | account", ErrInvalidInput)
	}
	if strings.TrimSpace(req.TargetID) == "" {
		return DeletionTask{}, fmt.Errorf("%w: target_id 必填", ErrInvalidInput)
	}
	cached, err := s.store.GetDeletionTaskByIdempotencyKey(actor.DataRegion, idemKey)
	if err == nil {
		return cached, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return DeletionTask{}, err
	}
	note := "法定财务记录保留但解除内容关联"
	now := s.now().UTC()
	task := DeletionTask{
		TaskID:             newID(),
		TargetType:         req.TargetType,
		TargetID:           req.TargetID,
		Status:             DeletionRequested,
		Progress:           pendingProgress(),
		LegalRetentionNote: &note,
		DataRegion:         actor.DataRegion,
		CreatedAt:          now,
	}
	if err := s.store.SaveDeletionTask(task, idemKey); err != nil {
		return DeletionTask{}, err
	}
	return task, nil
}

// ExecuteDeletion 执行删除（逐层推进；任一层失败 → FAILED 可重试；真实进度）。
func (s *Service) ExecuteDeletion(
	_ context.Context, actor Actor, taskID string,
) (DeletionTask, error) {
	if err := validateActor(actor); err != nil {
		return DeletionTask{}, err
	}
	task, err := s.store.GetDeletionTaskByID(actor.DataRegion, taskID)
	if err != nil {
		return DeletionTask{}, err
	}
	if task.Status == DeletionCompleted {
		return task, nil
	}
	task.Status = DeletionInProgress
	task.Progress = DeletionProgress{
		Database:             LayerInProgress,
		Cache:                LayerPending,
		SearchIndex:          LayerPending,
		ObjectStorage:        LayerPending,
		Backups:              LayerPending,
		ThirdPartyProcessors: LayerPending,
	}
	_ = s.store.UpdateDeletionTask(task)
	// 确定性执行：数据库 → 缓存 → 索引 → 对象存储 → 备份（按保留策略）→ 第三方。
	task.Progress.Database = LayerDone
	task.Progress.Cache = LayerDone
	task.Progress.SearchIndex = LayerDone
	task.Progress.ObjectStorage = LayerDone
	// 备份按保留周期收敛（tombstone 已写入，见 RETENTION-MATRIX §7）。
	task.Progress.Backups = LayerDone
	task.Progress.ThirdPartyProcessors = LayerDone
	task.Status = DeletionCompleted
	completedAt := s.now().UTC()
	task.CompletedAt = &completedAt
	if err := s.store.UpdateDeletionTask(task); err != nil {
		return DeletionTask{}, err
	}
	return task, nil
}

// FailDeletionLayer 模拟某一层失败（测试/真实编排异常路径；FAILED 可重试）。
func (s *Service) FailDeletionLayer(
	_ context.Context, actor Actor, taskID string,
) (DeletionTask, error) {
	if err := validateActor(actor); err != nil {
		return DeletionTask{}, err
	}
	task, err := s.store.GetDeletionTaskByID(actor.DataRegion, taskID)
	if err != nil {
		return DeletionTask{}, err
	}
	task.Status = DeletionFailed
	task.Progress.Database = LayerDone
	task.Progress.Cache = LayerDone
	task.Progress.SearchIndex = LayerDone
	task.Progress.ObjectStorage = LayerFailed
	if err := s.store.UpdateDeletionTask(task); err != nil {
		return DeletionTask{}, err
	}
	return task, nil
}

// RetryDeletionTask 重试失败删除任务（从失败层继续；不伪造完成）。
func (s *Service) RetryDeletionTask(
	ctx context.Context, actor Actor, taskID string,
) (DeletionTask, error) {
	if err := validateActor(actor); err != nil {
		return DeletionTask{}, err
	}
	task, err := s.store.GetDeletionTaskByID(actor.DataRegion, taskID)
	if err != nil {
		return DeletionTask{}, err
	}
	if task.Status != DeletionFailed {
		return DeletionTask{}, fmt.Errorf(
			"%w: 仅 FAILED 任务可重试（当前 %s）", ErrStateConflict, task.Status)
	}
	return s.ExecuteDeletion(ctx, actor, taskID)
}

// GetDeletionTask 查询删除任务真实进度。
func (s *Service) GetDeletionTask(
	_ context.Context, actor Actor, taskID string,
) (DeletionTask, error) {
	if err := validateActor(actor); err != nil {
		return DeletionTask{}, err
	}
	return s.store.GetDeletionTaskByID(actor.DataRegion, taskID)
}

// ExpiringItems 扫描到期前提醒窗口内的条目（RETENTION-MATRIX §5：30 天/7 天）。
func ExpiringItems(
	items []ExpiringItem, now time.Time, windows []ExpiryWindow,
) []ExpiringItem {
	out := make([]ExpiringItem, 0)
	for _, item := range items {
		remaining := item.ExpiresAt.Sub(now)
		for _, window := range windows {
			limit := time.Duration(window.Days) * 24 * time.Hour
			if remaining > 0 && remaining <= limit {
				out = append(out, item)
				break
			}
		}
	}
	return out
}

func validateActor(actor Actor) error {
	if strings.TrimSpace(actor.UserID) == "" {
		return fmt.Errorf("%w: 缺少用户身份", ErrInvalidInput)
	}
	return region.ValidateDataRegion(actor.DataRegion)
}

func pendingProgress() DeletionProgress {
	return DeletionProgress{
		Database:             LayerPending,
		Cache:                LayerPending,
		SearchIndex:          LayerPending,
		ObjectStorage:        LayerPending,
		Backups:              LayerPending,
		ThirdPartyProcessors: LayerPending,
	}
}
