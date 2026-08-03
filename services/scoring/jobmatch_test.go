package scoring

import (
	"context"
	"testing"
)

func jobMatchInput() *JobMatchInput {
	return &JobMatchInput{
		Requirements: []JobRequirement{
			{RequirementID: "r1", Bucket: BucketMustHave, Weight: 5},
			{RequirementID: "r2", Bucket: BucketMustHave, Weight: 5},
			{RequirementID: "n1", Bucket: BucketNiceToHave, Weight: 2},
			{RequirementID: "n2", Bucket: BucketNiceToHave, Weight: 2},
		},
		ResumeAvailable:   true,
		ProvenByResume:    []string{"r1"},
		ProvenByInterview: []string{"r2", "n1"},
	}
}

// 必备/加分分列：must_have = 10/10 = 1.0；nice_to_have = 2/4 = 0.5。
func TestJobMatchBucketsSeparate(t *testing.T) {
	jobMatch, err := ComputeJobMatch(jobMatchInput())
	if err != nil {
		t.Fatalf("计算失败: %v", err)
	}
	if jobMatch.MustHave.MatchRatio != 1.0 {
		t.Fatalf("must_have 应为 1.0，实际 %v", jobMatch.MustHave.MatchRatio)
	}
	if jobMatch.NiceToHave.MatchRatio != 0.5 {
		t.Fatalf("nice_to_have 应为 0.5，实际 %v", jobMatch.NiceToHave.MatchRatio)
	}
	if len(jobMatch.MustHave.Proven) != 2 || len(jobMatch.MustHave.Unproven) != 0 {
		t.Fatalf("must_have proven/unproven 异常：%+v", jobMatch.MustHave)
	}
	if len(jobMatch.NiceToHave.Proven) != 1 || len(jobMatch.NiceToHave.Unproven) != 1 {
		t.Fatalf("nice_to_have proven/unproven 异常：%+v", jobMatch.NiceToHave)
	}
	if jobMatch.NotDisplayedReason != nil {
		t.Fatalf("有 JD 时不应 not_displayed：%v", *jobMatch.NotDisplayedReason)
	}
}

// 权重参与比例：r1 权重 3 已证明 / r2 权重 7 未证明 → 0.3。
func TestJobMatchWeightedRatio(t *testing.T) {
	in := &JobMatchInput{
		Requirements: []JobRequirement{
			{RequirementID: "r1", Bucket: BucketMustHave, Weight: 3},
			{RequirementID: "r2", Bucket: BucketMustHave, Weight: 7},
		},
		ResumeAvailable: true,
		ProvenByResume:  []string{"r1"},
	}
	jobMatch, err := ComputeJobMatch(in)
	if err != nil {
		t.Fatalf("计算失败: %v", err)
	}
	if jobMatch.MustHave.MatchRatio != 0.3 {
		t.Fatalf("加权比例应为 0.3，实际 %v", jobMatch.MustHave.MatchRatio)
	}
	if len(jobMatch.MustHave.Unproven) != 1 || jobMatch.MustHave.Unproven[0] != "r2" {
		t.Fatalf("未证明清单异常：%v", jobMatch.MustHave.Unproven)
	}
}

// SC-EC-22：无 JD → not_displayed_reason = no_jd，不展示匹配百分比。
func TestNoJDNotDisplayed(t *testing.T) {
	jobMatch, err := ComputeJobMatch(&JobMatchInput{Requirements: []JobRequirement{}})
	if err != nil {
		t.Fatalf("计算失败: %v", err)
	}
	if jobMatch.NotDisplayedReason == nil || *jobMatch.NotDisplayedReason != ReasonNoJD {
		t.Fatalf("无 JD 应 not_displayed=no_jd，实际 %v", jobMatch.NotDisplayedReason)
	}
	if jobMatch.MustHave.MatchRatio != 0 || jobMatch.NiceToHave.MatchRatio != 0 {
		t.Fatalf("无 JD 不展示百分比：%+v", jobMatch)
	}
}

