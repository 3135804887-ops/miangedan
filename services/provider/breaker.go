package provider

import (
	"sync"
	"time"
)

// BreakerState 为熔断状态（PROVIDER-ADAPTERS §7.2）。
type BreakerState string

// 熔断状态。
const (
	StateClosed   BreakerState = "closed"
	StateOpen     BreakerState = "open"
	StateHalfOpen BreakerState = "half_open"
)

// BreakerConfig 为熔断参数。
type BreakerConfig struct {
	FailureThreshold int
	OpenCoolDown     time.Duration
	HalfOpenMax      int
	SuccessThreshold int
}

// DefaultBreakerConfig 返回默认熔断参数（合成值，随被动指标校准）。
func DefaultBreakerConfig() BreakerConfig {
	return BreakerConfig{
		FailureThreshold: 5,
		OpenCoolDown:     30 * time.Second,
		HalfOpenMax:      1,
		SuccessThreshold: 2,
	}
}

// CircuitBreaker 为每（区 × 供应商 × 能力）熔断器（§7.2：closed → open → half_open → closed）。
type CircuitBreaker struct {
	mu           sync.Mutex
	cfg          BreakerConfig
	state        BreakerState
	failures     int
	successes    int
	halfInFlight int
	openedAt     time.Time
	now          func() time.Time
}

// NewCircuitBreaker 创建熔断器。
func NewCircuitBreaker(cfg BreakerConfig, now func() time.Time) *CircuitBreaker {
	if now == nil {
		now = time.Now
	}
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.OpenCoolDown <= 0 {
		cfg.OpenCoolDown = 30 * time.Second
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = 1
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	return &CircuitBreaker{cfg: cfg, state: StateClosed, now: now}
}

// State 返回当前状态。
func (b *CircuitBreaker) State() BreakerState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

// Allow 判断请求是否放行；open 冷却结束后进入 half_open 放行探针流量。
func (b *CircuitBreaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case StateClosed:
		return true
	case StateOpen:
		if b.now().Sub(b.openedAt) >= b.cfg.OpenCoolDown {
			b.state = StateHalfOpen
			b.halfInFlight = 0
			b.successes = 0
		} else {
			return false
		}
		fallthrough
	case StateHalfOpen:
		if b.halfInFlight < b.cfg.HalfOpenMax {
			b.halfInFlight++
			return true
		}
		return false
	default:
		return false
	}
}

// RecordSuccess 记录成功：half_open 达标回 closed；closed 累计。
func (b *CircuitBreaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case StateHalfOpen:
		b.successes++
		if b.successes >= b.cfg.SuccessThreshold {
			b.state = StateClosed
			b.resetCounters()
		}
	case StateClosed:
		b.successes++
		b.failures = 0
	}
}

// RecordFailure 记录失败：closed 超阈值转 open；half_open 不达标回 open。
func (b *CircuitBreaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case StateClosed:
		b.failures++
		if b.failures >= b.cfg.FailureThreshold {
			b.state = StateOpen
			b.openedAt = b.now()
			b.resetCounters()
		}
	case StateHalfOpen:
		b.state = StateOpen
		b.openedAt = b.now()
		b.resetCounters()
	}
}

func (b *CircuitBreaker) resetCounters() {
	b.failures = 0
	b.successes = 0
	b.halfInFlight = 0
}
