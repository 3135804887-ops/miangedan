package scoring

import (
	"encoding/json"
	"testing"
)

func stabilityBaseline() Input {
	in := baseInput()
	in.CoverageAssessments = []CoverageAssessment{
		assessment(DimProfessional, AnswerAnswered, 4, 1, withScore(78,
			AnchorCitation{AnchorLevels: []int{4, 5}, EvidenceIDs: []string{"ev-1"}})),
		assessment(DimProblemSolving, AnswerAnswered, 4, 1, withScore(74,
			AnchorCitation{AnchorLevels: []int{4, 5}, EvidenceIDs: []string{"ev-2"}})),
		assessment(DimCommunication, AnswerAnswered, 3, 1, withCommunication(76, 72)),
		assessment(DimExperienceEvidence, AnswerAnswered, 4, 1, withScore(75,
			AnchorCitation{AnchorLevels: []int{4, 5}, EvidenceIDs: []string{"ev-4"}})),
		assessment(DimBehavioralCollaborate, AnswerAnswered, 3, 1, withScore(70,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-5"}})),
		assessment(DimLearningAdaptability, AnswerAnswered, 3, 1, withScore(66,
			AnchorCitation{AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-6"}})),
	}
	return in
}

// 门槛达标：重复评分 95% 维度差 ≤3、及格结论一致率 ≥98%（TASK-045 硬门槛）。
func TestStabilityRegressionThresholdsMet(t *testing.T) {
	svc, _ := newTestService(t)
	report, err := svc.RunStabilityRegression(stabilityBaseline(), DefaultStabilityConfig())
	if err != nil {
		t.Fatalf("稳定性回归失败: %v", err)
	}
	if !report.Passed {
		t.Fatalf("门槛未达标：diff_ratio=%v agreement=%v per_dim=%v",
			report.Metrics.DimensionDiffLe3Ratio,
			report.Metrics.PassAgreementRate,
			report.Metrics.PerDimensionRatio)
	}
	if report.Metrics.DimensionDiffLe3Ratio < StabilityDiffRatioMin {
		t.Fatalf("维度差 ≤3 比例 %v < %v", report.Metrics.DimensionDiffLe3Ratio,
			StabilityDiffRatioMin)
	}
	if report.Metrics.PassAgreementRate < StabilityPassAgreementMin {
		t.Fatalf("及格一致率 %v < %v", report.Metrics.PassAgreementRate,
			StabilityPassAgreementMin)
	}
	if len(report.Metrics.PerDimensionRatio) != len(DimensionKeys) {
		t.Fatalf("必须输出六维逐维比例，实际 %d", len(report.Metrics.PerDimensionRatio))
	}
	if report.ReportKind != "stability" {
		t.Fatalf("报告类型应为 stability，实际 %s", report.ReportKind)
	}
}

// 确定性：相同种子产生相同指标（可复现报告）。
func TestStabilityRegressionDeterministic(t *testing.T) {
	svc, _ := newTestService(t)
	first, err := svc.RunStabilityRegression(stabilityBaseline(), DefaultStabilityConfig())
	if err != nil {
		t.Fatalf("首次运行失败: %v", err)
	}
	second, err := svc.RunStabilityRegression(stabilityBaseline(), DefaultStabilityConfig())
	if err != nil {
		t.Fatalf("二次运行失败: %v", err)
	}
	if first.Metrics.DimensionDiffLe3Ratio != second.Metrics.DimensionDiffLe3Ratio ||
		first.Metrics.PassAgreementRate != second.Metrics.PassAgreementRate {
		t.Fatalf("同种子回归必须一致：%+v vs %+v",
			first.Metrics, second.Metrics)
	}
}

// 异常配置与基线必须拒绝。
func TestStabilityRegressionInvalidInputs(t *testing.T) {
	svc, _ := newTestService(t)
	cfg := DefaultStabilityConfig()
	cfg.Runs = 10
	if _, err := svc.RunStabilityRegression(stabilityBaseline(), cfg); err == nil {
		t.Fatal("runs < 20 必须拒绝")
	}
	cfg = DefaultStabilityConfig()
	cfg.PerturbationRate = 1.5
	if _, err := svc.RunStabilityRegression(stabilityBaseline(), cfg); err == nil {
		t.Fatal("perturbation_rate >1 必须拒绝")
	}
	// 基线为评估未完成 → 拒绝。
	in := stabilityBaseline()
	in.CriticalDimensions = []DimensionKey{DimExperienceEvidence}
	in.CoverageAssessments = append(in.CoverageAssessments,
		assessment(DimExperienceEvidence, AnswerSkipped, 0, 1),
		assessment(DimExperienceEvidence, AnswerSkipped, 0, 1))
	if _, err := svc.RunStabilityRegression(in, DefaultStabilityConfig()); err == nil {
		t.Fatal("不完整基线必须拒绝")
	}
}

// 报告 JSON 形状（与 mgd_evals.validate_stability_report 握手）。
func TestStabilityReportJSONShape(t *testing.T) {
	svc, _ := newTestService(t)
	report, err := svc.RunStabilityRegression(stabilityBaseline(), DefaultStabilityConfig())
	if err != nil {
		t.Fatalf("稳定性回归失败: %v", err)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	if decoded["report_kind"] != "stability" {
		t.Fatal("报告必须标记 report_kind=stability")
	}
	metrics, ok := decoded["metrics"].(map[string]any)
	if !ok {
		t.Fatal("报告缺少 metrics")
	}
	for _, key := range []string{"dimension_diff_le3_ratio", "pass_agreement_rate"} {
		if _, ok := metrics[key]; !ok {
			t.Fatalf("metrics 缺少 %s", key)
		}
	}
}
