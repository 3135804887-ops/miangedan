// Package slo 提供上线可用性与有效完成率门槛校验（TASK-090，NFR-001~NFR-003）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-090；docs/testing/ACCEPTANCE-MATRIX.md
// TC-NFR-001/002/003；docs/security/THREAT-MODEL.md（系统故障不判失败）。
package slo

import (
	"errors"
	"fmt"
)

// 可用性/有效完成率门槛（PRD NFR-001/NFR-002：≥99.95%；NFR-003：≥99.5%）。
const (
	CoreAvailabilityTarget    = 99.95
	RoomAvailabilityTarget    = 99.95
	EffectiveCompletionTarget = 99.5
)

// AvailabilityInput 为月度可用性统计输入（窗口与故障秒数）。
type AvailabilityInput struct {
	WindowSeconds   float64
	DowntimeSeconds float64
}

// MonthlyAvailability 计算窗口内可用性百分比。
func MonthlyAvailability(in AvailabilityInput) (float64, error) {
	if in.WindowSeconds <= 0 {
		return 0, fmt.Errorf("window_seconds 必须 >0")
	}
	if in.DowntimeSeconds < 0 || in.DowntimeSeconds > in.WindowSeconds {
		return 0, fmt.Errorf("downtime_seconds 越界（0 ≤ d ≤ window）")
	}
	return (1 - in.DowntimeSeconds/in.WindowSeconds) * 100, nil
}

// ValidateTarget 校验实际可用性/完成率是否达到门槛（未达标返回错误，部署/发布门禁据此阻断）。
func ValidateTarget(actual, target float64) error {
	if actual+1e-9 < target {
		return fmt.Errorf("实际 %.4f%% 低于门槛 %.2f%%", actual, target)
	}
	return nil
}

// CompletionInput 为有效完成率统计输入。
// Excluded 为主动退出与本地断网等非系统责任样本（不计入分母，也不判失败）。
type CompletionInput struct {
	Started   int
	Completed int
	Excluded  int
}

// EffectiveCompletionRate 计算有效完成率（NFR-003：排除主动退出与本地断网后统计达标）。
func EffectiveCompletionRate(in CompletionInput) (float64, error) {
	eligible := in.Started - in.Excluded
	if eligible <= 0 {
		return 0, errors.New("有效样本必须 >0")
	}
	if in.Completed < 0 || in.Completed > eligible {
		return 0, fmt.Errorf("completed 越界（0 ≤ completed ≤ eligible）")
	}
	return float64(in.Completed) / float64(eligible) * 100, nil
}

// FaultFailureRatio 计算注入系统故障中被判失败的比例（NFR-003-A01：必须为 0）。
func FaultFailureRatio(faults, faultFailures int) (float64, error) {
	if faults <= 0 {
		return 0, errors.New("faults 必须 >0")
	}
	if faultFailures < 0 || faultFailures > faults {
		return 0, fmt.Errorf("fault_failures 越界（0 ≤ f ≤ faults）")
	}
	return float64(faultFailures) / float64(faults) * 100, nil
}

// CoreAvailabilityExcludingComponent 计算单组件故障降级后仍计入核心可用性的统计
// （NFR-001-A01：组件故障走降级读取，不中断核心服务，不计入核心不可用时间）。
func CoreAvailabilityExcludingComponent(componentDowntime float64, in AvailabilityInput) (float64, error) {
	coreDowntime := in.DowntimeSeconds - componentDowntime
	if coreDowntime < 0 {
		return 0, errors.New("component_downtime 不能超过总停机时间")
	}
	return MonthlyAvailability(AvailabilityInput{
		WindowSeconds:   in.WindowSeconds,
		DowntimeSeconds: coreDowntime,
	})
}
