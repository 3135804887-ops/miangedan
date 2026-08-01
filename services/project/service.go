package project

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"miangedan/services/region"
)

// Actor 为业务令牌携带的身份（由 identity 服务签发）。
type Actor struct {
	UserID     string
	DataRegion string
}

func (a Actor) validate() error {
	if strings.TrimSpace(a.UserID) == "" {
		return fmt.Errorf("%w: 缺少用户身份", ErrInvalidInput)
	}
	return region.ValidateDataRegion(a.DataRegion)
}

// CreateInput 为创建项目入参（openapi createProject）。
type CreateInput struct {
	InterviewLanguage     string
	DegradedMode          DegradedMode
	ResumeRef             *MaterialRef
	JobRef                *MaterialRef
	DegradedModeConsentID string
	AssignmentID          string
	Name                  string
}

// Service 为项目/计划/轮次应用服务（TASK-016，FR-009~FR-011）。
type Service struct {
	store Store
	idem  IdempotencyStore
	flow  *FlowConfig
	now   func() time.Time
}

// NewService 创建项目服务；流程配置与存储必填。
func NewService(store Store, idem IdempotencyStore, flow *FlowConfig) (*Service, error) {
	if store == nil || idem == nil || flow == nil {
		return nil, fmt.Errorf("%w: 缺少存储/幂等存储/流程配置", ErrInvalidInput)
	}
	return &Service{store: store, idem: idem, flow: flow, now: time.Now}, nil
}

// idempotent 按幂等键执行写操作：仅显式幂等键启用缓存，重复键返回首次结果（NFR-006）；
// 无幂等键的请求不缓存（避免同路径后续请求被首次结果污染）。
func idempotent[T any](s *Service, prefix, idemKey string, run func() (T, error)) (T, error) {
	var zero T
	fullKey := prefix + idemKey
	if idemKey != "" {
		var cached T
		found, err := s.idem.Recall(fullKey, &cached)
		if err != nil {
			return zero, err
		}
		if found {
			return cached, nil
		}
	}
	result, err := run()
	if err != nil {
		return zero, err
	}
	if idemKey != "" {
		if err := s.idem.Remember(fullKey, result); err != nil {
			return zero, err
		}
	}
	return result, nil
}

// CreateProject 创建 DRAFT 项目；非 full 降级模式必须携带明确同意记录。
func (s *Service) CreateProject(_ context.Context, actor Actor, in CreateInput, idemKey string) (Project, error) {
	if err := actor.validate(); err != nil {
		return Project{}, err
	}
	if !contains(AllLanguages, in.InterviewLanguage) {
		return Project{}, fmt.Errorf("%w: 面试语言必须为 zh-CN | en-US", ErrInvalidInput)
	}
	switch in.DegradedMode {
	case ModeFull, ModeJDOnly, ModeResumeOnly, ModeNeither:
	default:
		return Project{}, fmt.Errorf("%w: 降级模式非法", ErrInvalidInput)
	}
	if in.DegradedMode != ModeFull && strings.TrimSpace(in.DegradedModeConsentID) == "" {
		return Project{}, fmt.Errorf("%w: 非 full 降级模式必须携带明确同意记录", ErrInvalidInput)
	}
	if err := validateMaterialRef(in.ResumeRef); err != nil {
		return Project{}, err
	}
	if err := validateMaterialRef(in.JobRef); err != nil {
		return Project{}, err
	}
	if len(in.Name) > 120 {
		return Project{}, fmt.Errorf("%w: 项目名最长 120 字符", ErrInvalidInput)
	}
	return idempotent(s, "create|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Project, error) {
		now := s.now()
		proj := Project{
			ProjectID:             newID(),
			UserID:                actor.UserID,
			DataRegion:            actor.DataRegion,
			Name:                  in.Name,
			InterviewLanguage:     in.InterviewLanguage,
			DegradedMode:          in.DegradedMode,
			DegradedModeConsentID: in.DegradedModeConsentID,
			ResumeRef:             cloneRef(in.ResumeRef),
			JobRef:                cloneRef(in.JobRef),
			Status:                StatusDraft,
			CurrentRoundSequence:  0,
			AssignmentID:          in.AssignmentID,
			CreatedAt:             now,
		}
		if err := s.store.CreateProject(proj); err != nil {
			return Project{}, err
		}
		return proj, nil
	})
}

