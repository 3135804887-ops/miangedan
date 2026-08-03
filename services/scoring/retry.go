package scoring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"miangedan/services/region"
)

// BeginRetryRequest 为发起正式重试请求（来源 = 该轮最新正式尝试）。
type BeginRetryRequest struct {
	ProjectID      string
	RoundSequence  int
	RequestID      string
	IdempotencyKey string
}

// BeginRetry 发起正式重试（TASK-053；SCORING-SPEC 6.7，DOMAIN-MODEL §6.14）。
// 前置：上一轮结果为 FAIL 或 EVALUATION_INCOMPLETE；重试使用新问题；
// locked = 上轮 score ≥60 的维度；rescore_scope = 失败维度 ∪ 未覆盖点对应维度。
func (s *Service) BeginRetry(_ context.Context, actor Actor, req BeginRetryRequest) (RetryAttempt, error) {
	if err := region.ValidateDataRegion(actor.DataRegion); err != nil {
		return RetryAttempt{}, err
	}
	if strings.TrimSpace(req.ProjectID) == "" ||
		strings.TrimSpace(req.IdempotencyKey) == "" {
		return RetryAttempt{}, fmt.Errorf("%w: project_id/idempotency_key 必填", ErrInvalidInput)
	}
	if req.RoundSequence < 1 || req.RoundSequence > 5 {
		return RetryAttempt{}, fmt.Errorf("%w: round_sequence 必须为 1-5", ErrInvalidInput)
	}
	cached, cacheErr := s.store.GetRetryAttemptByIdempotencyKey(actor.DataRegion, req.IdempotencyKey)
	if cacheErr == nil {
		return cached, nil
	}
	if !errors.Is(cacheErr, ErrNotFound) {
		return RetryAttempt{}, cacheErr
	}
	source, err := s.store.GetLatest(actor.DataRegion, req.ProjectID, req.RoundSequence)
	if err != nil {
		return RetryAttempt{}, err
	}
	if source.ResultStatus == ResultPass {
		return RetryAttempt{}, fmt.Errorf(
			"%w: 该轮已通过（PASS），不允许正式重试", ErrStateConflict)
	}
	if source.ResultStatus != ResultFail &&
		source.ResultStatus != ResultEvaluationIncomplete {
		return RetryAttempt{}, fmt.Errorf(
			"%w: 仅 FAIL/EVALUATION_INCOMPLETE 可发起正式重试（当前 %s）",
			ErrStateConflict, source.ResultStatus)
	}
	locked, rescope := retryDimensions(source)
	attempt := RetryAttempt{
		AttemptID:         newID(),
		ProjectID:         req.ProjectID,
		RoundSequence:     req.RoundSequence,
		SourceAttemptID:   source.AttemptID,
		Status:            RetryStatusScheduled,
		LockedDimensions:  locked,
		RescopeDimensions: rescope,
		DataRegion:        actor.DataRegion,
		CreatedAt:         s.now().UTC(),
	}
	if err := s.store.SaveRetryAttempt(attempt, req.IdempotencyKey); err != nil {
		return RetryAttempt{}, err
	}
	return attempt, nil
}

// retryDimensions 由来源结果派生锁定维度（≥60）与重评范围（<60 或未覆盖）。
func retryDimensions(source Result) ([]DimensionKey, []DimensionKey) {
	locked := make([]DimensionKey, 0, len(DimensionKeys))
	rescope := make([]DimensionKey, 0, len(DimensionKeys))
	for _, dr := range source.DimensionResults {
		switch dr.ScoreStatus {
		case StatusScored, StatusLockedCarried:
			if dr.Score != nil && *dr.Score >= PassLine {
				locked = append(locked, dr.Dimension)
			} else {
				rescope = append(rescope, dr.Dimension)
			}
		case StatusInsufficientEvidence, StatusUncovered:
			rescope = append(rescope, dr.Dimension)
		}
	}
	return locked, rescope
}

