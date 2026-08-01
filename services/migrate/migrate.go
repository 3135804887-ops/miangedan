// Package migrate 提供幂等、可校验的 PostgreSQL 迁移执行器（TASK-003 数据平台）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-003；docs/data/DATA-MODEL.md；
// docs/architecture/adr/ADR-0004-append-only-evidence-ledger.md；
// docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节。
package migrate

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

// migrationsFS 嵌入迁移 SQL，保证迁移文件与代码同版本发布。
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

var migrationNamePattern = regexp.MustCompile(`^(\d{4})_[a-z0-9_-]+\.sql$`)

// Migration 描述一份可执行迁移。
type Migration struct {
	// Version 为四位序号（如 0001）。
	Version string
	// Name 为迁移文件名。
	Name string
	// Checksum 为 SQL 内容的 SHA-256。
	Checksum string
	// SQL 为迁移正文。
	SQL string
}

// Store 抽象迁移状态与执行，便于无数据库单测。
type Store interface {
	// EnsureMigrationsTable 确保迁移记录表存在。
	EnsureMigrationsTable(ctx context.Context) error
	// AppliedChecksums 返回已应用迁移版本到校验和的映射。
	AppliedChecksums(ctx context.Context) (map[string]string, error)
	// ApplyMigration 在单个事务内执行迁移并记录版本。
	ApplyMigration(ctx context.Context, version, checksum, sqlText string) error
}

// LoadMigrations 读取并校验嵌入迁移：文件名合法、版本唯一且按序返回。
func LoadMigrations() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("读取迁移目录: %w", err)
	}
	migrations := make([]Migration, 0, len(entries))
	seen := make(map[string]bool)
	for _, entry := range entries {
		name := entry.Name()
		match := migrationNamePattern.FindStringSubmatch(name)
		if match == nil {
			return nil, fmt.Errorf("迁移文件名 %q 不符合 NNNN_name.sql", name)
		}
		version := match[1]
		if seen[version] {
			return nil, fmt.Errorf("迁移版本重复: %s", version)
		}
		seen[version] = true
		raw, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return nil, fmt.Errorf("读取迁移 %s: %w", name, err)
		}
		migrations = append(migrations, Migration{
			Version:  version,
			Name:     name,
			Checksum: checksum(raw),
			SQL:      string(raw),
		})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	return migrations, nil
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Apply 按版本顺序执行未应用迁移；已应用且校验和不变则跳过（幂等），
// 校验和变化直接失败，禁止修改已应用迁移。
func Apply(ctx context.Context, store Store) ([]string, error) {
	migrations, err := LoadMigrations()
	if err != nil {
		return nil, err
	}
	if err := store.EnsureMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("初始化迁移记录表: %w", err)
	}
	applied, err := store.AppliedChecksums(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取已应用迁移: %w", err)
	}
	appliedNow := make([]string, 0, len(migrations))
	for _, m := range migrations {
		if existing, ok := applied[m.Version]; ok {
			if existing != m.Checksum {
				return appliedNow, fmt.Errorf(
					"迁移 %s 校验和已变化：%s != %s（禁止修改已应用迁移）",
					m.Version, existing, m.Checksum,
				)
			}
			continue
		}
		if err := store.ApplyMigration(ctx, m.Version, m.Checksum, m.SQL); err != nil {
			return appliedNow, fmt.Errorf("执行迁移 %s: %w", m.Version, err)
		}
		appliedNow = append(appliedNow, m.Version)
	}
	return appliedNow, nil
}

// Status 为迁移状态报告。
type Status struct {
	// Applied 为已应用版本（按迁移顺序）。
	Applied []string
	// Pending 为待应用版本（按迁移顺序）。
	Pending []string
}

// Inspect 返回迁移状态报告。
func Inspect(ctx context.Context, store Store) (Status, error) {
	migrations, err := LoadMigrations()
	if err != nil {
		return Status{}, err
	}
	if err := store.EnsureMigrationsTable(ctx); err != nil {
		return Status{}, fmt.Errorf("初始化迁移记录表: %w", err)
	}
	applied, err := store.AppliedChecksums(ctx)
	if err != nil {
		return Status{}, fmt.Errorf("读取已应用迁移: %w", err)
	}
	status := Status{}
	for _, m := range migrations {
		if _, ok := applied[m.Version]; ok {
			status.Applied = append(status.Applied, m.Version)
		} else {
			status.Pending = append(status.Pending, m.Version)
		}
	}
	return status, nil
}

// SplitStatements 按分号切分 SQL 语句，保留单引号与注释内的分号。
func SplitStatements(sqlText string) []string {
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inLineComment := false
	inBlockComment := false
	for i := 0; i < len(sqlText); i++ {
		ch := sqlText[i]
		next := byte(0)
		if i+1 < len(sqlText) {
			next = sqlText[i+1]
		}
		switch {
		case inLineComment:
			current.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
			}
		case inBlockComment:
			current.WriteByte(ch)
			if ch == '*' && next == '/' {
				current.WriteByte(next)
				i++
				inBlockComment = false
			}
		case inSingleQuote:
			current.WriteByte(ch)
			if ch == '\'' {
				if next == '\'' {
					current.WriteByte(next)
					i++
				} else {
					inSingleQuote = false
				}
			}
		case ch == '-' && next == '-':
			current.WriteString("--")
			i++
			inLineComment = true
		case ch == '/' && next == '*':
			current.WriteString("/*")
			i++
			inBlockComment = true
		case ch == '\'':
			current.WriteByte(ch)
			inSingleQuote = true
		case ch == ';':
			if stmt := strings.TrimSpace(current.String()); stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		default:
			current.WriteByte(ch)
		}
	}
	if stmt := strings.TrimSpace(current.String()); stmt != "" {
		statements = append(statements, stmt)
	}
	return statements
}
