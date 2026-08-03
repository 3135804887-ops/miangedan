package scoring

import (
	"context"
	"errors"
	"testing"
)

func reviewRequest(attemptID string) ReviewRequest {
	return ReviewRequest{
		ProjectID:      testProject,
		RoundSequence:  1,
		AttemptID:      attemptID,
		Scope:          ReviewScopeRound,
		RequestID:      "req-review",
		IdempotencyKey: "review-idem-" + attemptID,
	}
}

// 通过真实评分流程：初始评分 → 正式复核（结果一致，产生新版本）。
func TestReviewAfterInitialScore(t *testing.T) {
	svc, store := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	original := mustScore(t, svc, in)
	result, err := svc.Review(context.Background(), testActor, reviewRequest(in.AttemptID))
	if err != nil {
		t.Fatalf("复核失败: %v", err)
	}
	if result.Review.ScoreVersion != original.ScoreVersion+1 {
		t.Fatalf("复核版本应为 %d，实际 %d", original.ScoreVersion+1, result.Review.ScoreVersion)
	}
	if result.Review.VersionLineage.SupersedesScoreID == nil ||
		*result.Review.VersionLineage.SupersedesScoreID != original.ScoreID {
		t.Fatalf("supersedes_score_id 应指向原版本：%v",
			result.Review.VersionLineage.SupersedesScoreID)
	}
	if len(result.Changes) != len(DimensionKeys) {
		t.Fatalf("前后对比应覆盖六维，实际 %d 维", len(result.Changes))
	}
	for _, change := range result.Changes {
		if change.BeforeScore == nil || change.AfterScore == nil ||
			*change.BeforeScore != *change.AfterScore {
			t.Fatalf("相同冻结输入重算应一致：%+v", change)
		}
	}
	items, _, err := store.ListVersions("cn", testProject, 1, 20, "")
	if err != nil || len(items) != 2 {
		t.Fatalf("复核后应保留两个版本：len=%d err=%v", len(items), err)
	}
}

// SC-EC-16：重算后关键维度 59 → 61；新 ScoreVersion、原版本保留、前后对比与原因。
func TestSCEC16ReviewChangesScoreAndKeepsHistory(t *testing.T) {
	svc, store := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 3, 1, withScore(61,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-p"}})),
		assessment(DimProblemSolving, AnswerAnswered, 3, 1),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(60, 60)),
		assessment(DimExperienceEvidence, AnswerAnswered, 3, 1),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1),
		assessment(DimLearningAdaptability, AnswerAnswered, 3, 1),
	}
	computed, err := svc.compute(in)
	if err != nil {
		t.Fatalf("预计算失败: %v", err)
	}
	original := computed
	original.ScoreID = "00000000-0000-4000-8000-00000000f001"
	original.ScoreVersion = 1
	original.VersionLineage.ScorerVersion = "scoring/v0.9"
	total59 := 59
	original.RoundTotal = &total59
	original.ResultStatus = ResultFail
	for i := range original.DimensionResults {
		if original.DimensionResults[i].Dimension == DimProfessional {
			score59 := 59
			original.DimensionResults[i].Score = &score59
		}
	}
	if err := store.SaveResult(original, "orig-idem-1"); err != nil {
		t.Fatalf("保存原始版本失败: %v", err)
	}
	if err := store.SaveInput("cn", original.ScoreID, in); err != nil {
		t.Fatalf("保存冻结输入失败: %v", err)
	}
	result, err := svc.Review(context.Background(), testActor, reviewRequest(in.AttemptID))
	if err != nil {
		t.Fatalf("复核失败: %v", err)
	}
	if result.Review.ResultStatus != ResultPass {
		t.Fatalf("重算后 61 应达标 PASS，实际 %s", result.Review.ResultStatus)
	}
	found := false
	for _, change := range result.Changes {
		if change.Dimension == DimProfessional {
			found = true
			if change.BeforeScore == nil || *change.BeforeScore != 59 ||
				change.AfterScore == nil || *change.AfterScore != 61 {
				t.Fatalf("professional 应 59 → 61，实际 %+v", change)
			}
		}
	}
	if !found {
		t.Fatal("前后对比缺少 professional 维度")
	}
	if result.Review.VersionLineage.SupersedesScoreID == nil ||
		*result.Review.VersionLineage.SupersedesScoreID != original.ScoreID {
		t.Fatal("复核版本必须 supersede 原版本")
	}
	if result.Reason == "" {
		t.Fatal("复核必须给出原因")
	}
	// 历史版本保留且不可改写。
	storedOriginal, err := store.GetByIdempotencyKey("cn", "orig-idem-1")
	if err != nil {
		t.Fatalf("读取历史版本失败: %v", err)
	}
	if storedOriginal.ResultStatus != ResultFail ||
		storedOriginal.VersionLineage.ScorerVersion != "scoring/v0.9" {
		t.Fatal("历史版本不得被复核改写")
	}
	items, _, err := store.ListVersions("cn", testProject, 1, 20, "")
	if err != nil || len(items) != 2 {
		t.Fatalf("复核后应保留两个版本：len=%d err=%v", len(items), err)
	}
}

