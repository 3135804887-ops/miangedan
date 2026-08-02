package room

import (
	"encoding/json"
	"sort"
	"sync"
)

// MemoryStore 为内存版 Store/MediaTokenStore/IdempotencyStore（开发与测试；生产替换为 PG+Redis）。
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
	consumed map[string]bool
	revoked  map[string]bool
	nonceSes map[string]string
	idem     map[string][]byte
}

// NewMemoryStore 创建空内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]Session),
		consumed: make(map[string]bool),
		revoked:  make(map[string]bool),
		nonceSes: make(map[string]string),
		idem:     make(map[string][]byte),
	}
}

func sessionKey(dataRegion, sessionID string) string {
	return dataRegion + "|" + sessionID
}

// SaveSession 保存会话。
func (m *MemoryStore) SaveSession(s Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sessionKey(s.DataRegion, s.SessionID)] = s
	return nil
}

// GetSession 读取会话。
func (m *MemoryStore) GetSession(dataRegion, sessionID string) (Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionKey(dataRegion, sessionID)]
	if !ok {
		return Session{}, ErrNotFound
	}
	return s, nil
}

// UpdateSession 覆盖保存会话。
func (m *MemoryStore) UpdateSession(s Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := sessionKey(s.DataRegion, s.SessionID)
	if _, ok := m.sessions[key]; !ok {
		return ErrNotFound
	}
	m.sessions[key] = s
	return nil
}

// ListSessionsByProject 列出项目会话（创建时间倒序）。
func (m *MemoryStore) ListSessionsByProject(dataRegion, projectID string) ([]Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Session
	for _, s := range m.sessions {
		if s.DataRegion == dataRegion && s.ProjectID == projectID {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// RecordNonce 记录 nonce 所属会话（吊销定位用）。
func (m *MemoryStore) RecordNonce(sessionID, _ string, nonce string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nonceSes[nonce] = sessionID
	return nil
}

// ConsumeNonce 一次性消费 nonce；重复消费返回 false。
func (m *MemoryStore) ConsumeNonce(nonce string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.consumed[nonce] {
		return false
	}
	m.consumed[nonce] = true
	return true
}

// RevokeSession 吊销会话已签发的全部令牌（按 nonce 定位）。
func (m *MemoryStore) RevokeSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for nonce, sid := range m.nonceSes {
		if sid == sessionID {
			m.revoked[nonce] = true
		}
	}
	return nil
}

// IsNonceRevoked 查询 nonce 是否已吊销。
func (m *MemoryStore) IsNonceRevoked(nonce string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.revoked[nonce]
}

// Remember 记录幂等结果。
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

// Recall 读取幂等结果。
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
