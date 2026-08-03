// Package main 生成评分稳定性回归报告（TASK-045；写出 ai/evals/reports/stability.json）。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"miangedan/services/scoring"
)

func main() {
	out := flag.String("out", "", "报告输出路径（默认 ai/evals/reports/stability.json）")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "稳定性回归失败:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	store := scoring.NewMemoryStore()
	svc, err := scoring.NewService(store)
	if err != nil {
		return err
	}
	base := syntheticBaseline()
	report, err := svc.RunStabilityRegression(base, scoring.DefaultStabilityConfig())
	if err != nil {
		return err
	}
	target := out
	if target == "" {
		_, sourceFile, _, _ := runtime.Caller(0)
		repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(sourceFile)))))
		target = filepath.Join(repoRoot, "ai", "evals", "reports", "stability.json")
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	if err := os.WriteFile(target, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	fmt.Printf("稳定性回归：维度差≤3 比例 %.4f，及格一致率 %.4f，通过=%v，报告=%s\n",
		report.Metrics.DimensionDiffLe3Ratio, report.Metrics.PassAgreementRate,
		report.Passed, target)
	return nil
}

// syntheticBaseline 构造合成基线输入（模拟 zh-normal-01 语音作答）。
func syntheticBaseline() scoring.Input {
	intPtr := func(v int) *int { return &v }
	assessments := []scoring.CoverageAssessment{
		{
			CoverageID: "cp-professional", Dimension: scoring.DimProfessional,
			EvidenceIDs: []string{"ev-syn-01"}, WeightInDimension: 1,
			AnswerStatus: scoring.AnswerAnswered, InputMode: scoring.ModeVoice,
			AnchorLevel: 4, InterpolatedScore: intPtr(78),
			AnchorCitations: []scoring.AnchorCitation{{
				AnchorLevels: []int{4, 5}, EvidenceIDs: []string{"ev-syn-01"},
			}},
		},
		{
			CoverageID: "cp-problem", Dimension: scoring.DimProblemSolving,
			EvidenceIDs: []string{"ev-syn-02"}, WeightInDimension: 1,
			AnswerStatus: scoring.AnswerAnswered, InputMode: scoring.ModeVoice,
			AnchorLevel: 4, InterpolatedScore: intPtr(74),
			AnchorCitations: []scoring.AnchorCitation{{
				AnchorLevels: []int{4, 5}, EvidenceIDs: []string{"ev-syn-02"},
			}},
		},
		{
			CoverageID: "cp-communication", Dimension: scoring.DimCommunication,
			EvidenceIDs: []string{"ev-syn-03"}, WeightInDimension: 1,
			AnswerStatus: scoring.AnswerAnswered, InputMode: scoring.ModeVoice,
			AnchorLevel: 4, StructureClarity: intPtr(76), OralDelivery: intPtr(72),
		},
		{
			CoverageID: "cp-experience", Dimension: scoring.DimExperienceEvidence,
			EvidenceIDs: []string{"ev-syn-04"}, WeightInDimension: 1,
			AnswerStatus: scoring.AnswerAnswered, InputMode: scoring.ModeVoice,
			AnchorLevel: 4, InterpolatedScore: intPtr(75),
			AnchorCitations: []scoring.AnchorCitation{{
				AnchorLevels: []int{4, 5}, EvidenceIDs: []string{"ev-syn-04"},
			}},
		},
		{
			CoverageID: "cp-behavior", Dimension: scoring.DimBehavioralCollaborate,
			EvidenceIDs: []string{"ev-syn-05"}, WeightInDimension: 1,
			AnswerStatus: scoring.AnswerAnswered, InputMode: scoring.ModeVoice,
			AnchorLevel: 3, InterpolatedScore: intPtr(70),
			AnchorCitations: []scoring.AnchorCitation{{
				AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-syn-05"},
			}},
		},
		{
			CoverageID: "cp-learning", Dimension: scoring.DimLearningAdaptability,
			EvidenceIDs: []string{"ev-syn-06"}, WeightInDimension: 1,
			AnswerStatus: scoring.AnswerAnswered, InputMode: scoring.ModeVoice,
			AnchorLevel: 3, InterpolatedScore: intPtr(66),
			AnchorCitations: []scoring.AnchorCitation{{
				AnchorLevels: []int{3, 4}, EvidenceIDs: []string{"ev-syn-06"},
			}},
		},
	}
	return scoring.Input{
		SchemaVersion:       "1.0.0",
		ScoringRequestID:    "req-stability-baseline",
		IdempotencyKey:      "idem-stability-baseline",
		ProjectID:           "00000000-0000-4000-8000-000000000001",
		RoundSequence:       1,
		AttemptID:           "00000000-0000-4000-8000-00000000s001",
		AttemptKind:         scoring.AttemptInitial,
		DataRegion:          "cn",
		InterviewLanguage:   "zh-CN",
		RubricVersion:       "rubrics/v1/default",
		DimensionWeights:    scoring.DefaultWeights,
		CriticalDimensions:  []scoring.DimensionKey{scoring.DimProfessional, scoring.DimProblemSolving},
		CoverageAssessments: assessments,
		InputModeContext: scoring.InputModeContext{
			CommunicationMode: scoring.ModeVoice,
			ModesUsed:         []string{"voice"},
		},
		SubmittedAt: time.Now(),
	}
}
