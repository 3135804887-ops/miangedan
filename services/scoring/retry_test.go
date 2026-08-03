package scoring

import (
	"context"
	"errors"
	"testing"
)

func beginRetryRequest(attemptID string) BeginRetryRequest {
	return BeginRetryRequest{
		ProjectID:      testProject,
		RoundSequence:  1,
		IdempotencyKey: "retry-idem-" + attemptID,
	}
}

func withEvidence(ids ...string) func(*CoverageAssessment) {
	return func(cp *CoverageAssessment) { cp.EvidenceIDs = ids }
}

// 发起正式重试：失败维度进入 rescope，≥60 维度锁定（SCORING-SPEC 6.7）。
func TestBeginRetryDerivesLockedAndRescope(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 2, 1, withScore(45,
			AnchorCitation{AnchorLevels: []int{2, 3}, EvidenceIDs: []string{"ev-p"}})),
		assessment(DimProblemSolving, AnswerAnswered, 3, 1),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(60, 60)),
		assessment(DimExperienceEvidence, AnswerAnswered, 3, 1),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1),
		assessment(DimLearningAdaptability, AnswerAnswered, 3, 1),
	}
	source := mustScore(t, svc, in)
	if source.ResultStatus != ResultFail {
		t.Fatalf("源结果应为 FAIL，实际 %s", source.ResultStatus)
	}
	attempt, err := svc.BeginRetry(context.Background(), testActor,
		beginRetryRequest(source.AttemptID))
	if err != nil {
		t.Fatalf("发起重试失败: %v", err)
	}
	if attempt.Status != RetryStatusScheduled {
		t.Fatalf("状态应为 RETRY_SCHEDULED，实际 %s", attempt.Status)
	}
	if attempt.SourceAttemptID != source.AttemptID {
		t.Fatalf("来源尝试应为 %s，实际 %s", source.AttemptID, attempt.SourceAttemptID)
	}
	if len(attempt.RescopeDimensions) != 1 ||
		attempt.RescopeDimensions[0] != DimProfessional {
		t.Fatalf("rescope 应为 professional：%v", attempt.RescopeDimensions)
	}
	foundLocked := map[DimensionKey]bool{}
	for _, d := range attempt.LockedDimensions {
		foundLocked[d] = true
	}
	if !foundLocked[DimProblemSolving] || !foundLocked[DimCommunication] {
		t.Fatalf("≥60 维度必须锁定：%v", attempt.LockedDimensions)
	}
	if foundLocked[DimProfessional] {
		t.Fatal("失败维度不得锁定")
	}
}

// 已通过轮次不可重试；无评分结果不可重试；幂等键去重。
func TestBeginRetryPreconditions(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	passed := mustScore(t, svc, in)
	if _, err := svc.BeginRetry(context.Background(), testActor,
		beginRetryRequest(passed.AttemptID)); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("PASS 轮次不可重试，实际 %v", err)
	}
	noResult := beginRetryRequest("00000000-0000-4000-8000-00000000a001")
	noResult.ProjectID = "00000000-0000-4000-8000-00000000ffff"
	if _, err := svc.BeginRetry(context.Background(), testActor, noResult); !errors.Is(err, ErrNotFound) {
		t.Fatalf("无评分结果不可重试，实际 %v", err)
	}
}

// 新题选择：不重复已通过相同问题；语义重复（措辞变化）命中丢弃；例外允许主题重验。
func TestSelectRetryQuestionsNoRepeat(t *testing.T) {
	svc2, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 2, 1, withScore(45,
			AnchorCitation{AnchorLevels: []int{2, 3}, EvidenceIDs: []string{"ev-p"}})),
		assessment(DimProblemSolving, AnswerAnswered, 3, 1),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(60, 60)),
		assessment(DimExperienceEvidence, AnswerAnswered, 3, 1),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1),
		assessment(DimLearningAdaptability, AnswerAnswered, 3, 1),
	}
	source := mustScore(t, svc2, in)
	retryAttempt, err := svc2.BeginRetry(context.Background(), testActor,
		beginRetryRequest(source.AttemptID))
	if err != nil {
		t.Fatalf("发起重试失败: %v", err)
	}
	selection, err := svc2.SelectRetryQuestions(context.Background(), testActor,
		SelectRetryQuestionsRequest{
			AttemptID: retryAttempt.AttemptID,
			CandidatePool: []string{
				"请介绍你在澄江云科实习期间负责的数据同步项目。",
				"请介绍你在澄江云科实习期间负责的数据同步项目（换个角度）。",
				"设计一个离线数仓分层方案。",
			},
			DoNotRepeat: []string{"请介绍你在澄江云科实习期间负责的数据同步项目。"},
		})
	if err != nil {
		t.Fatalf("选题失败: %v", err)
	}
	if len(selection.Selected) != 1 || selection.Selected[0] != "设计一个离线数仓分层方案。" {
		t.Fatalf("只应选中新题：%v", selection.Selected)
	}
	if len(selection.SkippedRepeats) != 2 {
		t.Fatalf("应丢弃 2 个重复题：%v", selection.SkippedRepeats)
	}
	// 例外：direct_contradiction 允许主题重验（仍不同措辞），相同措辞仍丢弃。
	exception, err := svc2.SelectRetryQuestions(context.Background(), testActor,
		SelectRetryQuestionsRequest{
			AttemptID: retryAttempt.AttemptID,
			CandidatePool: []string{
				"请介绍你在澄江云科实习期间负责的数据同步项目。",
				"再讲一次你在澄江云科的数据同步项目（新证据澄清）。",
			},
			DoNotRepeat:           []string{"请介绍你在澄江云科实习期间负责的数据同步项目。"},
			ReverificationReasons: []string{"direct_contradiction"},
		})
	if err != nil {
		t.Fatalf("例外选题失败: %v", err)
	}
	if len(exception.Selected) != 1 {
		t.Fatalf("例外应允许主题重验（不同措辞）：%v", exception.Selected)
	}
}

