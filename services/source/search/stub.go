package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"miangedan/services/source"
)

// StubAdapter 为搜索适配层的合成桩实现（TASK-030 未开工，仅落地 PROVIDER-ADAPTERS §4.5 契约语义）。
// 全部数据为虚构合成样例（synthetic），域名使用 example.com 保留域，禁止真实公司/个人数据；
// 真实供应商适配器随 TASK-030 实现，接口语义不变（ADR-0003）。
type StubAdapter struct {
	// Now 为可注入时钟（默认 time.Now）。
	Now func() time.Time
}

// syntheticSources 为内置合成来源样例（与 fixtures/synthetic/process-sources 对齐，synthetic: true）。
func syntheticSources(now time.Time) []source.ProcessSource {
	url := func(raw string) *string { return &raw }
	retrieved := now.Add(-48 * time.Hour)
	expires := now.Add(180 * 24 * time.Hour)
	expiredAt := now.Add(-24 * time.Hour)
	return []source.ProcessSource{
		{
			SourceID:    "src-fict-0001",
			URL:         url("https://careers.chengjiang-yunke.example.com/hiring-process"),
			SourceType:  source.OfficialCareersPage,
			RetrievedAt: retrieved,
			Credibility: source.CredibilityHigh,
			ExpiresAt:   &expires,
			Region:      "cn",
			JobFamily:   "data_engineering",
			Company:     "澄江云科（虚构）",
			Role:        "data_engineer",
			Level:       "senior",
			Summary:     "虚构公司澄江云科官方招聘页：数据工程岗位通常为 3 轮——HR 筛选、技术深挖、综合终面。",
			Status:      source.StatusActive,
			DataRegion:  "cn",
		},
		{
			SourceID:    "src-fict-0004",
			URL:         url("https://careers.novalake.example.com/recruiting/interview-process"),
			SourceType:  source.OfficialRecruitingContent,
			RetrievedAt: retrieved,
			Credibility: source.CredibilityMedium,
			ExpiresAt:   &expires,
			Region:      "intl",
			JobFamily:   "data_engineering",
			Company:     "Novalake Analytics（虚构）",
			Role:        "data_engineer",
			Level:       "senior",
			Summary:     "虚构公司 Novalake 官方招聘内容：技术轮包含白板设计题，终面为业务场景轮。",
			Status:      source.StatusActive,
			DataRegion:  "intl",
		},
		{
			SourceID:    "src-fict-0005",
			URL:         url("https://blog.example.com/posts/expired-official-process"),
			SourceType:  source.OfficialCareersPage,
			RetrievedAt: now.Add(-72 * 24 * time.Hour),
			Credibility: source.CredibilityHigh,
			ExpiresAt:   &expiredAt,
			Region:      "eu",
			JobFamily:   "software_engineering",
			Company:     "示例公司（已失效来源）",
			Summary:     "已失效的官方来源样例：失效来源自动回退通用模板（source_content 失效策略）。",
			Status:      source.StatusActive,
			DataRegion:  "eu",
		},
		{
			SourceID:               "src-fict-0002",
			URL:                    url("https://blog.example.com/posts/fictional-novalake-interview-exp"),
			SourceType:             source.CandidateExperience,
			RetrievedAt:            now.Add(-72 * time.Hour),
			Credibility:            source.CredibilityLow,
			ExpiresAt:              &expires,
			Region:                 "intl",
			JobFamily:              "data_engineering",
			Company:                "Novalake Analytics（虚构）",
			Summary:                "候选人经验帖（非官方，必须标记）：称 Novalake Analytics 技术轮包含白板设计题；仅作参考，不得冒充企业流程。",
			IsUnofficialExperience: true,
			Status:                 source.StatusActive,
			DataRegion:             "intl",
		},
	}
}

// SearchProcess 返回与检索条件匹配的合成来源（结构化元数据，无网页正文）。
// 公司名不区分大小写精确匹配；company 为空时不返回（由 Service 直接回退）。
func (s *StubAdapter) SearchProcess(_ context.Context, q source.SearchQuery) ([]source.ProcessSource, error) {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	t := now()
	all := syntheticSources(t)
	out := make([]source.ProcessSource, 0, len(all))
	for _, item := range all {
		if item.Region != q.Region {
			continue
		}
		if q.Company != "" && !strings.EqualFold(trimSpace(item.Company), trimSpace(q.Company)) {
			continue
		}
		item.RetrievedAt = t.Add(-48 * time.Hour)
		item.IdempotencyKey = fmt.Sprintf("stub-%s-%s", q.Region, item.SourceID)
		item.CreatedAt = item.RetrievedAt
		item.UpdatedAt = item.RetrievedAt
		item.DataRegion = q.Region
		item.Region = q.Region
		out = append(out, item)
	}
	return out, nil
}
