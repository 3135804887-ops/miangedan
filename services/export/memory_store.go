package export

import "sync"

// MemoryStore 为内存版任务存储（开发/测试；生产 PostgreSQL）。
type MemoryStore struct {
	mu           sync.RWMutex
	exports      map[string]Task
	exportIDems  map[string]Task
	deletions    map[string]DeletionTask
	deletionIDem map[string]DeletionTask
}

// NewMemoryStore 创建空内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		exports:      make(map[string]Task),
		exportIDems:  make(map[string]Task),
		deletions:    make(map[string]DeletionTask),
		deletionIDem: make(map[string]DeletionTask),
	}
}

// SaveTask 保存导出任务（幂等键去重由服务层保证）。
func (m *MemoryStore) SaveTask(t Task, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exports[t.DataRegion+"|"+t.TaskID] = t
	if idemKey != "" {
		m.exportIDems[t.DataRegion+"|"+idemKey] = t
	}
	return nil
}

// GetTaskByID 按任务 ID 查询导出任务。
func (m *MemoryStore) GetTaskByID(dataRegion, taskID string) (Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.exports[dataRegion+"|"+taskID]
	if !ok {
		return Task{}, ErrNotFound
	}
	return t, nil
}

// GetTaskByIdempotencyKey 幂等键查询导出任务。
func (m *MemoryStore) GetTaskByIdempotencyKey(dataRegion, key string) (Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.exportIDems[dataRegion+"|"+key]
	if !ok {
		return Task{}, ErrNotFound
	}
	return t, nil
}

// UpdateTask 更新导出任务进度。
func (m *MemoryStore) UpdateTask(t Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.exports[t.DataRegion+"|"+t.TaskID] = t
	return nil
}

// SaveDeletionTask 保存删除任务（幂等键去重由服务层保证）。
func (m *MemoryStore) SaveDeletionTask(t DeletionTask, idemKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletions[t.DataRegion+"|"+t.TaskID] = t
	if idemKey != "" {
		m.deletionIDem[t.DataRegion+"|"+idemKey] = t
	}
	return nil
}

// GetDeletionTaskByID 按任务 ID 查询删除任务。
func (m *MemoryStore) GetDeletionTaskByID(dataRegion, taskID string) (DeletionTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.deletions[dataRegion+"|"+taskID]
	if !ok {
		return DeletionTask{}, ErrNotFound
	}
	return t, nil
}

// GetDeletionTaskByIdempotencyKey 幂等键查询删除任务。
func (m *MemoryStore) GetDeletionTaskByIdempotencyKey(dataRegion, key string) (DeletionTask, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.deletionIDem[dataRegion+"|"+key]
	if !ok {
		return DeletionTask{}, ErrNotFound
	}
	return t, nil
}

// UpdateDeletionTask 更新删除任务进度。
func (m *MemoryStore) UpdateDeletionTask(t DeletionTask) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletions[t.DataRegion+"|"+t.TaskID] = t
	return nil
}
