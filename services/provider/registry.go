package provider

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Registry 为供应商注册表（按数据区隔离，§5）。
type Registry struct {
	mu       sync.RWMutex
	byID     map[string]Info
	health   map[string]HealthFunc
	breakers map[string]*CircuitBreaker
	now      func() time.Time
}

// NewRegistry 创建空注册表。
func NewRegistry(now func() time.Time) *Registry {
	if now == nil {
		now = time.Now
	}
	return &Registry{
		byID:     make(map[string]Info),
		health:   make(map[string]HealthFunc),
		breakers: make(map[string]*CircuitBreaker),
		now:      now,
	}
}

// Register 注册供应商（重复 ID 拒绝；health 为空则视为不提供主动探测）。
func (r *Registry) Register(info Info, health HealthFunc) error {
	if err := ValidateInfo(info); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[info.ProviderID]; ok {
		return fmt.Errorf("%w: 供应商 %q 已注册", ErrInvalidProvider, info.ProviderID)
	}
	r.byID[info.ProviderID] = info
	r.health[info.ProviderID] = health
	r.breakers[info.ProviderID] = NewCircuitBreaker(DefaultBreakerConfig(), r.now)
	return nil
}

// SetStatus 置供应商角色（紧急停用 → disabled，US-08 场景 5）。
func (r *Registry) SetStatus(providerID string, role Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	info, ok := r.byID[providerID]
	if !ok {
		return ErrNoVerifiedProvider
	}
	info.Role = role
	r.byID[providerID] = info
	return nil
}

// Providers 返回指定能力与区域下的供应商（按角色 primary → secondary 排序）。
func (r *Registry) Providers(capability Capability, dataRegion string) []Info {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Info
	for _, info := range r.byID {
		if info.Capability == capability && info.DataRegion == dataRegion && info.Role != RoleDisabled {
			out = append(out, info)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		ri, rj := out[i].Role, out[j].Role
		if ri != rj {
			return ri == RolePrimary
		}
		return out[i].ProviderID < out[j].ProviderID
	})
	return out
}

// Provider 按 ID 读取供应商信息。
func (r *Registry) Provider(providerID string) (Info, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.byID[providerID]
	if !ok {
		return Info{}, ErrNoVerifiedProvider
	}
	return info, nil
}

// Breaker 返回供应商熔断器。
func (r *Registry) Breaker(providerID string) (*CircuitBreaker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.breakers[providerID]
	if !ok {
		return nil, ErrNoVerifiedProvider
	}
	return b, nil
}

// Health 主动健康探测（§6 低频合成探针）。
func (r *Registry) Health(providerID string) error {
	r.mu.RLock()
	fn, ok := r.health[providerID]
	r.mu.RUnlock()
	if !ok {
		return errors.New("provider not registered")
	}
	if fn == nil {
		return nil // 未配置主动探测视为健康（以被动指标为准，§11）
	}
	return fn()
}
