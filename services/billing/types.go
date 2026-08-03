// Package billing 提供报价引擎与权益模型（TASK-060；FR-031，US-06 场景 1）。
// 追踪：docs/domain/BILLING-STATE-MACHINE.md §4/§5.1/§5.5；docs/data/DATA-MODEL.md。
// 红线：付费永不影响评分；已开始的正式轮次不因余额不足中断；开始后计费版本冻结。
package billing

import "time"

// 权益类型（openapi Entitlement kind 对齐）。
const (
	KindFreeCredit  = "free_credit"
	KindProjectPack = "project_pack"
	KindProSub      = "pro_subscription"
	KindTopup       = "topup_pack"
)

// FreeCreditSeconds 为首次登录免费额度（60 分钟）。
const FreeCreditSeconds = 60 * 60

// 报价状态（BILLING-STATE-MACHINE 5.1）。
const (
	QuoteDraft        = "QUOTE_DRAFT"
	QuotePresented    = "QUOTE_PRESENTED"
	QuoteAccepted     = "QUOTE_ACCEPTED"
	QuoteRecalculated = "QUOTE_RECALCULATED"
)

// Actor 为调用方身份。
type Actor struct {
	UserID     string
	DataRegion string
}

// Entitlement 为一条权益（免费/项目包/Pro/加油包）。
type Entitlement struct {
	EntitlementID   string
	UserID          string
	Kind            string
	ScopeProjectID  string
	TotalSeconds    int
	ConsumedSeconds int
	Status          string // active | consumed | expired | revoked
	ValidFrom       time.Time
	ValidTo         time.Time
	DataRegion      string
}

// Remaining 返回剩余秒数。
func (e Entitlement) Remaining() int {
	remaining := e.TotalSeconds - e.ConsumedSeconds
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RoundPlanInput 为计划中的单轮（时长/是否可重试）。
type RoundPlanInput struct {
	Sequence        int
	DurationMinutes int
	RetryEligible   bool
}

// PlanInput 为报价输入（冻结计划子集）。
type PlanInput struct {
	ProjectID string
	Rounds    []RoundPlanInput
}

// PriceConfig 为区域化合成定价（OD-02 未决前仅用于确定性报价与测试）。
type PriceConfig struct {
	Currency       string
	PerMinuteCents int
	TaxRate        float64
}

// PriceConfigFor 返回区域合成定价。
func PriceConfigFor(region string) PriceConfig {
	switch region {
	case "cn":
		return PriceConfig{Currency: "CNY", PerMinuteCents: 50, TaxRate: 0.06}
	case "eu":
		return PriceConfig{Currency: "EUR", PerMinuteCents: 12, TaxRate: 0.19}
	default:
		return PriceConfig{Currency: "USD", PerMinuteCents: 12, TaxRate: 0.0}
	}
}

// Quote 为一次报价（轮次/时长/重试/税费/有效期；开始后冻结）。
type Quote struct {
	QuoteID        string
	ProjectID      string
	PlanVersion    int
	Status         string
	TotalMinutes   int
	FreeRetries    int
	AmountCents    int
	Currency       string
	TaxDescription string
	ValidUntil     time.Time
	DataRegion     string
	CreatedAt      time.Time
}

// Freeze 为开始后的计费版本冻结（不可修改/不可重新报价）。
type Freeze struct {
	ProjectID   string
	QuoteID     string
	PlanVersion int
	Frozen      bool
	FrozenAt    time.Time
	DataRegion  string
}

// ProSubscription 为 Pro 订阅（结转规则：≤1 账期；余额 ≤ 2×月额度）。
type ProSubscription struct {
	SubscriptionID   string
	UserID           string
	Status           string
	MonthlySeconds   int
	PeriodStart      time.Time
	PeriodEnd        time.Time
	CarryoverSeconds int
	AutoRenew        bool
	DataRegion       string
}
