package scoring

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testProject = "00000000-0000-4000-8000-000000000001"
	testAttempt = "00000000-0000-4000-8000-00000000a001"
)

var testActor = Actor{UserID: "user-001", DataRegion: "cn"}

func newTestService(t *testing.T) (*Service, *MemoryStore) {
	t.Helper()
	store := NewMemoryStore()
	svc, err := NewService(store)
	if err != nil {
		t.Fatalf("创建评分服务失败: %v", err)
	}
	return svc, store
}

func defaultWeights() map[DimensionKey]int {
	return map[DimensionKey]int{
		DimProfessional:          25,
		DimProblemSolving:        20,
		DimCommunication:         15,
		DimExperienceEvidence:    15,
		DimBehavioralCollaborate: 15,
		DimLearningAdaptability:  10,
	}
}

func baseInput() Input {
	return Input{
		SchemaVersion:      "1.0.0",
		ScoringRequestID:   "req-001",
		IdempotencyKey:     "idem-001",
		ProjectID:          testProject,
		RoundSequence:      1,
		AttemptID:          testAttempt,
		AttemptKind:        AttemptInitial,
		DataRegion:         "cn",
		InterviewLanguage:  "zh-CN",
		RubricVersion:      "rubrics/v1/default",
		DimensionWeights:   defaultWeights(),
		CriticalDimensions: []DimensionKey{DimProfessional, DimProblemSolving},
		InputModeContext: InputModeContext{
			CommunicationMode: ModeVoice,
			ModesUsed:         []string{"voice"},
		},
		SubmittedAt: time.Now(),
	}
}

func intPtr(v int) *int { return &v }

// assessment 构造单个覆盖点评分判定。
func assessment(
	d DimensionKey, status AnswerStatus, level int, weight int, opts ...func(*CoverageAssessment),
) CoverageAssessment {
	cp := CoverageAssessment{
		CoverageID:        "cp-" + string(d) + "-" + strings.ToLower(string(status)),
		Dimension:         d,
		EvidenceIDs:       []string{"ev-" + string(d)},
		WeightInDimension: weight,
		AnswerStatus:      status,
		InputMode:         ModeVoice,
		AnchorLevel:       level,
	}
	for _, opt := range opts {
		opt(&cp)
	}
	return cp
}

func withScore(score int, citations ...AnchorCitation) func(*CoverageAssessment) {
	return func(cp *CoverageAssessment) {
		cp.InterpolatedScore = intPtr(score)
		cp.AnchorCitations = citations
	}
}

func withKeyTranscript() func(*CoverageAssessment) {
	return func(cp *CoverageAssessment) { cp.IsKeyTranscript = true }
}

func withCommunication(sc, od int) func(*CoverageAssessment) {
	return func(cp *CoverageAssessment) {
		cp.StructureClarity = intPtr(sc)
		cp.OralDelivery = intPtr(od)
	}
}

func voiceAssessments(level int) []CoverageAssessment {
	return []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, level, 1),
		assessment(DimProblemSolving, AnswerAnswered, level, 1),
		assessment(DimCommunication, AnswerAnswered, level, 1, withCommunication(60, 60)),
		assessment(DimExperienceEvidence, AnswerAnswered, level, 1),
		assessment(DimBehavioralCollaborate, AnswerAnswered, level, 1),
		assessment(DimLearningAdaptability, AnswerAnswered, level, 1),
	}
}

func mustScore(t *testing.T, svc *Service, in Input) Result {
	t.Helper()
	result, err := svc.Score(context.Background(), testActor, in)
	if err != nil {
		t.Fatalf("评分失败: %v", err)
	}
	return result
}

