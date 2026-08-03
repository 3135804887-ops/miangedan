package scoring

import (
	"fmt"
	"math"
	"strings"
)

// 岗位匹配度分列（SCORING-SPEC 6.8：必备/加分单独展示）。
const (
	BucketMustHave   = "must_have"
	BucketNiceToHave = "nice_to_have"
	ReasonNoJD       = "no_jd"
)

// ComputeJobMatch 计算岗位匹配度（TASK-042；SCORING-SPEC 6.8）。
// - match = Σ weight(已证明) / Σ weight(全部)，必备与加分分列；
// - "已证明" = 简历证据（仅当存在简历）∪ 面试证据（Result 引用）；
// - 无 JD：not_displayed_reason = no_jd，不展示匹配百分比；
// - 只有 JD（无简历）：只计算面试证明的覆盖，不得生成经历一致性评分；
// - 匹配度与面试分数相互独立，不作为单轮解锁的隐藏因素。
func ComputeJobMatch(in *JobMatchInput) (*JobMatch, error) {
	if in == nil {
		return nil, nil
	}
	if len(in.Requirements) == 0 {
		reason := ReasonNoJD
		empty := MatchBucket{MatchRatio: 0, Proven: []string{}, Unproven: []string{}}
		return &JobMatch{
			MustHave:           empty,
			NiceToHave:         empty,
			NotDisplayedReason: &reason,
		}, nil
	}
	if !in.ResumeAvailable && len(in.ProvenByResume) > 0 {
		return nil, fmt.Errorf("%w: 无简历时不得使用简历证明（SC-EC-21）", ErrInvalidInput)
	}
	byID := make(map[string]JobRequirement, len(in.Requirements))
	bucketWeight := map[string]int{BucketMustHave: 0, BucketNiceToHave: 0}
	for _, req := range in.Requirements {
		if strings.TrimSpace(req.RequirementID) == "" {
			return nil, fmt.Errorf("%w: requirement_id 必填", ErrInvalidInput)
		}
		if req.Bucket != BucketMustHave && req.Bucket != BucketNiceToHave {
			return nil, fmt.Errorf("%w: bucket 必须为 must_have | nice_to_have",
				ErrInvalidInput)
		}
		if req.Weight < 0 {
			return nil, fmt.Errorf("%w: 要求权重不能为负", ErrInvalidInput)
		}
		if _, dup := byID[req.RequirementID]; dup {
			return nil, fmt.Errorf("%w: 要求重复 %s", ErrInvalidInput, req.RequirementID)
		}
		byID[req.RequirementID] = req
		bucketWeight[req.Bucket] += req.Weight
	}
	proven := make(map[string]bool)
	if in.ResumeAvailable {
		for _, id := range in.ProvenByResume {
			if _, ok := byID[id]; !ok {
				return nil, fmt.Errorf("%w: 简历证明引用了不存在的 JD 要求 %s",
					ErrInvalidInput, id)
			}
			proven[id] = true
		}
	}
	for _, id := range in.ProvenByInterview {
		if _, ok := byID[id]; !ok {
			return nil, fmt.Errorf("%w: 面试证明引用了不存在的 JD 要求 %s",
				ErrInvalidInput, id)
		}
		proven[id] = true
	}
	compute := func(bucket string) MatchBucket {
		provenIDs, unprovenIDs := []string{}, []string{}
		provenWeight := 0
		totalWeight := bucketWeight[bucket]
		for _, req := range in.Requirements {
			if req.Bucket != bucket {
				continue
			}
			if proven[req.RequirementID] {
				provenIDs = append(provenIDs, req.RequirementID)
				provenWeight += req.Weight
			} else {
				unprovenIDs = append(unprovenIDs, req.RequirementID)
			}
		}
		ratio := 0.0
		if totalWeight > 0 {
			ratio = roundRatio(float64(provenWeight) / float64(totalWeight))
		}
		return MatchBucket{
			MatchRatio: ratio,
			Proven:     provenIDs,
			Unproven:   unprovenIDs,
		}
	}
	return &JobMatch{
		MustHave:   compute(BucketMustHave),
		NiceToHave: compute(BucketNiceToHave),
	}, nil
}

// roundRatio 将比率保留 4 位小数（确定性输出）。
func roundRatio(value float64) float64 {
	return math.Round(value*10000) / 10000
}
