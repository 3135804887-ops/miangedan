package scoring

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"miangedan/services/region"
)

// 正式复核范围声明（SCORING-SPEC 6.10：计算均基于完整冻结输入）。
const (
	ReviewScopeQuestion  = "question"
	ReviewScopeDimension = "dimension"
	ReviewScopeRound     = "round"
)

// Review 执行正式复核（TASK-043；SCORING-SPEC 6.10）。
// 规则：
//   - 每次正式尝试允许一次自动复核；第二次请求必须被拒绝（SC-EC-17）；
//   - 复核输入 = 与原始评分完全相同的冻结证据（以 evidence_snapshot_hash 校验）、
//     量表、权重与版本；不允许补充或改写回答；
//   - 复核产出新 ScoreVersion（supersedes_score_id 指向原版本），展示原结果、
//     新结果与改变原因；全部版本保留，历史分数不可改写（SC-EC-16）；
//   - 复核后是否达到门槛由工作流据 result_status 解锁，评分服务不直接改业务状态。
func (s *Service) Review(_ context.Context, actor Actor, req ReviewRequest) (ReviewResult, error) {
	if err := validateReviewRequest(actor, req); err != nil {
		return ReviewResult{}, err
	}
	cached, cacheErr := s.store.GetReviewByIdempotencyKey(actor.DataRegion, req.IdempotencyKey)
	if cacheErr == nil {
		return cached, nil
	}
	if !errors.Is(cacheErr, ErrNotFound) {
		return ReviewResult{}, cacheErr
	}
	count, err := s.store.CountReviews(actor.DataRegion, req.AttemptID)
	if err != nil {
		return ReviewResult{}, err
	}
	if count >= 1 {
		return ReviewResult{}, fmt.Errorf("%w: 每次正式尝试仅允许一次复核", ErrReviewLimit)
	}
	original, err := s.store.GetFirstByAttempt(actor.DataRegion, req.AttemptID)
	if err != nil {
		return ReviewResult{}, err
	}
	if original.ProjectID != req.ProjectID || original.RoundSequence != req.RoundSequence {
		return ReviewResult{}, fmt.Errorf(
			"%w: 复核请求与原始评分归属不一致", ErrInvalidInput)
	}
	frozen, err := s.store.GetInput(actor.DataRegion, original.ScoreID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ReviewResult{}, fmt.Errorf(
				"%w: 缺少冻结输入（疑似证据链异常，触发安全审计）", ErrEvidenceMismatch)
		}
		return ReviewResult{}, err
	}
	if evidenceSnapshotHash(frozen) != original.VersionLineage.EvidenceSnapshotHash {
		return ReviewResult{}, fmt.Errorf(
			"%w: 冻结输入散列与原始评分不一致（疑似证据篡改）", ErrEvidenceMismatch)
	}
	recomputed, err := s.compute(frozen)
	if err != nil {
		return ReviewResult{}, err
	}
	review := recomputed
	review.ScoreID = newID()
	review.ScoreVersion = original.ScoreVersion + 1
	supersedes := original.ScoreID
	review.VersionLineage.SupersedesScoreID = &supersedes
	review.ComputedAt = s.now().UTC()
	changes := compareDimensionResults(original, review)
	reason := reviewReason(original, review)
	if review.Explanations.Summary != "" {
		review.Explanations.Summary += "；" + reason
	} else {
		review.Explanations.Summary = reason
	}
	if err := s.store.SaveResult(review, req.IdempotencyKey); err != nil {
		return ReviewResult{}, fmt.Errorf("%w: 复核结果持久化失败: %v", ErrScoringFault, err)
	}
	if err := s.store.SaveInput(actor.DataRegion, review.ScoreID, frozen); err != nil {
		return ReviewResult{}, fmt.Errorf("%w: 复核冻结输入持久化失败: %v", ErrScoringFault, err)
	}
	if err := s.store.MarkReview(actor.DataRegion, req.AttemptID); err != nil {
		return ReviewResult{}, err
	}
	reviewResult := ReviewResult{
		Original: original,
		Review:   review,
		Changes:  changes,
		Reason:   reason,
	}
	if err := s.store.SaveReview(actor.DataRegion, req.IdempotencyKey, reviewResult); err != nil {
		return ReviewResult{}, err
	}
	return reviewResult, nil
}

func validateReviewRequest(actor Actor, req ReviewRequest) error {
	if err := region.ValidateDataRegion(actor.DataRegion); err != nil {
		return err
	}
	if strings.TrimSpace(req.ProjectID) == "" ||
		strings.TrimSpace(req.AttemptID) == "" ||
		strings.TrimSpace(req.IdempotencyKey) == "" {
		return fmt.Errorf("%w: project_id/attempt_id/idempotency_key 必填", ErrInvalidInput)
	}
	if req.RoundSequence < 1 || req.RoundSequence > 5 {
		return fmt.Errorf("%w: round_sequence 必须为 1-5", ErrInvalidInput)
	}
	switch req.Scope {
	case ReviewScopeQuestion, ReviewScopeDimension, ReviewScopeRound:
	default:
		return fmt.Errorf("%w: scope 必须为 question | dimension | round", ErrInvalidInput)
	}
	return nil
}

// compareDimensionResults 生成复核前后逐维对比（分数与状态；原结果不可改写）。
func compareDimensionResults(before, after Result) []DimensionChange {
	byDim := func(r Result) map[DimensionKey]DimensionResult {
		out := make(map[DimensionKey]DimensionResult, len(r.DimensionResults))
		for _, dr := range r.DimensionResults {
			out[dr.Dimension] = dr
		}
		return out
	}
	beforeDims := byDim(before)
	afterDims := byDim(after)
	changes := make([]DimensionChange, 0, len(DimensionKeys))
	for _, d := range DimensionKeys {
		bd, bok := beforeDims[d]
		ad, aok := afterDims[d]
		if !bok || !aok {
			continue
		}
		changes = append(changes, DimensionChange{
			Dimension:    d,
			BeforeScore:  bd.Score,
			AfterScore:   ad.Score,
			BeforeStatus: bd.ScoreStatus,
			AfterStatus:  ad.ScoreStatus,
		})
	}
	return changes
}

// reviewReason 生成改变原因（SC-EC-16：展示原结果、新结果与原因）。
func reviewReason(before, after Result) string {
	if before.VersionLineage.ScorerVersion != after.VersionLineage.ScorerVersion {
		return fmt.Sprintf(
			"正式复核：以完全相同的冻结证据与量表重算，计算版本由 %s 变更为 %s，产生新 ScoreVersion；历史版本保留",
			before.VersionLineage.ScorerVersion, after.VersionLineage.ScorerVersion)
	}
	if before.ResultStatus != after.ResultStatus {
		return "正式复核：以完全相同的冻结证据与量表重算，结论由 " +
			before.ResultStatus + " 变更为 " + after.ResultStatus + "；历史版本保留"
	}
	return "正式复核：以完全相同的冻结证据与量表重算，结果与原始评分一致；历史版本保留"
}
