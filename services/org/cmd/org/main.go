// Command org 是面个蛋控制面「机构租户」服务的最小入口（TASK-001 工程骨架）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-001；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 4 节；EPIC-08（TASK-070 ~ TASK-074）。
package main

import (
	"fmt"
	"os"
)

// validDataRegions 为 ADR-0005 批准的三个数据区代码（OD-09）。
var validDataRegions = map[string]bool{"cn": true, "eu": true, "intl": true}

// requireDataRegion 提供 fail-closed 区域自检的最小形态：DATA_REGION 缺失或不在
// 批准集合内即返回错误，进程必须拒绝启动。与所连基础设施区域的一致性校验在
// TASK-002 落地（EPIC-01-INFRA-DESIGN.md 第 5.2 节）。
func requireDataRegion(region string) error {
	if !validDataRegions[region] {
		return fmt.Errorf("DATA_REGION %q 非法：必须为 cn | eu | intl（fail-closed，ADR-0005）", region)
	}
	return nil
}

func main() {
	if err := requireDataRegion(os.Getenv("DATA_REGION")); err != nil {
		fmt.Fprintln(os.Stderr, "启动被拒绝:", err)
		os.Exit(1)
	}
	fmt.Printf("org 服务骨架已启动（data_region=%s, service_env=%s）\n",
		os.Getenv("DATA_REGION"), os.Getenv("SERVICE_ENV"))
}
