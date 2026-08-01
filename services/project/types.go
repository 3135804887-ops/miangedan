// Package project 提供面试项目、计划版本与轮次配置服务（TASK-016）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-016；docs/domain/DOMAIN-MODEL.md 6.8/6.9；
// FR-009 ~ FR-011；docs/domain/INTERVIEW-STATE-MACHINE.md 第 5 节。
package project

import (
	"errors"
	"time"
)

// Status 为项目状态；取值必须与 INTERVIEW-STATE-MACHINE.md 第 5.1 节一致，禁止服务自创状态。
type Status string

// 项目状态全集。
const (
	StatusDraft                Status = "DRAFT"
	StatusParsing              Status = "PARSING"
	StatusMaterialReview       Status = "MATERIAL_REVIEW"
	StatusParseFailed          Status = "PARSE_FAILED"
	StatusPlanGenerating       Status = "PLAN_GENERATING"
	StatusPlanReview           Status = "PLAN_REVIEW"
	StatusPlanFailed           Status = "PLAN_FAILED"
	StatusReady                Status = "READY"
	StatusInSession            Status = "IN_SESSION"
	StatusScoring              Status = "SCORING"
	StatusRoundPassed          Status = "ROUND_PASSED"
	StatusRoundFailed          Status = "ROUND_FAILED"
	StatusPracticing           Status = "PRACTICING"
	StatusEvaluationIncomplete Status = "EVALUATION_INCOMPLETE"
	StatusCompleted            Status = "COMPLETED"
)

// DegradedMode 为材料缺失降级模式（PRD FR-004/FR-005）。
type DegradedMode string

// 降级模式枚举。
const (
	ModeFull       DegradedMode = "full"
	ModeJDOnly     DegradedMode = "jd_only"
	ModeResumeOnly DegradedMode = "resume_only"
	ModeNeither    DegradedMode = "neither"
)

// 枚举常量（与 openapi.yaml / interview-flows 契约一致）。
var (
	AllLanguages = []string{"zh-CN", "en-US"}
	Difficulties = []string{"basic", "standard", "challenge"}
	RoundTypes   = []string{
		"screening_resume_deepdive", "role_professional", "comprehensive_final",
		"case_study", "portfolio_review", "management_scenario", "business_scenario",
	}
	ToolTypes     = []string{"code_editor", "whiteboard", "case_materials", "portfolio"}
	DimensionKeys = []string{
		"professional_competence", "problem_solving", "communication",
		"experience_evidence", "behavioral_collaboration", "learning_adaptability",
	}
	// Accommodations 为会前便利设置（确认计划时冻结，SCORING-SPEC 不视为弱点）。
	Accommodations = []string{
		"text_only", "mixed_input", "slower_avatar_speech", "repeat_questions",
		"extended_time", "silence_threshold_adjusted", "no_proactive_interruption",
		"reduced_motion", "tool_keyboard_alternative",
	}
)

// 服务错误（httpapi 映射为开放 API 错误码）。
var (
	ErrNotFound       = errors.New("project not found")
	ErrStateConflict  = errors.New("project state conflict")
	ErrPlanIncomplete = errors.New("plan incomplete")
	ErrInvalidInput   = errors.New("invalid input")
)

// MaterialRef 为冻结材料版本引用（简历/岗位）。
type MaterialRef struct {
	ID      string
	Version int
}

// Project 为面试项目聚合根（DOMAIN-MODEL 6.8）。
type Project struct {
	ProjectID             string
	UserID                string
	DataRegion            string
	Name                  string
	InterviewLanguage     string
	DegradedMode          DegradedMode
	DegradedModeConsentID string
	ResumeRef             *MaterialRef
	JobRef                *MaterialRef
	Status                Status
	CurrentRoundSequence  int
	PlanVersion           int
	ActiveDeviceID        string
	AssignmentID          string
	CreatedAt             time.Time
}

// RoundConfig 为单轮配置（RoundConfigInput + 冻结期就绪标记）。
type RoundConfig struct {
	Sequence                  int
	RoundType                 string
	Role                      string
	Focus                     string
	DurationMinutes           int
	Difficulty                string
	CriticalDimensions        []string
	Tools                     []string
	StyleParameters           map[string]any
	AvatarCharacterID         string
	VoiceID                   string
	RubricBound               bool
	QuestionCoveragePlanReady bool
}

// RoundWeight 为轮次权重（确认后冻结）。
type RoundWeight struct {
	Sequence int
	Weight   int
}

// ProcessSourceRef 为计划引用的企业流程来源元数据。
type ProcessSourceRef struct {
	SourceID               string
	SourceType             string
	URL                    string
	RetrievedAt            time.Time
	Credibility            string
	IsUnofficialExperience bool
}

// PlanVersion 为面试计划版本（DOMAIN-MODEL 6.9；ai/schemas/interview-plan.schema.json）。
type PlanVersion struct {
	ProjectID               string
	PlanVersion             int
	DataRegion              string
	InterviewLanguage       string
	ResumeRef               *MaterialRef
	JobRef                  *MaterialRef
	DegradedMode            DegradedMode
	DegradedModeConsentID   string
	RubricVersion           string
	DimensionWeights        map[string]int
	Rounds                  []RoundConfig
	RoundWeights            []RoundWeight
	ProcessSourceRefs       []ProcessSourceRef
	FlowUsesGenericTemplate bool
	Frozen                  bool
	CreatedAt               time.Time
}

// DeletionTask 为删除编排任务（真实进度由 TASK-083 编排）。
type DeletionTask struct {
	TaskID string
	Status string
}
