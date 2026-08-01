package project

import (
	"encoding/json"
	"sort"
	"strconv"
	"sync"
)

// MemoryStore 为内存版 Store/IdempotencyStore（单进程测试与开发用；生产替换为 PostgreSQL）。
type MemoryStore struct {
	mu       sync.RWMutex
	projects map[string]Project
	plans    map[string]PlanVersion
	library  map[string]LibraryEntry
	prefs    map[string]Preferences
	idem     map[string][]byte
}

// NewMemoryStore 创建空的内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		projects: make(map[string]Project),
		plans:    make(map[string]PlanVersion),
		library:  make(map[string]LibraryEntry),
		prefs:    make(map[string]Preferences),
		idem:     make(map[string][]byte),
	}
}

func projectKey(userID, dataRegion, projectID string) string {
	return userID + "|" + dataRegion + "|" + projectID
}

func planKey(dataRegion, projectID string, version int) string {
	return dataRegion + "|" + projectID + "|" + strconv.Itoa(version)
}

func libraryKey(userID, dataRegion string, kind LibraryKind, materialID string) string {
	return userID + "|" + dataRegion + "|" + string(kind) + "|" + materialID
}

func prefsKey(userID, dataRegion string) string {
	return userID + "|" + dataRegion
}

// CreateProject 保存新项目（重复写入视为幂等成功）。
func (m *MemoryStore) CreateProject(p Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projects[projectKey(p.UserID, p.DataRegion, p.ProjectID)] = p
	return nil
}

// GetProject 按用户/区域/项目 ID 读取。
func (m *MemoryStore) GetProject(userID, dataRegion, projectID string) (Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.projects[projectKey(userID, dataRegion, projectID)]
	if !ok {
		return Project{}, ErrNotFound
	}
	return p, nil
}

// ListProjects 按筛选条件返回项目（创建时间倒序）。
func (m *MemoryStore) ListProjects(userID, dataRegion string, f ListFilter) ([]Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Project
	for _, p := range m.projects {
		if p.UserID != userID || p.DataRegion != dataRegion {
			continue
		}
		if f.Status != "" && p.Status != f.Status {
			continue
		}
		if f.InterviewLanguage != "" && p.InterviewLanguage != f.InterviewLanguage {
			continue
		}
		if !f.DateFrom.IsZero() && p.CreatedAt.Before(f.DateFrom) {
			continue
		}
		if !f.DateTo.IsZero() && p.CreatedAt.After(f.DateTo) {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// UpdateProject 覆盖保存项目（调用方保证先 Get）。
func (m *MemoryStore) UpdateProject(p Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := projectKey(p.UserID, p.DataRegion, p.ProjectID)
	if _, ok := m.projects[key]; !ok {
		return ErrNotFound
	}
	m.projects[key] = p
	return nil
}

// SavePlan 保存计划版本。
func (m *MemoryStore) SavePlan(plan PlanVersion) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plans[planKey(plan.DataRegion, plan.ProjectID, plan.PlanVersion)] = plan
	return nil
}

// GetPlan 按版本读取计划。
func (m *MemoryStore) GetPlan(dataRegion, projectID string, version int) (PlanVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	plan, ok := m.plans[planKey(dataRegion, projectID, version)]
	if !ok {
		return PlanVersion{}, ErrNotFound
	}
	return plan, nil
}

// LatestPlan 返回最新版本计划。
func (m *MemoryStore) LatestPlan(dataRegion, projectID string) (PlanVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var latest PlanVersion
	found := false
	for key, plan := range m.plans {
		if plan.DataRegion != dataRegion || plan.ProjectID != projectID {
			continue
		}
		if !found || plan.PlanVersion > latest.PlanVersion {
			latest = plan
			found = true
		}
		_ = key
	}
	if !found {
		return PlanVersion{}, ErrNotFound
	}
	return latest, nil
}

// SaveLibraryEntry 保存/覆盖材料库条目。
func (m *MemoryStore) SaveLibraryEntry(e LibraryEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.library[libraryKey(e.UserID, e.DataRegion, e.Kind, e.MaterialID)] = e
	return nil
}

// ListLibrary 按类型返回用户材料库（保存时间倒序）。
func (m *MemoryStore) ListLibrary(userID, dataRegion string, kind LibraryKind) ([]LibraryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []LibraryEntry
	for _, e := range m.library {
		if e.UserID == userID && e.DataRegion == dataRegion && e.Kind == kind {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// DeleteLibraryEntry 删除材料库条目（不存在视为成功，幂等）。
func (m *MemoryStore) DeleteLibraryEntry(userID, dataRegion string, kind LibraryKind, materialID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.library, libraryKey(userID, dataRegion, kind, materialID))
	return nil
}

// GetLibraryEntry 读取单个材料库条目。
func (m *MemoryStore) GetLibraryEntry(userID, dataRegion string, kind LibraryKind, materialID string) (LibraryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.library[libraryKey(userID, dataRegion, kind, materialID)]
	if !ok {
		return LibraryEntry{}, ErrNotFound
	}
	return e, nil
}

// GetPreferences 读取用户语言偏好（缺省 zh-CN）。
func (m *MemoryStore) GetPreferences(userID, dataRegion string) (Preferences, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.prefs[prefsKey(userID, dataRegion)]; ok {
		return p, nil
	}
	return Preferences{UserID: userID, DataRegion: dataRegion, UILanguage: "zh-CN", InterviewLanguage: "zh-CN"}, nil
}

// SavePreferences 保存用户语言偏好。
func (m *MemoryStore) SavePreferences(p Preferences) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prefs[prefsKey(p.UserID, p.DataRegion)] = p
	return nil
}

// Remember 记录幂等键结果（JSON 序列化）。
func (m *MemoryStore) Remember(key string, result any) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idem[key] = raw
	return nil
}

// Recall 读取幂等键结果。
func (m *MemoryStore) Recall(key string, out any) (bool, error) {
	m.mu.RLock()
	raw, ok := m.idem[key]
	m.mu.RUnlock()
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return false, err
	}
	return true, nil
}
