package backup

import "fmt"

// RestorePlan 返回一键恢复的固定步骤序列（恢复脚本按此执行，RPO/RTO 达标后对外服务）。
func RestorePlan(cfg Config) []string {
	return []string{
		fmt.Sprintf("1. 从区域备份桶 %s 还原最近一次每日完整备份", cfg.Bucket),
		"2. 应用持续增量（WAL 归档）至目标时间点（PITR）",
		"3. 应用 tombstone 过滤：恢复后已删数据不可见（RETENTION-MATRIX）",
		"4. 校验证据 RPO=0 与关键状态一致性",
		"5. 恢复区域服务并验证 RTO ≤30 分钟",
		"6. 记录演练/事故报告并复盘",
	}
}

// PlanSummary 输出供 CLI/演练使用的配置摘要。
func PlanSummary(cfg Config) string {
	return fmt.Sprintf(
		"区域=%s 环境=%s 桶=%s 策略=%s PITR=%t 证据RPO=%ds 其他RPO=%ds RTO=%ds tombstone=%t",
		cfg.DataRegion, cfg.ServiceEnv, cfg.Bucket, cfg.Schedule, cfg.PITR,
		cfg.EvidenceRPOSeconds, cfg.OtherRPOSeconds, cfg.RTOSeconds, cfg.TombstoneFilter,
	)
}
