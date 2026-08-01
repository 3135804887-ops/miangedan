package main

import "testing"

// 正常路径：区域/基础设施/环境三者一致时通过启动自检。
func TestCheckStartupValid(t *testing.T) {
	t.Setenv("DATA_REGION", "cn")
	t.Setenv("INFRA_REGION", "cn")
	t.Setenv("SERVICE_ENV", "dev")
	if err := checkStartup(); err != nil {
		t.Fatalf("合法配置应通过：%v", err)
	}
}

// 异常路径：跨区配置必须 fail-closed 拒绝。
func TestCheckStartupRejected(t *testing.T) {
	t.Setenv("DATA_REGION", "cn")
	t.Setenv("INFRA_REGION", "eu")
	t.Setenv("SERVICE_ENV", "dev")
	if err := checkStartup(); err == nil {
		t.Fatal("跨区配置必须拒绝（fail-closed）")
	}
}
