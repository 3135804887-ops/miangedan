// Package scoring 评分服务（TASK-040；SCORING-SPEC 6.1-6.7 伪代码逐条实现）。
package scoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"miangedan/services/region"
)

// ScorerVersion 为评分计算代码版本（写入 version_lineage，可回溯）。
const ScorerVersion = "scoring/v1"

// 错误集。
var (
	ErrInvalidInput            = errors.New("invalid scoring input")
	ErrNotFound                = errors.New("scoring result not found")
	ErrInvalidCursor           = errors.New("invalid cursor")
	ErrFormalReviewUnsupported = errors.New("formal review is implemented in TASK-043")
	ErrScoringFault            = errors.New("scoring service fault")
	ErrReviewLimit             = errors.New("formal review limit reached")
	ErrEvidenceMismatch        = errors.New("frozen evidence mismatch")
)

// Service 为评分服务（独立、可重复、可解释、版本冻结；追加式 ScoreVersion）。
type Service struct {
	store   Store
	now     func() time.Time
	rubrics *RubricRegistry
}

// NewService 创建评分服务。
func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("%w: 缺少存储", ErrInvalidInput)
	}
	return &Service{store: store, now: time.Now}, nil
}

// SetRubricRegistry 注入版本化量表注册表（TASK-044；未注入时不校验版本）。
func (s *Service) SetRubricRegistry(registry *RubricRegistry) {
	s.rubrics = registry
}

// Score 执行单轮评分（SCORING-SPEC 6.1-6.7）。
// 幂等：同一 idempotency_key 重复提交返回首个结果，不产生新 ScoreVersion（NFR-006）。
// 服务故障（panic/持久化失败）降级为 EVALUATION_INCOMPLETE(scoring_service_failure)，
// 不判失败、不解锁、不产生已落库版本；恢复后可用同一幂等键重算。
func (s *Service) Score(_ context.Context, actor Actor, in Input) (result Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = s.faultResult(actor, in)
			err = nil
		}
	}()
	if err := s.validateInput(actor, in); err != nil {
		return Result{}, err
	}
	if in.IsFormalReview {
		return Result{}, ErrFormalReviewUnsupported
	}
	cached, cacheErr := s.store.GetByIdempotencyKey(in.DataRegion, in.IdempotencyKey)
	if cacheErr == nil {
		return cached, nil
	}
	if !errors.Is(cacheErr, ErrNotFound) {
		return Result{}, cacheErr
	}
	result, err = s.compute(in)
	if err != nil {
		return Result{}, err
	}
	latest, latestErr := s.store.GetLatestByAttempt(in.DataRegion, in.AttemptID)
	version := 1
	if latestErr == nil {
		version = latest.ScoreVersion + 1
	} else if !errors.Is(latestErr, ErrNotFound) {
		return Result{}, latestErr
	}
	result.ScoreID = newID()
	result.ScoreVersion = version
	result.ComputedAt = s.now().UTC()
	if saveErr := s.store.SaveResult(result, in.IdempotencyKey); saveErr != nil {
		// 持久化故障：不判失败，降级评估未完成；恢复后重算（SC-EC-18）。
		return s.faultResult(actor, in), nil
	}
	if inputErr := s.store.SaveInput(in.DataRegion, result.ScoreID, in); inputErr != nil {
		return s.faultResult(actor, in), nil
	}
	return result, nil
}

// GetLatest 查询项目轮次最新有效版本。
func (s *Service) GetLatest(_ context.Context, actor Actor, projectID string, roundSequence int) (Result, error) {
	if err := region.ValidateDataRegion(actor.DataRegion); err != nil {
		return Result{}, err
	}
	return s.store.GetLatest(actor.DataRegion, projectID, roundSequence)
}

// ListVersions 分页列出全部保留版本（追加式；历史永不改写）。
func (s *Service) ListVersions(
	_ context.Context, actor Actor, projectID string, roundSequence, limit int, cursor string,
) ([]Result, string, error) {
	if err := region.ValidateDataRegion(actor.DataRegion); err != nil {
		return nil, "", err
	}
	if limit <= 0 || limit > 100 {
		return nil, "", fmt.Errorf("%w: limit 必须为 1-100", ErrInvalidInput)
	}
	return s.store.ListVersions(actor.DataRegion, projectID, roundSequence, limit, cursor)
}

