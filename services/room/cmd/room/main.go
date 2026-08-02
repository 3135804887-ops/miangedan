// Command room 是面个蛋控制面「实时会话房间」服务入口。
// TASK-020 业务与 HTTP 适配器位于 services/room 与 services/room/httpapi。
// 追踪：TASK-020、FR-013、NFR-007、SEC-003、ADR-0005。
package main

import (
	"fmt"
	"os"

	"miangedan/services/region"
)

// checkStartup 提供 fail-closed 区域自检（TASK-002，ADR-0005）。
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
	fmt.Printf("room 服务已通过区域自检（data_region=%s, service_env=%s, api_prefix=/v1/sessions）\n",
		os.Getenv("DATA_REGION"), os.Getenv("SERVICE_ENV"))
}
