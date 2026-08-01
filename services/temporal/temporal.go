// Package temporal 提供 Temporal 区域命名空间与任务队列契约（TASK-004）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-004；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节；
// docs/architecture/adr/ADR-0001-separate-business-workflow-from-ai-graph.md；
// docs/architecture/adr/ADR-0005-three-data-regions.md。
package temporal

import (
	"fmt"
	"sort"

	"miangedan/services/region"
)

// 已批准的业务工作流域任务队列。
const (
	QueueIngestion = "ingestion"
	QueuePlan      = "plan"
	QueueInterview = "interview"
	QueueScoring   = "scoring"
	QueueReport    = "report"
	QueueBilling   = "billing"
	QueueDeletion  = "deletion"
)

// AllTaskQueues 返回全部已批准任务队列（七域固定）。
var AllTaskQueues = []string{
	QueueIngestion, QueuePlan, QueueInterview, QueueScoring,
	QueueReport, QueueBilling, QueueDeletion,
}

var validQueues = func() map[string]bool {
	m := make(map[string]bool, len(AllTaskQueues))
	for _, q := range AllTaskQueues {
		m[q] = true
	}
	return m
}()

// Namespace 生成区域命名空间 mgd-{region}-{env}-temporal。
func Namespace(regionCode, env string) (string, error) {
	if err := region.ValidateDataRegion(regionCode); err != nil {
		return "", err
	}
	if err := region.ValidateEnvironment(env); err != nil {
		return "", err
	}
	return fmt.Sprintf("mgd-%s-%s-temporal", regionCode, env), nil
}

// ValidateNamespace 校验命名空间与区域/环境一致（fail-closed，ADR-0005）。
func ValidateNamespace(regionCode, env, namespace string) error {
	expected, err := Namespace(regionCode, env)
	if err != nil {
		return err
	}
	if namespace != expected {
		return fmt.Errorf("namespace %q 与区域命名空间 %q 不一致（fail-closed，ADR-0005）", namespace, expected)
	}
	return nil
}

// ValidateTaskQueues 校验任务队列集合：七域齐全、无重复、无未知队列。
func ValidateTaskQueues(queues []string) error {
	if len(queues) != len(AllTaskQueues) {
		return fmt.Errorf("task_queues 数量必须为 %d，实际 %d", len(AllTaskQueues), len(queues))
	}
	seen := make(map[string]bool, len(queues))
	for _, q := range queues {
		if !validQueues[q] {
			return fmt.Errorf("未知任务队列 %q", q)
		}
		if seen[q] {
			return fmt.Errorf("任务队列重复 %q", q)
		}
		seen[q] = true
	}
	return nil
}

// ValidateConfig 校验 Temporal 区域实例配置：命名空间与任务队列。
func ValidateConfig(regionCode, env, namespace string, taskQueues []string) error {
	if err := ValidateNamespace(regionCode, env, namespace); err != nil {
		return err
	}
	return ValidateTaskQueues(taskQueues)
}

// TaskQueuesEqual 按集合语义比较任务队列（顺序无关）。
func TaskQueuesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA := append([]string(nil), a...)
	sortedB := append([]string(nil), b...)
	sort.Strings(sortedA)
	sort.Strings(sortedB)
	for i := range sortedA {
		if sortedA[i] != sortedB[i] {
			return false
		}
	}
	return true
}
