package migrate

import (
	"context"
	"strings"
	"testing"
)

type fakeStore struct {
	applied    map[string]string
	applyCalls []string
}

func (f *fakeStore) EnsureMigrationsTable(context.Context) error { return nil }

func (f *fakeStore) AppliedChecksums(context.Context) (map[string]string, error) {
	copied := make(map[string]string, len(f.applied))
	for version, checksum := range f.applied {
		copied[version] = checksum
	}
	return copied, nil
}

func (f *fakeStore) ApplyMigration(_ context.Context, version, checksum, _ string) error {
	f.applied[version] = checksum
	f.applyCalls = append(f.applyCalls, version)
	return nil
}

// 正常路径：迁移文件名合法、版本唯一、按序排列且校验和为 SHA-256。
func TestLoadMigrationsValidAndSorted(t *testing.T) {
	migrations, err := LoadMigrations()
	if err != nil {
		t.Fatalf("加载迁移失败: %v", err)
	}
	if len(migrations) == 0 {
		t.Fatal("至少应有一份迁移")
	}
	for i, m := range migrations {
		if len(m.Checksum) != 64 {
			t.Fatalf("迁移 %s 校验和长度异常: %d", m.Version, len(m.Checksum))
		}
		if i > 0 && migrations[i-1].Version >= m.Version {
			t.Fatalf("迁移版本未按序: %s >= %s", migrations[i-1].Version, m.Version)
		}
	}
}

// 幂等/重试：同一 Store 二次 Apply 必须全部跳过。
func TestApplyIdempotent(t *testing.T) {
	store := &fakeStore{applied: map[string]string{}}
	ctx := context.Background()
	first, err := Apply(ctx, store)
	if err != nil {
		t.Fatalf("首次执行失败: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("首次执行应返回待应用版本")
	}
	second, err := Apply(ctx, store)
	if err != nil {
		t.Fatalf("二次执行失败: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("二次执行应幂等跳过，实际应用: %v", second)
	}
}

// 异常路径：已应用迁移校验和变化必须拒绝，禁止改写历史迁移。
func TestApplyChecksumMismatchFails(t *testing.T) {
	migrations, err := LoadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{applied: map[string]string{migrations[0].Version: "wrong-checksum"}}
	_, err = Apply(context.Background(), store)
	if err == nil || !strings.Contains(err.Error(), "校验和") {
		t.Fatalf("校验和变化应拒绝，得到: %v", err)
	}
}

// 正常路径：SQL 切分保留引号与注释内的分号。
func TestSplitStatementsKeepsQuotedSemicolons(t *testing.T) {
	sqlText := `
-- 注释; 不切断
CREATE TABLE t (v text CHECK (v IN ('a;b')));
INSERT INTO t VALUES ('c');
`
	statements := SplitStatements(sqlText)
	if len(statements) != 2 {
		t.Fatalf("期望 2 条语句，实际 %d: %#v", len(statements), statements)
	}
	if !strings.Contains(statements[0], "CHECK (v IN ('a;b'))") {
		t.Fatalf("引号内分号被误切: %s", statements[0])
	}
}

// 契约校验：基线迁移必须包含四张追加式表、REVOKE 与幂等键，业务角色无 UPDATE/DELETE。
func TestBaselineMigrationEnforcesLedgerConstraints(t *testing.T) {
	migrations, err := LoadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	baseline := migrations[0].SQL
	for _, table := range []string{"evidence_items", "score_versions", "usage_ledger", "access_audits"} {
		if !strings.Contains(baseline, "CREATE TABLE "+table) {
			t.Errorf("基线迁移缺少表 %s", table)
		}
	}
	if !strings.Contains(baseline, "REVOKE UPDATE, DELETE") {
		t.Error("基线迁移缺少 REVOKE UPDATE, DELETE")
	}
	if !strings.Contains(baseline, "idempotency_key") {
		t.Error("基线迁移缺少幂等键")
	}
	for _, role := range []string{"mgd_app_runtime", "mgd_ledger_writer"} {
		for _, line := range strings.Split(baseline, "\n") {
			if strings.Contains(line, "TO "+role) && (strings.Contains(line, "UPDATE") || strings.Contains(line, "DELETE")) {
				t.Errorf("业务角色 %s 不得获得 UPDATE/DELETE: %s", role, line)
			}
		}
	}
}

// TASK-010 / US-05 / FR-027：身份迁移强制区域隔离、主体唯一、防误合并与零明文凭证。
func TestIdentityMigrationEnforcesBindingAndSecretConstraints(t *testing.T) {
	migrations, err := LoadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	var identitySQL string
	for _, migration := range migrations {
		if migration.Version == "0010" {
			identitySQL = migration.SQL
			break
		}
	}
	if identitySQL == "" {
		t.Fatal("缺少 TASK-010 身份迁移 0010")
	}
	for _, table := range []string{
		"users", "identities", "identity_verifications", "identity_sessions",
		"identity_conflicts", "identity_idempotency",
	} {
		if !strings.Contains(identitySQL, "CREATE TABLE "+table) {
			t.Errorf("身份迁移缺少表 %s", table)
		}
	}
	for _, required := range []string{
		"identities_provider_subject_region_unique",
		"UNIQUE (data_region, provider, provider_subject_hash)",
		"UNIQUE (data_region, provider, request_key)",
		"identity_conflicts",
		"UNIQUE (data_region, operation, idempotency_key)",
	} {
		if !strings.Contains(identitySQL, required) {
			t.Errorf("身份迁移缺少约束 %q", required)
		}
	}
	for _, forbidden := range []string{"source_proof_token text", "target_proof_token text", "authorization_code", "access_token text", "refresh_token text", "email text"} {
		if strings.Contains(identitySQL, forbidden) {
			t.Errorf("身份迁移出现明文凭证/邮箱列 %q", forbidden)
		}
	}
}

// 正常路径：状态报告区分已应用与待应用。
func TestInspectStatus(t *testing.T) {
	migrations, err := LoadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{applied: map[string]string{migrations[0].Version: migrations[0].Checksum}}
	status, err := Inspect(context.Background(), store)
	if err != nil {
		t.Fatalf("状态查询失败: %v", err)
	}
	if len(status.Applied) != 1 || status.Applied[0] != migrations[0].Version {
		t.Fatalf("已应用状态不符: %v", status.Applied)
	}
	if len(status.Pending) != len(migrations)-1 {
		t.Fatalf("待应用数量不符: %v", status.Pending)
	}
}
