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

// idempotent 按幂等键执行写操作：重复键返回首次结果（NFR-006）。
func idempotent[T any](s *Service, key string, run func() (T, error)) (T, error) {
	var zero T
	if key != "" {
		var cached T
		found, err := s.idem.Recall(key, &cached)
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
	if key != "" {
		if err := s.idem.Remember(key, result); err != nil {
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
	return idempotent(s, "create|"+actor.UserID+"|"+actor.DataRegion+"|"+idemKey, func() (Project, error) {
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

// ListProjects 按筛选列出项目。
func (s *Service) ListProjects(_ context.Context, actor Actor, f ListFilter) ([]Project, error) {
	if err := actor.validate(); err != nil {
		return nil, err
	}
	return s.store.ListProjects(actor.UserID, actor.DataRegion, f)
}

// RenameProject 重命名项目（名称不在冻结范围内）。
func (s *Service) RenameProject(_ context.Context, actor Actor, projectID, name, idemKey string) (Project, error) {
	if err := actor.validate(); err != nil {
		return Project{}, err
	}
	if len(name) < 1 || len(name) > 120 {
		return Project{}, fmt.Errorf("%w: 项目名长度必须为 1-120", ErrInvalidInput)
	}
	return idempotent(s, "rename|"+actor.UserID+"|"+actor.DataRegion+"|"+idemKey, func() (Project, error) {
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
	return idempotent(s, "delete|"+actor.UserID+"|"+actor.DataRegion+"|"+idemKey, func() (DeletionTask, error) {
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
	return idempotent(s, "duplicate|"+actor.UserID+"|"+actor.DataRegion+"|"+idemKey, func() (Project, error) {
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
	return idempotent(s, "editplan|"+actor.UserID+"|"+actor.DataRegion+"|"+idemKey, func() (PlanVersion, error) {
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
	return idempotent(s, "confirm|"+actor.UserID+"|"+actor.DataRegion+"|"+idemKey, func() (Project, error) {
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
