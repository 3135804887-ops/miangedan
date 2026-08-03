// Package room 的 TASK-061 计费挂接（BILLING-STATE-MACHINE §5.3/§6）。
package room

import (
	"context"
	"errors"

	"miangedan/services/project"
)

// ErrInsufficientEntitlement 为余额不足（阻止轮次开始，402）。
var ErrInsufficientEntitlement = errors.New("insufficient entitlement")

// BillingReserveInput 为每轮开始前预留入参。
type BillingReserveInput struct {
	ProjectID        string
	RoundSequence    int
	AttemptID        string
	SessionID        string
	EstimatedSeconds int
}

// BillingAPI 为秒级账本能力（生产由 services/billing 实现；测试用桩）。
// 规则（BILLING-STATE-MACHINE）：只计数字人已连接且正式进行中（LIVE）秒数；
// 故障/等待/重连/认证暂停与降级后文字面试不计；拒绝降级=系统责任全额返还。
type BillingAPI interface {
	Reserve(context.Context, project.Actor, BillingReserveInput) error
	StartMetering(context.Context, project.Actor, string) error
	StopMetering(context.Context, project.Actor, string) error
	Settle(context.Context, project.Actor, string, string) error
	RefundFull(context.Context, project.Actor, string, string) error
}

// billingNoop 为默认空实现（未注入计费时挂接点无副作用；生产必须注入）。
type billingNoop struct{}

func (billingNoop) Reserve(context.Context, project.Actor, BillingReserveInput) error {
	return nil
}

func (billingNoop) StartMetering(context.Context, project.Actor, string) error { return nil }
func (billingNoop) StopMetering(context.Context, project.Actor, string) error  { return nil }
func (billingNoop) Settle(context.Context, project.Actor, string, string) error {
	return nil
}

func (billingNoop) RefundFull(context.Context, project.Actor, string, string) error {
	return nil
}
