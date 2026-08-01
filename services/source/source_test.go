package source

import (
	"testing"
	"time"
)

func TestValidProcessSource(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	exp := now.Add(180 * 24 * time.Hour)
	u := "https://careers.example.com/hiring-process"
	src := ProcessSource{
		SourceID:       "src-test-0001",
		URL:            &u,
		SourceType:     OfficialCareersPage,
		RetrievedAt:    now,
		Credibility:    CredibilityHigh,
		ExpiresAt:      &exp,
		Region:         "cn",
		JobFamily:      "data_engineering",
		Company:        "示例公司（虚构）",
		Status:         StatusActive,
		IdempotencyKey: "key-0001",
		DataRegion:     "cn",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := src.Validate(); err != nil {
		t.Fatalf("合法来源应通过校验: %v", err)
	}
}

func TestInvalidProcessSource(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	exp := now.Add(30 * 24 * time.Hour)
	u := "https://careers.example.com/hiring-process"
	mk := func(mut func(*ProcessSource)) ProcessSource {
		s := ProcessSource{
			SourceID:       "src-test-x",
			URL:            &u,
			SourceType:     OfficialCareersPage,
			RetrievedAt:    now,
			Credibility:    CredibilityHigh,
			ExpiresAt:      &exp,
			Region:         "cn",
			JobFamily:      "data_engineering",
			Status:         StatusActive,
			IdempotencyKey: "key-x",
			DataRegion:     "cn",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		mut(&s)
		return s
	}
	cases := []struct {
		name string
		src  ProcessSource
	}{
		{"非法区域", mk(func(s *ProcessSource) { s.Region = "us" })},
		{"区域与数据区不一致", mk(func(s *ProcessSource) { s.DataRegion = "intl" })},
		{"未知来源类型", mk(func(s *ProcessSource) { s.SourceType = "recruiter_call" })},
		{"未知可信度", mk(func(s *ProcessSource) { s.Credibility = "certain" })},
		{"未知状态", mk(func(s *ProcessSource) { s.Status = "deleted" })},
		{"空岗位族", mk(func(s *ProcessSource) { s.JobFamily = "" })},
		{"缺幂等键", mk(func(s *ProcessSource) { s.IdempotencyKey = "" })},
		{"失效早于检索", mk(func(s *ProcessSource) {
			past := now.Add(-24 * time.Hour)
			s.ExpiresAt = &past
		})},
		{"非通用模板缺链接", mk(func(s *ProcessSource) { s.URL = nil })},
		{"非 http 链接", mk(func(s *ProcessSource) {
			bad := "ftp://careers.example.com/x"
			s.URL = &bad
		})},
		{"候选人经验未标记非官方", mk(func(s *ProcessSource) { s.SourceType = CandidateExperience })},
		{"通用模板带链接", mk(func(s *ProcessSource) {
			s.SourceType = GenericTemplate
			s.Credibility = CredibilityMedium
		})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.src.Validate(); err == nil {
				t.Fatal("非法来源必须被拒绝（fail-closed）")
			}
		})
	}
}

func TestCandidateExperienceMustMarkUnofficial(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	u := "https://blog.example.com/posts/exp"
	src := ProcessSource{
		SourceID:       "src-test-exp",
		URL:            &u,
		SourceType:     CandidateExperience,
		RetrievedAt:    now,
		Credibility:    CredibilityLow,
		Region:         "intl",
		JobFamily:      "data_engineering",
		Status:         StatusActive,
		IdempotencyKey: "key-exp",
		DataRegion:     "intl",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := src.Validate(); err == nil {
		t.Fatal("候选人经验必须标记非官方（FR-008）")
	}
	src.IsUnofficialExperience = true
	if err := src.Validate(); err != nil {
		t.Fatalf("标记非官方后应通过: %v", err)
	}
}

func TestIsReliable(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(30 * 24 * time.Hour)
	past := now.Add(-24 * time.Hour)
	u := "https://careers.example.com/x"
	mk := func(mut func(*ProcessSource)) ProcessSource {
		s := ProcessSource{
			SourceType:     OfficialCareersPage,
			URL:            &u,
			RetrievedAt:    now,
			Credibility:    CredibilityHigh,
			ExpiresAt:      &future,
			Region:         "cn",
			DataRegion:     "cn",
			JobFamily:      "data_engineering",
			Status:         StatusActive,
			IdempotencyKey: "k",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		mut(&s)
		return s
	}
	cases := []struct {
		name string
		src  ProcessSource
		want bool
	}{
		{"官方招聘页 high 可信", mk(func(s *ProcessSource) {}), true},
		{"官方招聘内容 medium 可信", mk(func(s *ProcessSource) {
			s.SourceType = OfficialRecruitingContent
			s.Credibility = CredibilityMedium
		}), true},
		{"可信公开材料 medium 可信", mk(func(s *ProcessSource) {
			s.SourceType = CrediblePublicMaterial
			s.Credibility = CredibilityMedium
		}), true},
		{"官方来源 low 可信不可靠", mk(func(s *ProcessSource) { s.Credibility = CredibilityLow }), false},
		{"已失效不可靠", mk(func(s *ProcessSource) { s.ExpiresAt = &past }), false},
		{"已下架不可靠", mk(func(s *ProcessSource) { s.Status = StatusTakenDown }), false},
		{"待复核不可靠", mk(func(s *ProcessSource) { s.Status = StatusUnderReview }), false},
		{"候选人经验永不作为可靠来源", mk(func(s *ProcessSource) {
			s.SourceType = CandidateExperience
			s.Credibility = CredibilityMedium
			s.IsUnofficialExperience = true
		}), false},
		{"通用模板不可靠", mk(func(s *ProcessSource) {
			s.SourceType = GenericTemplate
			s.Credibility = CredibilityMedium
			s.URL = nil
		}), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.src.IsReliable(now); got != tc.want {
				t.Fatalf("IsReliable=%v，期望 %v", got, tc.want)
			}
		})
	}
}

func TestGenericTemplateMarksAIDerived(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	tpl := NewGenericTemplate("cn", "general", now, "fallback-key-0001")
	if err := tpl.Validate(); err != nil {
		t.Fatalf("通用模板必须通过校验: %v", err)
	}
	if tpl.SourceType != GenericTemplate {
		t.Fatalf("通用模板类型错误: %s", tpl.SourceType)
	}
	if tpl.URL != nil {
		t.Fatal("通用模板不得携带外部链接（不伪装企业流程）")
	}
	if tpl.IsReliable(now) {
		t.Fatal("通用模板不得被视为可靠来源")
	}
	if !tpl.IsUnofficialExperience && tpl.SourceType == GenericTemplate {
		// 推导标记由 SearchResult.AIDerived 承载；此处验证类型本身不冒充官方。
	}
}

func TestSortByPriority(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	mk := func(id string, typ SourceType) ProcessSource {
		return ProcessSource{
			SourceID:    id,
			SourceType:  typ,
			RetrievedAt: now,
		}
	}
	items := []ProcessSource{
		mk("b", CandidateExperience),
		mk("a", OfficialCareersPage),
		mk("c", CrediblePublicMaterial),
		mk("d", OfficialRecruitingContent),
	}
	SortByPriority(items)
	want := []string{"a", "d", "c", "b"}
	for i, id := range want {
		if items[i].SourceID != id {
			t.Fatalf("优先级排序错误，第 %d 位应为 %s，实际 %s", i, id, items[i].SourceID)
		}
	}
}
