// Package scoring 提供独立、可重复、可解释、版本冻结的评分服务（TASK-040，FR-021）。
// 追踪：docs/ai/SCORING-SPEC.md；ai/schemas/scoring-input.schema.json /
// scoring-result.schema.json；config/rubrics/v1/default.yaml。
// 红线：分数一经冻结不可改写（ADR-0004 追加式）；提示词不产出最终评分；
// 练习永不进入本服务；付费/便利设置/保护属性永远不是输入。
package scoring

import "time"

// DimensionKey 为六维维度键（与 rubric/SCORING-SPEC 一致）。
type DimensionKey string

// 六维集合 D。
const (
	DimProfessional          DimensionKey = "professional_competence"
	DimProblemSolving        DimensionKey = "problem_solving"
	DimCommunication         DimensionKey = "communication"
	DimExperienceEvidence    DimensionKey = "experience_evidence"
	DimBehavioralCollaborate DimensionKey = "behavioral_collaboration"
	DimLearningAdaptability  DimensionKey = "learning_adaptability"
)

// DimensionKeys 为固定顺序（结果输出按此序）。
var DimensionKeys = []DimensionKey{
	DimProfessional, DimProblemSolving, DimCommunication,
	DimExperienceEvidence, DimBehavioralCollaborate, DimLearningAdaptability,
}

// DefaultWeights 为 PRD 默认权重（25/20/15/15/15/10，总和 100）。
var DefaultWeights = map[DimensionKey]int{
	DimProfessional:          25,
	DimProblemSolving:        20,
	DimCommunication:         15,
	DimExperienceEvidence:    15,
	DimBehavioralCollaborate: 15,
	DimLearningAdaptability:  10,
}

// AnchorScore 为行为锚点映射（1→20 … 5→100；rubric anchors）。
func AnchorScore(level int) int {
	switch level {
	case 1:
		return 20
	case 2:
		return 40
	case 3:
		return 60
	case 4:
		return 80
	case 5:
		return 100
	default:
		return -1
	}
}

// 状态与模式枚举。
const (
	AnswerAnswered      = "answered"
	AnswerPartial       = "partial"
	AnswerSkipped       = "skipped"
	AnswerUnrecoverable = "unrecoverable"

	StatusScored               = "scored"
	StatusInsufficientEvidence = "insufficient_evidence"
	StatusUncovered            = "uncovered"
	StatusNotApplicable        = "not_applicable"
	StatusLockedCarried        = "locked_carried"

	ResultPass                  = "PASS"
	ResultFail                  = "FAIL"
	ResultEvaluationIncomplete  = "EVALUATION_INCOMPLETE"
	ReasonInsufficientEvidence  = "insufficient_evidence"
	ReasonUnrecoverable         = "unrecoverable_transcript"
	ReasonScoringServiceFailure = "scoring_service_failure"

	ModeVoice = "voice"
	ModeText  = "text"
	ModeMixed = "mixed"

	AttemptInitial     = "initial"
	AttemptFormalRetry = "formal_retry"
)

// PassLine 为通过门槛（PRD 硬门槛：round_total ≥60 且全部关键维度 ≥60）。
const PassLine = 60

// MinCoverageRatio 为证据充分度阈值（OD-08：0.5，可校准参数）。
const MinCoverageRatio = 0.5

// Actor 为评分调用方身份（区域强绑定）。
type Actor struct {
	UserID     string
	DataRegion string
}

// AnswerStatus 为回答状态（turn-evidence answer.answer_status）。
type AnswerStatus string

// ScoreStatus 为维度证据状态机状态（SCORING-SPEC 6.1）。
type ScoreStatus string

// AnchorCitation 为插值引用（SCORING-SPEC 6.2：必须引用相邻锚点等级与证据 ID）。
type AnchorCitation struct {
	AnchorLevels []int    `json:"anchor_levels"`
	EvidenceIDs  []string `json:"evidence_ids"`
	Rationale    string   `json:"rationale,omitempty"`
}

// CoverageAssessment 为冻结的覆盖点评分判定（证据提取层产出；评分服务只消费）。
// 覆盖点集合必须包含该维度全部计划覆盖点（含未作答），weight_in_dimension 来自冻结计划。
type CoverageAssessment struct {
	CoverageID        string           `json:"coverage_id"`
	Dimension         DimensionKey     `json:"dimension"`
	EvidenceIDs       []string         `json:"evidence_ids"`
	WeightInDimension int              `json:"weight_in_dimension"`
	AnswerStatus      AnswerStatus     `json:"answer_status"`
	IsKeyTranscript   bool             `json:"is_key_transcript"`
	InputMode         string           `json:"input_mode"`
	AnchorLevel       int              `json:"anchor_level"`
	InterpolatedScore *int             `json:"interpolated_score,omitempty"`
	AnchorCitations   []AnchorCitation `json:"anchor_citations,omitempty"`
	StructureClarity  *int             `json:"structure_clarity,omitempty"`
	OralDelivery      *int             `json:"oral_delivery,omitempty"`
}

// InputModeContext 为输入模式与便利设置上下文（6.4：摄像头与便利设置不参与计算）。
type InputModeContext struct {
	ModesUsed              []string `json:"modes_used"`
	CommunicationMode      string   `json:"communication_mode"`
	MixedModeVoiceShare    *float64 `json:"mixed_mode_voice_share,omitempty"`
	AccommodationsInEffect []string `json:"accommodations_in_effect,omitempty"`
}