func validateMaterialRef(ref *MaterialRef) error {
	if ref == nil {
		return nil
	}
	if strings.TrimSpace(ref.ID) == "" || ref.Version < 1 {
		return fmt.Errorf("%w: 材料引用必须同时含 id 与 version≥1", ErrInvalidInput)
	}
	return nil
}

func cloneRef(ref *MaterialRef) *MaterialRef {
	if ref == nil {
		return nil
	}
	c := *ref
	return &c
}

// GetProject 获取项目详情。
func (s *Service) GetProject(_ context.Context, actor Actor, projectID string) (Project, error) {
	if err := actor.validate(); err != nil {
		return Project{}, err
	}
	if strings.TrimSpace(projectID) == "" {
		return Project{}, fmt.Errorf("%w: 项目 ID 为空", ErrInvalidInput)
	}
	return s.store.GetProject(actor.UserID, actor.DataRegion, projectID)
}

// ListProjects 按筛选列出项目；company/job_title 通过材料库元数据解析（TASK-018，FR-029）。
func (s *Service) ListProjects(_ context.Context, actor Actor, f ListFilter) ([]Project, error) {
	if err := actor.validate(); err != nil {
		return nil, err
	}
	items, err := s.store.ListProjects(actor.UserID, actor.DataRegion, f)
	if err != nil {
		return nil, err
	}
	if f.Company == "" && f.JobTitle == "" {
		return items, nil
	}
	filtered := make([]Project, 0, len(items))
	for _, p := range items {
		if matchesMaterialFilter(s, actor, p, f) {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
}

func matchesMaterialFilter(s *Service, actor Actor, p Project, f ListFilter) bool {
	refs := []struct {
		kind LibraryKind
		ref  *MaterialRef
	}{
		{KindResume, p.ResumeRef},
		{KindJob, p.JobRef},
	}
	for _, item := range refs {
		if item.ref == nil {
			continue
		}
		entry, err := s.store.GetLibraryEntry(actor.UserID, actor.DataRegion, item.kind, item.ref.ID)
		if err != nil {
			continue
		}
		if f.Company != "" && entry.Company != f.Company {
			continue
		}
		if f.JobTitle != "" && entry.JobTitle != f.JobTitle {
			continue
		}
		return true
	}
	return false
}

// RenameProject 重命名项目（名称不在冻结范围内）。
func (s *Service) RenameProject(_ context.Context, actor Actor, projectID, name, idemKey string) (Project, error) {
	if err := actor.validate(); err != nil {
		return Project{}, err
	}
	if len(name) < 1 || len(name) > 120 {
		return Project{}, fmt.Errorf("%w: 项目名长度必须为 1-120", ErrInvalidInput)
	}
	return idempotent(s, "rename|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Project, error) {
		proj, err := s.store.GetProject(actor.UserID, actor.DataRegion, projectID)
		if err != nil {
			return Project{}, err
		}
		proj.Name = name
		if err := s.store.UpdateProject(proj); err != nil {
			return Project{}, err
		}
		return proj, nil
	})
}

// DeleteProject 删除项目；活动项目（面试中/评分中）返回 state_conflict。
func (s *Service) DeleteProject(_ context.Context, actor Actor, projectID, idemKey string) (DeletionTask, error) {
	if err := actor.validate(); err != nil {
		return DeletionTask{}, err
	}
	return idempotent(s, "delete|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (DeletionTask, error) {
		proj, err := s.store.GetProject(actor.UserID, actor.DataRegion, projectID)
		if err != nil {
			return DeletionTask{}, err
		}
		if isActive(proj.Status) {
			return DeletionTask{}, fmt.Errorf("%w: 活动项目需先确认终止", ErrStateConflict)
		}
		return DeletionTask{TaskID: newID(), Status: "queued"}, nil
	})
}

func isActive(status Status) bool {
	switch status {
	case StatusInSession, StatusScoring:
		return true
	default:
		return false
	}
}

