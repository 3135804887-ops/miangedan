package scoring

import (
	"context"
	"errors"
	"testing"
)

func newTestRubricRegistry(t *testing.T) *RubricRegistry {
	t.Helper()
	registry, err := LoadDefaultRubricRegistry()
	if err != nil {
		t.Fatalf("加载默认量表失败: %v", err)
	}
	return registry
}

func v2Rubric() Rubric {
	return Rubric{
		ID:      "rubrics/v2/experimental",
		Version: "2.0.0",
		Status:  "draft_for_review",
		DefaultWeights: map[DimensionKey]int{
			DimProfessional:          25,
			DimProblemSolving:        20,
			DimCommunication:         15,
			DimExperienceEvidence:    15,
			DimBehavioralCollaborate: 15,
			DimLearningAdaptability:  10,
		},
		Anchors:           map[int]int{1: 20, 2: 40, 3: 60, 4: 80, 5: 100},
		MinCoverageRatio:  0.55,
		TotalWeight:       100,
		MaxDimensionDelta: 5,
	}
}

// 默认量表加载：六维权重/锚点/覆盖率阈值与 rubric 文件一致。
func TestLoadDefaultRubricRegistry(t *testing.T) {
	registry := newTestRubricRegistry(t)
	rubric, err := registry.Get("rubrics/v1/default")
	if err != nil {
		t.Fatalf("读取默认量表失败: %v", err)
	}
	if rubric.Status != "approved" {
		t.Fatalf("默认量表状态应为 approved，实际 %s", rubric.Status)
	}
	for level, score := range map[int]int{1: 20, 2: 40, 3: 60, 4: 80, 5: 100} {
		if rubric.Anchors[level] != score {
			t.Fatalf("锚点 %d 应为 %d，实际 %d", level, score, rubric.Anchors[level])
		}
	}
	if rubric.MinCoverageRatio != MinCoverageRatio {
		t.Fatalf("证据充分度阈值应为 %v，实际 %v", MinCoverageRatio, rubric.MinCoverageRatio)
	}
	sum := 0
	for _, d := range DimensionKeys {
		sum += rubric.DefaultWeights[d]
	}
	if sum != 100 {
		t.Fatalf("默认权重总和应为 100，实际 %d", sum)
	}
}

// SC-EC-19：单维 ±5 且总和 100 接受；+6 拒绝；0 权重重分配路径允许。
func TestRubricWeightsValidation(t *testing.T) {
	registry := newTestRubricRegistry(t)
	valid := defaultWeights()
	valid[DimProfessional] = 30
	valid[DimLearningAdaptability] = 5
	if err := registry.ValidateWeights("rubrics/v1/default", valid); err != nil {
		t.Fatalf("+5 调整应接受: %v", err)
	}
	tooMuch := defaultWeights()
	tooMuch[DimProfessional] = 31
	tooMuch[DimLearningAdaptability] = 4
	if err := registry.ValidateWeights("rubrics/v1/default", tooMuch); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("+6 调整应拒绝，实际 %v", err)
	}
	badSum := defaultWeights()
	badSum[DimProfessional] = 30
	if err := registry.ValidateWeights("rubrics/v1/default", badSum); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("权重和 !=100 应拒绝，实际 %v", err)
	}
	reallocated := defaultWeights()
	reallocated[DimExperienceEvidence] = 0
	reallocated[DimProfessional] = 40
	if err := registry.ValidateWeights("rubrics/v1/default", reallocated); err != nil {
		t.Fatalf("0 权重重新分配路径应接受（SC-EC-21）: %v", err)
	}
}

// 历史分数不因版本升级被修改：旧版本结果原样保留，新版本仅用于新评分。
func TestRubricVersionUpgradeKeepsHistory(t *testing.T) {
	store := NewMemoryStore()
	svc, err := NewService(store)
	if err != nil {
		t.Fatalf("创建服务失败: %v", err)
	}
	registry := newTestRubricRegistry(t)
	svc.SetRubricRegistry(registry)
	in := baseInput()
	in.CoverageAssessments = voiceAssessments(3)
	v1Result := mustScore(t, svc, in)
	// 注册 v2（校准参数变化，锚点/权重不变）。
	if err := registry.Register(v2Rubric()); err != nil {
		t.Fatalf("注册 v2 失败: %v", err)
	}
	in2 := in
	in2.ScoringRequestID = "req-v2"
	in2.IdempotencyKey = "idem-v2"
	in2.AttemptID = "00000000-0000-4000-8000-00000000v201"
	in2.RubricVersion = "rubrics/v2/experimental"
	v2Result := mustScore(t, svc, in2)
	if v2Result.RubricVersion != "rubrics/v2/experimental" {
		t.Fatalf("新评分应使用 v2，实际 %s", v2Result.RubricVersion)
	}
	// 历史结果原样保留。
	old, err := store.GetByIdempotencyKey("cn", in.IdempotencyKey)
	if err != nil {
		t.Fatalf("读取历史结果失败: %v", err)
	}
	if old.RubricVersion != "rubrics/v1/default" || old.ScoreID != v1Result.ScoreID {
		t.Fatal("历史结果不得因版本升级被修改")
	}
	replay := mustScore(t, svc, in)
	if replay.ScoreID != v1Result.ScoreID || replay.RubricVersion != "rubrics/v1/default" {
		t.Fatal("幂等重放必须返回 v1 结果")
	}
}

// 活跃正式会话固定版本：不匹配 fail-closed。
func TestRubricPinnedCheck(t *testing.T) {
	registry := newTestRubricRegistry(t)
	if err := registry.PinnedCheck("rubrics/v1/default", "rubrics/v1/default"); err != nil {
		t.Fatalf("同版本应通过: %v", err)
	}
	if err := registry.PinnedCheck("rubrics/v1/default", "rubrics/v2/experimental"); !errors.Is(
		err, ErrRubricPinnedMismatch) {
		t.Fatalf("版本不匹配应拒绝，实际 %v", err)
	}
}

// 未知量表版本拒绝（fail-closed）。
func TestRubricUnknownVersionRejected(t *testing.T) {
	svc, _ := newTestService(t)
	in := baseInput()
	in.RubricVersion = "rubrics/v9/unknown"
	in.CoverageAssessments = voiceAssessments(3)
	if _, err := svc.Score(context.Background(), testActor, in); !errors.Is(err, ErrRubricNotFound) {
		t.Fatalf("未知量表版本应拒绝，实际 %v", err)
	}
}

// 注册表约束：版本不可覆盖、锚点必须完整、六维权重齐全。
func TestRubricRegistryConstraints(t *testing.T) {
	registry := newTestRubricRegistry(t)
	if err := registry.Register(v2Rubric()); err != nil {
		t.Fatalf("首次注册 v2 应成功: %v", err)
	}
	if err := registry.Register(v2Rubric()); err == nil {
		t.Fatal("版本重复注册必须拒绝")
	}
	badAnchors := v2Rubric()
	badAnchors.Anchors = map[int]int{1: 30, 2: 40, 3: 60, 4: 80, 5: 100}
	if err := registry.Register(badAnchors); err == nil {
		t.Fatal("锚点映射非法必须拒绝")
	}
	missingDim := v2Rubric()
	delete(missingDim.DefaultWeights, DimProfessional)
	if err := registry.Register(missingDim); err == nil {
		t.Fatal("缺维度的量表必须拒绝")
	}
}
