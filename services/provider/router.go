package provider

import (
	"errors"
	"fmt"
	"sync"

	"miangedan/services/region"
)

// RouteOptions 为路由选项。
type RouteOptions struct {
	Language string
}

// Router 按（数据区、能力、语言）路由新会话；活跃正式面试经 Pin/Resolve 钉扎版本与供应商
// （§7.3：不中途无记录切换；§9 版本 pin）。
type Router struct {
	registry *Registry
	mu       sync.RWMutex
	pins     map[string]PinnedProvider
}

// PinnedProvider 为会话钉扎结果。
type PinnedProvider struct {
	ProviderID string
	Version    string
	Capability Capability
	DataRegion string
}

// NewRouter 创建路由。
func NewRouter(registry *Registry) *Router {
	return &Router{registry: registry, pins: make(map[string]PinnedProvider)}
}

// Route 为新会话选择已验证健康供应商；主 open 时切 secondary；无可用 → ErrNoVerifiedProvider。
func (rt *Router) Route(capability Capability, dataRegion string, opts RouteOptions) (Info, error) {
	if err := validateRouteArgs(capability, dataRegion, opts); err != nil {
		return Info{}, err
	}
	candidates := rt.registry.Providers(capability, dataRegion)
	if len(candidates) == 0 {
		return Info{}, ErrNoVerifiedProvider
	}
	for _, info := range candidates {
		if opts.Language != "" && !contains(info.Languages, opts.Language) {
			continue
		}
		b, err := rt.registry.Breaker(info.ProviderID)
		if err != nil {
			continue
		}
		if b.Allow() {
			return info, nil
		}
	}
	return Info{}, ErrNoVerifiedProvider
}

// Pin 为新会话钉扎供应商（含版本）；钉扎后活跃正式面试不得无记录切换（§7.3、§9）。
func (rt *Router) Pin(sessionKey string, info Info) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.pins[sessionKey] = PinnedProvider{
		ProviderID: info.ProviderID,
		Version:    info.Version,
		Capability: info.Capability,
		DataRegion: info.DataRegion,
	}
}

// Resolve 返回会话钉扎的供应商；被停用/不可用时返回 ErrPinnedUnavailable（不得静默切换）。
func (rt *Router) Resolve(sessionKey string) (PinnedProvider, error) {
	rt.mu.RLock()
	pin, ok := rt.pins[sessionKey]
	rt.mu.RUnlock()
	if !ok {
		return PinnedProvider{}, errors.New("session not pinned")
	}
	info, err := rt.registry.Provider(pin.ProviderID)
	if err != nil || info.Role == RoleDisabled || info.Version != pin.Version {
		return PinnedProvider{}, ErrPinnedUnavailable
	}
	return pin, nil
}

// RecordSuccess 记录供应商调用成功（熔断 half_open 恢复依据）。
func (rt *Router) RecordSuccess(providerID string) {
	if b, err := rt.registry.Breaker(providerID); err == nil {
		b.RecordSuccess()
	}
}

// RecordFailure 记录供应商调用失败（熔断指标；错误分类见 §7.1）。
func (rt *Router) RecordFailure(providerID string) {
	if b, err := rt.registry.Breaker(providerID); err == nil {
		b.RecordFailure()
	}
}

func validateRouteArgs(capability Capability, dataRegion string, opts RouteOptions) error {
	if !validCapability(capability) {
		return fmt.Errorf("%w: 未知能力", ErrInvalidProvider)
	}
	if err := region.ValidateDataRegion(dataRegion); err != nil {
		return err
	}
	if opts.Language != "" && !contains(AllLanguages, opts.Language) {
		return fmt.Errorf("%w: 未知语言", ErrInvalidProvider)
	}
	return nil
}
