// Package search 提供企业公开流程检索链路（TASK-015）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-015；PRD FR-007、FR-008；US-02 规则 1–4；
// docs/ai/PROVIDER-ADAPTERS.md 第 4.5 节（Search 能力契约）与第 7 节（重试/错误分类）；
// docs/security/SECURITY-REQUIREMENTS.md SEC-024、SEC-025；ADR-0003。
//
// 实现约束（TASK-030 未开工）：
//   - 搜索与 LLM 调用一律经本包定义的供应商中立能力接口（Adapter），业务代码禁止绑定厂商 SDK；
//   - 当前以合成桩（StubAdapter）落地契约；真实适配器随 TASK-030 实现并保持同一接口语义；
//   - 外部网页内容仅作为不可信数据进入结构化提取，绝不作为系统指令（SEC-024/SEC-025）。
package search

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"miangedan/services/source"
)

// Adapter 为供应商中立的搜索能力接口（PROVIDER-ADAPTERS §4.5 search_process）。
// 业务代码只依赖本接口；区域路由、健康检查、熔断与版本固定由 TASK-030 适配层实现。
type Adapter interface {
	// SearchProcess 按公司/岗位/级别/地区检索公开流程来源，返回来源元数据列表。
	// 实现方负责：只做结构化提取（不可信数据），禁止绕过网站协议/登录/验证码/反爬（SEC-025）。
	SearchProcess(ctx context.Context, q source.SearchQuery) ([]source.ProcessSource, error)
}

// Store 为来源持久化抽象（迁移 0002 已就绪；本任务提供内存实现，PostgreSQL 实现随存储接线任务落地）。
type Store interface {
	// Save 幂等保存来源：同幂等键重复写入返回 saved=false 且不产生重复副作用（NFR-006）。
	Save(ctx context.Context, s source.ProcessSource) (saved bool, err error)
}

// Service 为公开流程检索编排服务：
// 检索（经适配层）→ 元数据校验 → 可靠来源判定 → 无可靠来源自动回退通用模板并标记 AI 推导。
type Service struct {
	adapter Adapter
	store   Store
	now     func() time.Time

	retryMaxAttempts int
	retryable        func(error) bool
	backoffFn        func(attempt int) time.Duration

	mu      sync.Mutex
	results map[string]source.SearchResult // 幂等键 → 首次结果（建模数据库唯一约束）
}

// Options 为 Service 可选配置（默认值满足生产保守默认）。
type Options struct {
	Now              func() time.Time
	RetryMaxAttempts int
	Retryable        func(error) bool
	Backoff          func(attempt int) time.Duration
}

// NewService 创建检索编排服务；adapter 与 store 必填（fail-closed）。
func NewService(adapter Adapter, store Store, opts Options) (*Service, error) {
	if adapter == nil {
		return nil, errors.New("搜索适配器为空（业务代码必须经适配层，ADR-0003）")
	}
	if store == nil {
		return nil, errors.New("来源存储为空")
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	maxAttempts := opts.RetryMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3 // 1 次初始 + 2 次重试（PROVIDER-ADAPTERS §7.1：可重试错误指数退避重试 ≤2 次）
	}
	retryable := opts.Retryable
	if retryable == nil {
		retryable = IsRetryable
	}
	backoffFn := opts.Backoff
	if backoffFn == nil {
		backoffFn = backoff
	}
	return &Service{
		adapter:          adapter,
		store:            store,
		now:              now,
		retryMaxAttempts: maxAttempts,
		retryable:        retryable,
		backoffFn:        backoffFn,
		results:          make(map[string]source.SearchResult),
	}, nil
}

// IsRetryable 按 PROVIDER-ADAPTERS §7.1 分类错误：网络超时/限流/5xx 可重试；
// 鉴权失败、配额耗尽、参数错误等配置类错误不可重试。
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	var e *RetryableError
	return errors.As(err, &e) && e.Retryable
}

// RetryableError 为可重试错误包装。
type RetryableError struct {
	Err       error
	Retryable bool
}

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

// Retryable 包装可重试错误。
func Retryable(err error) error {
	return &RetryableError{Err: err, Retryable: true}
}

// NonRetryable 包装不可重试错误。
func NonRetryable(err error) error {
	return &RetryableError{Err: err, Retryable: false}
}

