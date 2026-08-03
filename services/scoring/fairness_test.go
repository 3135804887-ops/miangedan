package scoring

import (
	"encoding/json"
	"strings"
	"testing"
)

func passResult(projectID string) Result {
	total := 70
	return Result{
		ProjectID:    projectID,
		ResultStatus: ResultPass,
		RoundTotal:   &total,
		DimensionResults: []DimensionResult{
			{Dimension: DimProfessional, ScoreStatus: StatusScored, Score: intPtr(70)},
			{Dimension: DimProblemSolving, ScoreStatus: StatusScored, Score: intPtr(70)},
		},
	}
}

func failResult(projectID string) Result {
	total := 55
	return Result{
		ProjectID:    projectID,
		ResultStatus: ResultFail,
		RoundTotal:   &total,
		DimensionResults: []DimensionResult{
			{Dimension: DimProfessional, ScoreStatus: StatusScored, Score: intPtr(55)},
		},
	}
}

func incompleteResult(projectID string) Result {
	return Result{
		ProjectID:        projectID,
		ResultStatus:     ResultEvaluationIncomplete,
		IncompleteReason: strPtr(ReasonInsufficientEvidence),
	}
}

func strPtr(v string) *string { return &v }

// 切分聚合：语言/输入模式/岗位/年限/口音/便利设置。
func TestFairnessSlicesAggregation(t *testing.T) {
	monitor := NewFairnessMonitor()
	monitor.Record(passResult("p1"), FairnessMetadata{
		InterviewLanguage: "zh-CN", InputMode: ModeVoice, JobFamily: "data",
		ExperienceBand: "0-2", Accent: "mandarin",
	})
	monitor.Record(passResult("p2"), FairnessMetadata{
		InterviewLanguage: "zh-CN", InputMode: ModeVoice, JobFamily: "data",
		ExperienceBand: "0-2", Accent: "mandarin",
	})
	monitor.Record(failResult("p3"), FairnessMetadata{
		InterviewLanguage: "en-US", InputMode: ModeText, JobFamily: "frontend",
		ExperienceBand: "3-5", Accent: "neutral_en",
		Accommodations: []string{"extended_time"},
	})
	monitor.Record(incompleteResult("p4"), FairnessMetadata{
		InterviewLanguage: "en-US", InputMode: ModeText, JobFamily: "frontend",
		ExperienceBand: "3-5", Accent: "neutral_en",
		Accommodations: []string{"extended_time"},
	})
	snapshot := monitor.Snapshot()
	stats := map[string]SliceStat{}
	for _, s := range snapshot.Slices {
		stats[s.Slice+"|"+s.Value] = s
	}
	zh := stats["language|zh-CN"]
	if zh.Count != 2 || zh.PassCount != 2 || zh.PassRate != 1.0 || zh.MeanScore != 70 {
		t.Fatalf("zh-CN 切分异常：%+v", zh)
	}
	en := stats["language|en-US"]
	if en.Count != 2 || en.IncompleteCount != 1 || en.PassCount != 0 {
		t.Fatalf("en-US 切分异常：%+v", en)
	}
	accommodation := stats["accommodation:extended_time|enabled"]
	if accommodation.Count != 2 {
		t.Fatalf("便利设置切分异常：%+v", accommodation)
	}
	if _, ok := stats["job_family|data"]; !ok {
		t.Fatal("岗位切分缺失")
	}
	if _, ok := stats["accent|mandarin"]; !ok {
		t.Fatal("口音切分缺失")
	}
	if _, ok := stats["experience|0-2"]; !ok {
		t.Fatal("年限切分缺失")
	}
	// 均值维度分。
	if zh.MeanDimensionScores[DimProfessional] != 70 {
		t.Fatalf("维度均值异常：%v", zh.MeanDimensionScores)
	}
}

// 快照确定性排序 + 标签最小化（不含任何用户内容）。
func TestFairnessSnapshotDeterministicAndMinimal(t *testing.T) {
	monitor := NewFairnessMonitor()
	monitor.Record(passResult("p1"), FairnessMetadata{
		InterviewLanguage: "zh-CN", InputMode: ModeVoice,
	})
	monitor.Record(failResult("p2"), FairnessMetadata{
		InterviewLanguage: "en-US", InputMode: ModeText,
		Accommodations: []string{"extended_time", "repeat_questions"},
	})
	first := monitor.Snapshot()
	second := monitor.Snapshot()
	if len(first.Slices) != len(second.Slices) {
		t.Fatalf("快照长度应一致：%d vs %d", len(first.Slices), len(second.Slices))
	}
	for i := range first.Slices {
		if first.Slices[i].Slice != second.Slices[i].Slice ||
			first.Slices[i].Value != second.Slices[i].Value {
			t.Fatalf("快照切分顺序必须确定：%+v vs %+v",
				first.Slices[i], second.Slices[i])
		}
	}
	raw, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	blob := string(raw)
	for _, forbidden := range []string{"我的回答", "project_id", "p1", "p2", "revised_text"} {
		if strings.Contains(blob, forbidden) {
			t.Fatalf("监控快照不得携带用户内容/项目标识：%s", forbidden)
		}
	}
}

// 评估未完成：计数但不计入均分（无总分的轮次）。
func TestFairnessIncompleteNotInMean(t *testing.T) {
	monitor := NewFairnessMonitor()
	monitor.Record(passResult("p1"), FairnessMetadata{
		InterviewLanguage: "zh-CN", InputMode: ModeVoice,
	})
	monitor.Record(incompleteResult("p2"), FairnessMetadata{
		InterviewLanguage: "zh-CN", InputMode: ModeVoice,
	})
	snapshot := monitor.Snapshot()
	zh := snapshot.Slices[0]
	if zh.Count != 2 || zh.IncompleteCount != 1 || zh.MeanScore != 70 {
		t.Fatalf("评估未完成处理异常：%+v", zh)
	}
}