// ---- 校验 ----
func (s *Service) validateInput(actor Actor, in Input) error {
	if err := region.ValidateDataRegion(actor.DataRegion); err != nil {
		return err
	}
	if in.DataRegion != "" && in.DataRegion != actor.DataRegion {
		return fmt.Errorf("%w: 输入区域与调用方区域不一致", ErrInvalidInput)
	}
	if in.SchemaVersion != "" && in.SchemaVersion != "1.0.0" {
		return fmt.Errorf("%w: schema_version 必须为 1.0.0", ErrInvalidInput)
	}
	if strings.TrimSpace(in.ScoringRequestID) == "" ||
		strings.TrimSpace(in.IdempotencyKey) == "" ||
		strings.TrimSpace(in.ProjectID) == "" ||
		strings.TrimSpace(in.AttemptID) == "" {
		return fmt.Errorf("%w: scoring_request_id/idempotency_key/project_id/attempt_id 必填", ErrInvalidInput)
	}
	if in.RoundSequence < 1 || in.RoundSequence > 5 {
		return fmt.Errorf("%w: round_sequence 必须为 1-5", ErrInvalidInput)
	}
	if in.InterviewLanguage != "zh-CN" && in.InterviewLanguage != "en-US" {
		return fmt.Errorf("%w: interview_language 必须为 zh-CN | en-US", ErrInvalidInput)
	}
	if in.InputModeContext.CommunicationMode != ModeVoice &&
		in.InputModeContext.CommunicationMode != ModeText &&
		in.InputModeContext.CommunicationMode != ModeMixed {
		return fmt.Errorf("%w: communication_mode 必须为 voice | text | mixed",
			ErrInvalidInput)
	}
	if in.InputModeContext.CommunicationMode == ModeMixed &&
		(in.InputModeContext.MixedModeVoiceShare == nil ||
			*in.InputModeContext.MixedModeVoiceShare < 0 ||
			*in.InputModeContext.MixedModeVoiceShare > 1) {
		return fmt.Errorf("%w: mixed 模式必须提供 0-1 的 mixed_mode_voice_share",
			ErrInvalidInput)
	}
	if strings.TrimSpace(in.RubricVersion) == "" {
		return fmt.Errorf("%w: rubric_version 必填（冻结量表版本）", ErrInvalidInput)
	}
	if s.rubrics != nil {
		if _, err := s.rubrics.Get(in.RubricVersion); err != nil {
			return err
		}
		if err := s.rubrics.ValidateWeights(in.RubricVersion, in.DimensionWeights); err != nil {
			return err
		}
	}
	if in.AttemptKind != AttemptInitial && in.AttemptKind != AttemptFormalRetry {
		return fmt.Errorf("%w: attempt_kind 必须为 initial | formal_retry", ErrInvalidInput)
	}
	if in.AttemptKind == AttemptFormalRetry &&
		len(in.LockedDimensionScores) == 0 && len(in.DimensionsToRescore) == 0 {
		return fmt.Errorf("%w: formal_retry 必须携带锁定维度分或待重评维度", ErrInvalidInput)
	}
	if in.JobMatchInput != nil {
		if _, err := ComputeJobMatch(in.JobMatchInput); err != nil {
			return err
		}
		if len(in.JobMatchInput.Requirements) > 0 &&
			!in.JobMatchInput.ResumeAvailable &&
			in.DimensionWeights[DimExperienceEvidence] > 0 {
			return fmt.Errorf(
				"%w: JD-only 模式 experience_evidence 权重必须为 0（SC-EC-21，计划阶段重新分配）",
				ErrInvalidInput)
		}
	}
	if len(in.CriticalDimensions) == 0 {
		return fmt.Errorf("%w: critical_dimensions 至少 1 个", ErrInvalidInput)
	}
	weightSum := 0
	for _, d := range DimensionKeys {
		w, ok := in.DimensionWeights[d]
		if !ok || w < 0 || w > 100 {
			return fmt.Errorf("%w: 维度 %s 权重必须为 0-100", ErrInvalidInput, d)
		}
		weightSum += w
	}
	if weightSum != 100 {
		return fmt.Errorf("%w: 冻结权重总和必须为 100，实际 %d", ErrInvalidInput, weightSum)
	}
	seenCritical := map[DimensionKey]bool{}
	for _, d := range in.CriticalDimensions {
		if !validDimension(d) {
			return fmt.Errorf("%w: 未知关键维度 %s", ErrInvalidInput, d)
		}
		if seenCritical[d] {
			return fmt.Errorf("%w: 关键维度重复 %s", ErrInvalidInput, d)
		}
		seenCritical[d] = true
	}
	for _, cp := range in.CoverageAssessments {
		if !validDimension(cp.Dimension) {
			return fmt.Errorf("%w: 未知覆盖点维度 %s", ErrInvalidInput, cp.Dimension)
		}
		if strings.TrimSpace(cp.CoverageID) == "" {
			return fmt.Errorf("%w: coverage_id 必填", ErrInvalidInput)
		}
		if cp.WeightInDimension < 0 {
			return fmt.Errorf("%w: 覆盖点权重不能为负", ErrInvalidInput)
		}
		switch cp.AnswerStatus {
		case AnswerAnswered, AnswerPartial, AnswerSkipped, AnswerUnrecoverable:
		default:
			return fmt.Errorf("%w: 未知 answer_status %q", ErrInvalidInput, cp.AnswerStatus)
		}
		if (cp.AnswerStatus == AnswerAnswered || cp.AnswerStatus == AnswerPartial) &&
			AnchorScore(cp.AnchorLevel) < 0 {
			return fmt.Errorf("%w: 锚点等级必须为 1-5", ErrInvalidInput)
		}
		if in.InputModeContext.CommunicationMode == ModeVoice && cp.Dimension == DimCommunication &&
			cp.InputMode != "" && cp.InputMode != ModeVoice {
			return fmt.Errorf("%w: voice 模式下沟通维度覆盖点必须为语音证据", ErrInvalidInput)
		}
	}
	for _, locked := range in.LockedDimensionScores {
		if !validDimension(locked.Dimension) {
			return fmt.Errorf("%w: 未知锁定维度 %s", ErrInvalidInput, locked.Dimension)
		}
		if locked.Score < PassLine || locked.Score > 100 {
			return fmt.Errorf("%w: 锁定维度分必须为 60-100", ErrInvalidInput)
		}
		if strings.TrimSpace(locked.SourceAttemptID) == "" {
			return fmt.Errorf("%w: 锁定维度必须携带 source_attempt_id", ErrInvalidInput)
		}
	}
	return nil
}

