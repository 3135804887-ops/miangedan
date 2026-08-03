package scoring

import (
	"fmt"
)

// CongratulationCopy 为通过祝贺红线文案（SCR-10：与现有 result 前端语义一致）。
const CongratulationCopy = "恭喜你通过本轮面试，已进入下一轮"

// CriticalDimensionView 为关键维度通过情况展示。
type CriticalDimensionView struct {
	Dimension  DimensionKey `json:"dimension"`
	Score      *int         `json:"score"`
	GatePassed bool         `json:"gate_passed"`
}

// ProgressView 为流程进度（当前轮/总轮数）。
type ProgressView struct {
	CurrentRound int `json:"current_round"`
	TotalRounds  int `json:"total_rounds"`
}

// NextRoundView 为下一轮预告（角色/重点/难度/时长；不泄露题目与答案）。
type NextRoundView struct {
	Sequence        int    `json:"sequence"`
	Role            string `json:"role"`
	Focus           string `json:"focus"`
	Difficulty      string `json:"difficulty"`
	DurationMinutes int    `json:"duration_minutes"`
}

// RoundSummaryView 为累计纪要条目（只含已发生轮次的事实摘要，不含未来内容）。
type RoundSummaryView struct {
	Sequence        int      `json:"sequence"`
	ResultStatus    string   `json:"result_status"`
	RoundTotal      *int     `json:"round_total"`
	Strengths       []string `json:"strengths"`
	AttentionPoints []string `json:"attention_points"`
}

// TrainingEntry 为结果页训练入口（复盘练习/正式重试/报告/复核）。
type TrainingEntry struct {
	Kind      string `json:"kind"`
	Target    string `json:"target"`
	Available bool   `json:"available"`
}

// RoundResultView 为轮次结果页视图（SCR-10 语义；不泄露后续轮次完整答案）。
type RoundResultView struct {
	ProjectID           string                  `json:"project_id"`
	RoundSequence       int                     `json:"round_sequence"`
	ResultStatus        string                  `json:"result_status"`
	CongratulationText  *string                 `json:"congratulation_text,omitempty"`
	RoundTotal          *int                    `json:"round_total"`
	PassLine            int                     `json:"pass_line"`
	CriticalDimensions  []CriticalDimensionView `json:"critical_dimensions"`
	Strengths           []string                `json:"strengths"`
	AttentionPoints     []string                `json:"attention_points"`
	Progress            ProgressView            `json:"progress"`
	NextRound           *NextRoundView          `json:"next_round,omitempty"`
	FailureBlocked      bool                    `json:"failure_blocked"`
	CumulativeReview    []RoundSummaryView      `json:"cumulative_review"`
	IncompleteReason    *string                 `json:"incomplete_reason,omitempty"`
	NotAFailureNote     *string                 `json:"not_a_failure_note,omitempty"`
	EntitlementRefunded *bool                   `json:"entitlement_refunded,omitempty"`
	TrainingEntries     []TrainingEntry         `json:"training_entries"`
}

// PlanRoundSnapshot 为冻结计划中的轮次信息（下一轮预告来源）。
type PlanRoundSnapshot struct {
	Sequence        int
	Role            string
	Focus           string
	Difficulty      string
	DurationMinutes int
}