// SC-EC-01：总分恰好 60、全部关键维度 60 → PASS（60 ≥ 60）。
func TestSCEC01TotalExactly60Pass(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	result := mustScore(t, svc, in)
	if result.ResultStatus != ResultPass {
		t.Fatalf("应为 PASS，实际 %s", result.ResultStatus)
	}
	if result.RoundTotal == nil || *result.RoundTotal != PassLine {
		t.Fatalf("总分应为 60，实际 %v", result.RoundTotal)
	}
	if result.GateResult.TotalGatePassed == nil || !*result.GateResult.TotalGatePassed {
		t.Fatal("总分门槛应通过")
	}
}

// SC-EC-02：总分 72 但关键维度 experience_evidence=58 → FAIL（高总分不救）。
func TestSCEC02CriticalDimensionBelow60Fail(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CriticalDimensions = []DimensionKey{DimProfessional, DimExperienceEvidence}
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 4, 1),
		assessment(DimProblemSolving, AnswerAnswered, 4, 1),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(70, 70)),
		assessment(DimExperienceEvidence, AnswerAnswered, 2, 1, withScore(58,
			AnchorCitation{AnchorLevels: []int{2, 3}, EvidenceIDs: []string{"ev-exp"}})),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1, withScore(70,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-bc"}})),
		assessment(DimLearningAdaptability, AnswerAnswered, 3, 1),
	}
	result := mustScore(t, svc, in)
	if result.ResultStatus != ResultFail {
		t.Fatalf("应为 FAIL，实际 %s", result.ResultStatus)
	}
	if result.RoundTotal == nil || *result.RoundTotal != 72 {
		t.Fatalf("总分应为 72，实际 %v", *result.RoundTotal)
	}
	if len(result.GateResult.FailedCriticalDimensions) != 1 ||
		result.GateResult.FailedCriticalDimensions[0] != DimExperienceEvidence {
		t.Fatalf("失败关键维度应为 experience_evidence：%v",
			result.GateResult.FailedCriticalDimensions)
	}
}

// SC-EC-03：加权原始值 59.5 → 取整 60 参与门槛比较 → PASS。
func TestSCEC03RoundHalfUp595(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 3, 1),
		assessment(DimProblemSolving, AnswerAnswered, 3, 1),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(60, 60)),
		assessment(DimExperienceEvidence, AnswerAnswered, 3, 1),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1),
		assessment(DimLearningAdaptability, AnswerAnswered, 2, 1, withScore(55,
			AnchorCitation{AnchorLevels: []int{2, 3}, EvidenceIDs: []string{"ev-learning"}})), // 55 → 59.5
	}
	result := mustScore(t, svc, in)
	if result.RoundTotal == nil || *result.RoundTotal != 60 {
		t.Fatalf("59.5 应取整为 60，实际 %v", result.RoundTotal)
	}
	if result.ResultStatus != ResultPass {
		t.Fatalf("取整后 60 应 PASS，实际 %s", result.ResultStatus)
	}
}

// SC-EC-04：加权原始值 59.4 → 取整 59 → FAIL。
func TestSCEC04RoundHalfUp594(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 3, 1),
		assessment(DimProblemSolving, AnswerAnswered, 3, 1),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(60, 60)),
		assessment(DimExperienceEvidence, AnswerAnswered, 3, 1),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1),
		assessment(DimLearningAdaptability, AnswerAnswered, 2, 1), // 55 → 59.5
	}
	// 54 → 59.4（59.5 属于 SC-EC-03）。
	in.CoverageAssessments[5] = assessment(DimLearningAdaptability, AnswerAnswered, 2, 1,
		withScore(54, AnchorCitation{AnchorLevels: []int{2, 3}, EvidenceIDs: []string{"ev-learning"}}))
	result := mustScore(t, svc, in)
	if result.RoundTotal == nil || *result.RoundTotal != 59 {
		t.Fatalf("59.4 应取整为 59，实际 %v", result.RoundTotal)
	}
	if result.ResultStatus != ResultFail {
		t.Fatalf("59 < 60 应 FAIL，实际 %s", result.ResultStatus)
	}
}

