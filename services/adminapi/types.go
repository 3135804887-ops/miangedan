// Package adminapi 提供运营后台与治理服务（EPIC-09；TASK-080~085；
// FR-037~FR-040；SCREEN-SPEC SCR-17）。
// 红线：默认匿名技术指标；运营不可旁听/代答；后台无编辑分数/解锁控件。
package adminapi

import "time"

// 后台角色（US-08 规则 1；超级管理员不默认拥有用户内容访问权）。
const (
	RoleSuperAdmin        = "super_admin"
	RoleOps               = "ops"
	RoleAIConfig          = "ai_config"
	RoleScoringGovernance = "scoring_governance"
	RoleSourceContent     = "source_content"
	RoleSupport           = "support"
	RolePrivacySecurity   = "privacy_security"
	RoleFinanceStaff      = "finance_staff"
)

// 能力类别（PROVIDER-ADAPTERS §4）。
const (
	CapabilityLLM    = "llm"
	CapabilityASR    = "asr"
	CapabilityTTS    = "tts"
	CapabilityAvatar = "avatar"
	CapabilitySearch = "search"
)

// 供应商状态与熔断状态。
const (
	ProviderActive   = "active"
	ProviderRamping  = "ramping"
	ProviderDisabled = "disabled"

	BreakerClosed   = "closed"
	BreakerOpen     = "open"
	BreakerHalfOpen = "half_open"
)

// Actor 为后台调用方身份。
type Actor struct {
	StaffID    string
	DataRegion string
	Role       string
}

// ProviderInfo 为供应商/模型注册表条目（不展示完整密钥）。
type ProviderInfo struct {
	ProviderID     string
	Capability     string
	Region         string
	Status         string
	RampPercent    int
	LatencyP95Ms   int
	ErrorRate      float64
	CircuitBreaker string
	Note           string
	UpdatedAt      time.Time
}

// ProviderHealth 为匿名技术指标（延迟/错误率/熔断状态）。
type ProviderHealth struct {
	ProviderID     string
	Capability     string
	Status         string
	LatencyP95Ms   int
	ErrorRate      float64
	CircuitBreaker string
}

// RoomSnapshot 为匿名会话技术状态（不含姓名/简历/回答/媒体）。
type RoomSnapshot struct {
	SnapshotID         string
	Region             string
	AnonymousSessionID string
	State              string
	DurationSeconds    int
	FaultCode          string
	CreatedAt          time.Time
}

// RegionOpsStatus 为区域监控快照（SLO/错误预算）。
type RegionOpsStatus struct {
	DataRegion      string
	OnlineRooms     int
	QueuedSessions  int
	Capacity        int
	ProviderHealth  []ProviderHealth
	SLO             map[string]float64
	ErrorBudgetBurn float64
	UpdatedAt       time.Time
}

// AuditEntry 为追加式后台审计（SELECT/INSERT only）。
type AuditEntry struct {
	AuditID    string
	StaffID    string
	Role       string
	Action     string
	TargetRef  string
	DataRegion string
	CreatedAt  time.Time
}
