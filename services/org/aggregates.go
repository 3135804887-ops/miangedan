// Package org 提供聚合分析（TASK-073；FR-036，US-07 场景 3；
// openapi /v1/orgs/{orgId}/aggregates）。
// 红线：细分群体 <10 人隐藏；无个人排行榜/排名/候选人搜索。
package org

import (
	"context"
	"sort"
)

// MinAggregateGroupSize 为小样本保护阈值（少于 10 人不展示指标）。
const MinAggregateGroupSize = 10

// AggregateGroup 为聚合分组（<10 人时 hidden 且不返回指标）。
type AggregateGroup struct {
	GroupKey         string
	MemberCount      int
	Hidden           bool
	CompletionRate   *float64
	DimensionAvg     map[string]*float64
	ImprovementTrend *float64
}

// AggregateResult 为聚合结果（平台不提供个人排名/搜索）。
type AggregateResult struct {
	DataRegion               string
	Groups                   []AggregateGroup
	PersonalRankingAvailable bool
}

// DimensionSample 为注入的单人单任务分数样本（由评分服务提供，机构侧不持久化个人分）。
type DimensionSample struct {
	UserID       string
	AssignmentID string
	Dimension    string
	Score        float64
	CompletedAt  int64 // 时间戳，用于提升趋势（首末对比）。
}

// ComputeAggregates 聚合分析：岗位类别/能力维度/完成率/提升趋势；
// 细分 <10 人隐藏；无个人排行榜/排名/候选人搜索。
func (s *Service) ComputeAggregates(
	_ context.Context, actor Actor, orgID, jobCategory string,
	samples []DimensionSample,
) (AggregateResult, error) {
	if err := s.require(actor, orgID, PermViewAggregates); err != nil {
		return AggregateResult{}, err
	}
	assignments, err := s.store.ListAssignments(actor.DataRegion, orgID)
	if err != nil {
		return AggregateResult{}, err
	}
	groups := make(map[string]*groupAccumulator)
	order := make([]string, 0)
	ensure := func(key string) *groupAccumulator {
		if g, ok := groups[key]; ok {
			return g
		}
		g := &groupAccumulator{key: key, seen: make(map[string]bool)}
		groups[key] = g
		order = append(order, key)
		return g
	}
	total := ensure("overall")
	assignmentCategory := make(map[string]string)
	for _, a := range assignments {
		if jobCategory != "" && a.JobCategory != jobCategory {
			continue
		}
		key := a.JobCategory
		if key == "" {
			key = "uncategorized"
		}
		assignmentCategory[a.AssignmentID] = key
		group := ensure(key)
		members, err := s.store.ListAssignmentMembers(a.AssignmentID)
		if err != nil {
			return AggregateResult{}, err
		}
		for _, m := range members {
			if !group.seen[m.UserID] {
				group.seen[m.UserID] = true
				group.memberCount++
			}
			if !total.seen[m.UserID] {
				total.seen[m.UserID] = true
				total.memberCount++
			}
			if m.Status == MemberCompleted {
				group.completedCount++
				total.completedCount++
			}
		}
	}
	// 维度均值与提升趋势（仅对样本所属任务的分组累计）。
	dimScores := make(map[string]map[string][]float64)
	trends := make(map[string]map[string]*firstLast)
	for _, smp := range samples {
		key := assignmentCategory[smp.AssignmentID]
		if key == "" {
			key = "overall"
		}
		if dimScores[key] == nil {
			dimScores[key] = make(map[string][]float64)
			trends[key] = make(map[string]*firstLast)
		}
		dimScores[key][smp.Dimension] = append(dimScores[key][smp.Dimension], smp.Score)
		fl := trends[key][smp.UserID]
		if fl == nil {
			fl = &firstLast{}
			trends[key][smp.UserID] = fl
		}
		if !fl.hasFirst || smp.CompletedAt < fl.firstAt {
			fl.first = smp.Score
			fl.firstAt = smp.CompletedAt
			fl.hasFirst = true
		}
		if smp.CompletedAt > fl.lastAt {
			fl.last = smp.Score
			fl.lastAt = smp.CompletedAt
			fl.hasLast = true
		}
	}
	result := AggregateResult{
		DataRegion:               actor.DataRegion,
		PersonalRankingAvailable: false,
	}
	for _, key := range order {
		g := groups[key]
		if g.memberCount == 0 {
			continue
		}
		group := AggregateGroup{GroupKey: key, MemberCount: g.memberCount}
		if g.memberCount < MinAggregateGroupSize {
			group.Hidden = true
			result.Groups = append(result.Groups, group)
			continue
		}
		if g.completedCount > 0 {
			rate := float64(g.completedCount) / float64(g.memberCount)
			group.CompletionRate = &rate
		}
		avg := make(map[string]*float64)
		for dim, scores := range dimScores[key] {
			if len(scores) == 0 {
				continue
			}
			sum := 0.0
			for _, v := range scores {
				sum += v
			}
			val := sum / float64(len(scores))
			avg[dim] = &val
		}
		group.DimensionAvg = avg
		if len(trends[key]) >= MinAggregateGroupSize {
			sum := 0.0
			count := 0
			for _, fl := range trends[key] {
				if fl.hasFirst && fl.hasLast {
					sum += fl.last - fl.first
					count++
				}
			}
			if count > 0 {
				val := sum / float64(count)
				group.ImprovementTrend = &val
			}
		}
		result.Groups = append(result.Groups, group)
	}
	sort.Slice(result.Groups, func(i, j int) bool {
		return result.Groups[i].GroupKey < result.Groups[j].GroupKey
	})
	return result, nil
}

// groupAccumulator 为分组内部累计状态。
type groupAccumulator struct {
	key            string
	seen           map[string]bool
	memberCount    int
	completedCount int
}

// firstLast 为单人首末分数（提升趋势）。
type firstLast struct {
	first    float64
	firstAt  int64
	last     float64
	lastAt   int64
	hasFirst bool
	hasLast  bool
}
