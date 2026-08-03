package scoring

import (
	"fmt"
	"math/rand"
	"time"
)

// 稳定性回归门槛（SCORING-SPEC 第 10 节 / TASK-045 硬门槛）：
// 重复评分 95% 维度差 ≤3 分；及格结论一致率 ≥98%。
const (
	StabilityDimensionDiffMax = 3
	StabilityDiffRatioMin     = 0.95
	StabilityPassAgreementMin = 0.98
	StabilityMinRuns          = 20
)

// StabilityConfig 为稳定性回归配置（确定性：固定种子可复现）。
type StabilityConfig struct {
	Runs             int     `json:"runs"`
	PerturbationRate float64 `json:"perturbation_rate"`
	ScoreJitter      int     `json:"score_jitter"`
	Seed             int64   `json:"seed"`
}

// DefaultStabilityConfig 返回默认配置（模拟重复证据提取的锚点评分微扰）。
func DefaultStabilityConfig() StabilityConfig {
	return StabilityConfig{
		Runs:             200,
		PerturbationRate: 0.04,
		ScoreJitter:      1,
		Seed:             20260803,
	}
}

// StabilityThresholds 为回归门槛声明。
type StabilityThresholds struct {
	DimensionDiffMax int     `json:"dimension_diff_max"`
	DimensionDiffMin float64 `json:"dimension_diff_ratio_min"`
	PassAgreementMin float64 `json:"pass_agreement_min"`
}

// StabilityMetrics 为回归指标。
type StabilityMetrics struct {
	Runs                  int                      `json:"runs"`
	DimensionDiffLe3Ratio float64                  `json:"dimension_diff_le3_ratio"`
	PassAgreementRate     float64                  `json:"pass_agreement_rate"`
	PerDimensionRatio     map[DimensionKey]float64 `json:"per_dimension_diff_le3_ratio"`
	PassStatusCounts      map[string]int           `json:"pass_status_counts"`
}

// StabilityReport 为稳定性回归报告（report_kind=stability，TASK-036 框架校验）。
type StabilityReport struct {
	ReportKind  string              `json:"report_kind"`
	Dataset     string              `json:"dataset"`
	GeneratedAt time.Time           `json:"generated_at"`
	Config      StabilityConfig     `json:"config"`
	Metrics     StabilityMetrics    `json:"metrics"`
	Thresholds  StabilityThresholds `json:"thresholds"`
	Passed      bool                `json:"passed"`
}

