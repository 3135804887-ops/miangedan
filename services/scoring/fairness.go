package scoring

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// FairnessMetadata 为公平性监控切分元数据（SCORING-SPEC 10.4 / PROMPT-POLICY 10.1）。
// 只记录受控类别值，不携带任何用户敏感内容或原始回答。
type FairnessMetadata struct {
	InterviewLanguage string   `json:"interview_language"`
	InputMode         string   `json:"input_mode"`
	Accommodations    []string `json:"accommodations,omitempty"`
	JobFamily         string   `json:"job_family,omitempty"`
	ExperienceBand    string   `json:"experience_band,omitempty"`
	Accent            string   `json:"accent,omitempty"`
}

// SliceStat 为单个切分的聚合统计（语言/口音/岗位/年限/输入模式/便利设置）。
type SliceStat struct {
	Slice               string                   `json:"slice"`
	Value               string                   `json:"value"`
	Count               int                      `json:"count"`
	PassCount           int                      `json:"pass_count"`
	PassRate            float64                  `json:"pass_rate"`
	MeanScore           float64                  `json:"mean_score"`
	IncompleteCount     int                      `json:"incomplete_count"`
	MeanDimensionScores map[DimensionKey]float64 `json:"mean_dimension_scores"`
}

// FairnessSnapshot 为公平性监控快照。
type FairnessSnapshot struct {
	GeneratedAt time.Time   `json:"generated_at"`
	Slices      []SliceStat `json:"slices"`
}

type sliceAccumulator struct {
	count           int
	passCount       int
	scoreSum        float64
	scoredCount     int
	incompleteCount int
	dimSums         map[DimensionKey]float64
	dimCounts       map[DimensionKey]int
}

func newSliceAccumulator() *sliceAccumulator {
	return &sliceAccumulator{
		dimSums:   make(map[DimensionKey]float64),
		dimCounts: make(map[DimensionKey]int),
	}
}

// FairnessMonitor 为内存版公平性监控聚合器（生产接入指标后端；标签最小化）。
type FairnessMonitor struct {
	mu  sync.Mutex
	acc map[string]*sliceAccumulator
	now func() time.Time
}

// NewFairnessMonitor 创建监控器。
func NewFairnessMonitor() *FairnessMonitor {
	return &FairnessMonitor{acc: make(map[string]*sliceAccumulator), now: time.Now}
}

// Record 记录一次正式评分结果并按切分聚合。
func (m *FairnessMonitor) Record(r Result, meta FairnessMetadata) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for slice, value := range m.sliceValues(meta) {
		acc := m.acc[slice+"|"+value]
		if acc == nil {
			acc = newSliceAccumulator()
			m.acc[slice+"|"+value] = acc
		}
		acc.count++
		if r.ResultStatus == ResultPass {
			acc.passCount++
		}
		if r.ResultStatus == ResultEvaluationIncomplete {
			acc.incompleteCount++
		}
		if r.RoundTotal != nil {
			acc.scoreSum += float64(*r.RoundTotal)
			acc.scoredCount++
		}
		for _, dr := range r.DimensionResults {
			if dr.Score == nil {
				continue
			}
			acc.dimSums[dr.Dimension] += float64(*dr.Score)
			acc.dimCounts[dr.Dimension]++
		}
	}
}

// sliceValues 派生全部切分（语言/输入模式/便利设置/岗位/年限/口音）。
func (m *FairnessMonitor) sliceValues(meta FairnessMetadata) map[string]string {
	out := map[string]string{
		"language":   meta.InterviewLanguage,
		"input_mode": meta.InputMode,
		"job_family": meta.JobFamily,
		"experience": meta.ExperienceBand,
		"accent":     meta.Accent,
	}
	for _, accommodation := range meta.Accommodations {
		out["accommodation:"+accommodation] = "enabled"
	}
	return out
}

// Snapshot 输出全部切分聚合（确定性排序）。
func (m *FairnessMonitor) Snapshot() FairnessSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	keys := make([]string, 0, len(m.acc))
	for key := range m.acc {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	slices := make([]SliceStat, 0, len(keys))
	for _, key := range keys {
		acc := m.acc[key]
		sliceName, value := splitSliceKey(key)
		stat := SliceStat{
			Slice:               sliceName,
			Value:               value,
			Count:               acc.count,
			PassCount:           acc.passCount,
			IncompleteCount:     acc.incompleteCount,
			MeanDimensionScores: make(map[DimensionKey]float64),
		}
		if acc.count > 0 {
			stat.PassRate = float64(acc.passCount) / float64(acc.count)
		}
		if acc.scoredCount > 0 {
			stat.MeanScore = roundRatio(acc.scoreSum / float64(acc.scoredCount))
		}
		for d, sum := range acc.dimSums {
			if acc.dimCounts[d] > 0 {
				stat.MeanDimensionScores[d] = roundRatio(sum / float64(acc.dimCounts[d]))
			}
		}
		slices = append(slices, stat)
	}
	return FairnessSnapshot{GeneratedAt: m.now().UTC(), Slices: slices}
}

// splitSliceKey 从 "slice|value" 键中拆分（value 受控枚举，不含分隔符）。
func splitSliceKey(key string) (string, string) {
	idx := strings.LastIndex(key, "|")
	if idx < 0 {
		return key, ""
	}
	return key[:idx], key[idx+1:]
}
