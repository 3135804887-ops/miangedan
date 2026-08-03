// Package adminapi 提供模型/提示词/量表/工作流版本治理：离线→影子→灰度→放量、
// 冻结与回滚、活跃正式面试固定开始版本（TASK-081；FR-038，US-08 场景 2/6；
// docs/ai/PROVIDER-ADAPTERS.md；TASK-031 pinned 机制）。
package adminapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 版本阶段（US-08 规则 4：离线 → 影子 → 灰度 → 放量）。
const (
	StageOffline = "offline"
	StageShadow  = "shadow"
	StageCanary  = "canary"
	StageFull    = "full"
)

// 版本资产类型（模型/提示词/量表/工作流）。
const (
	AssetModel    = "model"
	AssetPrompt   = "prompt"
	AssetRubric   = "rubric"
	AssetWorkflow = "workflow"
)

// ArtifactVersion 为版本注册表条目（离线-影子-灰度-放量；冻结与回滚）。
type ArtifactVersion struct {
	VersionID    string
	AssetType    string
	AssetKey     string
	Version      string
	Stage        string
	Compatible   bool
	SafetyTested bool
	MetricsOK    bool
	Deprecated   bool
	Note         string
	DataRegion   string
	CreatedAt    time.Time
}

// VersionPin 为项目版本固定（活跃正式面试固定开始版本；不可中途改变）。
type VersionPin struct {
	ProjectID  string
	AssetType  string
	AssetKey   string
	VersionID  string
	DataRegion string
	PinnedAt   time.Time
}

// RegisterVersion 注册新版本（初始 offline；兼容性与安全测试标记来自评测门槛）。
func (s *Service) RegisterVersion(
	_ context.Context, actor Actor, v ArtifactVersion,
) (ArtifactVersion, error) {
	if err := requireRole(actor, RoleAIConfig); err != nil {
		return ArtifactVersion{}, err
	}
	if !validAsset(v.AssetType) || strings.TrimSpace(v.AssetKey) == "" ||
		strings.TrimSpace(v.Version) == "" {
		return ArtifactVersion{}, fmt.Errorf("%w: 资产类型/键/版本必填", ErrInvalidInput)
	}
	v.VersionID = newID()
	v.Stage = StageOffline
	v.DataRegion = actor.DataRegion
	v.CreatedAt = s.now().UTC()
	if err := s.store.SaveVersion(v); err != nil {
		return ArtifactVersion{}, err
	}
	_ = s.appendAudit(actor, "version.registered", v.VersionID)
	return v, nil
}

// PromoteVersion 升级阶段：offline → shadow → canary → full；
// 灰度门槛：结构兼容 + 安全测试；放量门槛：影子/灰度指标通过（MetricsOK）。
func (s *Service) PromoteVersion(
	_ context.Context, actor Actor, versionID, targetStage string,
) (ArtifactVersion, error) {
	if err := requireRole(actor, RoleAIConfig); err != nil {
		return ArtifactVersion{}, err
	}
	v, err := s.store.GetVersion(actor.DataRegion, versionID)
	if err != nil {
		return ArtifactVersion{}, err
	}
	if !validTransition(v.Stage, targetStage) {
		return ArtifactVersion{}, fmt.Errorf("%w: %s 不可直接到 %s", ErrStateConflict, v.Stage, targetStage)
	}
	if targetStage == StageCanary && (!v.Compatible || !v.SafetyTested) {
		return ArtifactVersion{}, fmt.Errorf("%w: 未通过结构兼容/安全测试不可灰度", ErrStateConflict)
	}
	if targetStage == StageFull && !v.MetricsOK {
		return ArtifactVersion{}, fmt.Errorf("%w: 未通过影子/灰度指标不可放量", ErrStateConflict)
	}
	v.Stage = targetStage
	if err := s.store.UpdateVersion(v); err != nil {
		return ArtifactVersion{}, err
	}
	_ = s.appendAudit(actor, "version.promoted", v.VersionID+"->"+targetStage)
	return v, nil
}

