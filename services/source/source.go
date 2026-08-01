// Package source 提供企业公开流程来源（ProcessSource）领域模型与可信度规则（TASK-015）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-015；PRD FR-007、FR-008；US-02 规则 1–4；
// docs/domain/DOMAIN-MODEL.md 第 6.6 节；docs/ai/PROVIDER-ADAPTERS.md 第 4.5 节；
// docs/security/SECURITY-REQUIREMENTS.md SEC-024、SEC-025。
//
// 红线：外部网页内容仅作为不可信数据进入结构化提取（来源元数据），绝不作为系统指令；
// 来源内容不得进入评分证据（FR-008、DOMAIN-MODEL 第 8 节）。
package source

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"miangedan/services/region"
)

// ai/schemas/interview-plan.schema.json）对齐，避免领域词汇漂移（OD-10）。
// SourceType 为公开流程来源类型（OD-10：稳定英文命名，与 interview-plan.schema 对齐）。
//
//nolint:revive // 类型名与包名前缀一致属刻意命名：与 API 契约字段 source_type（openapi.yaml、
type SourceType string

// 已批准来源类型枚举（PRD US-02 规则 2、FR-008；interview-plan.schema process_source_refs）。
const (
	OfficialCareersPage       SourceType = "official_careers_page"
	OfficialRecruitingContent SourceType = "official_recruiting_content"
	CrediblePublicMaterial    SourceType = "credible_public_material"
	CandidateExperience       SourceType = "candidate_experience"
	GenericTemplate           SourceType = "generic_template"
)

// Credibility 为来源可信度。
type Credibility string

// 已批准可信度枚举。
const (
	CredibilityHigh   Credibility = "high"
	CredibilityMedium Credibility = "medium"
	CredibilityLow    Credibility = "low"
)

// SourceStatus 为来源生命周期状态（active → under_review → taken_down；支持版权投诉与下架）。
//
//nolint:revive // 同上：与 API 契约字段 source_status 对齐（openapi.yaml SourceStatus）。
type SourceStatus string

// 已批准来源状态枚举。
const (
	StatusActive      SourceStatus = "active"
	StatusUnderReview SourceStatus = "under_review"
	StatusTakenDown   SourceStatus = "taken_down"
)

// ProcessSource 为公开面试流程参考的来源元数据（DOMAIN-MODEL §6.6）。
// 仅保存结构化元数据（链接/日期/类型/可信度/失效状态/摘要），不保存网页正文；
// 网页内容一律视为不可信数据（SEC-024、SEC-025）。
type ProcessSource struct {
	SourceID               string       `json:"source_id"`
	URL                    *string      `json:"url"`
	SourceType             SourceType   `json:"source_type"`
	RetrievedAt            time.Time    `json:"retrieved_at"`
	Credibility            Credibility  `json:"credibility"`
	ExpiresAt              *time.Time   `json:"expires_at"`
	Region                 string       `json:"region"`
	JobFamily              string       `json:"job_family"`
	Company                string       `json:"company,omitempty"`
	Role                   string       `json:"role,omitempty"`
	Level                  string       `json:"level,omitempty"`
	IsUnofficialExperience bool         `json:"is_unofficial_experience"`
	Summary                string       `json:"summary,omitempty"`
	Status                 SourceStatus `json:"status"`
	IdempotencyKey         string       `json:"-"`
	DataRegion             string       `json:"data_region"`
	CreatedAt              time.Time    `json:"created_at"`
	UpdatedAt              time.Time    `json:"updated_at"`
}

// FallbackReason 为回退通用模板的稳定原因码（与 openapi.yaml SourceSearchResult 对齐）。
type FallbackReason string

// 已批准回退原因。
const (
	FallbackNone           FallbackReason = ""
	FallbackMissingCompany FallbackReason = "missing_company"
	FallbackSearchFailed   FallbackReason = "search_failed"
	FallbackNoReliable     FallbackReason = "no_reliable_source"
)

// SearchQuery 为公开流程检索条件（PROVIDER-ADAPTERS §4.5 search_process(company, role, level, region)）。
type SearchQuery struct {
	Company string
	Role    string
	Level   string
	Region  string
}

// SearchResult 为检索链路的确定性输出：来源列表 + 是否回退通用模板 + AI 推导标记。
type SearchResult struct {
	Sources                 []ProcessSource `json:"sources"`
	FlowUsesGenericTemplate bool            `json:"flow_uses_generic_template"`
	AIDerived               bool            `json:"ai_derived"`
	FallbackReason          FallbackReason  `json:"fallback_reason,omitempty"`
	DataRegion              string          `json:"data_region"`
}

// SourceTypeOrder 为来源优先级（FR-008：官方来源优先）。
// 官方招聘页 > 官方招聘内容 > 可信公开材料 > 候选人经验；通用模板为回退，不参与排序展示。
var SourceTypeOrder = map[SourceType]int{
	OfficialCareersPage:       1,
	OfficialRecruitingContent: 2,
	CrediblePublicMaterial:    3,
	CandidateExperience:       4,
	GenericTemplate:           5,
}

