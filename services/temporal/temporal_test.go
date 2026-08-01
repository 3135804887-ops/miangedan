package temporal

import "testing"

// 正常路径：三个区域 × 三个环境生成约定命名空间。
func TestNamespaceValid(t *testing.T) {
	cases := map[string]string{
		"cn":   "cn",
		"eu":   "eu",
		"intl": "intl",
	}
	for env := range cases {
		expected := "mgd-" + env + "-dev-temporal"
		got, err := Namespace(env, "dev")
		if err != nil {
			t.Fatalf("区域 %s 应生成命名空间: %v", env, err)
		}
		if got != expected {
			t.Fatalf("命名空间应为 %s，实际 %s", expected, got)
		}
	}
}

// 异常路径：非法区域/环境必须拒绝。
func TestNamespaceRejected(t *testing.T) {
	for _, c := range [][2]string{{"", "dev"}, {"us", "dev"}, {"cn", ""}, {"cn", "qa"}} {
		if _, err := Namespace(c[0], c[1]); err == nil {
			t.Fatalf("区域 %q 环境 %q 必须拒绝", c[0], c[1])
		}
	}
}

// 异常路径：命名空间与区域/环境不一致必须 fail-closed 拒绝。
func TestValidateNamespaceMismatch(t *testing.T) {
	if err := ValidateNamespace("cn", "dev", "mgd-eu-dev-temporal"); err == nil {
		t.Fatal("跨区命名空间必须拒绝")
	}
}

// 正常路径：七域队列齐全即通过，顺序无关。
func TestValidateTaskQueuesValid(t *testing.T) {
	if err := ValidateTaskQueues(AllTaskQueues); err != nil {
		t.Fatalf("七域队列应通过: %v", err)
	}
	shuffled := []string{QueueDeletion, QueueReport, QueueScoring, QueueInterview, QueuePlan, QueueIngestion, QueueBilling}
	if err := ValidateTaskQueues(shuffled); err != nil {
		t.Fatalf("顺序无关应通过: %v", err)
	}
}

// 异常路径：缺失/额外/重复/未知队列必须拒绝。
func TestValidateTaskQueuesRejected(t *testing.T) {
	cases := [][]string{
		{QueueIngestion, QueuePlan, QueueInterview, QueueScoring, QueueReport, QueueBilling},
		append(append([]string(nil), AllTaskQueues...), "extra"),
		{QueueIngestion, QueueIngestion, QueuePlan, QueueInterview, QueueScoring, QueueReport, QueueBilling},
		{QueueIngestion, QueuePlan, QueueInterview, QueueScoring, QueueReport, QueueBilling, "unknown"},
	}
	for _, queues := range cases {
		if err := ValidateTaskQueues(queues); err == nil {
			t.Fatalf("队列集合 %v 必须拒绝", queues)
		}
	}
}

// 幂等性：同一配置重复校验结果一致（DoD 第 3 条）。
func TestValidateConfigIdempotent(t *testing.T) {
	namespace, err := Namespace("cn", "production")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := ValidateConfig("cn", "production", namespace, AllTaskQueues); err != nil {
			t.Fatalf("配置校验必须幂等通过: %v", err)
		}
	}
}

// 正常路径：队列集合比较顺序无关。
func TestTaskQueuesEqual(t *testing.T) {
	if !TaskQueuesEqual(AllTaskQueues, []string{QueueDeletion, QueueBilling, QueueReport, QueueScoring, QueueInterview, QueuePlan, QueueIngestion}) {
		t.Fatal("顺序无关的同一集合应相等")
	}
	if TaskQueuesEqual(AllTaskQueues, []string{QueueIngestion}) {
		t.Fatal("不同集合不应相等")
	}
}
