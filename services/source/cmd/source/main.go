// Command source 是面个蛋控制面「企业公开流程来源服务」最小入口（TASK-015）。
// 启动前执行 fail-closed 区域自检：DATA_REGION 必须与所连基础设施区域 INFRA_REGION 一致，
// 且 SERVICE_ENV 合法（TASK-002、ADR-0005）。业务逻辑见 services/source 包。
package main

import (
	"fmt"
	"os"

	"miangedan/services/region"
)

// checkStartup 提供 fail-closed 区域自检（与 services/ingestion 等控制面服务一致）。
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
	fmt.Printf("source 服务骨架已启动（data_region=%s, service_env=%s）\n",
		os.Getenv("DATA_REGION"), os.Getenv("SERVICE_ENV"))
}
