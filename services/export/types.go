// Package export 提供数据导出与删除编排服务（TASK-055；FR-040，US-05 场景 5）。
// 追踪：docs/data/RETENTION-MATRIX.md；docs/data/DATA-MODEL.md；
// 红线：删除必须是真实删除或不可逆匿名化（禁止软删除冒充）；进度逐层可见、
// 失败可重试；法定财务记录保留但解除内容关联；导出物必带训练用途标记。
package export

import "time"

// 任务状态（与 openapi AsyncTask / DeletionTask 对齐）。
const (
	TaskQueued    = "queued"
	TaskRunning   = "running"
	TaskSucceeded = "succeeded"
	TaskFailed    = "failed"

	DeletionRequested  = "REQUESTED"
	DeletionVerifying  = "VERIFYING"
	DeletionInProgress = "IN_PROGRESS"
	DeletionCompleted  = "COMPLETED"
	DeletionFailed     = "FAILED"

	LayerPending    = "pending"
	LayerInProgress = "in_progress"
	LayerDone       = "done"
	LayerFailed     = "failed"

	TargetProject = "project"
	TargetResume  = "resume"
	TargetJob     = "job"
	TargetAccount = "account"
)

// TrainingUseDisclaimer 为导出物强制标记（REPORT-SPEC/SCR-11 导出必带）。
const TrainingUseDisclaimer = "模拟训练结果，不代表真实企业录用结论"

// Actor 为调用方身份（区域强绑定）。
type Actor struct {
	UserID     string
	DataRegion string
}

// Task 为导出任务（异步；真实进度可查；导出物含训练用途标记）。
type Task struct {
	TaskID           string    `json:"task_id"`
	TaskType         string    `json:"task_type"` // export | report_generate
	Status           string    `json:"status"`
	ProgressNote     *string   `json:"progress_note,omitempty"`
	DataRegion       string    `json:"data_region"`
	ProjectID        string    `json:"project_id,omitempty"`
	Scope            string    `json:"scope"` // account | project
	TrainingMarker   bool      `json:"training_marker"`
	ExportContentRef string    `json:"export_content_ref,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// DeletionProgress 为六层真实进度（RETENTION-MATRIX 6.3）。
type DeletionProgress struct {
	Database             string `json:"database"`
	Cache                string `json:"cache"`
	SearchIndex          string `json:"search_index"`
	ObjectStorage        string `json:"object_storage"`
	Backups              string `json:"backups"`
	ThirdPartyProcessors string `json:"third_party_processors"`
}

// DeletionTask 为删除任务（级联；财务记录保留但解除内容关联）。
type DeletionTask struct {
	TaskID             string           `json:"task_id"`
	TargetType         string           `json:"target_type"`
	TargetID           string           `json:"target_id"`
	Status             string           `json:"status"`
	Progress           DeletionProgress `json:"progress"`
	LegalRetentionNote *string          `json:"legal_retention_note,omitempty"`
	DataRegion         string           `json:"data_region"`
	CreatedAt          time.Time        `json:"created_at"`
	CompletedAt        *time.Time       `json:"completed_at,omitempty"`
}

// DeletionRequest 为删除请求（target_type/target_id）。
type DeletionRequest struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	UserID     string `json:"user_id"`
}

// ExpiringItem 为到期提醒扫描条目（RETENTION-MATRIX §5：到期前 30/7 天提醒）。
type ExpiringItem struct {
	Kind      string    `json:"kind"`
	ItemID    string    `json:"item_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ExpiryWindow 为提醒窗口。
type ExpiryWindow struct {
	Days int
}

// DefaultExpiryWindows 为平台默认提醒窗口（30 天与 7 天）。
var DefaultExpiryWindows = []ExpiryWindow{{Days: 30}, {Days: 7}}
