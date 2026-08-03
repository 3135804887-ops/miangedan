package scoring

import (
	"sort"
	"strconv"
	"sync"
)

// MemoryStore 为内存版 ScoreVersion 存储（开发/测试；生产 PostgreSQL 只读+插入角色）。
type MemoryStore struct {
	mu        sync.RWMutex
	byIDem    map[string]Result
	byKey     map[string][]Result
	byAttempt map[string][]Result
}

// NewMemoryStore 创建空内存评分存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byIDem:    make(map[string]Result),
		byKey:     make(map[string][]Result),
		byAttempt: make(map[string][]Result),
	}
}

func resultKey(dataRegion, projectID string, roundSequence int) string {
	return dataRegion + "|" + projectID + "|" + strconv.Itoa(roundSequence)
}

// SaveResult 保存结果（同 idempotency_key 覆盖等价，幂等由服务层保证）。
func (m *MemoryStore) SaveResult(r Result, idempotencyKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byIDem[r.DataRegion+"|"+idempotencyKey] = r
	key := resultKey(r.DataRegion, r.ProjectID, r.RoundSequence)
	items := m.byKey[key]
	items = append(items, r)
	sort.Slice(items, func(i, j int) bool { return items[i].ScoreVersion < items[j].ScoreVersion })
	m.byKey[key] = items
	attemptKey := r.DataRegion + "|" + r.AttemptID
	attemptItems := m.byAttempt[attemptKey]
	attemptItems = append(attemptItems, r)
	sort.Slice(attemptItems, func(i, j int) bool {
		return attemptItems[i].ScoreVersion < attemptItems[j].ScoreVersion
	})
	m.byAttempt[attemptKey] = attemptItems
	return nil
}

// GetByIdempotencyKey 幂等键查询。
func (m *MemoryStore) GetByIdempotencyKey(dataRegion, idempotencyKey string) (Result, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.byIDem[dataRegion+"|"+idempotencyKey]
	if !ok {
		return Result{}, ErrNotFound
	}
	return r, nil
}

// GetLatestByAttempt 查询某次正式尝试的最新版本（版本号递增依据）。
func (m *MemoryStore) GetLatestByAttempt(dataRegion, attemptID string) (Result, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.byAttempt[dataRegion+"|"+attemptID]
	if len(items) == 0 {
		return Result{}, ErrNotFound
	}
	return items[len(items)-1], nil
}

// GetLatest 查询项目轮次最新有效版本。
func (m *MemoryStore) GetLatest(dataRegion, projectID string, roundSequence int) (Result, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.byKey[resultKey(dataRegion, projectID, roundSequence)]
	if len(items) == 0 {
		return Result{}, ErrNotFound
	}
	return items[len(items)-1], nil
}

// ListVersions 分页列出版本（score_version 升序；cursor 为版本序号偏移）。
func (m *MemoryStore) ListVersions(
	dataRegion, projectID string, roundSequence, limit int, cursor string,
) ([]Result, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.byKey[resultKey(dataRegion, projectID, roundSequence)]
	if len(items) == 0 {
		return nil, "", ErrNotFound
	}
	start := 0
	if cursor != "" {
		parsed, err := strconv.Atoi(cursor)
		if err != nil || parsed < 0 {
			return nil, "", ErrInvalidCursor
		}
		start = parsed
	}
	if start >= len(items) {
		return nil, "", nil
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	out := make([]Result, end-start)
	copy(out, items[start:end])
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return out, next, nil
}