// DuplicateProject 复制项目（复用冻结材料引用，生成独立 DRAFT 项目）。
func (s *Service) DuplicateProject(_ context.Context, actor Actor, projectID, interviewLanguage, idemKey string) (Project, error) {
	if err := actor.validate(); err != nil {
		return Project{}, err
	}
	if interviewLanguage != "" && !contains(AllLanguages, interviewLanguage) {
		return Project{}, fmt.Errorf("%w: 面试语言必须为 zh-CN | en-US", ErrInvalidInput)
	}
	return idempotent(s, "duplicate|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Project, error) {
		base, err := s.store.GetProject(actor.UserID, actor.DataRegion, projectID)
		if err != nil {
			return Project{}, err
		}
		lang := interviewLanguage
		if lang == "" {
			lang = base.InterviewLanguage
		}
		copyProj := Project{
			ProjectID:             newID(),
			UserID:                actor.UserID,
			DataRegion:            actor.DataRegion,
			Name:                  base.Name,
			InterviewLanguage:     lang,
			DegradedMode:          base.DegradedMode,
			DegradedModeConsentID: base.DegradedModeConsentID,
			ResumeRef:             cloneRef(base.ResumeRef),
			JobRef:                cloneRef(base.JobRef),
			Status:                StatusDraft,
			CurrentRoundSequence:  0,
			CreatedAt:             s.now(),
		}
		if err := s.store.CreateProject(copyProj); err != nil {
			return Project{}, err
		}
		return copyProj, nil
	})
}

// GetPlan 返回项目最新计划版本。
func (s *Service) GetPlan(_ context.Context, actor Actor, projectID string) (PlanVersion, error) {
	if err := actor.validate(); err != nil {
		return PlanVersion{}, err
	}
	if _, err := s.store.GetProject(actor.UserID, actor.DataRegion, projectID); err != nil {
		return PlanVersion{}, err
	}
	return s.store.LatestPlan(actor.DataRegion, projectID)
}

// EditPlan 编辑计划（开始前）：基于指定版本追加新版本，轮次按流程边界校验；
// 计划已冻结或项目已进入 READY+ 时拒绝（FR-011）。
func (s *Service) EditPlan(_ context.Context, actor Actor, projectID string, baseVersion int, rounds []RoundConfig, idemKey string) (PlanVersion, error) {
	if err := actor.validate(); err != nil {
		return PlanVersion{}, err
	}
	if baseVersion < 1 {
		return PlanVersion{}, fmt.Errorf("%w: base_plan_version 必须 ≥1", ErrInvalidInput)
	}
	if err := s.validateRounds(rounds); err != nil {
		return PlanVersion{}, err
	}
	return idempotent(s, "editplan|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (PlanVersion, error) {
		proj, err := s.store.GetProject(actor.UserID, actor.DataRegion, projectID)
		if err != nil {
			return PlanVersion{}, err
		}
		if planLocked(proj) {
			return PlanVersion{}, fmt.Errorf("%w: 计划已冻结或面试已开始，不可修改（FR-011）", ErrStateConflict)
		}
		base, err := s.store.GetPlan(actor.DataRegion, projectID, baseVersion)
		if err != nil {
			return PlanVersion{}, err
		}
		next := base
		next.PlanVersion = base.PlanVersion + 1
		next.Rounds = normalizeRounds(rounds)
		next.RoundWeights = defaultRoundWeights(rounds)
		next.Frozen = false
		next.CreatedAt = s.now()
		if err := s.store.SavePlan(next); err != nil {
			return PlanVersion{}, err
		}
		return next, nil
	})
}

// ConfirmPlan 确认并冻结计划：全部轮次量表与覆盖方案就绪才可冻结，项目进入 READY。
func (s *Service) ConfirmPlan(_ context.Context, actor Actor, projectID string, planVersion int, accommodations []string, _ string, idemKey string) (Project, error) {
	if err := actor.validate(); err != nil {
		return Project{}, err
	}
	if planVersion < 1 {
		return Project{}, fmt.Errorf("%w: plan_version 必须 ≥1", ErrInvalidInput)
	}
	for _, a := range accommodations {
		if !contains(Accommodations, a) {
			return Project{}, fmt.Errorf("%w: 未知便利设置 %q", ErrInvalidInput, a)
		}
	}
	return idempotent(s, "confirm|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Project, error) {
		proj, err := s.store.GetProject(actor.UserID, actor.DataRegion, projectID)
		if err != nil {
			return Project{}, err
		}
		if planLocked(proj) {
			return Project{}, fmt.Errorf("%w: 计划已冻结或面试已开始", ErrStateConflict)
		}
		plan, err := s.store.GetPlan(actor.DataRegion, projectID, planVersion)
		if err != nil {
			return Project{}, err
		}
		if !planComplete(plan) {
			return Project{}, fmt.Errorf("%w: 存在轮次缺问题覆盖方案或评分量表", ErrPlanIncomplete)
		}
		plan.Frozen = true
		if err := s.store.SavePlan(plan); err != nil {
			return Project{}, err
		}
		proj.Status = StatusReady
		proj.PlanVersion = planVersion
		proj.CurrentRoundSequence = 1
		if err := s.store.UpdateProject(proj); err != nil {
			return Project{}, err
		}
		return proj, nil
	})
}

