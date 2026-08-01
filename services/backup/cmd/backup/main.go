// Command backup 是备份/恢复契约 CLI（TASK-008）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-008；SECURITY-REQUIREMENTS SEC-050、SEC-052。
package main

import (
	"flag"
	"fmt"
	"os"

	"miangedan/services/backup"
)

func main() {
	configPath := flag.String("config", "", "备份配置文件路径（YAML）")
	mode := flag.String("mode", "plan", "plan | restore-dry-run")
	flag.Parse()
	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "用法: backup -config <path> [-mode plan|restore-dry-run]")
		os.Exit(2)
	}
	cfg, err := backup.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "备份配置校验失败:", err)
		os.Exit(1)
	}
	fmt.Println("备份配置校验通过：", backup.PlanSummary(cfg))
	if *mode == "restore-dry-run" {
		fmt.Println("一键恢复步骤（dry-run）：")
		for _, step := range backup.RestorePlan(cfg) {
			fmt.Println("  " + step)
		}
	} else if *mode != "plan" {
		fmt.Fprintf(os.Stderr, "未知模式 %q（支持 plan | restore-dry-run）\n", *mode)
		os.Exit(2)
	}
}
