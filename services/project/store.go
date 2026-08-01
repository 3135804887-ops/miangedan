package project

import "time"

// ListFilter 为项目列表筛选（材料派生筛选 company/job_title 随 TASK-018 落地）。
type ListFilter struct {
	Status            Status
	InterviewLanguage string
	DateFrom          time.Time
	DateTo            time.Time
	Company           string
	JobTitle          string
}

// Store 为项目与计划版本的持久化接口（当前内存实现，真实持久化随数据平台接入）。
type Store interface {
	CreateProject(p Project) error
	GetProject(userID, dataRegion, projectID string) (Project, error)
	ListProjects(userID, dataRegion string, f ListFilter) ([]Project, error)
	UpdateProject(p Project) error
	SavePlan(plan PlanVersion) error
	GetPlan(dataRegion, projectID string, version int) (PlanVersion, error)
	LatestPlan(dataRegion, projectID string) (PlanVersion, error)
	SaveLibraryEntry(e LibraryEntry) error
	ListLibrary(userID, dataRegion string, kind LibraryKind) ([]LibraryEntry, error)
	DeleteLibraryEntry(userID, dataRegion string, kind LibraryKind, materialID string) error
	GetLibraryEntry(userID, dataRegion string, kind LibraryKind, materialID string) (LibraryEntry, error)
	GetPreferences(userID, dataRegion string) (Preferences, error)
	SavePreferences(p Preferences) error
}

// IdempotencyStore 为写操作幂等键存储（NFR-006：重复请求不产生重复副作用）。
type IdempotencyStore interface {
	Remember(key string, result any) error
	Recall(key string, out any) (bool, error)
}
