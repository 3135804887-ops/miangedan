package scoring

import (
	"encoding/json"
	"testing"
)

func planRounds(sequences ...int) []PlanRoundSnapshot {
	out := make([]PlanRoundSnapshot, 0, len(sequences))
	for _, seq := range sequences {
		out = append(out, PlanRoundSnapshot{
			Sequence:        seq,
			Role:            "面试官",
			Focus:           "重点",
			Difficulty:      "standard",
			DurationMinutes: 30,
		})
	}
	return out
}

func flowResult(seq int, status string, total *int, reason string, weak []DimensionKey) Result {
	dims := []DimensionResult{
		{Dimension: DimProfessional, ScoreStatus: StatusScored, Score: intPtr(72), IsCritical: true},
		{Dimension: DimProblemSolving, ScoreStatus: StatusScored, Score: intPtr(70), IsCritical: true},
		{Dimension: DimCommunication, ScoreStatus: StatusScored, Score: intPtr(65), IsCritical: false},
	}
	gate := GateResult{WeakDimensions: weak}
	if status == ResultFail {
		gate.FailedCriticalDimensions = []DimensionKey{DimProblemSolving}
	}
	reasonPtr := reason
	return Result{
		ProjectID:        testProject,
		RoundSequence:    seq,
		ResultStatus:     status,
		RoundTotal:       total,
		DimensionResults: dims,
		GateResult:       gate,
		IncompleteReason: &reasonPtr,
		Explanations: Explanations{
			Summary:      "摘要",
			Strengths:    []string{"优势一"},
			Improvements: []string{"改进一"},
		},
	}
}

func flowResultNoReason(seq int, status string, total *int) Result {
	r := flowResult(seq, status, total, "", nil)
	r.IncompleteReason = nil
	return r
}

// 通过态：固定祝贺文案、下一轮预告、进度与优势；不展示失败阻断。
func TestFlowPassView(t *testing.T) {
	view, err := BuildRoundResultView(
		testProject,
		flowResultNoReason(1, ResultPass, intPtr(70)),
		nil,
		planRounds(1, 2, 3),
	)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	if view.CongratulationText == nil || *view.CongratulationText != CongratulationCopy {
		t.Fatalf("通过态必须展示固定祝贺文案：%v", view.CongratulationText)
	}
	if view.NextRound == nil || view.NextRound.Sequence != 2 {
		t.Fatalf("应有下一轮预告：%+v", view.NextRound)
	}
	if view.Progress.TotalRounds != 3 || view.Progress.CurrentRound != 1 {
		t.Fatalf("进度异常：%+v", view.Progress)
	}
	if view.FailureBlocked {
		t.Fatal("通过态不得阻断")
	}
	if len(view.CriticalDimensions) != 2 {
		t.Fatalf("关键维度展示应为 2 个：%v", view.CriticalDimensions)
	}
	// 最后一轮通过时无下一轮预告。
	last, err := BuildRoundResultView(
		testProject,
		flowResultNoReason(3, ResultPass, intPtr(70)),
		nil,
		planRounds(1, 2, 3),
	)
	if err != nil || last.NextRound != nil {
		t.Fatalf("最后一轮不应有下一轮预告：%+v err=%v", last.NextRound, err)
	}
}

// 未通过态：阻断下一轮、累计纪要（含前序轮次）、训练入口（复盘/重试/报告/复核）。
func TestFlowFailViewBlocksAndCumulative(t *testing.T) {
	prior := []Result{
		flowResultNoReason(1, ResultPass, intPtr(75)),
		flowResultNoReason(2, ResultFail, intPtr(58)),
	}
	view, err := BuildRoundResultView(
		testProject,
		flowResultNoReason(2, ResultFail, intPtr(58)),
		prior[:1],
		planRounds(1, 2, 3),
	)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	if !view.FailureBlocked {
		t.Fatal("失败态必须阻断下一轮")
	}
	if view.CongratulationText != nil {
		t.Fatal("失败态不得出现祝贺文案")
	}
	if len(view.CumulativeReview) != 2 {
		t.Fatalf("累计纪要应含两轮：%v", view.CumulativeReview)
	}
	if view.CumulativeReview[0].RoundTotal == nil || *view.CumulativeReview[0].RoundTotal != 75 {
		t.Fatalf("累计纪要首轮异常：%+v", view.CumulativeReview[0])
	}
	kinds := map[string]bool{}
	for _, entry := range view.TrainingEntries {
		kinds[entry.Kind] = entry.Available
	}
	for _, kind := range []string{"practice", "formal_retry", "report", "review"} {
		if !kinds[kind] {
			t.Fatalf("失败态必须提供 %s 入口：%v", kind, view.TrainingEntries)
		}
	}
}

// 评估未完成态：明确"这不是失败"；系统责任展示额度已返还；保留重试入口。
func TestFlowIncompleteView(t *testing.T) {
	system := flowResult(1, ResultEvaluationIncomplete, nil, ReasonScoringServiceFailure, nil)
	system.IncompleteReason = strPtr(ReasonScoringServiceFailure)
	view, err := BuildRoundResultView(testProject, system, nil, planRounds(1, 2))
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	if view.NotAFailureNote == nil || *view.NotAFailureNote == "" {
		t.Fatal("评估未完成必须说明这不是失败")
	}
	if view.EntitlementRefunded == nil || !*view.EntitlementRefunded {
		t.Fatal("系统责任必须展示额度已返还")
	}
	if view.FailureBlocked {
		t.Fatal("评估未完成不是失败，不得按失败阻断展示")
	}
	insufficient := flowResult(1, ResultEvaluationIncomplete, nil, ReasonInsufficientEvidence, nil)
	insufficient.IncompleteReason = strPtr(ReasonInsufficientEvidence)
	view2, err := BuildRoundResultView(testProject, insufficient, nil, planRounds(1, 2))
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	if view2.EntitlementRefunded == nil || *view2.EntitlementRefunded {
		t.Fatal("证据不足不是系统责任，不得展示额度返还")
	}
}

// 关键规则：任何结果视图都不泄露后续轮次完整答案/考点（视图无题目字段）。
func TestFlowNeverLeaksFutureAnswers(t *testing.T) {
	views := []RoundResultView{
		mustFlow(t, flowResultNoReason(1, ResultPass, intPtr(70)), nil, planRounds(1, 2)),
		mustFlow(t, flowResultNoReason(2, ResultFail, intPtr(55)), []Result{
			flowResultNoReason(1, ResultPass, intPtr(75)),
		}, planRounds(1, 2, 3)),
	}
	for _, view := range views {
		raw, err := json.Marshal(view)
		if err != nil {
			t.Fatalf("序列化失败: %v", err)
		}
		blob := string(raw)
		for _, forbidden := range []string{"question", "answer", "标准答案", "考点"} {
			if containsJSONKey(blob, forbidden) {
				t.Fatalf("结果视图不得携带题目/答案内容：%s", forbidden)
			}
		}
	}
}

func mustFlow(t *testing.T, current Result, prior []Result, plan []PlanRoundSnapshot) RoundResultView {
	t.Helper()
	view, err := BuildRoundResultView(testProject, current, prior, plan)
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}
	return view
}

func containsJSONKey(blob, key string) bool {
	for i := 0; i+len(key) <= len(blob); i++ {
		if blob[i:i+len(key)] == key {
			return true
		}
	}
	return false
}
