// Package backup 提供备份与恢复契约（TASK-008）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-008；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节；
// SECURITY-REQUIREMENTS SEC-050、SEC-052；RETENTION-MATRIX（tombstone 过滤）。
package backup

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"miangedan/services/region"
)

// 备份策略常量。
const (
	// ScheduleDailyFullPlusWAL 为每日完整 + 持续增量（WAL 归档）+ PITR 基线。
	ScheduleDailyFullPlusWAL = "daily_full_plus_wal"

	// EvidenceRPOZero 为已确认回答与评分证据的 RPO 硬目标（NFR/PRD 容灾：证据 RPO=0）。
	EvidenceRPOZero = 0
	// OtherRPOSecondsMax 为其他非关键状态 RPO 上限（≤5 秒）。
	OtherRPOSecondsMax = 5
	// RTOSecondsMax 为区域级严重故障 RTO 上限（≤30 分钟）。
	RTOSecondsMax = 1800
)

// Config 描述单数据区的备份与恢复目标。
type Config struct {
	DataRegion         string `yaml:"data_region"`
	ServiceEnv         string `yaml:"service_env"`
	Bucket             string `yaml:"bucket"`
	Schedule           string `yaml:"schedule"`
	PITR               bool   `yaml:"pitr"`
	EvidenceRPOSeconds int    `yaml:"evidence_rpo_seconds"`
	OtherRPOSeconds    int    `yaml:"other_rpo_seconds"`
	RTOSeconds         int    `yaml:"rto_seconds"`
	TombstoneFilter    bool   `yaml:"tombstone_filter"`
}

// Validate 校验备份配置：区域/环境合法、策略为每日完整+WAL、启用 PITR、
// 证据 RPO=0、其他 RPO ≤5s、RTO ≤30min、恢复前强制 tombstone 过滤（fail-closed）。
func (c Config) Validate() error {
	if err := region.ValidateDataRegion(c.DataRegion); err != nil {
		return err
	}
	if err := region.ValidateEnvironment(c.ServiceEnv); err != nil {
		return err
	}
	if strings.TrimSpace(c.Bucket) == "" {
		return errors.New("备份桶为空")
	}
	if !strings.Contains(c.Bucket, c.DataRegion) {
		return fmt.Errorf("备份桶 %q 必须位于数据区 %s（区域内备份，SEC-050）", c.Bucket, c.DataRegion)
	}
	if c.Schedule != ScheduleDailyFullPlusWAL {
		return fmt.Errorf("备份策略 %q 非法：必须为 %s", c.Schedule, ScheduleDailyFullPlusWAL)
	}
	if !c.PITR {
		return errors.New("必须启用时间点恢复（PITR=true，SEC-052）")
	}
	if c.EvidenceRPOSeconds != EvidenceRPOZero {
		return fmt.Errorf("证据 RPO 必须为 0 秒，实际 %d", c.EvidenceRPOSeconds)
	}
	if c.OtherRPOSeconds <= 0 || c.OtherRPOSeconds > OtherRPOSecondsMax {
		return fmt.Errorf("其他状态 RPO 必须为 1-%d 秒，实际 %d", OtherRPOSecondsMax, c.OtherRPOSeconds)
	}
	if c.RTOSeconds <= 0 || c.RTOSeconds > RTOSecondsMax {
		return fmt.Errorf("RTO 必须为 1-%d 秒（≤30 分钟），实际 %d", RTOSecondsMax, c.RTOSeconds)
	}
	if !c.TombstoneFilter {
		return errors.New("恢复流程必须启用 tombstone 过滤（RETENTION-MATRIX：恢复服务前先过滤已删数据）")
	}
	return nil
}

// LoadConfig 从区域拓扑 YAML（infra/regions/{cn,eu,intl}/envs/*.yaml）加载并校验备份配置：
// 顶层 region/environment 与 topology.backup 段合并后整体校验。
func LoadConfig(path string) (Config, error) {
	// #nosec G304 -- 备份配置路径由 CLI 显式传入（本工具唯一用途），非不可信网络输入
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("读取备份配置失败: %w", err)
	}
	var doc struct {
		Region      string `yaml:"region"`
		Environment string `yaml:"environment"`
		Topology    struct {
			Backup Config `yaml:"backup"`
		} `yaml:"topology"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return Config{}, fmt.Errorf("解析备份配置失败: %w", err)
	}
	cfg := doc.Topology.Backup
	cfg.DataRegion = doc.Region
	cfg.ServiceEnv = doc.Environment
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