func validDimension(d DimensionKey) bool {
	for _, k := range DimensionKeys {
		if k == d {
			return true
		}
	}
	return false
}

// ---- 核心计算（SCORING-SPEC 6.1-6.6） ----
func (s *Service) compute(in Input) (Result, error) {
	results := make(map[DimensionKey]DimensionResult, len(DimensionKeys))
	critical := keySet(in.CriticalDimensions)
	contradictions := keySet(in.ContradictionUnlocks)
	rescore := keySet(in.DimensionsToRescore)
	locked := lockedMap(in.LockedDimensionScores)

	for _, d := range DimensionKeys {
		weight := in.DimensionWeights[d]
		base := DimensionResult{
			Dimension:  d,
			Weight:     weight,
			IsCritical: critical[d],
		}
		if lockedScore, ok := locked[d]; ok && !contradictions[d] && !rescore[d] {
			score := lockedScore.Score
			base.ScoreStatus = StatusLockedCarried
			base.Score = &score
			results[d] = base
			continue
		}
		if weight == 0 {
			base.ScoreStatus = StatusNotApplicable
			results[d] = base
			continue
		}
		cps := assessmentsFor(d, in.CoverageAssessments)
		if len(cps) == 0 {
			base.ScoreStatus = StatusUncovered
			results[d] = base
			continue
		}
		for _, cp := range cps {
			if cp.IsKeyTranscript && cp.AnswerStatus == AnswerUnrecoverable {
				return s.incomplete(in, ReasonUnrecoverable, results,
					"关键转写不可恢复（ASR 故障且未修订）"), nil
			}
		}
		totalWeight, answeredWeight := 0, 0
		for _, cp := range cps {
			totalWeight += cp.WeightInDimension
			if cp.AnswerStatus == AnswerAnswered || cp.AnswerStatus == AnswerPartial {
				answeredWeight += cp.WeightInDimension
			}
		}
		if totalWeight <= 0 {
			base.ScoreStatus = StatusUncovered
			results[d] = base
			continue
		}
		if float64(answeredWeight)/float64(totalWeight) < MinCoverageRatio {
			base.ScoreStatus = StatusInsufficientEvidence
			results[d] = base
			continue
		}
		if d == DimCommunication {
			results[d] = s.computeCommunication(in, cps, base)
			continue
		}
		results[d] = scoredDimension(cps, base)
	}
	result := s.finish(in, results)
	attachJobMatch(in, &result)
	return result, nil
}

