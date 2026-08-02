package provider

import (
	"testing"
	"time"
)

// 正常/异常路径：closed 超阈值 → open；冷却 → half_open 放行探针；达标 → closed；不达标 → open。
func TestCircuitBreakerTransitions(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	cfg := BreakerConfig{FailureThreshold: 3, OpenCoolDown: 30 * time.Second, HalfOpenMax: 1, SuccessThreshold: 2}
	b := NewCircuitBreaker(cfg, clock)

	for i := 0; i < 3; i++ {
		if !b.Allow() {
			t.Fatalf("closed 应放行（第 %d 次）", i+1)
		}
		b.RecordFailure()
	}
	if b.State() != StateOpen {
		t.Fatalf("3 次失败后应为 open，实际 %s", b.State())
	}
	if b.Allow() {
		t.Fatal("open 冷却期内必须拒绝")
	}
	now = now.Add(31 * time.Second)
	if !b.Allow() {
		t.Fatal("冷却结束后应进入 half_open 放行探针")
	}
	b.RecordFailure()
	if b.State() != StateOpen {
		t.Fatalf("half_open 失败应回 open，实际 %s", b.State())
	}
	now = now.Add(31 * time.Second)
	if !b.Allow() {
		t.Fatal("再次冷却后应放行")
	}
	b.RecordSuccess()
	b.RecordSuccess()
	if b.State() != StateClosed {
		t.Fatalf("half_open 连续成功应回 closed，实际 %s", b.State())
	}
}

// 幂等性：连续放行与状态查询结果确定。
func TestCircuitBreakerDeterministic(t *testing.T) {
	b := NewCircuitBreaker(DefaultBreakerConfig(), nil)
	for i := 0; i < 3; i++ {
		if !b.Allow() || b.State() != StateClosed {
			t.Fatal("默认状态必须 closed 且放行")
		}
	}
}
