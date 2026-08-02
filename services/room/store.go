package room

// Store 为会话与令牌短期登记存储（生产：会话 PostgreSQL + 令牌 Redis，Redis 非证据存储）。
type Store interface {
	SaveSession(Session) error
	GetSession(dataRegion, sessionID string) (Session, error)
	UpdateSession(Session) error
	ListSessionsByProject(dataRegion, projectID string) ([]Session, error)
}

// IdempotencyStore 为写操作幂等键存储（NFR-006）。
type IdempotencyStore interface {
	Remember(key string, result any) error
	Recall(key string, out any) (bool, error)
}