// Validate 校验来源元数据（fail-closed）：
// 区域/类型/可信度/状态白名单、来源链接（非通用模板必填且仅 http/https）、
// 候选人经验必须标记非官方、通用模板不得携带链接、幂等键必填、区域归属一致。
func (s ProcessSource) Validate() error {
	if strings.TrimSpace(s.SourceID) == "" {
		return errors.New("来源缺少 source_id")
	}
	if err := region.ValidateDataRegion(s.Region); err != nil {
		return fmt.Errorf("来源区域非法: %w", err)
	}
	if err := region.ValidateDataRegion(s.DataRegion); err != nil {
		return fmt.Errorf("来源数据区归属非法: %w", err)
	}
	if s.Region != s.DataRegion {
		return fmt.Errorf("来源区域 %q 与数据区归属 %q 不一致（ADR-0005）", s.Region, s.DataRegion)
	}
	if !validSourceType(s.SourceType) {
		return fmt.Errorf("来源类型 %q 非法（允许：官方招聘页/官方招聘内容/可信公开材料/候选人经验/通用模板）", s.SourceType)
	}
	if !validCredibility(s.Credibility) {
		return fmt.Errorf("来源可信度 %q 非法（允许：high/medium/low）", s.Credibility)
	}
	if !validSourceStatus(s.Status) {
		return fmt.Errorf("来源状态 %q 非法（允许：active/under_review/taken_down）", s.Status)
	}
	if strings.TrimSpace(s.JobFamily) == "" {
		return errors.New("来源缺少 job_family（FR-007 按岗位族检索）")
	}
	if strings.TrimSpace(s.IdempotencyKey) == "" {
		return errors.New("来源缺少幂等键（NFR-006：写操作必须幂等）")
	}
	if s.RetrievedAt.IsZero() {
		return errors.New("来源缺少检索时间 retrieved_at")
	}
	if s.ExpiresAt != nil && !s.ExpiresAt.After(s.RetrievedAt) {
		return errors.New("来源失效时间 expires_at 必须晚于检索时间 retrieved_at")
	}
	switch s.SourceType {
	case GenericTemplate:
		if s.URL != nil {
			return errors.New("通用模板来源不得携带外部链接（回退内容不伪装企业流程，US-02 规则 3）")
		}
		if s.IsUnofficialExperience {
			return errors.New("通用模板不得标记为候选人经验")
		}
	default:
		if s.URL == nil || strings.TrimSpace(*s.URL) == "" {
			return errors.New("非通用模板来源必须携带可验证来源链接（FR-007）")
		}
		if err := validateHTTPURL(*s.URL); err != nil {
			return fmt.Errorf("来源链接非法: %w", err)
		}
	}
	if s.SourceType == CandidateExperience && !s.IsUnofficialExperience {
		return errors.New("候选人经验必须标记非官方（is_unofficial_experience=true，FR-008）")
	}
	if s.SourceType != CandidateExperience && s.IsUnofficialExperience {
		return errors.New("仅候选人经验可标记 is_unofficial_experience=true")
	}
	if s.CreatedAt.IsZero() {
		return errors.New("来源缺少创建时间 created_at")
	}
	if s.UpdatedAt.IsZero() {
		return errors.New("来源缺少更新时间 updated_at")
	}
	return nil
}

// Validate 校验检索条件：数据区域合法；公司为空时服务将直接回退通用模板（US-02 规则 3）。
func (q SearchQuery) Validate() error {
	if err := region.ValidateDataRegion(q.Region); err != nil {
		return fmt.Errorf("检索区域非法: %w", err)
	}
	return nil
}

// IsReliable 判断来源是否可支撑企业流程（FR-008、US-02 规则 3）。
// 可靠 = 官方招聘页/官方招聘内容/可信公开材料，状态 active、未失效、可信度 high/medium；
// 候选人经验永远不是可靠来源（仅参考，必须标记非官方）；通用模板为回退产物。
func (s ProcessSource) IsReliable(now time.Time) bool {
	if s.Status != StatusActive {
		return false
	}
	if s.ExpiresAt != nil && !s.ExpiresAt.After(now) {
		return false
	}
	switch s.SourceType {
	case OfficialCareersPage, OfficialRecruitingContent, CrediblePublicMaterial:
		return s.Credibility == CredibilityHigh || s.Credibility == CredibilityMedium
	case CandidateExperience, GenericTemplate:
		return false
	}
	return false
}

// IsExpired 判断来源是否已失效（失效任务用于过期巡检与自动回退）。
func (s ProcessSource) IsExpired(now time.Time) bool {
	return s.ExpiresAt != nil && !s.ExpiresAt.After(now)
}

// NewGenericTemplate 构造无可靠来源时的通用岗位/级别模板来源（US-02 规则 3、FR-008）。
// 返回内容明确标记为 AI 推导（SourceType=generic_template），不伪装企业流程。
func NewGenericTemplate(regionCode, jobFamily string, now time.Time, idempotencyKey string) ProcessSource {
	src := ProcessSource{
		SourceID:               NewSourceID(),
		SourceType:             GenericTemplate,
		RetrievedAt:            now,
		Credibility:            CredibilityMedium,
		Region:                 regionCode,
		JobFamily:              jobFamily,
		IsUnofficialExperience: false,
		Summary:                "无可靠来源时使用的通用岗位/级别模板（标记 AI 推导）：筛选与简历深挖 → 岗位专业能力 → 综合终面。",
		Status:                 StatusActive,
		IdempotencyKey:         idempotencyKey,
		DataRegion:             regionCode,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	return src
}

// SortByPriority 按 FR-008 来源优先级稳定排序（官方 > 官方内容 > 可信公开 > 经验 > 通用模板）。
func SortByPriority(items []ProcessSource) {
	sortSources(items)
}

func validSourceType(t SourceType) bool {
	switch t {
	case OfficialCareersPage, OfficialRecruitingContent, CrediblePublicMaterial, CandidateExperience, GenericTemplate:
		return true
	}
	return false
}

func validCredibility(c Credibility) bool {
	switch c {
	case CredibilityHigh, CredibilityMedium, CredibilityLow:
		return true
	}
	return false
}

func validSourceStatus(st SourceStatus) bool {
	switch st {
	case StatusActive, StatusUnderReview, StatusTakenDown:
		return true
	}
	return false
}

func validateHTTPURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("仅允许 http/https 协议，实际 %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("缺少主机名")
	}
	return nil
}