// scoredDimension 计算普通维度分（6.2-6.3：锚点+插值加权平均，half-up 取整）。
func scoredDimension(cps []CoverageAssessment, base DimensionResult) DimensionResult {
	sum, weightSum := 0, 0
	var citations []AnchorCitation
	var evidenceIDs []string
	for _, cp := range cps {
		if cp.AnswerStatus != AnswerAnswered && cp.AnswerStatus != AnswerPartial {
			continue
		}
		score := interpolatedScore(cp)
		sum += score * cp.WeightInDimension
		weightSum += cp.WeightInDimension
		citations = append(citations, cp.AnchorCitations...)
		evidenceIDs = append(evidenceIDs, cp.EvidenceIDs...)
	}
	score := roundHalfUp(float64(sum) / float64(weightSum))
	base.ScoreStatus = StatusScored
	base.Score = &score
	base.AnchorCitations = citations
	base.EvidenceIDs = uniqueStrings(evidenceIDs)
	return base
}

// interpolatedScore 实现 6.2：插值必须引用相邻锚点与证据，否则回退到下锚点。
func interpolatedScore(cp CoverageAssessment) int {
	low := AnchorScore(cp.AnchorLevel)
	if low < 0 {
		return 0
	}
	if cp.InterpolatedScore == nil {
		return low
	}
	high := low + 20
	if high > 100 {
		high = 100
	}
	if !validInterpolation(cp) || *cp.InterpolatedScore < low || *cp.InterpolatedScore > high {
		return low
	}
	return *cp.InterpolatedScore
}

func validInterpolation(cp CoverageAssessment) bool {
	if len(cp.AnchorCitations) == 0 {
		return false
	}
	for _, citation := range cp.AnchorCitations {
		if len(citation.AnchorLevels) == 0 || len(citation.EvidenceIDs) == 0 {
			return false
		}
		minLevel, maxLevel := 6, 0
		for _, level := range citation.AnchorLevels {
			if level < 1 || level > 5 {
				return false
			}
			if level < minLevel {
				minLevel = level
			}
			if level > maxLevel {
				maxLevel = level
			}
		}
		if maxLevel-minLevel > 1 {
			return false
		}
		if cp.AnchorLevel < minLevel || cp.AnchorLevel > maxLevel {
			return false
		}
	}
	return true
}

