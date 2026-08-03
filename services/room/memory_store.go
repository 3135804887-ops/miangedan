package room

import (
	"encoding/json"
	"sort"
	"strconv"
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
	// TASK-023：字幕与回合冻结。
	transcripts map[string]Transcript
	turns       map[string]TurnState
	// TASK-024：工具事件。
	toolEvents map[string][]ToolEvent
	// TASK-027：会前冻结。
	prechecks map[string]PreCheck
}

// NewMemoryStore 创建空内存存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions:    make(map[string]Session),
		consumed:    make(map[string]bool),
		revoked:     make(map[string]bool),
		nonceSes:    make(map[string]string),
		idem:        make(map[string][]byte),
		transcripts: make(map[string]Transcript),
		turns:       make(map[string]TurnState),
		toolEvents:  make(map[string][]ToolEvent),
		prechecks:   make(map[string]PreCheck),
	}
}

func sessionKey(dataRegion, sessionID string) string {
	return dataRegion + "|" + sessionID
}

func transcriptKey(dataRegion, sessionID, utteranceID string) string {
	return dataRegion + "|" + sessionID + "|" + utteranceID
}

func turnKey(dataRegion, sessionID string, turnIndex int) string {
	return dataRegion + "|" + sessionID + "|" + strconv.Itoa(turnIndex)
}

func toolKey(dataRegion, sessionID string) string {
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

// SaveTranscript 保存字幕/转写（同一 utterance 覆盖为最新版本）。
func (m *MemoryStore) SaveTranscript(t Transcript) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transcripts[transcriptKey(t.DataRegion, t.SessionID, t.UtteranceID)] = t
	return nil
}

// GetTranscript 读取单条字幕/转写。
func (m *MemoryStore) GetTranscript(dataRegion, sessionID, utteranceID string) (Transcript, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.transcripts[transcriptKey(dataRegion, sessionID, utteranceID)]
	if !ok {
		return Transcript{}, ErrNotFound
	}
	return t, nil
}

// ListTranscripts 列出会话全部字幕/转写（按创建时间升序）。
func (m *MemoryStore) ListTranscripts(dataRegion, sessionID string) ([]Transcript, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	prefix := dataRegion + "|" + sessionID + "|"
	var out []Transcript
	for k, t := range m.transcripts {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

// SaveTurn 保存回合冻结状态。
func (m *MemoryStore) SaveTurn(t TurnState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.turns[turnKey(t.DataRegion, t.SessionID, t.TurnIndex)] = t
	return nil
}

// GetTurn 读取回合冻结状态；不存在返回 ErrNotFound。
func (m *MemoryStore) GetTurn(dataRegion, sessionID string, turnIndex int) (TurnState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.turns[turnKey(dataRegion, sessionID, turnIndex)]
	if !ok {
		return TurnState{}, ErrNotFound
	}
	return t, nil
}

// SaveToolEvent 保存工具事件（幂等键由服务层去重）。
func (m *MemoryStore) SaveToolEvent(ev ToolEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := toolKey(ev.DataRegion, ev.SessionID)
	m.toolEvents[key] = append(m.toolEvents[key], ev)
	return nil
}

// ListToolEvents 列出会话工具事件。
func (m *MemoryStore) ListToolEvents(dataRegion, sessionID string) ([]ToolEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	items := m.toolEvents[toolKey(dataRegion, sessionID)]
	out := make([]ToolEvent, len(items))
	copy(out, items)
	return out, nil
}

// SavePreCheck 保存会前冻结配置（同会话覆盖）。
func (m *MemoryStore) SavePreCheck(pc PreCheck) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prechecks[sessionKey(pc.DataRegion, pc.SessionID)] = pc
	return nil
}

// GetPreCheck 读取会前冻结配置；不存在返回 ErrNotFound。
func (m *MemoryStore) GetPreCheck(dataRegion, sessionID string) (PreCheck, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pc, ok := m.prechecks[sessionKey(dataRegion, sessionID)]
	if !ok {
		return PreCheck{}, ErrNotFound
	}
	return pc, nil
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
