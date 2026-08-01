package main

import "testing"

// 正常路径：区域/基础设施/环境三者一致时启动自检通过。
func TestCheckStartupValid(t *testing.T) {
	t.Setenv("DATA_REGION", "cn")
	t.Setenv("INFRA_REGION", "cn")
	t.Setenv("SERVICE_ENV", "dev")
	if err := checkStartup(); err != nil {
		t.Fatalf("合法配置应通过：%v", err)
	}
}

// 异常路径：缺失/非法区域、跨区不一致、非法环境必须 fail-closed 拒绝。
func TestCheckStartupRejected(t *testing.T) {
	cases := []struct {
		name, data, infra, env string
	}{
		{"缺失 DATA_REGION", "", "cn", "dev"},
		{"非法 DATA_REGION", "us", "cn", "dev"},
		{"缺失 INFRA_REGION", "cn", "", "dev"},
		{"跨区不一致", "cn", "eu", "dev"},
		{"非法 SERVICE_ENV", "cn", "cn", "qa"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATA_REGION", tc.data)
			t.Setenv("INFRA_REGION", tc.infra)
			t.Setenv("SERVICE_ENV", tc.env)
			if err := checkStartup(); err == nil {
				t.Fatal("必须拒绝启动（fail-closed）")
			}
		})
	}
}