// SetRoundReadiness 由计划生成/安全检查链路（TASK-033）调用，标记轮次就绪；
// 该字段非用户可编辑，用户界面不暴露。
func (s *Service) SetRoundReadiness(dataRegion, projectID string, planVersion, sequence int, rubricBound, coverageReady bool) error {
	if planVersion < 1 || sequence < 1 {
		return fmt.Errorf("%w: 版本/轮次序号非法", ErrInvalidInput)
	}
	if err := region.ValidateDataRegion(dataRegion); err != nil {
		return err
	}
	plan, err := s.store.LatestPlan(dataRegion, projectID)
	if err != nil {
		return err
	}
	if plan.PlanVersion != planVersion {
		return fmt.Errorf("%w: 就绪标记仅允许作用于最新计划版本", ErrStateConflict)
	}
	for i := range plan.Rounds {
		if plan.Rounds[i].Sequence == sequence {
			plan.Rounds[i].RubricBound = rubricBound
			plan.Rounds[i].QuestionCoveragePlanReady = coverageReady
		}
	}
	return s.store.SavePlan(plan)
}

// SaveLibraryEntry 保存材料库条目（幂等：同一材料 ID+版本覆盖）。
func (s *Service) SaveLibraryEntry(_ context.Context, actor Actor, kind LibraryKind, materialID string, version int, company, jobTitle, idemKey string) (LibraryEntry, error) {
	if err := actor.validate(); err != nil {
		return LibraryEntry{}, err
	}
	if kind != KindResume && kind != KindJob {
		return LibraryEntry{}, fmt.Errorf("%w: 材料库类型必须为 resume | job", ErrInvalidInput)
	}
	if strings.TrimSpace(materialID) == "" || version < 1 {
		return LibraryEntry{}, fmt.Errorf("%w: 材料 ID 与 version≥1 必填", ErrInvalidInput)
	}
	if len(company) > 120 || len(jobTitle) > 120 {
		return LibraryEntry{}, fmt.Errorf("%w: 公司/岗位名最长 120 字符", ErrInvalidInput)
	}
	return idempotent(s, "library|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (LibraryEntry, error) {
		entry := LibraryEntry{
			UserID:     actor.UserID,
			DataRegion: actor.DataRegion,
			Kind:       kind,
			MaterialID: materialID,
			Version:    version,
			Company:    company,
			JobTitle:   jobTitle,
			CreatedAt:  s.now(),
		}
		if err := s.store.SaveLibraryEntry(entry); err != nil {
			return LibraryEntry{}, err
		}
		return entry, nil
	})
}

// ListLibrary 列出用户材料库（FR-029）。
func (s *Service) ListLibrary(_ context.Context, actor Actor, kind LibraryKind) ([]LibraryEntry, error) {
	if err := actor.validate(); err != nil {
		return nil, err
	}
	if kind != KindResume && kind != KindJob {
		return nil, fmt.Errorf("%w: 材料库类型必须为 resume | job", ErrInvalidInput)
	}
	return s.store.ListLibrary(actor.UserID, actor.DataRegion, kind)
}

