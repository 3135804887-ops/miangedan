// Package adminapi 版本治理测试（TASK-081；FR-038，US-08 场景 2/6）。
package adminapi

import (
	"context"
	"errors"
	"testing"
)

var configActor = Actor{StaffID: "staff-ai", DataRegion: "cn", Role: RoleAIConfig}

func mustRegisterVersion(t *testing.T, svc *Service, assetType, assetKey string,
	compatible, safety bool) ArtifactVersion {
	t.Helper()
	v, err := svc.RegisterVersion(context.Background(), configActor, ArtifactVersion{
		AssetType: assetType, AssetKey: assetKey, Version: "v1.0.0",
		Compatible: compatible, SafetyTested: safety,
	})
	if err != nil {
		t.Fatalf("注册版本失败: %v", err)
	}
	return v
}

// 阶段门禁：未过兼容/安全测试不可灰度；未过指标不可放量。
func TestPromotionGates(t *testing.T) {
	svc, _ := newTestService(t)
	v := mustRegisterVersion(t, svc, AssetModel, "llm-main", false, false)
	if _, err := svc.PromoteVersion(context.Background(), configActor,
		v.VersionID, StageCanary); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("未过门槛灰度应被阻止，实际 err=%v", err)
	}
	// 补齐门槛后逐级推进。
	v.Compatible = true
	v.SafetyTested = true
	if err := svc.store.UpdateVersion(v); err != nil {
		t.Fatalf("更新版本失败: %v", err)
	}
	if _, err := svc.PromoteVersion(context.Background(), configActor,
		v.VersionID, StageShadow); err != nil {
		t.Fatalf("升影子失败: %v", err)
	}
	canary, err := svc.PromoteVersion(context.Background(), configActor,
		v.VersionID, StageCanary)
	if err != nil || canary.Stage != StageCanary {
		t.Fatalf("升灰度失败：%+v err=%v", canary, err)
	}
	if _, err := svc.PromoteVersion(context.Background(), configActor,
		v.VersionID, StageFull); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("未过指标放量应被阻止，实际 err=%v", err)
	}
	canary.MetricsOK = true
	_ = svc.store.UpdateVersion(canary)
	full, err := svc.PromoteVersion(context.Background(), configActor, canary.VersionID, StageFull)
	if err != nil || full.Stage != StageFull {
		t.Fatalf("放量失败：%+v err=%v", full, err)
	}
}

// 冻结：项目固定开始版本；不可重复固定；进行中会话不可回滚。
func TestFreezeAndRollback(t *testing.T) {
	svc, store := newTestService(t)
	v := mustRegisterVersion(t, svc, AssetPrompt, "main-prompt", true, true)
	pin, err := svc.FreezeVersion(context.Background(), configActor, "p-1", v.VersionID)
	if err != nil || pin.VersionID != v.VersionID {
		t.Fatalf("固定版本失败：%+v err=%v", pin, err)
	}
	if _, err := svc.FreezeVersion(context.Background(), configActor, "p-1", v.VersionID); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("重复固定应被阻止，实际 err=%v", err)
	}
	store.activeSess["cn|p-1"] = true
	if _, err := svc.RollbackVersion(context.Background(), configActor, "p-1", v.VersionID); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("进行中会话回滚应被阻止，实际 err=%v", err)
	}
	store.activeSess["cn|p-1"] = false
	stable := mustRegisterVersion(t, svc, AssetPrompt, "stable-prompt", true, true)
	rolled, err := svc.RollbackVersion(context.Background(), configActor, "p-1", stable.VersionID)
	if err != nil || rolled.VersionID != stable.VersionID {
		t.Fatalf("回滚失败：%+v err=%v", rolled, err)
	}
}

// 量表停用：三方审批齐全才可停用；缺审批拒绝。
func TestRubricDeprecateRequiresThreeApprovals(t *testing.T) {
	svc, _ := newTestService(t)
	v := mustRegisterVersion(t, svc, AssetRubric, "rubric-main", true, true)
	governance := Actor{StaffID: "staff-governance", DataRegion: "cn", Role: RoleScoringGovernance}
	if _, err := svc.DeprecateRubric(context.Background(), governance,
		"rubric-main", "系统性偏差", map[string]string{"product": "a"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("缺审批应拒绝，实际 err=%v", err)
	}
	deprecated, err := svc.DeprecateRubric(context.Background(), governance,
		"rubric-main", "系统性偏差", map[string]string{
			"product": "a", "interview_professional": "b", "safety_fairness": "c",
		})
	if err != nil || !deprecated.Deprecated {
		t.Fatalf("停用量表失败：%+v err=%v", deprecated, err)
	}
	_ = v
}

// 角色门禁：非 ai_config 角色不可注册/推进版本。
func TestVersionRoleGuard(t *testing.T) {
	svc, _ := newTestService(t)
	support := Actor{StaffID: "staff-support", DataRegion: "cn", Role: RoleSupport}
	if _, err := svc.RegisterVersion(context.Background(), support, ArtifactVersion{
		AssetType: AssetWorkflow, AssetKey: "wf", Version: "v1",
	}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("support 注册版本应被拒，实际 err=%v", err)
	}
}