// SelectRetryQuestions 为新题选择（HANDOFF-SPEC 5.3：禁止重复已通过相同问题）。
// 相同措辞一律丢弃；仅 direct_contradiction / new_job_scenario_transfer
// 例外允许重新验证同一主题（仍使用不同措辞）。
func (s *Service) SelectRetryQuestions(
	_ context.Context, actor Actor, req SelectRetryQuestionsRequest,
) (RetryQuestionSelection, error) {
	if err := region.ValidateDataRegion(actor.DataRegion); err != nil {
		return RetryQuestionSelection{}, err
	}
	attempt, err := s.store.GetRetryAttempt(actor.DataRegion, req.AttemptID)
	if err != nil {
		return RetryQuestionSelection{}, err
	}
	if attempt.Status != RetryStatusScheduled && attempt.Status != RetryStatusInProgress {
		return RetryQuestionSelection{}, fmt.Errorf(
			"%w: 重试状态 %s 不允许选题", ErrStateConflict, attempt.Status)
	}
	reasons := map[string]bool{}
	for _, reason := range req.ReverificationReasons {
		reasons[reason] = true
	}
	allowTopic := reasons["direct_contradiction"] || reasons["new_job_scenario_transfer"]
	selected := make([]string, 0, len(req.CandidatePool))
	skipped := make([]string, 0)
	for _, candidate := range req.CandidatePool {
		if RepeatsQuestion(candidate, req.DoNotRepeat) {
			// 相同措辞永不重复；主题重复仅在例外下允许（仍不同措辞）。
			if allowTopic && !exactRepeat(candidate, req.DoNotRepeat) {
				selected = append(selected, candidate)
				continue
			}
			skipped = append(skipped, candidate)
			continue
		}
		selected = append(selected, candidate)
	}
	if len(selected) == 0 {
		return RetryQuestionSelection{}, fmt.Errorf(
			"%w: 候选池全部与已通过问题重复，请重新生成新题", ErrInvalidInput)
	}
	return RetryQuestionSelection{
		AttemptID:      req.AttemptID,
		Selected:       selected,
		SkippedRepeats: skipped,
	}, nil
}

// RepeatsQuestion 语义去重：规范化后相同或互相包含即命中（跨语言标点归一）。
func RepeatsQuestion(candidate string, doNotRepeat []string) bool {
	norm := normalizeQuestion(candidate)
	if norm == "" {
		return false
	}
	for _, stored := range doNotRepeat {
		normalized := normalizeQuestion(stored)
		if normalized == "" {
			continue
		}
		if norm == normalized ||
			strings.Contains(norm, normalized) ||
			strings.Contains(normalized, norm) {
			return true
		}
	}
	return false
}

func exactRepeat(candidate string, doNotRepeat []string) bool {
	norm := normalizeQuestion(candidate)
	for _, stored := range doNotRepeat {
		if normalizeQuestion(stored) == norm {
			return true
		}
	}
	return false
}

func normalizeQuestion(text string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(text)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// ScoreRetry 执行正式重试评分（TASK-053）：校验重试尝试登记、新分替换失败维度、
// 新旧证据替换（旧证据保留、新证据引用进入结果）、状态推进 COMPLETED。
func (s *Service) ScoreRetry(
	ctx context.Context, actor Actor, in Input,
) (RetryResult, error) {
	if in.AttemptKind != AttemptFormalRetry {
		return RetryResult{}, fmt.Errorf(
			"%w: ScoreRetry 仅接受 formal_retry 输入", ErrInvalidInput)
	}
	attempt, err := s.store.GetRetryAttempt(actor.DataRegion, in.AttemptID)
	if err != nil {
		return RetryResult{}, fmt.Errorf(
			"%w: 重试尝试不存在（练习/未知尝试不能作为正式重试）: %v",
			ErrInvalidInput, err)
	}
	if attempt.ProjectID != in.ProjectID || attempt.RoundSequence != in.RoundSequence {
		return RetryResult{}, fmt.Errorf(
			"%w: 重试尝试与评分输入归属不一致", ErrInvalidInput)
	}
	score, err := s.Score(ctx, actor, in)
	if err != nil {
		return RetryResult{}, err
	}
	replaced, missingEvidence := replacedDimensions(attempt, score)
	if len(missingEvidence) > 0 {
		return RetryResult{}, fmt.Errorf(
			"%w: 重评维度缺少新证据引用：%v（新旧证据替换要求）",
			ErrInvalidInput, missingEvidence)
	}
	if err := s.store.UpdateRetryStatus(actor.DataRegion, attempt.AttemptID, RetryStatusCompleted); err != nil {
		return RetryResult{}, err
	}
	attempt.Status = RetryStatusCompleted
	return RetryResult{
		Attempt:            attempt,
		Score:              score,
		ReplacedDimensions: replaced,
	}, nil
}

// replacedDimensions 计算重评维度中分数发生替换的维度，并校验新证据引用。
func replacedDimensions(attempt RetryAttempt, score Result) ([]DimensionKey, []DimensionKey) {
	byDim := map[DimensionKey]DimensionResult{}
	for _, dr := range score.DimensionResults {
		byDim[dr.Dimension] = dr
	}
	replaced := make([]DimensionKey, 0, len(attempt.RescopeDimensions))
	missing := make([]DimensionKey, 0)
	for _, d := range attempt.RescopeDimensions {
		dr, ok := byDim[d]
		if !ok {
			continue
		}
		if dr.Score != nil && dr.ScoreStatus == StatusScored {
			replaced = append(replaced, d)
		}
		if len(dr.EvidenceIDs) == 0 {
			missing = append(missing, d)
		}
	}
	return replaced, missing
}