// FreezeVersion 固定项目开始版本（活跃正式面试固定开始版本；不可中途改变）。
func (s *Service) FreezeVersion(
	_ context.Context, actor Actor, projectID, versionID string,
) (VersionPin, error) {
	if err := requireRole(actor, RoleAIConfig); err != nil {
		return VersionPin{}, err
	}
	if existing, err := s.store.GetPin(actor.DataRegion, projectID); err == nil {
		return VersionPin{}, fmt.Errorf("%w: 项目已固定版本 %s（不可中途改变）",
			ErrStateConflict, existing.VersionID)
	} else if !errors.Is(err, ErrNotFound) {
		return VersionPin{}, err
	}
	v, err := s.store.GetVersion(actor.DataRegion, versionID)
	if err != nil {
		return VersionPin{}, err
	}
	pin := VersionPin{
		ProjectID:  projectID,
		AssetType:  v.AssetType,
		AssetKey:   v.AssetKey,
		VersionID:  versionID,
		DataRegion: actor.DataRegion,
		PinnedAt:   s.now().UTC(),
	}
	if err := s.store.SavePin(pin); err != nil {
		return VersionPin{}, err
	}
	_ = s.appendAudit(actor, "version.frozen", projectID+"->"+versionID)
	return pin, nil
}

// RollbackVersion 回滚：新会话回到稳定版本；进行中的正式会话不被中途改变。
func (s *Service) RollbackVersion(
	_ context.Context, actor Actor, projectID, stableVersionID string,
) (VersionPin, error) {
	if err := requireRole(actor, RoleAIConfig); err != nil {
		return VersionPin{}, err
	}
	pin, err := s.store.GetPin(actor.DataRegion, projectID)
	if err != nil {
		return VersionPin{}, err
	}
	if s.store.HasActiveSession(actor.DataRegion, projectID) {
		return VersionPin{}, fmt.Errorf("%w: 进行中的正式会话不可中途改版本", ErrStateConflict)
	}
	pin.VersionID = stableVersionID
	if err := s.store.UpdatePin(pin); err != nil {
		return VersionPin{}, err
	}
	_ = s.appendAudit(actor, "version.rolled_back", projectID+"->"+stableVersionID)
	return pin, nil
}

// DeprecateRubric 停用存在系统性偏差的量表版本（产品/面试专业/安全公平三方审批；
// 不批量改写历史分数；受影响项目标记与免费重试由上层异步任务处理）。
func (s *Service) DeprecateRubric(
	_ context.Context, actor Actor, rubricID, reason string, approvals map[string]string,
) (ArtifactVersion, error) {
	if err := requireRole(actor, RoleScoringGovernance); err != nil {
		return ArtifactVersion{}, err
	}
	required := []string{"product", "interview_professional", "safety_fairness"}
	for _, role := range required {
		if strings.TrimSpace(approvals[role]) == "" {
			return ArtifactVersion{}, fmt.Errorf("%w: 缺少 %s 审批", ErrInvalidInput, role)
		}
	}
	v, err := s.store.GetVersionByKey(actor.DataRegion, AssetRubric, rubricID)
	if err != nil {
		return ArtifactVersion{}, err
	}
	v.Deprecated = true
	v.Note = "rubric_deprecated: " + reason
	if err := s.store.UpdateVersion(v); err != nil {
		return ArtifactVersion{}, err
	}
	_ = s.appendAudit(actor, "rubric.deprecated", rubricID)
	return v, nil
}

func validAsset(assetType string) bool {
	switch assetType {
	case AssetModel, AssetPrompt, AssetRubric, AssetWorkflow:
		return true
	}
	return false
}

func validTransition(from, to string) bool {
	order := map[string]int{StageOffline: 0, StageShadow: 1, StageCanary: 2, StageFull: 3}
	f, ok1 := order[from]
	t, ok2 := order[to]
	return ok1 && ok2 && t == f+1
}

func requireRole(actor Actor, role string) error {
	if err := validateActor(actor); err != nil {
		return err
	}
	if actor.Role != role && actor.Role != RoleSuperAdmin {
		return fmt.Errorf("%w: 需要角色 %s", ErrForbidden, role)
	}
	return nil
}