// SC-EC-05：关键维度覆盖率 0.3（<0.5）→ EVALUATION_INCOMPLETE(insufficient_evidence)。
func TestSCEC05InsufficientEvidenceIncomplete(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CriticalDimensions = []DimensionKey{DimExperienceEvidence}
	assessments := voiceAssessments(3)
	var experience []CoverageAssessment
	for i := 0; i < 10; i++ {
		status := AnswerStatus(AnswerSkipped)
		if i < 3 {
			status = AnswerStatus(AnswerAnswered)
		}
		cp := assessment(DimExperienceEvidence, status, 3, 1)
		cp.CoverageID = "cp-exp-" + string(rune('a'+i))
		experience = append(experience, cp)
	}
	assessments = append(assessments, experience...)
	in.CoverageAssessments = assessments
	result := mustScore(t, svc, in)
	if result.ResultStatus != ResultEvaluationIncomplete {
		t.Fatalf("应为 EVALUATION_INCOMPLETE，实际 %s", result.ResultStatus)
	}
	if result.IncompleteReason == nil || *result.IncompleteReason != ReasonInsufficientEvidence {
		t.Fatalf("原因应为 insufficient_evidence，实际 %v", result.IncompleteReason)
	}
	if result.RoundTotal != nil {
		t.Fatalf("评估未完成不应有总分，实际 %v", *result.RoundTotal)
	}
}

// SC-EC-06：关键转写 unrecoverable → EVALUATION_INCOMPLETE(unrecoverable_transcript)。
func TestSCEC06UnrecoverableKeyTranscript(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerUnrecoverable, 0, 1, withKeyTranscript()),
		assessment(DimProblemSolving, AnswerAnswered, 3, 1),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(60, 60)),
		assessment(DimExperienceEvidence, AnswerAnswered, 3, 1),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1),
		assessment(DimLearningAdaptability, AnswerAnswered, 3, 1),
	}
	result := mustScore(t, svc, in)
	if result.ResultStatus != ResultEvaluationIncomplete {
		t.Fatalf("应为 EVALUATION_INCOMPLETE，实际 %s", result.ResultStatus)
	}
	if result.IncompleteReason == nil || *result.IncompleteReason != ReasonUnrecoverable {
		t.Fatalf("原因应为 unrecoverable_transcript，实际 %v", result.IncompleteReason)
	}
}

// SC-EC-07：非关键维度 learning_adaptability=45 → PASS + weak_dimensions。
func TestSCEC07NonCriticalWeakDimension(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 4, 1),
		assessment(DimProblemSolving, AnswerAnswered, 4, 1),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(70, 70)),
		assessment(DimExperienceEvidence, AnswerAnswered, 4, 1),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1, withScore(70,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-bc"}})),
		assessment(DimLearningAdaptability, AnswerAnswered, 2, 1, withScore(45,
			AnchorCitation{AnchorLevels: []int{2, 3}, EvidenceIDs: []string{"ev-learning"}})),
	}
	result := mustScore(t, svc, in)
	if result.ResultStatus != ResultPass {
		t.Fatalf("非关键弱项不应导致失败，实际 %s", result.ResultStatus)
	}
	found := false
	for _, d := range result.GateResult.WeakDimensions {
		if d == DimLearningAdaptability {
			found = true
		}
	}
	if !found {
		t.Fatalf("weak_dimensions 应包含 learning_adaptability：%v",
			result.GateResult.WeakDimensions)
	}
}

