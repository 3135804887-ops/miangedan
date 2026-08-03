// Package adminapi 禁止改分与破窗访问测试（TASK-082；FR-039，US-08 场景 3）。
package adminapi

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

var securityActor = Actor{StaffID: "staff-sec", DataRegion: "cn", Role: RolePrivacySecurity}

// 改分约束：编辑分数/解锁/改证据一律拒绝并写审计；无分数修改存储路径。
func TestScoreWriteBlockedWithAudit(t *testing.T) {
	svc, _ := newTestService(t)
	for _, action := range []string{"edit_score", "unlock_round", "edit_evidence"} {
		if err := svc.AttemptScoreWrite(context.Background(), securityActor, action, "p-1"); !errors.Is(err, ErrForbidden) {
			t.Fatalf("%s 应被拒绝，实际 err=%v", action, err)
		}
	}
	audits, _ := svc.store.ListAudits("cn")
	if len(audits) != 3 {
		t.Fatalf("拒绝尝试应写审计，实际 %d 条", len(audits))
	}
}

// 破窗访问：限定理由与时长；开启者不可自审；72 小时窗口内事后复核。
func TestBreakGlassOpenAndReview(t *testing.T) {
	svc, _ := newTestService(t)
	if _, err := svc.OpenBreakGlass(context.Background(), securityActor,
		"user-001", "", "u-1", 30); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("空理由应拒绝，实际 err=%v", err)
	}
	if _, err := svc.OpenBreakGlass(context.Background(), securityActor,
		"user-001", "法律事件", "u-1", 999); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("超长破窗应拒绝，实际 err=%v", err)
	}
	glass, err := svc.OpenBreakGlass(context.Background(), securityActor,
		"user-001", "重大安全事件", "session-1", 30)
	if err != nil {
		t.Fatalf("开启破窗失败: %v", err)
	}
	if _, err := svc.ReviewBreakGlass(context.Background(), securityActor,
		glass.GlassID, "approved", "自审"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("开启者自审应被拒，实际 err=%v", err)
	}
	reviewer := Actor{StaffID: "staff-sec2", DataRegion: "cn", Role: RolePrivacySecurity}
	review, err := svc.ReviewBreakGlass(context.Background(), reviewer,
		glass.GlassID, "approved", "复核通过")
	if err != nil || review.Decision != "approved" {
		t.Fatalf("事后复核失败：%+v err=%v", review, err)
	}
	if _, err := svc.ReviewBreakGlass(context.Background(), reviewer,
		glass.GlassID, "rejected", "重复"); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("重复复核应被拒，实际 err=%v", err)
	}
	_, status, err := svc.GetBreakGlass(context.Background(), reviewer, glass.GlassID)
	if err != nil || status != GlassReviewed {
		t.Fatalf("破窗状态应为 reviewed：%s err=%v", status, err)
	}
}

// 破窗到期自动 expired；敏感访问通知记录存在。
func TestBreakGlassExpiryAndNotificationTrace(t *testing.T) {
	svc, _ := newTestService(t)
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	glass, err := svc.OpenBreakGlass(context.Background(), securityActor,
		"user-002", "法律事件", "u-2", 30)
	if err != nil {
		t.Fatalf("开启破窗失败: %v", err)
	}
	svc.now = func() time.Time { return now.Add(31 * time.Minute) }
	_, status, err := svc.GetBreakGlass(context.Background(), securityActor, glass.GlassID)
	if err != nil || status != GlassExpired {
		t.Fatalf("破窗应过期：%s err=%v", status, err)
	}
	byTarget, err := svc.store.ListBreakGlassByTarget("cn", "user-002")
	if err != nil || len(byTarget) != 1 {
		t.Fatalf("目标用户破窗记录应可查：%+v err=%v", byTarget, err)
	}
}

// 存储接口只 SELECT/INSERT：破窗与审计方法无 Update/Delete 修改路径。
func TestScoreGuardStoreHasNoMutationPaths(t *testing.T) {
	storeType := reflect.TypeOf((*Store)(nil)).Elem()
	for i := 0; i < storeType.NumMethod(); i++ {
		name := storeType.Method(i).Name
		if strings.Contains(name, "Score") ||
			(strings.Contains(name, "BreakGlass") && strings.Contains(name, "Update")) ||
			strings.Contains(name, "Delete") {
			t.Fatalf("存储接口存在被禁止的修改路径: %s", name)
		}
	}
	// 审计仅 Append/List。
	for i := 0; i < storeType.NumMethod(); i++ {
		name := storeType.Method(i).Name
		if strings.HasPrefix(name, "Audit") && !strings.HasPrefix(name, "AppendAudit") &&
			!strings.HasPrefix(name, "ListAudits") {
			t.Fatalf("审计存储存在非追加路径: %s", name)
		}
	}
}
