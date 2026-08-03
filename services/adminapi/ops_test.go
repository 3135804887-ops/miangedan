// Package adminapi 运营后台测试（TASK-080；FR-037，US-08 场景 1）。
package adminapi

import (
	"context"
	"errors"
	"testing"
)

var opsActor = Actor{StaffID: "staff-ops", DataRegion: "cn", Role: RoleOps}

func newTestService(t *testing.T) (*Service, *MemoryStore) {
	t.Helper()
	store := NewMemoryStore()
	svc, err := NewService(store)
	if err != nil {
		t.Fatalf("创建服务失败: %v", err)
	}
	return svc, store
}

func seedProvider(t *testing.T, store *MemoryStore) {
	t.Helper()
	provider := ProviderInfo{
		ProviderID: "psp_cn_primary", Capability: CapabilityAvatar, Region: "cn",
		Status: ProviderActive, CircuitBreaker: BreakerClosed,
	}
	if err := store.SaveProvider(provider); err != nil {
		t.Fatalf("保存供应商失败: %v", err)
	}
}

// 区域监控：匿名房间与技术指标（不含身份与内容）。
func TestAnonymousRoomsAndRegionStatus(t *testing.T) {
	svc, store := newTestService(t)
	seedProvider(t, store)
	snapshot := RoomSnapshot{
		Region: "cn", AnonymousSessionID: "anon-0001",
		State: "LIVE", DurationSeconds: 120, FaultCode: "fault_asr_timeout",
	}
	if err := svc.RecordRoomSnapshot(context.Background(), opsActor, snapshot); err != nil {
		t.Fatalf("记录房间快照失败: %v", err)
	}
	if err := svc.RecordProviderHealth(context.Background(), opsActor, ProviderHealth{
		ProviderID: "psp_cn_primary", Capability: CapabilityAvatar, Status: ProviderActive,
		LatencyP95Ms: 800, ErrorRate: 0.01, CircuitBreaker: BreakerClosed,
	}); err != nil {
		t.Fatalf("记录供应商健康失败: %v", err)
	}
	status := RegionOpsStatus{
		DataRegion: "cn", OnlineRooms: 1, QueuedSessions: 0, Capacity: 100,
		SLO: map[string]float64{"availability": 0.999},
	}
	if err := store.SaveRegionStatus(status); err != nil {
		t.Fatalf("保存区域状态失败: %v", err)
	}
	rooms, err := svc.ListRooms(context.Background(), opsActor, "cn")
	if err != nil || len(rooms) != 1 || rooms[0].AnonymousSessionID != "anon-0001" {
		t.Fatalf("房间列表异常：%+v err=%v", rooms, err)
	}
	got, err := svc.ListRegionStatus(context.Background(), opsActor, "cn")
	if err != nil || got.OnlineRooms != 1 || got.Capacity != 100 {
		t.Fatalf("区域状态异常：%+v err=%v", got, err)
	}
}

// 供应商状态变更：停用必须记录原因并写审计。
func TestProviderStatusChangeRequiresReasonAndAudit(t *testing.T) {
	svc, store := newTestService(t)
	seedProvider(t, store)
	if _, err := svc.UpdateProviderStatus(context.Background(), opsActor,
		"psp_cn_primary", ProviderDisabled, 0, ""); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("停用必须记录原因，实际 err=%v", err)
	}
	updated, err := svc.UpdateProviderStatus(context.Background(), opsActor,
		"psp_cn_primary", ProviderDisabled, 0, "紧急故障切换")
	if err != nil || updated.Status != ProviderDisabled {
		t.Fatalf("停用失败：%+v err=%v", updated, err)
	}
	audits, _ := svc.store.ListAudits("cn")
	hasAudit := false
	for _, a := range audits {
		if a.Action == "provider.status_changed" {
			hasAudit = true
		}
	}
	if !hasAudit {
		t.Fatal("状态变更应写审计")
	}
}

// 运营红线：加入/旁听/代答一律拒绝并写审计。
func TestOperatorCannotJoinOrEavesdrop(t *testing.T) {
	svc, _ := newTestService(t)
	for _, action := range []string{"join", "eavesdrop", "answer"} {
		if err := svc.OperatorSessionGuard(context.Background(), opsActor, action); !errors.Is(err, ErrForbidden) {
			t.Fatalf("%s 应被拒绝，实际 err=%v", action, err)
		}
	}
	audits, _ := svc.store.ListAudits("cn")
	if len(audits) != 3 {
		t.Fatalf("拒绝尝试应写审计，实际 %d 条", len(audits))
	}
}

// 角色与跨区：非 ops 角色拒绝；跨区查询拒绝。
func TestRoleAndRegionGuards(t *testing.T) {
	svc, _ := newTestService(t)
	support := Actor{StaffID: "staff-support", DataRegion: "cn", Role: RoleSupport}
	if _, err := svc.ListRooms(context.Background(), support, "cn"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("support 查房间应被拒，实际 err=%v", err)
	}
	euOps := Actor{StaffID: "staff-eu", DataRegion: "eu", Role: RoleOps}
	if _, err := svc.ListRooms(context.Background(), euOps, "cn"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("跨区查询应被拒，实际 err=%v", err)
	}
}
