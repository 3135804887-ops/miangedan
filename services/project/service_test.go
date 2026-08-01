package project

import (
	"context"
	"errors"
	"testing"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	flow, err := LoadFlowConfig("")
	if err != nil {
		t.Fatalf("加载流程配置失败: %v", err)
	}
	store := NewMemoryStore()
	svc, err := NewService(store, store, flow)
	if err != nil {
		t.Fatalf("创建服务失败: %v", err)
	}
	return svc
}

var testActor = Actor{UserID: "user-001", DataRegion: "cn"}

func createTestProject(t *testing.T, svc *Service, mode DegradedMode) Project {
	t.Helper()
	consent := ""
	if mode != ModeFull {
		consent = "consent-001"
	}
	proj, err := svc.CreateProject(context.Background(), testActor, CreateInput{
		InterviewLanguage:     "zh-CN",
		DegradedMode:          mode,
		DegradedModeConsentID: consent,
		ResumeRef:             &MaterialRef{ID: "resume-1", Version: 1},
		JobRef:                &MaterialRef{ID: "job-1", Version: 1},
	}, "create-"+string(mode))
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}
	return proj
}

func basePlan(proj Project) PlanVersion {
	rounds := make([]RoundConfig, 0, 3)
	for i := 1; i <= 3; i++ {
		rounds = append(rounds, RoundConfig{
			Sequence:           i,
			RoundType:          RoundTypes[i-1],
			Role:               "合成角色",
			Focus:              "合成重点",
			DurationMinutes:    30,
			Difficulty:         "standard",
			CriticalDimensions: []string{DimensionKeys[0], DimensionKeys[1]},
			Tools:              []string{},
		})
	}
	return PlanVersion{
		ProjectID:         proj.ProjectID,
		PlanVersion:       1,
		DataRegion:        proj.DataRegion,
		InterviewLanguage: proj.InterviewLanguage,
		ResumeRef:         cloneRef(proj.ResumeRef),
		JobRef:            cloneRef(proj.JobRef),
		DegradedMode:      proj.DegradedMode,
		RubricVersion:     "rubrics/v1/default",
		DimensionWeights: map[string]int{
			"professional_competence": 25, "problem_solving": 20, "communication": 15,
			"experience_evidence": 15, "behavioral_collaboration": 15, "learning_adaptability": 10,
		},
		Rounds:       rounds,
		RoundWeights: defaultRoundWeights(rounds),
	}
}

func seedPlan(t *testing.T, svc *Service, proj Project, rounds []RoundConfig) PlanVersion {
	t.Helper()
	plan := basePlan(proj)
	if rounds != nil {
		plan.Rounds = rounds
		plan.RoundWeights = defaultRoundWeights(rounds)
	}
	if err := svc.store.SavePlan(plan); err != nil {
		t.Fatalf("保存基础计划失败: %v", err)
	}
	return plan
}

// 正常路径：创建 DRAFT 项目（full 模式）。
func TestCreateProjectValid(t *testing.T) {
	svc := newTestService(t)
	proj := createTestProject(t, svc, ModeFull)
	if proj.Status != StatusDraft || proj.CurrentRoundSequence != 0 {
		t.Fatalf("新项目应为 DRAFT 且轮次 0：%+v", proj)
	}
	if proj.ResumeRef == nil || proj.JobRef == nil {
		t.Fatal("材料引用应保留")
	}
}

