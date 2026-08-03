package scoring

// Store 为追加式 ScoreVersion 存储（生产 PostgreSQL；仅 SELECT/INSERT，ADR-0004）。
// 红线：不存在更新/删除路径；纠正只产生新版本（supersedes_score_id 链接）。
type Store interface {
	// SaveResult 保存结果；idempotencyKey 为幂等键（NFR-006：重复提交返回首条）。
	SaveResult(ScoringResult, string) error
	GetByIdempotencyKey(dataRegion, idempotencyKey string) (ScoringResult, error)
	GetLatestByAttempt(dataRegion, attemptID string) (ScoringResult, error)
	GetLatest(dataRegion, projectID string, roundSequence int) (ScoringResult, error)
	ListVersions(dataRegion, projectID string, roundSequence, limit int, cursor string) ([]ScoringResult, string, error)
}
