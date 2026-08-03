package scoring

// Store 为追加式 ScoreVersion 存储（生产 PostgreSQL；仅 SELECT/INSERT，ADR-0004）。
// 红线：不存在更新/删除路径；纠正只产生新版本（supersedes_score_id 链接）。
type Store interface {
	// SaveResult 保存结果；idempotencyKey 为幂等键（NFR-006：重复提交返回首条）。
	SaveResult(Result, string) error
	GetByIdempotencyKey(dataRegion, idempotencyKey string) (Result, error)
	GetLatestByAttempt(dataRegion, attemptID string) (Result, error)
	GetLatest(dataRegion, projectID string, roundSequence int) (Result, error)
	ListVersions(dataRegion, projectID string, roundSequence, limit int, cursor string) ([]Result, string, error)
	// TASK-043 正式复核：冻结输入与复核次数必须持久化（追加式；只增不改）。
	SaveInput(dataRegion, scoreID string, in Input) error
	GetInput(dataRegion, scoreID string) (Input, error)
	GetFirstByAttempt(dataRegion, attemptID string) (Result, error)
	CountReviews(dataRegion, attemptID string) (int, error)
	MarkReview(dataRegion, attemptID string) error
	SaveReview(dataRegion, idempotencyKey string, r ReviewResult) error
	GetReviewByIdempotencyKey(dataRegion, idempotencyKey string) (ReviewResult, error)
}
