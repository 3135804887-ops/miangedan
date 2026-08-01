// Command worker 是项目状态机 Temporal Worker（TASK-017）。
// 追踪：TASK-017；ADR-0001；workflows/README（队列 interview 属 project）。
package main

import (
	"fmt"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"miangedan/services/region"
	"miangedan/services/temporal"
	"miangedan/workflows/workflow"
)

func main() {
	if err := region.CheckStartup(
		os.Getenv("DATA_REGION"),
		os.Getenv("INFRA_REGION"),
		os.Getenv("SERVICE_ENV"),
	); err != nil {
		fmt.Fprintln(os.Stderr, "启动被拒绝:", err)
		os.Exit(1)
	}
	namespace, err := temporal.Namespace(os.Getenv("DATA_REGION"), os.Getenv("SERVICE_ENV"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "命名空间校验失败:", err)
		os.Exit(1)
	}
	c, err := client.Dial(client.Options{
		HostPort:  os.Getenv("TEMPORAL_ADDRESS"),
		Namespace: namespace,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "连接 Temporal 失败:", err)
		os.Exit(1)
	}
	defer c.Close()

	w := worker.New(c, workflow.TaskQueueInterview, worker.Options{})
	w.RegisterWorkflow(workflow.ProjectWorkflow)
	w.RegisterActivity(workflow.AuditTransitionActivity)
	w.RegisterActivity(workflow.PersistProjectStateActivity)
	if err := w.Run(worker.InterruptCh()); err != nil {
		fmt.Fprintln(os.Stderr, "Worker 运行失败:", err)
		os.Exit(1)
	}
}