// 异常路径：非法语言/降级缺同意/材料引用残缺/超长名称必须拒绝。
func TestCreateProjectRejected(t *testing.T) {
	svc := newTestService(t)
	cases := map[string]CreateInput{
		"非法语言":   {InterviewLanguage: "fr-FR", DegradedMode: ModeFull},
		"降级缺同意":  {InterviewLanguage: "zh-CN", DegradedMode: ModeJDOnly},
		"材料引用残缺": {InterviewLanguage: "zh-CN", DegradedMode: ModeFull, ResumeRef: &MaterialRef{ID: "r1"}},
		"超长名称":   {InterviewLanguage: "zh-CN", DegradedMode: ModeFull, Name: string(make([]byte, 121))},
	}
	for name, in := range cases {
		if _, err := svc.CreateProject(context.Background(), testActor, in, "k-"+name); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// 幂等性：同一幂等键重复创建返回同一项目（NFR-006）。
func TestCreateProjectIdempotent(t *testing.T) {
	svc := newTestService(t)
	in := CreateInput{InterviewLanguage: "zh-CN", DegradedMode: ModeFull, ResumeRef: &MaterialRef{ID: "r1", Version: 1}}
	first, err := svc.CreateProject(context.Background(), testActor, in, "idem-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateProject(context.Background(), testActor, in, "idem-1")
	if err != nil {
		t.Fatal(err)
	}
	if first.ProjectID != second.ProjectID {
		t.Fatal("幂等键应返回同一项目")
	}
}

// 正常路径：计划编辑生成新版本（未冻结）；越界配置拒绝（FR-009/FR-010）。
func TestEditPlan(t *testing.T) {
	svc := newTestService(t)
	proj := createTestProject(t, svc, ModeFull)
	seedPlan(t, svc, proj, nil)
	edited, err := svc.EditPlan(context.Background(), testActor, proj.ProjectID, 1, []RoundConfig{{
		Sequence: 1, RoundType: RoundTypes[0], DurationMinutes: 40, Difficulty: "challenge",
		CriticalDimensions: []string{DimensionKeys[2]},
	}}, "edit-1")
	if err != nil {
		t.Fatalf("合法编辑应通过: %v", err)
	}
	if edited.PlanVersion != 2 || edited.Frozen {
		t.Fatalf("新版本应为 v2 且未冻结：%+v", edited)
	}
	if edited.Rounds[0].QuestionCoveragePlanReady {
		t.Fatal("编辑后的轮次就绪标记必须重置")
	}
}

// 异常路径：轮次越界/未知类型/重复序号必须拒绝。
func TestEditPlanRejected(t *testing.T) {
	svc := newTestService(t)
	proj := createTestProject(t, svc, ModeFull)
	seedPlan(t, svc, proj, nil)
	validRound := RoundConfig{Sequence: 1, RoundType: RoundTypes[0], DurationMinutes: 30, Difficulty: "standard", CriticalDimensions: []string{DimensionKeys[0]}}
	rounds6 := make([]RoundConfig, 0, 6)
	for i := 1; i <= 6; i++ {
		r := validRound
		r.Sequence = i
		rounds6 = append(rounds6, r)
	}
	cases := map[string][]RoundConfig{
		"6轮越界":   rounds6,
		"时长70越界": {{Sequence: 1, RoundType: RoundTypes[0], DurationMinutes: 70, Difficulty: "standard", CriticalDimensions: []string{DimensionKeys[0]}}},
		"未知轮次类型": {{Sequence: 1, RoundType: "unknown_type", DurationMinutes: 30, Difficulty: "standard", CriticalDimensions: []string{DimensionKeys[0]}}},
		"重复序号":   {{Sequence: 1, RoundType: RoundTypes[0], DurationMinutes: 30, Difficulty: "standard", CriticalDimensions: []string{DimensionKeys[0]}}, {Sequence: 1, RoundType: RoundTypes[1], DurationMinutes: 30, Difficulty: "standard", CriticalDimensions: []string{DimensionKeys[0]}}},
		"空关键维度":  {{Sequence: 1, RoundType: RoundTypes[0], DurationMinutes: 30, Difficulty: "standard", CriticalDimensions: nil}},
		"未知维度":   {{Sequence: 1, RoundType: RoundTypes[0], DurationMinutes: 30, Difficulty: "standard", CriticalDimensions: []string{"unknown_dim"}}},
	}
	for name, rounds := range cases {
		if _, err := svc.EditPlan(context.Background(), testActor, proj.ProjectID, 1, rounds, "k-"+name); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// 异常路径：不完整计划（缺覆盖方案/量表）确认被拒（不完整计划禁止开始）。
func TestConfirmPlanIncomplete(t *testing.T) {
	svc := newTestService(t)
	proj := createTestProject(t, svc, ModeFull)
	seedPlan(t, svc, proj, nil)
	if _, err := svc.ConfirmPlan(context.Background(), testActor, proj.ProjectID, 1, nil, "", "confirm-1"); !errors.Is(err, ErrPlanIncomplete) {
		t.Fatalf("不完整计划必须返回 ErrPlanIncomplete，实际 %v", err)
	}
}

// 正常路径：就绪后确认 → 冻结 + READY；确认后编辑被拒（FR-011）。
func TestConfirmPlanCompleteAndFreeze(t *testing.T) {
	svc := newTestService(t)
	proj := createTestProject(t, svc, ModeFull)
	plan := seedPlan(t, svc, proj, nil)
	for _, r := range plan.Rounds {
		if err := svc.SetRoundReadiness(proj.DataRegion, proj.ProjectID, 1, r.Sequence, true, true); err != nil {
			t.Fatalf("标记就绪失败: %v", err)
		}
	}
	confirmed, err := svc.ConfirmPlan(context.Background(), testActor, proj.ProjectID, 1, []string{"text_only"}, "", "confirm-ok")
	if err != nil {
		t.Fatalf("完整计划确认应通过: %v", err)
	}
	if confirmed.Status != StatusReady || confirmed.PlanVersion != 1 || confirmed.CurrentRoundSequence != 1 {
		t.Fatalf("确认后应为 READY/plan v1/轮次 1：%+v", confirmed)
	}
	frozen, err := svc.store.GetPlan(proj.DataRegion, proj.ProjectID, 1)
	if err != nil || !frozen.Frozen {
		t.Fatal("确认后计划必须冻结")
	}
	if _, err := svc.EditPlan(context.Background(), testActor, proj.ProjectID, 1, []RoundConfig{{
		Sequence: 1, RoundType: RoundTypes[0], DurationMinutes: 30, Difficulty: "standard", CriticalDimensions: []string{DimensionKeys[0]},
	}}, "edit-after"); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("冻结后编辑必须返回 ErrStateConflict，实际 %v", err)
	}
}

// 异常路径：活动项目删除被拒；正常路径：非活动项目返回删除任务。
func TestDeleteProject(t *testing.T) {
	svc := newTestService(t)
	proj := createTestProject(t, svc, ModeFull)
	task, err := svc.DeleteProject(context.Background(), testActor, proj.ProjectID, "del-1")
	if err != nil || task.TaskID == "" {
		t.Fatalf("DRAFT 项目应返回删除任务: %v", err)
	}
	active := proj
	active.Status = StatusInSession
	if err := svc.store.UpdateProject(active); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DeleteProject(context.Background(), testActor, proj.ProjectID, "del-2"); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("活动项目删除必须返回 ErrStateConflict，实际 %v", err)
	}
}

// 正常路径：复制项目生成独立 DRAFT 项目并复用材料引用。
func TestDuplicateProject(t *testing.T) {
	svc := newTestService(t)
	proj := createTestProject(t, svc, ModeFull)
	dup, err := svc.DuplicateProject(context.Background(), testActor, proj.ProjectID, "en-US", "dup-1")
	if err != nil {
		t.Fatalf("复制失败: %v", err)
	}
	if dup.ProjectID == proj.ProjectID || dup.Status != StatusDraft || dup.InterviewLanguage != "en-US" {
		t.Fatalf("复制项目应为独立 DRAFT：%+v", dup)
	}
	if dup.ResumeRef == nil || dup.ResumeRef.ID != proj.ResumeRef.ID {
		t.Fatal("复制应复用材料引用")
	}
}

// 正常路径：列表筛选按状态/语言生效。
func TestListProjectsFilter(t *testing.T) {
	svc := newTestService(t)
	createTestProject(t, svc, ModeFull)
	items, err := svc.ListProjects(context.Background(), testActor, ListFilter{Status: StatusDraft, InterviewLanguage: "zh-CN"})
	if err != nil || len(items) != 1 {
		t.Fatalf("筛选应返回 1 项: %v %v", len(items), err)
	}
	items, err = svc.ListProjects(context.Background(), testActor, ListFilter{Status: StatusReady})
	if err != nil || len(items) != 0 {
		t.Fatalf("筛选 READY 应为 0 项: %v %v", len(items), err)
	}
}