// SC-EC-08：非关键维度 uncovered → 按其余五维重新归一化计算。
func TestSCEC08UncoveredNonCriticalRenormalize(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 3, 1, withScore(70,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-p"}})),
		assessment(DimProblemSolving, AnswerAnswered, 3, 1, withScore(70,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-ps"}})),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(70, 70)),
		assessment(DimExperienceEvidence, AnswerAnswered, 3, 1, withScore(70,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-exp"}})),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1, withScore(70,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-bc"}})),
		// learning_adaptability 无任何覆盖点 → uncovered
	}
	result := mustScore(t, svc, in)
	if result.ResultStatus != ResultPass {
		t.Fatalf("五维达标应 PASS，实际 %s", result.ResultStatus)
	}
	if result.RoundTotal == nil || *result.RoundTotal != 70 {
		t.Fatalf("按五维归一化应为 70，实际 %v", *result.RoundTotal)
	}
	found := false
	for _, d := range result.GateResult.UncoveredDimensions {
		if d == DimLearningAdaptability {
			found = true
		}
	}
	if !found {
		t.Fatalf("uncovered_dimensions 应包含 learning_adaptability：%v",
			result.GateResult.UncoveredDimensions)
	}
}

// SC-EC-13：正式重试锁定沿用 + 失败维度新分替换。
func TestSCEC13RetryLockedCarryAndRescore(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.AttemptKind = AttemptFormalRetry
	in.LockedDimensionScores = []LockedDimensionScore{
		{Dimension: DimProblemSolving, Score: 68, SourceAttemptID: "prev-attempt", SourceScoreVersion: 1},
		{Dimension: DimCommunication, Score: 70, SourceAttemptID: "prev-attempt", SourceScoreVersion: 1},
		{Dimension: DimExperienceEvidence, Score: 66, SourceAttemptID: "prev-attempt", SourceScoreVersion: 1},
		{Dimension: DimBehavioralCollaborate, Score: 64, SourceAttemptID: "prev-attempt", SourceScoreVersion: 1},
		{Dimension: DimLearningAdaptability, Score: 62, SourceAttemptID: "prev-attempt", SourceScoreVersion: 1},
	}
	in.DimensionsToRescore = []DimensionKey{DimProfessional}
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 3, 1, withScore(70,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-new"}})),
	}
	result := mustScore(t, svc, in)
	if result.ResultStatus != ResultPass {
		t.Fatalf("重试新分 70 应通过，实际 %s", result.ResultStatus)
	}
	byDim := map[DimensionKey]DimensionResult{}
	for _, dr := range result.DimensionResults {
		byDim[dr.Dimension] = dr
	}
	if byDim[DimProfessional].Score == nil || *byDim[DimProfessional].Score != 70 {
		t.Fatalf("professional 新分应为 70，实际 %v", byDim[DimProfessional].Score)
	}
	if byDim[DimProblemSolving].ScoreStatus != StatusLockedCarried ||
		byDim[DimProblemSolving].Score == nil || *byDim[DimProblemSolving].Score != 68 {
		t.Fatal("problem_solving 应锁定沿用 68")
	}
	if result.RoundTotal == nil || *result.RoundTotal != 67 {
		t.Fatalf("按锁定+新分重新加权应为 67，实际 %v", result.RoundTotal)
	}
}

