// Package scoring 评分服务（TASK-044：量表/权重版本化）。
package scoring

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"gopkg.in/yaml.v3"
)

// 版本化错误。
var (
	ErrRubricNotFound       = errors.New("rubric version not found")
	ErrRubricPinnedMismatch = errors.New("active rubric version pinned mismatch")
	ErrRubricInvalid        = errors.New("invalid rubric configuration")
)

// Rubric 为一份冻结量表版本（config/rubrics/v1/default.yaml 的结构化形态）。
// 版本规则：rubric_version 在计划确认时冻结；活跃正式面试固定开始时版本；
// 历史分数保留各自 rubric_version，绝不因版本升级被改写。
type Rubric struct {
	ID                string
	Version           string
	Status            string
	DefaultWeights    map[DimensionKey]int
	Anchors           map[int]int
	MinCoverageRatio  float64
	TotalWeight       int
	MaxDimensionDelta int
}

// LoadRubric 从 YAML 加载量表版本。
func LoadRubric(path string) (Rubric, error) {
	// #nosec G304 -- 路径来自受控配置（仓库 config/rubrics 或测试固定路径），非用户输入。
	raw, err := os.ReadFile(path)
	if err != nil {
		return Rubric{}, fmt.Errorf("%w: %v", ErrRubricInvalid, err)
	}
	var doc struct {
		RubricVersion string `yaml:"rubric_version"`
		Version       string `yaml:"version"`
		Status        string `yaml:"status"`
		Dimensions    []struct {
			Key           string `yaml:"key"`
			DefaultWeight int    `yaml:"default_weight"`
		} `yaml:"dimensions"`
		Anchors []struct {
			Level       int `yaml:"level"`
			MappedScore int `yaml:"mapped_score"`
		} `yaml:"anchors"`
		EvidenceSufficiency struct {
			MinCoverageRatio float64 `yaml:"min_coverage_ratio"`
		} `yaml:"evidence_sufficiency"`
		WeightRules struct {
			TotalMustEqual        int `yaml:"total_must_equal"`
			PerDimensionAdjustMax int `yaml:"per_dimension_adjustment_max"`
		} `yaml:"weight_rules"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return Rubric{}, fmt.Errorf("%w: %v", ErrRubricInvalid, err)
	}
	weights := make(map[DimensionKey]int, len(doc.Dimensions))
	for _, d := range doc.Dimensions {
		if !validDimension(DimensionKey(d.Key)) {
			return Rubric{}, fmt.Errorf("%w: 未知维度 %s", ErrRubricInvalid, d.Key)
		}
		weights[DimensionKey(d.Key)] = d.DefaultWeight
	}
	anchors := make(map[int]int, len(doc.Anchors))
	for _, a := range doc.Anchors {
		anchors[a.Level] = a.MappedScore
	}
	return Rubric{
		ID:                doc.RubricVersion,
		Version:           doc.Version,
		Status:            doc.Status,
		DefaultWeights:    weights,
		Anchors:           anchors,
		MinCoverageRatio:  doc.EvidenceSufficiency.MinCoverageRatio,
		TotalWeight:       doc.WeightRules.TotalMustEqual,
		MaxDimensionDelta: doc.WeightRules.PerDimensionAdjustMax,
	}, nil
}

// RubricRegistry 为版本化量表注册表（追加注册；版本不可覆盖）。
type RubricRegistry struct {
	mu      sync.RWMutex
	rubrics map[string]Rubric
	order   []string
}

// NewRubricRegistry 创建空注册表。
func NewRubricRegistry() *RubricRegistry {
	return &RubricRegistry{rubrics: make(map[string]Rubric)}
}

// Register 注册量表版本（版本唯一；锚点必须为 1→20 … 5→100）。
func (r *RubricRegistry) Register(rubric Rubric) error {
	if rubric.ID == "" {
		return fmt.Errorf("%w: rubric_version 必填", ErrRubricInvalid)
	}
	if len(rubric.DefaultWeights) != len(DimensionKeys) {
		return fmt.Errorf("%w: 量表必须覆盖六维", ErrRubricInvalid)
	}
	sum := 0
	for _, d := range DimensionKeys {
		w, ok := rubric.DefaultWeights[d]
		if !ok || w < 0 {
			return fmt.Errorf("%w: 维度 %s 权重缺失或非法", ErrRubricInvalid, d)
		}
		sum += w
	}
	if sum != 100 {
		return fmt.Errorf("%w: 默认权重总和必须为 100，实际 %d", ErrRubricInvalid, sum)
	}
	for level := 1; level <= 5; level++ {
		if AnchorScore(level) != rubric.Anchors[level] {
			return fmt.Errorf("%w: 锚点映射必须完整且为 1→20…5→100", ErrRubricInvalid)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rubrics[rubric.ID]; ok {
		return fmt.Errorf("%w: 量表版本 %s 已存在（版本不可覆盖）", ErrRubricInvalid, rubric.ID)
	}
	r.rubrics[rubric.ID] = rubric
	r.order = append(r.order, rubric.ID)
	return nil
}

// Get 按版本取量表（fail-closed：未知版本拒绝）。
func (r *RubricRegistry) Get(version string) (Rubric, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rubric, ok := r.rubrics[version]
	if !ok {
		return Rubric{}, fmt.Errorf("%w: %s", ErrRubricNotFound, version)
	}
	return rubric, nil
}

// Latest 返回最新注册版本。
func (r *RubricRegistry) Latest() (Rubric, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.order) == 0 {
		return Rubric{}, ErrRubricNotFound
	}
	return r.rubrics[r.order[len(r.order)-1]], nil
}

// ValidateWeights 校验冻结权重（SC-EC-19：单维 ±5 且总和 100；
// 允许 0 权重重新分配路径，SC-EC-21 JD-only 计划阶段重分配）。
func (r *RubricRegistry) ValidateWeights(version string, weights map[DimensionKey]int) error {
	rubric, err := r.Get(version)
	if err != nil {
		return err
	}
	sum := 0
	for _, d := range DimensionKeys {
		w, ok := weights[d]
		if !ok || w < 0 {
			return fmt.Errorf("%w: 维度 %s 权重缺失或为负", ErrInvalidInput, d)
		}
		sum += w
	}
	if sum != rubric.TotalWeight {
		return fmt.Errorf("%w: 冻结权重总和必须为 %d，实际 %d",
			ErrInvalidInput, rubric.TotalWeight, sum)
	}
	hasZero := false
	for _, d := range DimensionKeys {
		if weights[d] == 0 {
			hasZero = true
			break
		}
	}
	if !hasZero {
		for _, d := range DimensionKeys {
			delta := weights[d] - rubric.DefaultWeights[d]
			if delta < -rubric.MaxDimensionDelta || delta > rubric.MaxDimensionDelta {
				return fmt.Errorf(
					"%w: 维度 %s 调整 %d 超出 ±%d（SC-EC-19 计划确认拒绝）",
					ErrInvalidInput, d, delta, rubric.MaxDimensionDelta)
			}
		}
	}
	return nil
}

// PinnedCheck 校验活跃正式会话固定版本（不匹配 fail-closed）。
func (r *RubricRegistry) PinnedCheck(activeVersion, requestedVersion string) error {
	if activeVersion != requestedVersion {
		return fmt.Errorf("%w: 活跃会话固定 %s，请求 %s",
			ErrRubricPinnedMismatch, activeVersion, requestedVersion)
	}
	return nil
}

// LoadDefaultRubricRegistry 加载仓库默认量表（config/rubrics/v1/default.yaml）。
func LoadDefaultRubricRegistry() (*RubricRegistry, error) {
	_, sourceFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(sourceFile)))
	rubric, err := LoadRubric(repoRoot + "/config/rubrics/v1/default.yaml")
	if err != nil {
		return nil, err
	}
	registry := NewRubricRegistry()
	if err := registry.Register(rubric); err != nil {
		return nil, err
	}
	return registry, nil
}
