// Command project 是面个蛋控制面「项目/计划/状态机 API」服务入口。
// TASK-016 业务与 HTTP 适配器位于 services/project 与 services/project/httpapi。
// 追踪：TASK-001、TASK-002、TASK-016、FR-009 ~ FR-011、ADR-0005。
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
	fmt.Printf("project 服务已通过区域自检（data_region=%s, service_env=%s, api_prefix=/v1/projects）\n",
		os.Getenv("DATA_REGION"), os.Getenv("SERVICE_ENV"))
}
