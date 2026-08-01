// Command consent 是面个蛋控制面「授权中心」服务的最小入口（TASK-001 工程骨架；
// 区域自检在 TASK-002 升级为 DATA_REGION/INFRA_REGION 一致性校验）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-001、TASK-002；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 5 节；TASK-011。
package main

import (
	"fmt"
	"os"

	"miangedan/services/region"
)

// checkStartup 提供 fail-closed 区域自检：DATA_REGION 必须与所连基础设施区域
// INFRA_REGION 一致，且 SERVICE_ENV 合法（TASK-002，ADR-0005）。
func checkStartup() error {
	return region.CheckStartup(
		os.Getenv("DATA_REGION"),
		os.Getenv("INFRA_REGION"),
		os.Getenv("SERVICE_ENV"),
	)
}

func main() {
	if err := checkStartup(); err != nil {
		fmt.Fprintln(os.Stderr, "启动被拒绝:", err)
		os.Exit(1)
	}
	fmt.Printf("consent 服务骨架已启动（data_region=%s, service_env=%s）\n",
		os.Getenv("DATA_REGION"), os.Getenv("SERVICE_ENV"))
}