// SC-EC-17：同一正式尝试第二次复核必须被拒绝。
func TestSCEC17SecondReviewRejected(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	mustScore(t, svc, in)
	if _, err := svc.Review(context.Background(), testActor, reviewRequest(in.AttemptID)); err != nil {
		t.Fatalf("首次复核失败: %v", err)
	}
	second := reviewRequest(in.AttemptID)
	second.IdempotencyKey = "review-idem-second"
	if _, err := svc.Review(context.Background(), testActor, second); !errors.Is(err, ErrReviewLimit) {
		t.Fatalf("第二次复核应拒绝（ErrReviewLimit），实际 %v", err)
	}
}

// 幂等：同一复核幂等键重复提交返回首个结果，不产生新版本、不重复计数。
func TestReviewIdempotent(t *testing.T) {
	svc, store := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	mustScore(t, svc, in)
	req := reviewRequest(in.AttemptID)
	first, err := svc.Review(context.Background(), testActor, req)
	if err != nil {
		t.Fatalf("首次复核失败: %v", err)
	}
	second, err := svc.Review(context.Background(), testActor, req)
	if err != nil {
		t.Fatalf("幂等复核失败: %v", err)
	}
	if second.Review.ScoreID != first.Review.ScoreID {
		t.Fatalf("幂等应返回首个复核结果")
	}
	count, err := store.CountReviews("cn", in.AttemptID)
	if err != nil || count != 1 {
		t.Fatalf("复核计数应为 1，实际 %d err=%v", count, err)
	}
	items, _, err := store.ListVersions("cn", testProject, 1, 20, "")
	if err != nil || len(items) != 2 {
		t.Fatalf("幂等复核不应产生新版本：len=%d err=%v", len(items), err)
	}
}

// 异常：无原始评分、非法 scope、区域不一致、证据散列不一致必须拒绝。
func TestReviewInvalidRequests(t *testing.T) {
	svc, store := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	mustScore(t, svc, in)
	cases := map[string]func() error{
		"无原始评分": func() error {
			_, err := svc.Review(context.Background(), testActor,
				reviewRequest("00000000-0000-4000-8000-00000000ffff"))
			return err
		},
		"非法scope": func() error {
			req := reviewRequest(in.AttemptID)
			req.Scope = "all"
			_, err := svc.Review(context.Background(), testActor, req)
			return err
		},
		"区域不一致": func() error {
			_, err := svc.Review(context.Background(),
				Actor{UserID: "user-001", DataRegion: "intl"}, reviewRequest(in.AttemptID))
			return err
		},
		"证据散列不一致": func() error {
			tamperedAttempt := "00000000-0000-4000-8000-00000000t001"
			orig := Result{
				ScoreID:       "00000000-0000-4000-8000-00000000f002",
				ProjectID:     testProject,
				RoundSequence: 1,
				AttemptID:     tamperedAttempt,
				DataRegion:    "cn",
				ScoreVersion:  1,
				ResultStatus:  ResultPass,
				VersionLineage: VersionLineage{
					ScorerVersion:        ScorerVersion,
					EvidenceSnapshotHash: "deadbeef",
				},
			}
			if err := store.SaveResult(orig, "tampered-idem"); err != nil {
				return err
			}
			if err := store.SaveInput("cn", orig.ScoreID, in); err != nil {
				return err
			}
			_, err := svc.Review(context.Background(), testActor, reviewRequest(tamperedAttempt))
			return err
		},
	}
	for name, run := range cases {
		if err := run(); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// 复核不改变业务状态：只产出新版本，锁定/解锁由工作流据 result_status 处理。
func TestReviewDoesNotMutateStoreOutsideVersions(t *testing.T) {
	svc, store := newTestService(t)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	mustScore(t, svc, in)
	if _, err := svc.Review(context.Background(), testActor, reviewRequest(in.AttemptID)); err != nil {
		t.Fatalf("复核失败: %v", err)
	}
	reviews, err := store.CountReviews("cn", in.AttemptID)
	if err != nil || reviews != 1 {
		t.Fatalf("复核计数应为 1，实际 %d", reviews)
	}
}
