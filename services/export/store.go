package export

// Store 为导出/删除任务存储（生产 PostgreSQL；任务记录可更新进度，不可伪造完成）。
type Store interface {
	SaveTask(Task, string) error
	GetTaskByID(dataRegion, taskID string) (Task, error)
	GetTaskByIdempotencyKey(dataRegion, key string) (Task, error)
	UpdateTask(Task) error
	SaveDeletionTask(DeletionTask, string) error
	GetDeletionTaskByID(dataRegion, taskID string) (DeletionTask, error)
	GetDeletionTaskByIdempotencyKey(dataRegion, key string) (DeletionTask, error)
	UpdateDeletionTask(DeletionTask) error
}