// DeleteLibraryEntry 从材料库移除条目（不存在视为成功，幂等）。
func (s *Service) DeleteLibraryEntry(_ context.Context, actor Actor, kind LibraryKind, materialID, idemKey string) error {
	if err := actor.validate(); err != nil {
		return err
	}
	if kind != KindResume && kind != KindJob {
		return fmt.Errorf("%w: 材料库类型必须为 resume | job", ErrInvalidInput)
	}
	_, err := idempotent(s, "librarydel|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (struct{}, error) {
		if err := s.store.DeleteLibraryEntry(actor.UserID, actor.DataRegion, kind, materialID); err != nil {
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return err
}

// GetPreferences 读取界面语言与面试语言独立配置（FR-028）。
func (s *Service) GetPreferences(_ context.Context, actor Actor) (Preferences, error) {
	if err := actor.validate(); err != nil {
		return Preferences{}, err
	}
	return s.store.GetPreferences(actor.UserID, actor.DataRegion)
}

// SetPreferences 更新界面语言与面试语言偏好（面试语言仍须按项目由用户确认，FR-028）。
func (s *Service) SetPreferences(_ context.Context, actor Actor, uiLanguage, interviewLanguage, idemKey string) (Preferences, error) {
	if err := actor.validate(); err != nil {
		return Preferences{}, err
	}
	if !contains(AllLanguages, uiLanguage) || !contains(AllLanguages, interviewLanguage) {
		return Preferences{}, fmt.Errorf("%w: 语言必须为 zh-CN | en-US", ErrInvalidInput)
	}
	return idempotent(s, "prefs|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Preferences, error) {
		p := Preferences{
			UserID:            actor.UserID,
			DataRegion:        actor.DataRegion,
			UILanguage:        uiLanguage,
			InterviewLanguage: interviewLanguage,
		}
		if err := s.store.SavePreferences(p); err != nil {
			return Preferences{}, err
		}
		return p, nil
	})
}

// formalActive 为正式面试活动状态集合（单活动设备锁生效区间，FR-030）。
func formalActive(status Status) bool {
	switch status {
	case StatusReady, StatusInSession, StatusScoring, StatusRoundPassed,
		StatusRoundFailed, StatusPracticing, StatusEvaluationIncomplete:
		return true
	default:
		return false
	}
}

// ClaimDevice 申请活动设备；正式面试已被另一设备占用时返回 ErrDeviceActive（FR-030）。
func (s *Service) ClaimDevice(_ context.Context, actor Actor, projectID, deviceID, idemKey string) (Project, error) {
	if err := actor.validate(); err != nil {
		return Project{}, err
	}
	if strings.TrimSpace(deviceID) == "" {
		return Project{}, fmt.Errorf("%w: device_id 必填", ErrInvalidInput)
	}
	return idempotent(s, "claim|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Project, error) {
		proj, err := s.store.GetProject(actor.UserID, actor.DataRegion, projectID)
		if err != nil {
			return Project{}, err
		}
		if formalActive(proj.Status) && proj.ActiveDeviceID != "" && proj.ActiveDeviceID != deviceID {
			return Project{}, fmt.Errorf("%w: 正式面试已在设备 %s 上活动", ErrDeviceActive, proj.ActiveDeviceID)
		}
		proj.ActiveDeviceID = deviceID
		if err := s.store.UpdateProject(proj); err != nil {
			return Project{}, err
		}
		return proj, nil
	})
}

// TransferDevice 安全转移活动设备：仅当前活动设备可发起，转移后原设备会话失效（US-05 场景 3）。
func (s *Service) TransferDevice(_ context.Context, actor Actor, projectID, currentDeviceID, newDeviceID, idemKey string) (Project, error) {
	if err := actor.validate(); err != nil {
		return Project{}, err
	}
	if strings.TrimSpace(currentDeviceID) == "" || strings.TrimSpace(newDeviceID) == "" {
		return Project{}, fmt.Errorf("%w: current_device_id 与 new_device_id 必填", ErrInvalidInput)
	}
	return idempotent(s, "transfer|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Project, error) {
		proj, err := s.store.GetProject(actor.UserID, actor.DataRegion, projectID)
		if err != nil {
			return Project{}, err
		}
		if proj.ActiveDeviceID != currentDeviceID {
			return Project{}, fmt.Errorf("%w: 仅当前活动设备可发起转移", ErrDeviceActive)
		}
		proj.ActiveDeviceID = newDeviceID
		if err := s.store.UpdateProject(proj); err != nil {
			return Project{}, err
		}
		return proj, nil
	})
}