// finish 执行 6.5 加权总分与 6.6 双门槛判定。
func (s *Service) finish(in Input, results map[DimensionKey]DimensionResult) Result {
	critical := keySet(in.CriticalDimensions)
	for _, d := range DimensionKeys {
		status := results[d].ScoreStatus
		if critical[d] && (status == StatusInsufficientEvidence || status == StatusUncovered) {
			return s.incomplete(in, ReasonInsufficientEvidence, results,
				"关键维度证据不足或未覆盖（评估未完成，不判失败）")
		}
	}
	sum, weightSum := 0, 0
	for _, d := range DimensionKeys {
		dr := results[d]
		if dr.ScoreStatus != StatusScored && dr.ScoreStatus != StatusLockedCarried {
			continue
		}
		sum += dr.Weight * *dr.Score
		weightSum += dr.Weight
	}
	total := roundHalfUp(float64(sum) / float64(weightSum))
	totalOK := total >= PassLine
	criticalOK := true
	var failedCritical, weak, insufficient, uncovered []DimensionKey
	for _, d := range DimensionKeys {
		dr := results[d]
		score := dr.Score
		if score != nil && dr.IsCritical && *score < PassLine {
			criticalOK = false
			failedCritical = append(failedCritical, d)
		}
		if score != nil && !dr.IsCritical && dr.ScoreStatus == StatusScored && *score < PassLine {
			weak = append(weak, d)
		}
		switch dr.ScoreStatus {
		case StatusInsufficientEvidence:
			insufficient = append(insufficient, d)
		case StatusUncovered:
			uncovered = append(uncovered, d)
		}
	}
	status := ResultFail
	if totalOK && criticalOK {
		status = ResultPass
	}
	totalCopy := total
	gate := GateResult{
		TotalGatePassed:          &totalOK,
		CriticalGatePassed:       &criticalOK,
		FailedCriticalDimensions: failedCritical,
		WeakDimensions:           weak,
		InsufficientDimensions:   insufficient,
		UncoveredDimensions:      uncovered,
	}
	summary := fmt.Sprintf("六维加权总分 %d，关键维度全部达标（通过）", total)
	if status == ResultFail {
		summary = fmt.Sprintf("总分 %d 未达 60 或关键维度未过线（未通过）", total)
	}
	strengths, improvements := explainDimensions(results)
	return Result{
		SchemaVersion:    "1.0.0",
		ScoringRequestID: in.ScoringRequestID,
		ProjectID:        in.ProjectID,
		RoundSequence:    in.RoundSequence,
		AttemptID:        in.AttemptID,
		DataRegion:       in.DataRegion,
		RubricVersion:    in.RubricVersion,
		DimensionResults: orderedResults(results),
		RoundTotal:       &totalCopy,
		GateResult:       gate,
		ResultStatus:     status,
		Explanations: Explanations{
			Summary:        summary,
			Strengths:      strengths,
			Improvements:   improvements,
			InputModeNotes: inputModeNotes(in),
		},
		VersionLineage: VersionLineage{
			ScorerVersion:        ScorerVersion,
			ModelVersions:        map[string]string{},
			PromptVersions:       map[string]string{},
			EvidenceSnapshotHash: evidenceSnapshotHash(in),
		},
	}
}

func (s *Service) incomplete(
	in Input, reason string, results map[DimensionKey]DimensionResult, summary string,
) Result {
	reasonCopy := reason
	notes := summary
	result := Result{
		SchemaVersion:    "1.0.0",
		ScoringRequestID: in.ScoringRequestID,
		ProjectID:        in.ProjectID,
		RoundSequence:    in.RoundSequence,
		AttemptID:        in.AttemptID,
		DataRegion:       in.DataRegion,
		RubricVersion:    in.RubricVersion,
		DimensionResults: orderedResults(results),
		ResultStatus:     ResultEvaluationIncomplete,
		IncompleteReason: &reasonCopy,
		Explanations: Explanations{
			Summary:        summary,
			InputModeNotes: &notes,
		},
		VersionLineage: VersionLineage{
			ScorerVersion:        ScorerVersion,
			ModelVersions:        map[string]string{},
			PromptVersions:       map[string]string{},
			EvidenceSnapshotHash: evidenceSnapshotHash(in),
		},
	}
	attachJobMatch(in, &result)
	return result
}

// attachJobMatch 附加岗位匹配度（SCORING-SPEC 6.8；输入已在校验阶段通过）。
func attachJobMatch(in Input, result *Result) {
	if in.JobMatchInput == nil {
		return
	}
	jobMatch, err := ComputeJobMatch(in.JobMatchInput)
	if err != nil {
		return
	}
	result.JobMatch = jobMatch
}

func (s *Service) faultResult(actor Actor, in Input) Result {
	results := make(map[DimensionKey]DimensionResult, len(DimensionKeys))
	for _, d := range DimensionKeys {
		results[d] = DimensionResult{
			Dimension: d, Weight: in.DimensionWeights[d],
			ScoreStatus: StatusInsufficientEvidence, IsCritical: containsKey(in.CriticalDimensions, d),
		}
	}
	reason := ReasonScoringServiceFailure
	summary := "评分服务中途故障：评估未完成（不判失败，恢复后可重算）"
	return Result{
		SchemaVersion:    "1.0.0",
		ScoringRequestID: in.ScoringRequestID,
		ProjectID:        in.ProjectID,
		RoundSequence:    in.RoundSequence,
		AttemptID:        in.AttemptID,
		DataRegion:       actor.DataRegion,
		RubricVersion:    in.RubricVersion,
		DimensionResults: orderedResults(results),
		ResultStatus:     ResultEvaluationIncomplete,
		IncompleteReason: &reason,
		Explanations:     Explanations{Summary: summary, InputModeNotes: &summary},
		VersionLineage: VersionLineage{
			ScorerVersion:        ScorerVersion,
			ModelVersions:        map[string]string{},
			PromptVersions:       map[string]string{},
			EvidenceSnapshotHash: evidenceSnapshotHash(in),
		},
	}
}

