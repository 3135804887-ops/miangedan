package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

// PGStore 基于 database/sql + pgx 驱动执行迁移。
type PGStore struct {
	// DB 为 PostgreSQL 连接池。
	DB *sql.DB
}

// EnsureMigrationsTable 创建迁移记录表（幂等）。
func (s *PGStore) EnsureMigrationsTable(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version text PRIMARY KEY,
		checksum text NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`)
	return err
}

// AppliedChecksums 返回已应用版本到校验和的映射。
func (s *PGStore) AppliedChecksums(ctx context.Context) (map[string]string, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT version, checksum FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	applied := make(map[string]string)
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, err
		}
		applied[version] = checksum
	}
	return applied, rows.Err()
}

// ApplyMigration 在单个事务中执行迁移 SQL 并写入迁移记录。
func (s *PGStore) ApplyMigration(ctx context.Context, version, checksum, sqlText string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range SplitStatements(sqlText) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("语句执行失败: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, checksum) VALUES ($1, $2)`, version, checksum); err != nil {
		return err
	}
	return tx.Commit()
}
