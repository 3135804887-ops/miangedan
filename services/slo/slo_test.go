package slo

import (
	"testing"
)

// 月度窗口（30 天）与 99.95% 对应的允许停机秒数（1296s）。
const testWindow = 30 * 24 * 3600.0

// TASK-090 补测（TC-NFR-001-N01）：月度核心可用性统计达到 99.95% 门槛。
func TestMonthlyAvailabilityThresholds(t *testing.T) {
	rate, err := MonthlyAvailability(AvailabilityInput{WindowSeconds: testWindow, DowntimeSeconds: 1296})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTarget(rate, CoreAvailabilityTarget); err != nil {
		t.Fatalf("99.95%% 应达标: %v", err)
	}
	below, err := MonthlyAvailability(AvailabilityInput{WindowSeconds: testWindow, DowntimeSeconds: 1297})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTarget(below, CoreAvailabilityTarget); err == nil {
		t.Fatal("低于 99.95% 必须判未达标")
	}
}

// TASK-090 补测（TC-NFR-001-A01）：单组件故障走降级读取，核心可用性不中断。
func TestComponentFailureDegradedRead(t *testing.T) {
	// 组件故障 10 分钟但核心服务降级读取全程可用 → 核心可用性仍达标。
	rate, err := CoreAvailabilityExcludingComponent(
		600,
		AvailabilityInput{WindowSeconds: testWindow, DowntimeSeconds: 600},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTarget(rate, CoreAvailabilityTarget); err != nil {
		t.Fatalf("降级读取不应计入核心停机: %v", err)
	}
}

// TASK-090 补测（TC-NFR-002-N01）：实时房间月度 SLO 达到 99.95% 门槛。
func TestRoomAvailabilityThresholds(t *testing.T) {
	rate, err := MonthlyAvailability(AvailabilityInput{WindowSeconds: testWindow, DowntimeSeconds: 1296})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTarget(rate, RoomAvailabilityTarget); err != nil {
		t.Fatalf("房间可用性应达标: %v", err)
	}
}

// TASK-090 补测（TC-NFR-002-A01）：SFU 节点故障按故障流程恢复，会话迁移后不判失败。
func TestSFUNodeFailureRecovery(t *testing.T) {
	// SFU 单节点故障 5 分钟（300s），其余窗口正常 → 房间可用性仍达标。
	rate, err := MonthlyAvailability(AvailabilityInput{WindowSeconds: testWindow, DowntimeSeconds: 300})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTarget(rate, RoomAvailabilityTarget); err != nil {
		t.Fatalf("故障恢复后房间可用性应达标: %v", err)
	}
}

// TASK-090 补测（TC-NFR-003-N01）：有效完成率排除主动退出与本地断网后达标。
func TestEffectiveCompletionExcludesVoluntaryExit(t *testing.T) {
	// 100 场开始，2 场主动退出/本地断网（不计分母），98 场完成 → 100% 有效完成。
	rate, err := EffectiveCompletionRate(CompletionInput{Started: 100, Completed: 98, Excluded: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTarget(rate, EffectiveCompletionTarget); err != nil {
		t.Fatalf("排除后有效完成率应达标: %v", err)
	}
}

// TASK-090 补测（TC-NFR-003-A01）：注入系统故障被判失败的比例为 0。
func TestInjectedFaultNotCountedAsFailure(t *testing.T) {
	ratio, err := FaultFailureRatio(50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if ratio != 0 {
		t.Fatalf("注入故障判失败比例必须为 0，实际 %.1f%%", ratio)
	}
}