// 端到端：失败维度新分替换、锁定沿用、新证据引用进入结果、历史版本保留。
func TestScoreRetryEndToEnd(t *testing.T) {
	svc, store := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 2, 1, withScore(45,
			AnchorCitation{AnchorLevels: []int{2, 3}, EvidenceIDs: []string{"ev-old-p"}})),
		assessment(DimProblemSolving, AnswerAnswered, 3, 1),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(60, 60)),
		assessment(DimExperienceEvidence, AnswerAnswered, 3, 1),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1),
		assessment(DimLearningAdaptability, AnswerAnswered, 3, 1),
	}
	source := mustScore(t, svc, in)
	retryAttempt, err := svc.BeginRetry(context.Background(), testActor,
		beginRetryRequest(source.AttemptID))
	if err != nil {
		t.Fatalf("发起重试失败: %v", err)
	}
	retryIn := baseInput()
	retryIn.AttemptKind = AttemptFormalRetry
	retryIn.AttemptID = retryAttempt.AttemptID
	retryIn.ScoringRequestID = "req-retry-1"
	retryIn.IdempotencyKey = "idem-retry-1"
	retryIn.CriticalDimensions = []DimensionKey{DimProfessional, DimProblemSolving}
	retryIn.LockedDimensionScores = []LockedDimensionScore{
		{Dimension: DimProblemSolving, Score: 60, SourceAttemptID: source.AttemptID, SourceScoreVersion: 1},
		{Dimension: DimCommunication, Score: 60, SourceAttemptID: source.AttemptID, SourceScoreVersion: 1},
		{Dimension: DimExperienceEvidence, Score: 60, SourceAttemptID: source.AttemptID, SourceScoreVersion: 1},
		{Dimension: DimBehavioralCollaborate, Score: 60, SourceAttemptID: source.AttemptID, SourceScoreVersion: 1},
		{Dimension: DimLearningAdaptability, Score: 60, SourceAttemptID: source.AttemptID, SourceScoreVersion: 1},
	}
	retryIn.DimensionsToRescore = []DimensionKey{DimProfessional}
	retryIn.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 3, 1, withScore(70,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-new-p"}}),
			withEvidence("ev-new-p")),
	}
	result, err := svc.ScoreRetry(context.Background(), testActor, retryIn)
	if err != nil {
		t.Fatalf("重试评分失败: %v", err)
	}
	if result.Score.ResultStatus != ResultPass {
		t.Fatalf("重试后应 PASS，实际 %s", result.Score.ResultStatus)
	}
	if len(result.ReplacedDimensions) != 1 ||
		result.ReplacedDimensions[0] != DimProfessional {
		t.Fatalf("应替换 professional：%v", result.ReplacedDimensions)
	}
	byDim := map[DimensionKey]DimensionResult{}
	for _, dr := range result.Score.DimensionResults {
		byDim[dr.Dimension] = dr
	}
	if byDim[DimProfessional].Score == nil || *byDim[DimProfessional].Score != 70 {
		t.Fatalf("新分应为 70，实际 %v", byDim[DimProfessional].Score)
	}
	evidence := byDim[DimProfessional].EvidenceIDs
	if len(evidence) != 1 || evidence[0] != "ev-new-p" {
		t.Fatalf("重评维度必须携带新证据引用：%v", evidence)
	}
	if byDim[DimProblemSolving].ScoreStatus != StatusLockedCarried {
		t.Fatal("≥60 维度必须锁定沿用")
	}
	// 历史版本保留且不可改写。
	old, err := store.GetByIdempotencyKey("cn", in.IdempotencyKey)
	if err != nil || old.ResultStatus != ResultFail {
		t.Fatalf("历史版本必须保留（FAIL），err=%v", err)
	}
	attempt, err := store.GetRetryAttempt("cn", retryAttempt.AttemptID)
	if err != nil || attempt.Status != RetryStatusCompleted {
		t.Fatalf("重试状态应为 COMPLETED，实际 %s err=%v", attempt.Status, err)
	}
}