// Search 执行公开流程检索并输出来源元数据；无可靠来源自动回退通用模板并标记 AI 推导（FR-008）。
// 幂等：同一 idempotencyKey 重复调用返回首次结果（NFR-006）。
func (s *Service) Search(ctx context.Context, q source.SearchQuery, idempotencyKey string) (source.SearchResult, error) {
	if ctx == nil {
		return source.SearchResult{}, errors.New("context 不能为 nil")
	}
	if err := q.Validate(); err != nil {
		return source.SearchResult{}, fmt.Errorf("检索条件非法: %w", err)
	}
	if len(idempotencyKey) < 8 {
		return source.SearchResult{}, errors.New("幂等键过短（Idempotency-Key 至少 8 字符，NFR-006）")
	}

	s.mu.Lock()
	if cached, ok := s.results[idempotencyKey]; ok {
		s.mu.Unlock()
		return cached, nil
	}
	s.mu.Unlock()

	result, err := s.searchOnce(ctx, q, idempotencyKey)
	if err != nil {
		return source.SearchResult{}, err
	}

	s.mu.Lock()
	// 并发同键：后到者复用首次结果，保证重复副作用为 0（NFR-006）。
	if cached, ok := s.results[idempotencyKey]; ok {
		s.mu.Unlock()
		return cached, nil
	}
	s.results[idempotencyKey] = result
	s.mu.Unlock()
	return result, nil
}

func (s *Service) searchOnce(ctx context.Context, q source.SearchQuery, idempotencyKey string) (source.SearchResult, error) {
	now := s.now()

	// 无公司信息：不发起检索，直接回退通用模板并标记 AI 推导（US-02 规则 3、FR-008）。
	if trimSpace(q.Company) == "" {
		return s.fallback(ctx, q, idempotencyKey, source.FallbackMissingCompany, now), nil
	}

	items, err := s.callAdapterWithRetry(ctx, q)
	if err != nil {
		// 断网/检索服务故障：自动回退通用模板并标记 AI 推导（US-02 场景 2）。
		return s.fallback(ctx, q, idempotencyKey, source.FallbackSearchFailed, now), nil
	}

	valid := make([]source.ProcessSource, 0, len(items))
	for _, item := range items {
		if err := item.Validate(); err != nil {
			// 适配层返回的非法元数据丢弃并告警（日志不含正文；由观测层埋点），不阻塞链路。
			continue
		}
		valid = append(valid, item)
	}

	reliable := false
	for _, item := range valid {
		if item.IsReliable(now) {
			reliable = true
			break
		}
	}
	if !reliable {
		// 来源全部失效/不可信/仅候选人经验：回退通用模板并标记 AI 推导（US-02 场景 2、FR-008）。
		return s.fallback(ctx, q, idempotencyKey, source.FallbackNoReliable, now), nil
	}

	// 保存来源元数据（幂等去重：同幂等键重复写入不产生重复副作用）。
	for _, item := range valid {
		if _, err := s.store.Save(ctx, item); err != nil {
			return source.SearchResult{}, fmt.Errorf("保存来源元数据失败: %w", err)
		}
	}

	source.SortByPriority(valid)
	return source.SearchResult{
		Sources:                 valid,
		FlowUsesGenericTemplate: false,
		AIDerived:               false,
		DataRegion:              q.Region,
	}, nil
}

// fallback 构造通用模板回退结果：AI 推导标记 + 无可靠来源原因（US-02 场景 2）。
func (s *Service) fallback(ctx context.Context, q source.SearchQuery, idempotencyKey string, reason source.FallbackReason, now time.Time) source.SearchResult {
	tpl := source.NewGenericTemplate(q.Region, "general", now, idempotencyKey)
	if _, err := s.store.Save(ctx, tpl); err != nil {
		// 存储故障不改变回退结论；来源为通用模板时允许不落库（避免回退被阻塞），告警由观测层负责。
		_ = err
	}
	return source.SearchResult{
		Sources:                 []source.ProcessSource{tpl},
		FlowUsesGenericTemplate: true,
		AIDerived:               true,
		FallbackReason:          reason,
		DataRegion:              q.Region,
	}
}

// callAdapterWithRetry 按 PROVIDER-ADAPTERS §7.1 对可重试错误执行有限重试（≤2 次重试）。
func (s *Service) callAdapterWithRetry(ctx context.Context, q source.SearchQuery) ([]source.ProcessSource, error) {
	var lastErr error
	for attempt := 1; attempt <= s.retryMaxAttempts; attempt++ {
		items, err := s.adapter.SearchProcess(ctx, q)
		if err == nil {
			return items, nil
		}
		lastErr = err
		if !s.retryable(err) || attempt == s.retryMaxAttempts {
			break
		}
		select {
		case <-time.After(s.backoffFn(attempt)):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}

// backoff 为指数退避（1s、2s）；测试通过短超时用例验证重试行为，不使用 sleep 依赖。
func backoff(attempt int) time.Duration {
	if attempt <= 1 {
		return time.Second
	}
	return time.Duration(1<<(attempt-1)) * time.Second
}

func trimSpace(v string) string {
	out := v
	for len(out) > 0 && (out[0] == ' ' || out[0] == '\t' || out[0] == '\n' || out[0] == '\r') {
		out = out[1:]
	}
	for len(out) > 0 && (out[len(out)-1] == ' ' || out[len(out)-1] == '\t' || out[len(out)-1] == '\n' || out[len(out)-1] == '\r') {
		out = out[:len(out)-1]
	}
	return out
}
