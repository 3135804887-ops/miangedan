// Package org 聚合分析测试（TASK-073；FR-036，US-07 场景 3）。
package org

import (
	"context"
	"testing"
	"time"
)

func seedAssignmentMembers(t *testing.T, svc *Service, orgID, assignmentID string, count, completed int) {
	t.Helper()
	for i := 0; i < count; i++ {
		userID := "agg-user-" + string(rune('a'+i))
		status := MemberNotStarted
		var completedAt *time.Time
		if i < completed {
			status = MemberCompleted
			now := time.Now().UTC()
			completedAt = &now
		}
		member := AssignmentMember{
			AssignmentID: assignmentID,
			OrgID:        orgID,
			UserID:       userID,
			Status:       status,
			CompletedAt:  completedAt,
		}
		if err := svc.store.SaveAssignmentMember(member); err != nil {
			t.Fatalf("保存成员失败: %v", err)
		}
	}
}

// ≥10 人细分正常展示完成率/维度均值/提升趋势。
func TestAggregateOverTenShowsMetrics(t *testing.T) {
	svc, _ := newTestService(t)
	org := mustOrg(t, svc, "idem-org-g1")
	assignment, err := svc.CreateAssignment(context.Background(), testActor,
		org.OrgID, sampleAssignmentInput(), "idem-assign-g1")
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	seedAssignmentMembers(t, svc, org.OrgID, assignment.AssignmentID, 12, 8)
	samples := make([]DimensionSample, 0)
	for i := 0; i < 8; i++ {
		samples = append(samples, DimensionSample{
			UserID:       "agg-user-" + string(rune('a'+i)),
			AssignmentID: assignment.AssignmentID,
			Dimension:    "communication",
			Score:        70 + float64(i),
			CompletedAt:  1000,
		})
	}
	result, err := svc.ComputeAggregates(context.Background(), testActor,
		org.OrgID, "", samples)
	if err != nil {
		t.Fatalf("聚合失败: %v", err)
	}
	found := false
	for _, g := range result.Groups {
		if g.GroupKey == "backend" {
			found = true
			if g.Hidden || g.MemberCount != 12 || g.CompletionRate == nil {
				t.Fatalf("≥10 人群组应展示指标：%+v", g)
			}
			if g.DimensionAvg["communication"] == nil {
				t.Fatalf("维度均值缺失：%+v", g.DimensionAvg)
			}
		}
	}
	if !found {
		t.Fatalf("缺少 backend 分组：%+v", result.Groups)
	}
	if result.PersonalRankingAvailable {
		t.Fatal("平台不应提供个人排名")
	}
}

// <10 人细分隐藏：不返回任何指标。
func TestAggregateUnderTenHidden(t *testing.T) {
	svc, _ := newTestService(t)
	org := mustOrg(t, svc, "idem-org-g2")
	assignment, err := svc.CreateAssignment(context.Background(), testActor,
		org.OrgID, sampleAssignmentInput(), "idem-assign-g2")
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	seedAssignmentMembers(t, svc, org.OrgID, assignment.AssignmentID, 9, 5)
	result, err := svc.ComputeAggregates(context.Background(), testActor,
		org.OrgID, "", nil)
	if err != nil {
		t.Fatalf("聚合失败: %v", err)
	}
	for _, g := range result.Groups {
		if g.GroupKey == "backend" {
			if !g.Hidden || g.CompletionRate != nil || g.DimensionAvg != nil || g.ImprovementTrend != nil {
				t.Fatalf("<10 人群组应隐藏且无指标：%+v", g)
			}
		}
	}
}

// 按岗位类别过滤；overall 汇总始终存在。
func TestAggregateFilterByCategory(t *testing.T) {
	svc, _ := newTestService(t)
	org := mustOrg(t, svc, "idem-org-g3")
	assignment, err := svc.CreateAssignment(context.Background(), testActor,
		org.OrgID, sampleAssignmentInput(), "idem-assign-g3")
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	seedAssignmentMembers(t, svc, org.OrgID, assignment.AssignmentID, 10, 10)
	result, err := svc.ComputeAggregates(context.Background(), testActor,
		org.OrgID, "backend", nil)
	if err != nil {
		t.Fatalf("聚合失败: %v", err)
	}
	found := false
	for _, g := range result.Groups {
		if g.GroupKey == "backend" {
			found = true
			if g.CompletionRate == nil || *g.CompletionRate != 1.0 {
				t.Fatalf("完成率异常：%+v", g.CompletionRate)
			}
		}
	}
	if !found {
		t.Fatalf("按类别过滤应包含 backend：%+v", result.Groups)
	}
}

// 无个人排行榜/排名/候选人搜索：结果常量 false，且无个人粒度方法。
func TestNoPersonalRankingSurface(t *testing.T) {
	svc, _ := newTestService(t)
	org := mustOrg(t, svc, "idem-org-g4")
	assignment, err := svc.CreateAssignment(context.Background(), testActor,
		org.OrgID, sampleAssignmentInput(), "idem-assign-g4")
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}
	seedAssignmentMembers(t, svc, org.OrgID, assignment.AssignmentID, 10, 10)
	result, err := svc.ComputeAggregates(context.Background(), testActor,
		org.OrgID, "", nil)
	if err != nil {
		t.Fatalf("聚合失败: %v", err)
	}
	if result.PersonalRankingAvailable {
		t.Fatal("个人排行榜必须不可用")
	}
}
