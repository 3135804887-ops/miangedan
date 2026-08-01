package consent

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	errDuplicateRecord = errors.New("duplicate consent record")
	errInvalidVersion  = errors.New("invalid consent version chain")
)

type memoryState struct {
	grants         map[string]Grant
	grantByRequest map[string]string
	historyByScope map[string][]string
	audits         map[string]AuditEvent
}

func newMemoryState() memoryState {
	return memoryState{
		grants:         make(map[string]Grant),
		grantByRequest: make(map[string]string),
		historyByScope: make(map[string][]string),
		audits:         make(map[string]AuditEvent),
	}
}

// MemoryStore is a serializable, rollback-on-error regional reference store.
type MemoryStore struct {
	mu    sync.Mutex
	state memoryState
}

// NewMemoryStore creates an empty consent store.
func NewMemoryStore() *MemoryStore { return &MemoryStore{state: newMemoryState()} }

// Transact commits a cloned state only when every grant and audit write succeeds.
func (s *MemoryStore) Transact(ctx context.Context, fn func(Tx) error) error {
	if ctx == nil || fn == nil {
		return errors.New("consent transaction requires context and callback")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	working := cloneMemoryState(s.state)
	if err := fn((*memoryTx)(&working)); err != nil {
		return err
	}
	s.state = working
	return nil
}

type memoryTx memoryState

func requestIndex(userID, dataRegion, operation, requestKey string) string {
	return dataRegion + "\x00" + userID + "\x00" + operation + "\x00" + requestKey
}

func scopeIndex(userID, dataRegion string, consentType Type, scopeHash string) string {
	return dataRegion + "\x00" + userID + "\x00" + string(consentType) + "\x00" + scopeHash
}

func (tx *memoryTx) GrantByRequest(userID, dataRegion, operation, requestKey string) (Grant, bool, error) {
	grantID, ok := tx.grantByRequest[requestIndex(userID, dataRegion, operation, requestKey)]
	if !ok {
		return Grant{}, false, nil
	}
	grant, ok := tx.grants[grantID]
	return cloneGrant(grant), ok, nil
}

func (tx *memoryTx) LatestGrant(userID, dataRegion string, consentType Type, scopeHash string) (Grant, bool, error) {
	history := tx.historyByScope[scopeIndex(userID, dataRegion, consentType, scopeHash)]
	if len(history) == 0 {
		return Grant{}, false, nil
	}
	grant, ok := tx.grants[history[len(history)-1]]
	return cloneGrant(grant), ok, nil
}

func (tx *memoryTx) LatestGrantsByUser(userID, dataRegion string) ([]Grant, error) {
	result := make([]Grant, 0)
	for _, history := range tx.historyByScope {
		if len(history) == 0 {
			continue
		}
		grant := tx.grants[history[len(history)-1]]
		if grant.UserID == userID && grant.DataRegion == dataRegion {
			result = append(result, cloneGrant(grant))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Type == result[j].Type {
			return result[i].ScopeHash < result[j].ScopeHash
		}
		return result[i].Type < result[j].Type
	})
	return result, nil
}

func (tx *memoryTx) HistoryByType(userID, dataRegion string, consentType Type) ([]Grant, error) {
	result := make([]Grant, 0)
	for _, grant := range tx.grants {
		if grant.UserID == userID && grant.DataRegion == dataRegion && grant.Type == consentType {
			result = append(result, cloneGrant(grant))
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ScopeHash == result[j].ScopeHash {
			return result[i].Version < result[j].Version
		}
		return result[i].ScopeHash < result[j].ScopeHash
	})
	return result, nil
}

func (tx *memoryTx) AppendGrant(grant Grant) error {
	if _, exists := tx.grants[grant.GrantID]; exists {
		return errDuplicateRecord
	}
	requestKey := requestIndex(grant.UserID, grant.DataRegion, grant.RequestOperation, grant.RequestKey)
	if _, exists := tx.grantByRequest[requestKey]; exists {
		return errDuplicateRecord
	}
	historyKey := scopeIndex(grant.UserID, grant.DataRegion, grant.Type, grant.ScopeHash)
	history := tx.historyByScope[historyKey]
	if len(history) == 0 {
		if grant.Version != 1 || grant.SupersedesGrantID != nil {
			return errInvalidVersion
		}
	} else {
		latestID := history[len(history)-1]
		latest := tx.grants[latestID]
		if grant.Version != latest.Version+1 || grant.SupersedesGrantID == nil || *grant.SupersedesGrantID != latestID {
			return errInvalidVersion
		}
	}
	tx.grants[grant.GrantID] = cloneGrant(grant)
	tx.grantByRequest[requestKey] = grant.GrantID
	tx.historyByScope[historyKey] = append(append([]string(nil), history...), grant.GrantID)
	return nil
}

func (tx *memoryTx) AppendAudit(event AuditEvent) error {
	if _, exists := tx.audits[event.AuditID]; exists {
		return errDuplicateRecord
	}
	tx.audits[event.AuditID] = event
	return nil
}

// MemoryStats exposes record counts without exposing evidence or identifiers.
type MemoryStats struct {
	Grants int
	Audits int
}

// Stats returns append-only record counts.
func (s *MemoryStore) Stats() MemoryStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return MemoryStats{Grants: len(s.state.grants), Audits: len(s.state.audits)}
}

// AuditsBySubject returns test/reference audit metadata without evidence content.
func (s *MemoryStore) AuditsBySubject(userID string) []AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]AuditEvent, 0)
	for _, event := range s.state.audits {
		if event.SubjectID == userID {
			result = append(result, event)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].AuditID < result[j].AuditID
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

func cloneMemoryState(source memoryState) memoryState {
	destination := newMemoryState()
	for key, grant := range source.grants {
		destination.grants[key] = cloneGrant(grant)
	}
	for key, grantID := range source.grantByRequest {
		destination.grantByRequest[key] = grantID
	}
	for key, history := range source.historyByScope {
		destination.historyByScope[key] = append([]string(nil), history...)
	}
	for key, event := range source.audits {
		destination.audits[key] = event
	}
	return destination
}

func cloneGrant(source Grant) Grant {
	cloned := source
	cloned.Scope = cloneScope(source.Scope)
	cloned.ExpiresAt = cloneTime(source.ExpiresAt)
	cloned.WithdrawnAt = cloneTime(source.WithdrawnAt)
	cloned.SupersedesGrantID = cloneString(source.SupersedesGrantID)
	return cloned
}

func cloneScope(source Scope) Scope {
	cloned := source
	cloned.AssignmentID = cloneString(source.AssignmentID)
	cloned.DataCategories = append([]DataCategory(nil), source.DataCategories...)
	cloned.MediaCategories = append([]MediaCategory(nil), source.MediaCategories...)
	cloned.Channels = append([]Channel(nil), source.Channels...)
	return cloned
}

func cloneTime(source *time.Time) *time.Time {
	if source == nil {
		return nil
	}
	cloned := source.UTC()
	return &cloned
}

func cloneString(source *string) *string {
	if source == nil {
		return nil
	}
	cloned := *source
	return &cloned
}