// BuildRoundResultView 构建轮次结果页视图（TASK-054；FR-021/FR-022，SCR-10）。
// 红线：通过态祝贺文案固定；失败态阻断下一轮并给出累计复盘与训练入口；
// 评估未完成明确"这不是失败"并按原因展示返还；任何分支都不输出
// 后续轮次正式考点或完整标准答案。
func BuildRoundResultView(
	projectID string,
	current Result,
	priorRounds []Result,
	planRounds []PlanRoundSnapshot,
) (RoundResultView, error) {
	if current.ProjectID != "" && current.ProjectID != projectID {
		return RoundResultView{}, fmt.Errorf("%w: 结果与项目不一致", ErrInvalidInput)
	}
	view := RoundResultView{
		ProjectID:       projectID,
		RoundSequence:   current.RoundSequence,
		ResultStatus:    current.ResultStatus,
		RoundTotal:      current.RoundTotal,
		PassLine:        PassLine,
		Strengths:       current.Explanations.Strengths,
		AttentionPoints: attentionPoints(current),
		Progress: ProgressView{
			CurrentRound: current.RoundSequence,
			TotalRounds:  len(planRounds),
		},
		CriticalDimensions: criticalDimensionViews(current),
		CumulativeReview:   cumulativeReview(append(priorRounds, current)),
		TrainingEntries:    []TrainingEntry{},
	}
	switch current.ResultStatus {
	case ResultPass:
		text := CongratulationCopy
		view.CongratulationText = &text
		view.NextRound = nextRound(current.RoundSequence, planRounds)
		view.TrainingEntries = append(view.TrainingEntries,
			TrainingEntry{Kind: "report", Target: "report", Available: true})
	case ResultFail:
		view.FailureBlocked = true
		view.TrainingEntries = []TrainingEntry{
			{Kind: "practice", Target: "coach", Available: true},
			{Kind: "formal_retry", Target: "retry", Available: true},
			{Kind: "report", Target: "report", Available: true},
			{Kind: "review", Target: "report#review", Available: true},
		}
	case ResultEvaluationIncomplete:
		reason := "unknown"
		if current.IncompleteReason != nil {
			reason = *current.IncompleteReason
		}
		view.IncompleteReason = &reason
		note := incompleteNote(reason)
		view.NotAFailureNote = &note
		refunded := reason == ReasonScoringServiceFailure || reason == "system_fault"
		view.EntitlementRefunded = &refunded
		view.TrainingEntries = []TrainingEntry{
			{Kind: "practice", Target: "coach", Available: true},
			{Kind: "formal_retry", Target: "retry", Available: true},
			{Kind: "report", Target: "report", Available: true},
		}
	default:
		return RoundResultView{}, fmt.Errorf("%w: 未知结果状态 %s", ErrInvalidInput, current.ResultStatus)
	}
	return view, nil
}

func criticalDimensionViews(r Result) []CriticalDimensionView {
	out := make([]CriticalDimensionView, 0, len(r.DimensionResults))
	failed := map[DimensionKey]bool{}
	for _, d := range r.GateResult.FailedCriticalDimensions {
		failed[d] = true
	}
	for _, dr := range r.DimensionResults {
		if !dr.IsCritical {
			continue
		}
		passed := dr.Score != nil && *dr.Score >= PassLine && !failed[dr.Dimension]
		out = append(out, CriticalDimensionView{
			Dimension:  dr.Dimension,
			Score:      dr.Score,
			GatePassed: passed,
		})
	}
	return out
}

func attentionPoints(r Result) []string {
	points := append([]string{}, r.Explanations.Improvements...)
	for _, d := range r.GateResult.WeakDimensions {
		points = append(points, fmt.Sprintf("维度 %s 未达 60 分，建议复盘", d))
	}
	return points
}

func cumulativeReview(rounds []Result) []RoundSummaryView {
	out := make([]RoundSummaryView, 0, len(rounds))
	for _, r := range rounds {
		out = append(out, RoundSummaryView{
			Sequence:        r.RoundSequence,
			ResultStatus:    r.ResultStatus,
			RoundTotal:      r.RoundTotal,
			Strengths:       r.Explanations.Strengths,
			AttentionPoints: attentionPoints(r),
		})
	}
	return out
}

func nextRound(sequence int, planRounds []PlanRoundSnapshot) *NextRoundView {
	for _, p := range planRounds {
		if p.Sequence == sequence+1 {
			return &NextRoundView{
				Sequence:        p.Sequence,
				Role:            p.Role,
				Focus:           p.Focus,
				Difficulty:      p.Difficulty,
				DurationMinutes: p.DurationMinutes,
			}
		}
	}
	return nil
}

func incompleteNote(reason string) string {
	switch reason {
	case ReasonInsufficientEvidence:
		return "这不是失败：证据不足导致评估未完成，已保留作答内容，可重试补齐"
	case ReasonScoringServiceFailure:
		return "这不是失败：评分服务故障导致评估未完成，恢复后自动重算，系统责任额度已返还"
	case "system_fault":
		return "这不是失败：系统故障导致评估未完成，已保留作答内容，系统责任额度已返还"
	case "user_exit":
		return "这不是失败：你主动结束面试，已保留作答内容，可重试"
	default:
		return "这不是失败：评估未完成，可重试"
	}
}
