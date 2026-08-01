// Command identity 是面个蛋控制面「身份与账户」服务入口。
// 追踪：TASK-002、TASK-010；US-05、FR-027；ADR-0005。
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
	fmt.Printf("identity 服务区域配置检查通过（data_region=%s, service_env=%s）\n",
		os.Getenv("DATA_REGION"), os.Getenv("SERVICE_ENV"))
}