// 矛盾解锁：新证据直接否定锁定维度 → 解锁并用旧+新证据重评。
func TestScoreRetryContradictionUnlock(t *testing.T) {
	svc, store := newTestService(t)
	attempt := RetryAttempt{
		AttemptID:       "00000000-0000-4000-8000-00000000c001",
		ProjectID:       testProject,
		RoundSequence:   1,
		SourceAttemptID: "00000000-0000-4000-8000-00000000c000",
		Status:          RetryStatusScheduled,
		LockedDimensions: []DimensionKey{
			DimProfessional, DimCommunication,
			DimBehavioralCollaborate, DimLearningAdaptability,
		},
		RescopeDimensions: []DimensionKey{DimProblemSolving, DimExperienceEvidence},
		DataRegion:        "cn",
	}
	if err := store.SaveRetryAttempt(attempt, "idem-contra-begin"); err != nil {
		t.Fatalf("登记重试尝试失败: %v", err)
	}
	retryIn := baseInput()
	retryIn.AttemptKind = AttemptFormalRetry
	retryIn.AttemptID = attempt.AttemptID
	retryIn.ScoringRequestID = "req-retry-contra"
	retryIn.IdempotencyKey = "idem-retry-contra"
	retryIn.DimensionsToRescore = []DimensionKey{DimProblemSolving}
	retryIn.ContradictionUnlocks = []DimensionKey{DimExperienceEvidence}
	retryIn.LockedDimensionScores = []LockedDimensionScore{
		{Dimension: DimProfessional, Score: 80, SourceAttemptID: attempt.SourceAttemptID, SourceScoreVersion: 1},
		{Dimension: DimCommunication, Score: 60, SourceAttemptID: attempt.SourceAttemptID, SourceScoreVersion: 1},
		{Dimension: DimBehavioralCollaborate, Score: 60, SourceAttemptID: attempt.SourceAttemptID, SourceScoreVersion: 1},
		{Dimension: DimLearningAdaptability, Score: 60, SourceAttemptID: attempt.SourceAttemptID, SourceScoreVersion: 1},
	}
	retryIn.CoverageAssessments = []CoverageAssessment{
		// 旧+新证据一起重评（新证据为主要证据）。
		assessment(DimExperienceEvidence, AnswerAnswered, 2, 1, withScore(40,
			AnchorCitation{AnchorLevels: []int{2, 3}, EvidenceIDs: []string{"ev-old-x", "ev-new-x"}}),
			withEvidence("ev-old-x", "ev-new-x")),
		assessment(DimProblemSolving, AnswerAnswered, 3, 1, withScore(70,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-new-ps"}}),
			withEvidence("ev-new-ps")),
	}
	result, err := svc.ScoreRetry(context.Background(), testActor, retryIn)
	if err != nil {
		t.Fatalf("矛盾重评失败: %v", err)
	}
	byDim := map[DimensionKey]DimensionResult{}
	for _, dr := range result.Score.DimensionResults {
		byDim[dr.Dimension] = dr
	}
	if byDim[DimExperienceEvidence].ScoreStatus == StatusLockedCarried {
		t.Fatal("矛盾维度必须解锁重评")
	}
	if byDim[DimExperienceEvidence].Score == nil || *byDim[DimExperienceEvidence].Score != 40 {
		t.Fatalf("矛盾维度新分应为 40，实际 %v", byDim[DimExperienceEvidence].Score)
	}
	evidence := byDim[DimExperienceEvidence].EvidenceIDs
	if len(evidence) < 2 {
		t.Fatalf("旧+新证据必须一起重评：%v", evidence)
	}
}

// 未登记的尝试不能作为正式重试（练习/未知尝试隔离）。
func TestScoreRetryRejectsUnknownAttempt(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.AttemptKind = AttemptFormalRetry
	in.AttemptID = "00000000-0000-4000-8000-00000000p001"
	in.CoverageAssessments = voiceAssessments(3)
	if _, err := svc.ScoreRetry(context.Background(), testActor, in); err == nil {
		t.Fatal("未登记的重试尝试必须拒绝")
	}
}