// SC-EC-14：新证据直接否定锁定维度 → 解锁重评（新证据为主要证据）。
func TestSCEC14ContradictionUnlocksDimension(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.AttemptKind = AttemptFormalRetry
	in.LockedDimensionScores = []LockedDimensionScore{
		{Dimension: DimProfessional, Score: 75, SourceAttemptID: "prev-attempt", SourceScoreVersion: 1},
		{Dimension: DimProblemSolving, Score: 68, SourceAttemptID: "prev-attempt", SourceScoreVersion: 1},
		{Dimension: DimCommunication, Score: 70, SourceAttemptID: "prev-attempt", SourceScoreVersion: 1},
		{Dimension: DimExperienceEvidence, Score: 72, SourceAttemptID: "prev-attempt", SourceScoreVersion: 1},
		{Dimension: DimBehavioralCollaborate, Score: 64, SourceAttemptID: "prev-attempt", SourceScoreVersion: 1},
		{Dimension: DimLearningAdaptability, Score: 62, SourceAttemptID: "prev-attempt", SourceScoreVersion: 1},
	}
	in.DimensionsToRescore = []DimensionKey{DimCommunication}
	in.ContradictionUnlocks = []DimensionKey{DimExperienceEvidence}
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(70, 70)),
		// 旧+新证据一起重评（新证据为主要证据）：锚点 2 → 40。
		assessment(DimExperienceEvidence, AnswerAnswered, 2, 1, withScore(40)),
	}
	result := mustScore(t, svc, in)
	byDim := map[DimensionKey]DimensionResult{}
	for _, dr := range result.DimensionResults {
		byDim[dr.Dimension] = dr
	}
	if byDim[DimExperienceEvidence].ScoreStatus == StatusLockedCarried {
		t.Fatal("矛盾维度必须解锁重评")
	}
	if byDim[DimExperienceEvidence].Score == nil || *byDim[DimExperienceEvidence].Score != 40 {
		t.Fatalf("矛盾维度新分应为 40，实际 %v", byDim[DimExperienceEvidence].Score)
	}
	if byDim[DimProfessional].ScoreStatus != StatusLockedCarried {
		t.Fatal("无矛盾的锁定维度应沿用")
	}
}

// SC-EC-20：插值 70 未引用锚点与证据 → 回退下锚点 60。
func TestSCEC20InterpolationRequiresCitations(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 3, 1, withScore(70)),
		assessment(DimProblemSolving, AnswerAnswered, 3, 1),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(60, 60)),
		assessment(DimExperienceEvidence, AnswerAnswered, 3, 1),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1),
		assessment(DimLearningAdaptability, AnswerAnswered, 3, 1),
	}
	result := mustScore(t, svc, in)
	professional := result.DimensionResults[0]
	if professional.Score == nil || *professional.Score != 60 {
		t.Fatalf("缺少引用应回退 60，实际 %v", professional.Score)
	}
}