// LockedDimensionScore 为正式重试时来自前次尝试的锁定维度分（≥60）。
type LockedDimensionScore struct {
	Dimension          DimensionKey `json:"dimension"`
	Score              int          `json:"score"`
	SourceAttemptID    string       `json:"source_attempt_id"`
	SourceScoreVersion int          `json:"source_score_version"`
}

// EvidenceRef 为冻结证据引用（复核时以 evidence_snapshot_hash 校验未被改写）。
type EvidenceRef struct {
	EvidenceRef  string `json:"evidence_ref"`
	EvidenceHash string `json:"evidence_hash,omitempty"`
}

// ScoringInput 为评分服务输入（冻结证据包；对齐 scoring-input.schema.json）。
type ScoringInput struct {
	SchemaVersion         string                 `json:"schema_version"`
	ScoringRequestID      string                 `json:"scoring_request_id"`
	IdempotencyKey        string                 `json:"idempotency_key"`
	ProjectID             string                 `json:"project_id"`
	RoundSequence         int                    `json:"round_sequence"`
	AttemptID             string                 `json:"attempt_id"`
	AttemptKind           string                 `json:"attempt_kind"`
	DataRegion            string                 `json:"data_region"`
	InterviewLanguage     string                 `json:"interview_language"`
	RubricVersion         string                 `json:"rubric_version"`
	DimensionWeights      map[DimensionKey]int   `json:"dimension_weights"`
	CriticalDimensions    []DimensionKey         `json:"critical_dimensions"`
	EvidenceItems         []EvidenceRef          `json:"evidence_items"`
	CoverageAssessments   []CoverageAssessment   `json:"coverage_assessments,omitempty"`
	InputModeContext      InputModeContext       `json:"input_mode_context"`
	LockedDimensionScores []LockedDimensionScore `json:"locked_dimension_scores,omitempty"`
	DimensionsToRescore   []DimensionKey         `json:"dimensions_to_rescore,omitempty"`
	ContradictionUnlocks  []DimensionKey         `json:"contradiction_unlocks,omitempty"`
	IsFormalReview        bool                   `json:"is_formal_review"`
	OriginalScoreVersion  int                    `json:"original_score_version,omitempty"`
	SubmittedAt           time.Time              `json:"submitted_at"`
}

// CommunicationSubscores 为沟通维度子项（6.4；文字模式 oral_delivery=not_evaluated）。
type CommunicationSubscores struct {
	StructureClarity         *int `json:"structure_clarity"`
	OralDelivery             *int `json:"oral_delivery"`
	OralDeliveryNotEvaluated bool `json:"oral_delivery_not_evaluated"`
}

// DimensionResult 为单维评分结果。
type DimensionResult struct {
	Dimension       DimensionKey            `json:"dimension"`
	Weight          int                     `json:"weight"`
	ScoreStatus     ScoreStatus             `json:"score_status"`
	Score           *int                    `json:"score"`
	IsCritical      bool                    `json:"is_critical"`
	GatePassed      *bool                   `json:"gate_passed"`
	AnchorCitations []AnchorCitation        `json:"anchor_citations,omitempty"`
	Subscores       *CommunicationSubscores `json:"subscores,omitempty"`
	EvidenceIDs     []string                `json:"evidence_ids,omitempty"`
}

// GateResult 为双门槛判定（6.6）。
type GateResult struct {
	TotalGatePassed          *bool          `json:"total_gate_passed"`
	CriticalGatePassed       *bool          `json:"critical_gate_passed"`
	FailedCriticalDimensions []DimensionKey `json:"failed_critical_dimensions,omitempty"`
	WeakDimensions           []DimensionKey `json:"weak_dimensions,omitempty"`
	InsufficientDimensions   []DimensionKey `json:"insufficient_dimensions,omitempty"`
	UncoveredDimensions      []DimensionKey `json:"uncovered_dimensions,omitempty"`
}

// Explanations 为可解释评分摘要（确定性生成；报告措辞不得伪造分数）。
type Explanations struct {
	Summary        string   `json:"summary"`
	Strengths      []string `json:"strengths,omitempty"`
	Improvements   []string `json:"improvements,omitempty"`
	InputModeNotes *string  `json:"input_mode_notes,omitempty"`
}

// VersionLineage 为版本血缘（每次推理可回溯；证据散列防篡改）。
type VersionLineage struct {
	ScorerVersion        string            `json:"scorer_version"`
	ModelVersions        map[string]string `json:"model_versions"`
	PromptVersions       map[string]string `json:"prompt_versions"`
	EvidenceSnapshotHash string            `json:"evidence_snapshot_hash"`
	SupersedesScoreID    *string           `json:"supersedes_score_id,omitempty"`
}

// ScoringResult 为评分服务输出（ScoreVersion，追加式；对齐 scoring-result.schema.json）。
type ScoringResult struct {
	SchemaVersion    string            `json:"schema_version"`
	ScoreID          string            `json:"score_id"`
	ScoreVersion     int               `json:"score_version"`
	ScoringRequestID string            `json:"scoring_request_id"`
	ProjectID        string            `json:"project_id"`
	RoundSequence    int               `json:"round_sequence"`
	AttemptID        string            `json:"attempt_id"`
	DataRegion       string            `json:"data_region"`
	RubricVersion    string            `json:"rubric_version"`
	DimensionResults []DimensionResult `json:"dimension_results"`
	RoundTotal       *int              `json:"round_total"`
	GateResult       GateResult        `json:"gate_result"`
	ResultStatus     string            `json:"result_status"`
	IncompleteReason *string           `json:"incomplete_reason,omitempty"`
	Explanations     Explanations      `json:"explanations"`
	VersionLineage   VersionLineage    `json:"version_lineage"`
	ComputedAt       time.Time         `json:"computed_at"`
}
