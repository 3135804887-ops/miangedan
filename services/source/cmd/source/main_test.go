package main

import (
	"testing"
)

func TestCheckStartupValid(t *testing.T) {
	t.Setenv("DATA_REGION", "cn")
	t.Setenv("INFRA_REGION", "cn")
	t.Setenv("SERVICE_ENV", "dev")
	if err := checkStartup(); err != nil {
		t.Fatalf("合法环境应通过自检: %v", err)
	}
}

func TestCheckStartupRejected(t *testing.T) {
	cases := []struct {
		name        string
		dataRegion  string
		infraRegion string
		serviceEnv  string
	}{
		{"非法区域", "us", "us", "dev"},
		{"区域不一致", "cn", "intl", "dev"},
		{"非法环境", "cn", "cn", "qa"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATA_REGION", tc.dataRegion)
			t.Setenv("INFRA_REGION", tc.infraRegion)
			t.Setenv("SERVICE_ENV", tc.serviceEnv)
			if err := checkStartup(); err == nil {
				t.Fatal("非法环境必须拒绝启动（fail-closed，ADR-0005）")
			}
		})
	}
}