// SC-EC-21：JD-only 模式只按面试证明计算；无简历时禁止简历证明与经历一致性评分。
func TestJDOnlyNoResumeConsistency(t *testing.T) {
	// 计划阶段已把 experience_evidence 权重重新分配为 0。
	weights := defaultWeights()
	weights[DimExperienceEvidence] = 0
	weights[DimProfessional] = 40
	in := baseInput()
	in.DimensionWeights = weights
	in.JobMatchInput = &JobMatchInput{
		Requirements: []JobRequirement{
			{RequirementID: "r1", Bucket: BucketMustHave, Weight: 4},
			{RequirementID: "r2", Bucket: BucketMustHave, Weight: 6},
		},
		ResumeAvailable:   false,
		ProvenByInterview: []string{"r1"},
	}
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 3, 1),
		assessment(DimProblemSolving, AnswerAnswered, 3, 1),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(60, 60)),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1),
		assessment(DimLearningAdaptability, AnswerAnswered, 3, 1),
	}
	svc, _ := newTestService(t)
	result := mustScore(t, svc, in)
	if result.JobMatch == nil || result.JobMatch.MustHave.MatchRatio != 0.4 {
		t.Fatalf("JD-only 应按面试证明计算 0.4，实际 %+v", result.JobMatch)
	}
	// 无简历但携带简历证明 → 拒绝。
	bad := *in.JobMatchInput
	bad.ProvenByResume = []string{"r1"}
	badInput := in
	badInput.JobMatchInput = &bad
	if _, err := svc.Score(context.Background(), testActor, badInput); err == nil {
		t.Fatal("无简历时简历证明必须拒绝")
	}
	// 无简历且 experience_evidence 权重 >0 → 拒绝（经历一致性评分禁止）。
	badWeights := defaultWeights()
	badInput2 := in
	badInput2.DimensionWeights = badWeights
	if _, err := svc.Score(context.Background(), testActor, badInput2); err == nil {
		t.Fatal("JD-only 模式 experience_evidence 权重必须为 0")
	}
}

// 匹配度不参与解锁：匹配度为 0 也不影响 PASS/FAIL 判定。
func TestJobMatchNotUnlockFactor(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.JobMatchInput = &JobMatchInput{
		Requirements: []JobRequirement{
			{RequirementID: "r1", Bucket: BucketMustHave, Weight: 1},
		},
		ResumeAvailable: true,
	}
	in.CoverageAssessments = voiceAssessments(3)
	result := mustScore(t, svc, in)
	if result.ResultStatus != ResultPass {
		t.Fatalf("匹配度不得影响通过判定，实际 %s", result.ResultStatus)
	}
	if result.JobMatch == nil || result.JobMatch.MustHave.MatchRatio != 0 {
		t.Fatalf("匹配度应为 0 且独立展示：%+v", result.JobMatch)
	}
}

// 异常路径：非法分列、重复要求、未注册证明引用必须拒绝。
func TestJobMatchInvalidInputs(t *testing.T) {
	cases := map[string]*JobMatchInput{
		"非法分列": {
			Requirements: []JobRequirement{{RequirementID: "r1", Bucket: "other", Weight: 1}},
		},
		"重复要求": {
			Requirements: []JobRequirement{
				{RequirementID: "r1", Bucket: BucketMustHave, Weight: 1},
				{RequirementID: "r1", Bucket: BucketMustHave, Weight: 1},
			},
		},
		"未注册证明": {
			Requirements:    []JobRequirement{{RequirementID: "r1", Bucket: BucketMustHave, Weight: 1}},
			ResumeAvailable: true,
			ProvenByResume:  []string{"ghost"},
		},
	}
	for name, in := range cases {
		if _, err := ComputeJobMatch(in); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// 幂等：同一键重复评分返回相同岗位匹配度（结果整体缓存）。
func TestJobMatchIdempotent(t *testing.T) {
	svc, store := newTestService(t)
	in := baseInput()
	in.JobMatchInput = jobMatchInput()
	in.CoverageAssessments = voiceAssessments(3)
	first := mustScore(t, svc, in)
	second := mustScore(t, svc, in)
	if first.JobMatch == nil || second.JobMatch == nil {
		t.Fatal("两个结果都必须包含岗位匹配度")
	}
	if first.JobMatch.MustHave.MatchRatio != second.JobMatch.MustHave.MatchRatio {
		t.Fatalf("幂等结果岗位匹配度必须一致")
	}
	items, _, err := store.ListVersions("cn", testProject, 1, 20, "")
	if err != nil || len(items) != 1 {
		t.Fatalf("幂等不应产生新版本：len=%d err=%v", len(items), err)
	}
}
