// Command migrate 是面个蛋数据平台 PostgreSQL 迁移工具（TASK-003）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-003；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节；
// docs/architecture/adr/ADR-0004-append-only-evidence-ledger.md。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"miangedan/services/migrate"
	"miangedan/services/region"
)

// checkStartup 提供 fail-closed 区域自检（DATA_REGION == INFRA_REGION，TASK-002 基线）。
func checkStartup() error {
	return region.CheckStartup(
		os.Getenv("DATA_REGION"),
		os.Getenv("INFRA_REGION"),
		os.Getenv("SERVICE_ENV"),
	)
}

func main() {
	flag.Parse()
	if err := checkStartup(); err != nil {
		fmt.Fprintln(os.Stderr, "启动被拒绝:", err)
		os.Exit(1)
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "启动被拒绝: DATABASE_URL 未设置（[REGION-SCOPED] 必须指向本区 PostgreSQL）")
		os.Exit(1)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "打开数据库失败:", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	store := &migrate.PGStore{DB: db}
	command := flag.Arg(0)
	switch command {
	case "up":
		applied, err := migrate.Apply(ctx, store)
		if err != nil {
			fmt.Fprintln(os.Stderr, "迁移失败:", err)
			os.Exit(1)
		}
		if len(applied) == 0 {
			fmt.Println("无待执行迁移（幂等：已是最新）")
			return
		}
		fmt.Printf("已应用迁移: %v\n", applied)
	case "status":
		status, err := migrate.Inspect(ctx, store)
		if err != nil {
			fmt.Fprintln(os.Stderr, "查询迁移状态失败:", err)
			os.Exit(1)
		}
		fmt.Printf("已应用: %v\n待应用: %v\n", status.Applied, status.Pending)
	default:
		fmt.Fprintln(os.Stderr, "用法: migrate up|status（环境：DATA_REGION/INFRA_REGION/SERVICE_ENV/DATABASE_URL）")
		os.Exit(2)
	}
}
