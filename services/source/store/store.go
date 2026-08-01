// Package store 提供企业流程来源持久化抽象（TASK-015）。
// 追踪：docs/data/DATA-MODEL.md 第 5.2 节（process_sources 表）；NFR-006（幂等）；
// services/migrate/migrations/0002_process_sources.sql（TASK-003 迁移工具，追加式迁移、可重复执行）。
//
// 本任务提供线程安全的内存实现用于契约落地与单测；PostgreSQL 实现按同一接口随存储接线任务落地。
package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"miangedan/services/region"
	"miangedan/services/source"
)

// ListQuery 为来源库筛选条件（与 openapi.yaml GET /v1/sources 对齐）。
type ListQuery struct {
	DataRegion     string
	JobFamily      string
	Status         source.SourceStatus
	IncludeExpired bool
	Cursor         int
	Limit          int
}

// Store 为来源存储接口：幂等保存、按 ID 读取、分页列表。
type Store interface {
	// Save 幂等保存来源；同幂等键或同 source_id 已存在时返回 saved=false（NFR-006 重复副作用为 0）。
	Save(ctx context.Context, s source.ProcessSource) (saved bool, err error)
	// Get 按来源 ID 读取；不存在返回 ErrNotFound。
	Get(ctx context.Context, sourceID string) (source.ProcessSource, error)
	// List 按查询条件分页列出（默认 limit 20，最大 100）。
	List(ctx context.Context, q ListQuery) ([]source.ProcessSource, error)
}

// ErrNotFound 为来源不存在错误。
var ErrNotFound = errors.New("来源不存在")

// Memory 为线程安全的内存存储实现（合成桩阶段；接口语义与迁移表约束一致）。
type Memory struct {
	mu      sync.RWMutex
	byID    map[string]source.ProcessSource
	byKey   map[string]string // idempotency_key → source_id（唯一约束）
	ordered []string
}

// NewMemory 创建空内存存储。
func NewMemory() *Memory {
	return &Memory{
		byID:  make(map[string]source.ProcessSource),
		byKey: make(map[string]string),
	}
}

// Save 实现 Store；同幂等键/同 ID 重复写入返回 saved=false（幂等去重）。
func (m *Memory) Save(_ context.Context, s source.ProcessSource) (bool, error) {
	if err := s.Validate(); err != nil {
		return false, fmt.Errorf("保存来源元数据非法: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byKey[s.IdempotencyKey]; ok {
		return false, nil
	}
	if _, ok := m.byID[s.SourceID]; ok {
		return false, nil
	}
	m.byID[s.SourceID] = s
	m.byKey[s.IdempotencyKey] = s.SourceID
	m.ordered = append(m.ordered, s.SourceID)
	return true, nil
}

// Get 实现 Store。
func (m *Memory) Get(_ context.Context, sourceID string) (source.ProcessSource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.byID[sourceID]
	if !ok {
		return source.ProcessSource{}, ErrNotFound
	}
	return s, nil
}

// List 实现 Store；区域必填（跨区列表拒绝，ADR-0005），支持岗位族/状态/失效过滤与游标分页。
func (m *Memory) List(_ context.Context, q ListQuery) ([]source.ProcessSource, error) {
	if err := region.ValidateDataRegion(q.DataRegion); err != nil {
		return nil, fmt.Errorf("列表区域非法: %w", err)
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var matches []source.ProcessSource
	for _, id := range m.ordered {
		s := m.byID[id]
		if s.DataRegion != q.DataRegion {
			continue
		}
		if q.JobFamily != "" && s.JobFamily != q.JobFamily {
			continue
		}
		if q.Status != "" && s.Status != q.Status {
			continue
		}
		if !q.IncludeExpired && s.ExpiresAt != nil && !s.ExpiresAt.After(timeNow()) {
			continue
		}
		matches = append(matches, s)
	}
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].CreatedAt.After(matches[j].CreatedAt)
	})
	if q.Cursor < 0 {
		q.Cursor = 0
	}
	if q.Cursor > len(matches) {
		return nil, nil
	}
	end := q.Cursor + limit
	if end > len(matches) {
		end = len(matches)
	}
	return matches[q.Cursor:end], nil
}
