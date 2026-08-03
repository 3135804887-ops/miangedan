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
	inputs    map[string]Input
	reviews   map[string]int
	byReview  map[string]ReviewResult
}

// NewMemoryStore 创建空内存评分存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byIDem:    make(map[string]Result),
		byKey:     make(map[string][]Result),
		byAttempt: make(map[string][]Result),
		inputs:    make(map[string]Input),
		reviews:   make(map[string]int),
		byReview:  make(map[string]ReviewResult),
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

// SaveInput 保存冻结输入（复核必须使用完全相同的输入）。
func (m *MemoryStore) SaveInput(dataRegion, scoreID string, in Input) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inputs[dataRegion+"|"+scoreID] = in
	return nil
}

// GetInput 读取冻结输入。
func (m *MemoryStore) GetInput(dataRegion, scoreID string) (Input, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	in, ok := m.inputs[dataRegion+"|"+scoreID]
	if !ok {
		return Input{}, ErrNotFound
	}
	return in, nil
}

// GetFirstByAttempt 查询该次正式尝试的首个 ScoreVersion（复核基准）。
func (m *MemoryStore) GetFirstByAttempt(dataRegion, attemptID string) (Result, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.byAttempt[dataRegion+"|"+attemptID]
	if len(items) == 0 {
		return Result{}, ErrNotFound
	}
	return items[0], nil
}

// CountReviews 统计某次正式尝试的复核次数（每次正式尝试仅一次）。
func (m *MemoryStore) CountReviews(dataRegion, attemptID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.reviews[dataRegion+"|"+attemptID], nil
}

// MarkReview 登记一次复核（只增；无回退路径）。
func (m *MemoryStore) MarkReview(dataRegion, attemptID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reviews[dataRegion+"|"+attemptID]++
	return nil
}

// SaveReview 保存复核结果（幂等键去重由服务层保证）。
func (m *MemoryStore) SaveReview(dataRegion, idempotencyKey string, r ReviewResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byReview[dataRegion+"|"+idempotencyKey] = r
	return nil
}

// GetReviewByIdempotencyKey 幂等键查询复核结果。
func (m *MemoryStore) GetReviewByIdempotencyKey(
	dataRegion, idempotencyKey string,
) (ReviewResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.byReview[dataRegion+"|"+idempotencyKey]
	if !ok {
		return ReviewResult{}, ErrNotFound
	}
	return r, nil
}