// ---- 辅助 ----
func assessmentsFor(d DimensionKey, all []CoverageAssessment) []CoverageAssessment {
	var out []CoverageAssessment
	for _, cp := range all {
		if cp.Dimension == d {
			out = append(out, cp)
		}
	}
	return out
}

func lockedMap(scores []LockedDimensionScore) map[DimensionKey]LockedDimensionScore {
	out := make(map[DimensionKey]LockedDimensionScore, len(scores))
	for _, item := range scores {
		out[item.Dimension] = item
	}
	return out
}

func keySet(keys []DimensionKey) map[DimensionKey]bool {
	out := make(map[DimensionKey]bool, len(keys))
	for _, k := range keys {
		out[k] = true
	}
	return out
}

func containsKey(keys []DimensionKey, target DimensionKey) bool {
	for _, k := range keys {
		if k == target {
			return true
		}
	}
	return false
}

func orderedResults(results map[DimensionKey]DimensionResult) []DimensionResult {
	out := make([]DimensionResult, 0, len(DimensionKeys))
	for _, d := range DimensionKeys {
		out = append(out, results[d])
	}
	return out
}

func explainDimensions(results map[DimensionKey]DimensionResult) ([]string, []string) {
	var strengths, improvements []string
	for _, d := range DimensionKeys {
		dr := results[d]
		if dr.Score != nil && dr.ScoreStatus == StatusScored && *dr.Score >= 75 {
			strengths = append(strengths, fmt.Sprintf("维度 %s 表现良好（%d 分）", d, *dr.Score))
		}
		if dr.Score != nil && dr.ScoreStatus == StatusScored && *dr.Score < PassLine {
			improvements = append(improvements, fmt.Sprintf("维度 %s 需加强（%d 分）", d, *dr.Score))
		}
	}
	return strengths, improvements
}

// inputModeNotes 生成输入模式与证据限制说明（SC-EC-09/10；报告必须标注）。
func inputModeNotes(in Input) *string {
	switch in.InputModeContext.CommunicationMode {
	case ModeText:
		notes := "输入模式：text——口语表现未评估（not_evaluated，不记 0 分）；" +
			"沟通维度按结构与清晰度归一化；报告须标注输入模式与证据限制"
		return &notes
	case ModeMixed:
		share := 0.5
		if in.InputModeContext.MixedModeVoiceShare != nil {
			share = *in.InputModeContext.MixedModeVoiceShare
		}
		notes := fmt.Sprintf(
			"输入模式：mixed——按语音/文字有效证据占比合并（语音占比 %.2f）；"+
				"报告须标注混合模式与证据限制", share)
		return &notes
	default:
		return nil
	}
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item != "" && !seen[item] {
			seen[item] = true
			out = append(out, item)
		}
	}
	return out
}

func roundHalfUp(value float64) int {
	return int(math.Floor(value + 0.5))
}

// evidenceSnapshotHash 计算冻结输入散列（复核输入必须一致；防篡改）。
func evidenceSnapshotHash(in Input) string {
	payload := struct {
		Assessments []CoverageAssessment   `json:"assessments"`
		Evidence    []EvidenceRef          `json:"evidence"`
		Locked      []LockedDimensionScore `json:"locked"`
		Critical    []DimensionKey         `json:"critical"`
		Weights     map[DimensionKey]int   `json:"weights"`
		Rubric      string                 `json:"rubric_version"`
		Rescore     []DimensionKey         `json:"rescore,omitempty"`
		Contradict  []DimensionKey         `json:"contradictions,omitempty"`
		InputMode   InputModeContext       `json:"input_mode"`
	}{
		Assessments: in.CoverageAssessments,
		Evidence:    in.EvidenceItems,
		Locked:      in.LockedDimensionScores,
		Critical:    in.CriticalDimensions,
		Weights:     in.DimensionWeights,
		Rubric:      in.RubricVersion,
		Rescore:     in.DimensionsToRescore,
		Contradict:  in.ContradictionUnlocks,
		InputMode:   in.InputModeContext,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