// SC-EC-24：同一 idempotency_key 重复提交返回首个结果，不产生新版本。
func TestSCEC24IdempotentScore(t *testing.T) {
	svc, store := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	first := mustScore(t, svc, in)
	second := mustScore(t, svc, in)
	if first.ScoreID != second.ScoreID || first.ScoreVersion != second.ScoreVersion {
		t.Fatalf("幂等应返回首个结果：%s/%d vs %s/%d",
			first.ScoreID, first.ScoreVersion, second.ScoreID, second.ScoreVersion)
	}
	items, _, err := store.ListVersions("cn", testProject, 1, 20, "")
	if err != nil {
		t.Fatalf("列出版本失败: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("重复提交不应产生新 ScoreVersion，实际 %d 个", len(items))
	}
}

// 正常路径：GET 最新结果与分页版本列表。
func TestGetLatestAndListVersions(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	mustScore(t, svc, in)
	in2 := in
	in2.ScoringRequestID = "req-002"
	in2.IdempotencyKey = "idem-002"
	in2.AttemptID = "00000000-0000-4000-8000-00000000a002"
	mustScore(t, svc, in2)
	latest, err := svc.GetLatest(context.Background(), testActor, testProject, 1)
	if err != nil {
		t.Fatalf("获取最新结果失败: %v", err)
	}
	if latest.AttemptID != in2.AttemptID {
		t.Fatalf("最新结果应为后提交的尝试：%s", latest.AttemptID)
	}
	items, next, err := svc.ListVersions(context.Background(), testActor, testProject, 1, 1, "")
	if err != nil {
		t.Fatalf("分页失败: %v", err)
	}
	if len(items) != 1 || next == "" {
		t.Fatalf("分页异常：items=%d next=%q", len(items), next)
	}
}

// 异常路径：非法输入全部拒绝。
func TestInvalidInputsRejected(t *testing.T) {
	svc, _ := newTestService(t)
	cases := map[string]Input{
		"缺项目":  func() Input { in := baseInput(); in.ProjectID = ""; return in }(),
		"非法区域": func() Input { in := baseInput(); in.DataRegion = "xx"; return in }(),
		"权重和不为100": func() Input {
			in := baseInput()
			in.DimensionWeights[DimProfessional] = 30
			return in
		}(),
		"非法锚点": func() Input {
			in := baseInput()
			in.CoverageAssessments = []CoverageAssessment{assessment(DimProfessional, AnswerAnswered, 9, 1)}
			return in
		}(),
		"文字模式未实现": func() Input {
			in := baseInput()
			in.InputModeContext.CommunicationMode = ModeText
			return in
		}(),
		"正式复核未实现": func() Input {
			in := baseInput()
			in.IsFormalReview = true
			return in
		}(),
		"重试缺范围": func() Input {
			in := baseInput()
			in.AttemptKind = AttemptFormalRetry
			return in
		}(),
	}
	for name, in := range cases {
		if _, err := svc.Score(context.Background(), testActor, in); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// SC-EC-18：评分服务持久化故障 → EVALUATION_INCOMPLETE(scoring_service_failure)。
func TestSCEC18ScoringServiceFaultIncomplete(t *testing.T) {
	store := NewMemoryStore()
	svc, err := NewService(&faultStore{MemoryStore: store, failSave: true})
	if err != nil {
		t.Fatalf("创建服务失败: %v", err)
	}
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	result := mustScore(t, svc, in)
	if result.ResultStatus != ResultEvaluationIncomplete {
		t.Fatalf("服务故障应判评估未完成，实际 %s", result.ResultStatus)
	}
	if result.IncompleteReason == nil || *result.IncompleteReason != ReasonScoringServiceFailure {
		t.Fatalf("原因应为 scoring_service_failure，实际 %v", result.IncompleteReason)
	}
	if _, err := store.GetLatest("cn", testProject, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("故障结果不应落库，实际 err=%v", err)
	}
}

// 故障恢复：恢复后可重算（同一幂等键）。
func TestScoringFaultRecoveryRecalculates(t *testing.T) {
	store := NewMemoryStore()
	faulty := &faultStore{MemoryStore: store, failSave: true}
	svc, _ := NewService(faulty)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	result := mustScore(t, svc, in)
	if result.ResultStatus != ResultEvaluationIncomplete {
		t.Fatalf("故障应未完成，实际 %s", result.ResultStatus)
	}
	faulty.failSave = false
	recovered := mustScore(t, svc, in)
	if recovered.ResultStatus != ResultPass {
		t.Fatalf("恢复后应正常重算，实际 %s", recovered.ResultStatus)
	}
}

// 摄像头/便利设置不影响计算（TASK-040 基线；SC-EC-11/12 详细用例见 TASK-041）。
func TestCameraAndAccommodationsNoEffect(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	baseline := mustScore(t, svc, in)
	in2 := baseInput()
	in2.IdempotencyKey = "idem-camera"
	in2.ScoringRequestID = "req-camera"
	in2.AttemptID = "00000000-0000-4000-8000-00000000a003"
	in2.InputModeContext.ModesUsed = []string{"voice", "camera"}
	in2.InputModeContext.AccommodationsInEffect = []string{"extended_time", "no_proactive_interruption"}
	in2.CoverageAssessments = voiceAssessments(3)
	withCamera := mustScore(t, svc, in2)
	if *baseline.RoundTotal != *withCamera.RoundTotal {
		t.Fatalf("摄像头与便利设置不得影响分数：%d vs %d",
			*baseline.RoundTotal, *withCamera.RoundTotal)
	}
}

// faultStore 模拟持久化故障（SC-EC-18）。
type faultStore struct {
	*MemoryStore
	failSave bool
}

func (f *faultStore) SaveResult(r Result, key string) error {
	if f.failSave {
		return errors.New("persistence unavailable")
	}
	return f.MemoryStore.SaveResult(r, key)
}
