package room

// Store 为会话与令牌短期登记存储（生产：会话 PostgreSQL + 令牌 Redis，Redis 非证据存储）。
type Store interface {
	SaveSession(Session) error
	GetSession(dataRegion, sessionID string) (Session, error)
	UpdateSession(Session) error
	ListSessionsByProject(dataRegion, projectID string) ([]Session, error)
	// TASK-023 字幕与回合冻结存储。
	SaveTranscript(Transcript) error
	GetTranscript(dataRegion, sessionID, utteranceID string) (Transcript, error)
	ListTranscripts(dataRegion, sessionID string) ([]Transcript, error)
	SaveTurn(TurnState) error
	GetTurn(dataRegion, sessionID string, turnIndex int) (TurnState, error)
	// TASK-024 工具事件存储。
	SaveToolEvent(ToolEvent) error
	ListToolEvents(dataRegion, sessionID string) ([]ToolEvent, error)
}

// IdempotencyStore 为写操作幂等键存储（NFR-006）。
type IdempotencyStore interface {
	Remember(key string, result any) error
	Recall(key string, out any) (bool, error)
}
