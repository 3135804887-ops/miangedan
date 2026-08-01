package backup

import (
	"os"
	"path/filepath"
	"testing"

	"miangedan/services/region"
)

func validConfig(regionCode string) Config {
	return Config{
		DataRegion:         regionCode,
		ServiceEnv:         "production",
		Bucket:             "mgd-" + regionCode + "-prod-backups",
		Schedule:           ScheduleDailyFullPlusWAL,
		PITR:               true,
		EvidenceRPOSeconds: EvidenceRPOZero,
		OtherRPOSeconds:    OtherRPOSecondsMax,
		RTOSeconds:         RTOSecondsMax,
		TombstoneFilter:    true,
	}
}

// 正常路径：三区合法备份配置通过。
func TestBackupConfigValid(t *testing.T) {
	for _, r := range region.AllRegions {
		if err := validConfig(r.String()).Validate(); err != nil {
			t.Fatalf("区域 %s 合法配置应通过: %v", r, err)
		}
	}
}

// 异常路径：跨区桶、禁用 PITR、证据 RPO>0、RTO 超限、关闭 tombstone 必须拒绝。
func TestBackupConfigRejected(t *testing.T) {
	cases := map[string]Config{
		"非法区域":        {DataRegion: "us", ServiceEnv: "production", Bucket: "mgd-us-prod-backups", Schedule: ScheduleDailyFullPlusWAL, PITR: true, EvidenceRPOSeconds: 0, OtherRPOSeconds: 5, RTOSeconds: 1800, TombstoneFilter: true},
		"跨区桶":         {DataRegion: "cn", ServiceEnv: "production", Bucket: "mgd-eu-prod-backups", Schedule: ScheduleDailyFullPlusWAL, PITR: true, EvidenceRPOSeconds: 0, OtherRPOSeconds: 5, RTOSeconds: 1800, TombstoneFilter: true},
		"非法策略":        {DataRegion: "cn", ServiceEnv: "production", Bucket: "mgd-cn-prod-backups", Schedule: "weekly", PITR: true, EvidenceRPOSeconds: 0, OtherRPOSeconds: 5, RTOSeconds: 1800, TombstoneFilter: true},
		"禁用PITR":      {DataRegion: "cn", ServiceEnv: "production", Bucket: "mgd-cn-prod-backups", Schedule: ScheduleDailyFullPlusWAL, PITR: false, EvidenceRPOSeconds: 0, OtherRPOSeconds: 5, RTOSeconds: 1800, TombstoneFilter: true},
		"证据RPO>0":     {DataRegion: "cn", ServiceEnv: "production", Bucket: "mgd-cn-prod-backups", Schedule: ScheduleDailyFullPlusWAL, PITR: true, EvidenceRPOSeconds: 60, OtherRPOSeconds: 5, RTOSeconds: 1800, TombstoneFilter: true},
		"其他RPO超限":     {DataRegion: "cn", ServiceEnv: "production", Bucket: "mgd-cn-prod-backups", Schedule: ScheduleDailyFullPlusWAL, PITR: true, EvidenceRPOSeconds: 0, OtherRPOSeconds: 30, RTOSeconds: 1800, TombstoneFilter: true},
		"RTO超限":       {DataRegion: "cn", ServiceEnv: "production", Bucket: "mgd-cn-prod-backups", Schedule: ScheduleDailyFullPlusWAL, PITR: true, EvidenceRPOSeconds: 0, OtherRPOSeconds: 5, RTOSeconds: 3600, TombstoneFilter: true},
		"关闭tombstone": {DataRegion: "cn", ServiceEnv: "production", Bucket: "mgd-cn-prod-backups", Schedule: ScheduleDailyFullPlusWAL, PITR: true, EvidenceRPOSeconds: 0, OtherRPOSeconds: 5, RTOSeconds: 1800, TombstoneFilter: false},
	}
	for name, cfg := range cases {
		if err := cfg.Validate(); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// 正常路径：配置 YAML 加载并校验；恢复步骤序列非空且包含 tombstone 过滤。
func TestLoadConfigAndRestorePlan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.yaml")
	content := `
region: cn
environment: production
topology:
  backup:
    bucket: mgd-cn-prod-backups
    schedule: daily_full_plus_wal
    pitr: true
    evidence_rpo_seconds: 0
    other_rpo_seconds: 5
    rto_seconds: 1800
    tombstone_filter: true
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("合法 YAML 应加载成功: %v", err)
	}
	steps := RestorePlan(cfg)
	if len(steps) == 0 {
		t.Fatal("恢复步骤序列不得为空")
	}
	joined := ""
	for _, s := range steps {
		joined += s
	}
	if !containsStr(joined, "tombstone") {
		t.Fatal("恢复步骤必须包含 tombstone 过滤")
	}
	if PlanSummary(cfg) == "" {
		t.Fatal("配置摘要不得为空")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// 异常路径：损坏 YAML 必须拒绝。
func TestLoadConfigRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("data_region: [unclosed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("损坏 YAML 必须拒绝")
	}
}

// 幂等性：同一配置重复校验与加载结果一致（DoD 第 3 条）。
func TestBackupIdempotent(t *testing.T) {
	cfg := validConfig("cn")
	for i := 0; i < 3; i++ {
		if err := cfg.Validate(); err != nil {
			t.Fatalf("配置校验必须幂等通过: %v", err)
		}
		if len(RestorePlan(cfg)) != 6 {
			t.Fatal("恢复步骤必须确定")
		}
	}
}
