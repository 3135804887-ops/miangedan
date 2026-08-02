// TASK-033 计划生成链路（FR-009/FR-011；US-02 场景 5）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-033；ai/prompts/plan-generation.md；
// ai/schemas/interview-plan.schema.json；PROMPT-POLICY（PII/注入安全过滤）。
package project

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

// PlanGenerator 为计划生成器接口：TASK-033 提供合成实现（供应商选型前不绑定厂商 SDK，
// 真实 LLM 接入经 TASK-030 适配层 + ai/services/orchestrator 的 PlanGenerator）。
type PlanGenerator interface {
	Generate(ctx context.Context, actor Actor, proj Project) (PlanVersion, error)
}

// StubPlanGenerator 为确定性合成生成器：轮次建议（默认 3 轮、1-5 轮与 10-60 分钟边界）、
// 六维权重默认 25/20/15/15/15/10、覆盖方案（全部关键维度）、PII/注入安全过滤。
type StubPlanGenerator struct{}

var (
	phonePattern  = regexp.MustCompile(`1[3-9]\d{9}`)
	emailPattern  = regexp.MustCompile(`[\w.+-]+@[\w-]+\.[\w.-]+`)
	idCardPattern = regexp.MustCompile(`\d{17}[\dXx]`)
	injectPattern = regexp.MustCompile(`(?i)(忽略(之前的|以上|所有)指令|你现在(是|变成|扮演)|ignore (all )?(previous|above) instructions|you are now )`)
)

// Generate 生成计划草稿（Frozen=false，进入 PLAN_REVIEW；不安全内容不进入房间）。
func (StubPlanGenerator) Generate(_ context.Context, actor Actor, proj Project) (PlanVersion, error) {
	if err := actor.validate(); err != nil {
		return PlanVersion{}, err
	}
	weights := map[string]int{
		"professional_competence": 25, "problem_solving": 20, "communication": 15,
		"experience_evidence": 15, "behavioral_collaboration": 15, "learning_adaptability": 10,
	}
	rounds := []RoundConfig{
		{
			Sequence: 1, RoundType: "role_professional", Role: "专业面试官",
			Focus: "考察岗位核心能力与问题解决", DurationMinutes: 30, Difficulty: "standard",
			CriticalDimensions: []string{"professional_competence", "problem_solving"},
			RubricBound:        false, QuestionCoveragePlanReady: true,
			CoveragePlan: coveragePlan("role_professional", "professional_competence", "problem_solving"),
		},
		{
			Sequence: 2, RoundType: "comprehensive_final", Role: "综合面试官",
			Focus: "综合评估学习适应性与跨场景行为", DurationMinutes: 20, Difficulty: "standard",
			CriticalDimensions: []string{"communication", "learning_adaptability"},
			RubricBound:        false, QuestionCoveragePlanReady: true,
			CoveragePlan: coveragePlan("comprehensive_final", "communication", "learning_adaptability"),
		},
	}
	if proj.ResumeRef != nil && proj.DegradedMode != ModeJDOnly && proj.DegradedMode != ModeNeither {
		rounds = append([]RoundConfig{{
			Sequence: 1, RoundType: "screening_resume_deepdive", Role: "招聘角色",
			Focus: "围绕简历经历结构化深挖，验证经历一致性", DurationMinutes: 25, Difficulty: "standard",
			CriticalDimensions: []string{"experience_evidence", "behavioral_collaboration"},
			RubricBound:        false, QuestionCoveragePlanReady: true,
			CoveragePlan: coveragePlan("screening_resume_deepdive", "experience_evidence", "behavioral_collaboration"),
		}}, rounds...)
		for i := range rounds {
			rounds[i].Sequence = i + 1
		}
	}
	roundWeights := make([]RoundWeight, 0, len(rounds))
	base := 100 / len(rounds)
	remainder := 100 - base*len(rounds)
	for i, r := range rounds {
		w := base
		if i < remainder {
			w++
		}
		roundWeights = append(roundWeights, RoundWeight{Sequence: r.Sequence, Weight: w})
	}
	return PlanVersion{
		ProjectID:         proj.ProjectID,
		PlanVersion:       proj.PlanVersion + 1,
		DataRegion:        proj.DataRegion,
		InterviewLanguage: proj.InterviewLanguage,
		ResumeRef:         proj.ResumeRef,
		JobRef:            proj.JobRef,
		DegradedMode:      proj.DegradedMode,
		RubricVersion:     "rubrics/v1/default",
		DimensionWeights:  weights,
		Rounds:            rounds,
		RoundWeights:      roundWeights,
		Frozen:            false,
		CreatedAt:         time.Now().UTC(),
	}, nil
}

func coveragePlan(roundType string, dims ...string) *QuestionCoveragePlan {
	points := make([]CoveragePoint, 0, len(dims))
	for _, d := range dims {
		points = append(points, CoveragePoint{
			CoverageID: roundType + "-" + d, Dimension: d,
			Description: "考察" + d + "维度", WeightInDimension: 1,
		})
	}
	return &QuestionCoveragePlan{
		CapabilityTargets: []string{roundType}, CoveragePoints: points,
		BackupQuestionCount: 2,
	}
}

// CheckPlanSafety 校验计划内容不包含 PII 复述与注入模式（fail-closed；不安全内容不进入房间）。
func CheckPlanSafety(p PlanVersion) error {
	for _, r := range p.Rounds {
		text := r.Role + r.Focus + r.RoundType
		for _, pat := range []*regexp.Regexp{phonePattern, emailPattern, idCardPattern, injectPattern} {
			if pat.MatchString(text) {
				return fmt.Errorf("%w: 计划包含不安全内容（PII 复述或注入模式）", ErrInvalidInput)
			}
		}
	}
	return nil
}