// ReleaseDevice 释放活动设备（结束会话/退出时调用）。
func (s *Service) ReleaseDevice(_ context.Context, actor Actor, projectID, deviceID, idemKey string) (Project, error) {
	if err := actor.validate(); err != nil {
		return Project{}, err
	}
	return idempotent(s, "release|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (Project, error) {
		proj, err := s.store.GetProject(actor.UserID, actor.DataRegion, projectID)
		if err != nil {
			return Project{}, err
		}
		if proj.ActiveDeviceID == deviceID {
			proj.ActiveDeviceID = ""
			if err := s.store.UpdateProject(proj); err != nil {
				return Project{}, err
			}
		}
		return proj, nil
	})
}

func planLocked(proj Project) bool {
	if proj.PlanVersion > 0 {
		return true
	}
	switch proj.Status {
	case StatusReady, StatusInSession, StatusScoring, StatusRoundPassed,
		StatusRoundFailed, StatusPracticing, StatusEvaluationIncomplete, StatusCompleted:
		return true
	default:
		return false
	}
}

func planComplete(plan PlanVersion) bool {
	if len(plan.Rounds) == 0 {
		return false
	}
	for _, r := range plan.Rounds {
		if !r.RubricBound || !r.QuestionCoveragePlanReady {
			return false
		}
	}
	return true
}

func (s *Service) validateRounds(rounds []RoundConfig) error {
	minRounds := s.flow.Bounds.Rounds.UserConfigurable.Min
	maxRounds := s.flow.Bounds.Rounds.UserConfigurable.Max
	if len(rounds) < minRounds || len(rounds) > maxRounds {
		return fmt.Errorf("%w: 轮次数必须为 %d-%d（FR-009）", ErrInvalidInput, minRounds, maxRounds)
	}
	seen := make(map[int]bool, len(rounds))
	for _, r := range rounds {
		if r.Sequence < 1 || r.Sequence > 5 {
			return fmt.Errorf("%w: 轮次序号必须为 1-5", ErrInvalidInput)
		}
		if seen[r.Sequence] {
			return fmt.Errorf("%w: 轮次序号重复 %d", ErrInvalidInput, r.Sequence)
		}
		seen[r.Sequence] = true
		if !contains(RoundTypes, r.RoundType) {
			return fmt.Errorf("%w: 未知轮次类型 %q", ErrInvalidInput, r.RoundType)
		}
		minDur := s.flow.Bounds.DurationMinutes.UserConfigurable.Min
		maxDur := s.flow.Bounds.DurationMinutes.UserConfigurable.Max
		if r.DurationMinutes < minDur || r.DurationMinutes > maxDur {
			return fmt.Errorf("%w: 时长必须为 %d-%d 分钟（FR-009）", ErrInvalidInput, minDur, maxDur)
		}
		if !contains(Difficulties, r.Difficulty) {
			return fmt.Errorf("%w: 难度必须为 basic | standard | challenge", ErrInvalidInput)
		}
		if len(r.CriticalDimensions) == 0 {
			return fmt.Errorf("%w: 每轮至少一个关键维度", ErrInvalidInput)
		}
		dimSeen := make(map[string]bool, len(r.CriticalDimensions))
		for _, d := range r.CriticalDimensions {
			if !contains(DimensionKeys, d) {
				return fmt.Errorf("%w: 未知评分维度 %q", ErrInvalidInput, d)
			}
			if dimSeen[d] {
				return fmt.Errorf("%w: 关键维度重复 %q", ErrInvalidInput, d)
			}
			dimSeen[d] = true
		}
		for _, tool := range r.Tools {
			if !contains(ToolTypes, tool) {
				return fmt.Errorf("%w: 未知岗位工具 %q", ErrInvalidInput, tool)
			}
		}
	}
	return nil
}

func normalizeRounds(rounds []RoundConfig) []RoundConfig {
	out := append([]RoundConfig(nil), rounds...)
	sort.Slice(out, func(i, j int) bool { return out[i].Sequence < out[j].Sequence })
	for i := range out {
		out[i].RubricBound = false
		out[i].QuestionCoveragePlanReady = false
	}
	return out
}

func defaultRoundWeights(rounds []RoundConfig) []RoundWeight {
	out := make([]RoundWeight, 0, len(rounds))
	weight := 100 / len(rounds)
	remainder := 100 % len(rounds)
	for i, r := range rounds {
		w := weight
		if i < remainder {
			w++
		}
		out = append(out, RoundWeight{Sequence: r.Sequence, Weight: w})
	}
	return out
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