// RunStabilityRegression 执行重复评分稳定性回归（TASK-045）。
// 基线为冻结输入的一次确定性评分；每次运行对覆盖点评分施加受控微扰
// （模拟重复证据提取的锚点/插值抖动），统计维度差 ≤3 占比与及格结论一致率。
func (s *Service) RunStabilityRegression(
	base Input, cfg StabilityConfig,
) (StabilityReport, error) {
	if cfg.Runs < StabilityMinRuns {
		return StabilityReport{}, fmt.Errorf(
			"%w: runs 必须 ≥%d", ErrInvalidInput, StabilityMinRuns)
	}
	if cfg.PerturbationRate < 0 || cfg.PerturbationRate > 1 {
		return StabilityReport{}, fmt.Errorf(
			"%w: perturbation_rate 必须为 0-1", ErrInvalidInput)
	}
	if cfg.ScoreJitter < 0 {
		return StabilityReport{}, fmt.Errorf(
			"%w: score_jitter 不能为负", ErrInvalidInput)
	}
	baseline, err := s.compute(base)
	if err != nil {
		return StabilityReport{}, err
	}
	if baseline.ResultStatus == ResultEvaluationIncomplete {
		return StabilityReport{}, fmt.Errorf(
			"%w: 基线必须是完整评分（当前 %s）",
			ErrInvalidInput, baseline.ResultStatus)
	}
	rng := rand.New(rand.NewSource(cfg.Seed)) // #nosec G404 -- 回归为可复现伪随机，非安全用途
	perDimension := make(map[DimensionKey]int, len(DimensionKeys))
	statusCounts := make(map[string]int, 3)
	statusCounts[baseline.ResultStatus] = 1
	for _, d := range DimensionKeys {
		perDimension[d] = 0
	}
	for run := 1; run <= cfg.Runs; run++ {
		perturbed := perturbAssessments(base, rng, cfg)
		result, err := s.compute(perturbed)
		if err != nil {
			return StabilityReport{}, err
		}
		if result.ResultStatus == baseline.ResultStatus {
			statusCounts[result.ResultStatus]++
		}
		baselineDims := dimensionScores(baseline)
		for _, dr := range result.DimensionResults {
			baseScore, ok := baselineDims[dr.Dimension]
			if !ok || dr.Score == nil || baseScore == nil {
				continue
			}
			diff := *dr.Score - *baseScore
			if diff < 0 {
				diff = -diff
			}
			if diff <= StabilityDimensionDiffMax {
				perDimension[dr.Dimension]++
			}
		}
	}
	metrics := StabilityMetrics{
		Runs:              cfg.Runs,
		PerDimensionRatio: make(map[DimensionKey]float64, len(DimensionKeys)),
		PassStatusCounts:  statusCounts,
	}
	overall := 1.0
	for _, d := range DimensionKeys {
		ratio := float64(perDimension[d]) / float64(cfg.Runs)
		metrics.PerDimensionRatio[d] = roundRatio(ratio)
		if ratio < overall {
			overall = ratio
		}
	}
	metrics.DimensionDiffLe3Ratio = roundRatio(overall)
	agreement := float64(statusCounts[baseline.ResultStatus]) / float64(cfg.Runs+1)
	metrics.PassAgreementRate = roundRatio(agreement)
	thresholds := StabilityThresholds{
		DimensionDiffMax: StabilityDimensionDiffMax,
		DimensionDiffMin: StabilityDiffRatioMin,
		PassAgreementMin: StabilityPassAgreementMin,
	}
	return StabilityReport{
		ReportKind:  "stability",
		Dataset:     "scoring-stability",
		GeneratedAt: s.now().UTC(),
		Config:      cfg,
		Metrics:     metrics,
		Thresholds:  thresholds,
		Passed: metrics.DimensionDiffLe3Ratio >= StabilityDiffRatioMin &&
			metrics.PassAgreementRate >= StabilityPassAgreementMin,
	}, nil
}

// perturbAssessments 对覆盖点评分施加受控微扰（锚点带内插值 ±jitter）。
func perturbAssessments(in Input, rng *rand.Rand, cfg StabilityConfig) Input {
	out := in
	out.CoverageAssessments = make([]CoverageAssessment, len(in.CoverageAssessments))
	copy(out.CoverageAssessments, in.CoverageAssessments)
	for i := range out.CoverageAssessments {
		cp := &out.CoverageAssessments[i]
		if cp.AnswerStatus != AnswerAnswered && cp.AnswerStatus != AnswerPartial {
			continue
		}
		if cp.Dimension == DimCommunication {
			continue // 沟通维度按子项计分，锚点不参与
		}
		if rng.Float64() > cfg.PerturbationRate {
			continue
		}
		jitter := rng.Intn(2*cfg.ScoreJitter+1) - cfg.ScoreJitter
		if cp.InterpolatedScore != nil {
			low := AnchorScore(cp.AnchorLevel)
			high := low + 20
			if high > 100 {
				high = 100
			}
			value := *cp.InterpolatedScore + jitter
			if value < low || value > high {
				value = *cp.InterpolatedScore
			}
			cp.InterpolatedScore = &value
			continue
		}
		low := AnchorScore(cp.AnchorLevel)
		if low >= 100 {
			continue
		}
		value := low + jitter
		if value < low || value > low+20 {
			value = low
		}
		if value != low {
			next := cp.AnchorLevel + 1
			if next > 5 {
				next = 5
			}
			cp.InterpolatedScore = &value
			cp.AnchorCitations = []AnchorCitation{{
				AnchorLevels: []int{cp.AnchorLevel, next},
				EvidenceIDs:  cp.EvidenceIDs,
			}}
		}
	}
	return out
}

func dimensionScores(r Result) map[DimensionKey]*int {
	out := make(map[DimensionKey]*int, len(r.DimensionResults))
	for _, dr := range r.DimensionResults {
		out[dr.Dimension] = dr.Score
	}
	return out
}
