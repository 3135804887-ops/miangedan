package region

import "testing"

// 正常路径：三个批准区域 × 三个合法环境全部通过启动自检。
func TestCheckStartupValid(t *testing.T) {
	for _, r := range AllRegions {
		for _, env := range []string{"dev", "staging", "production"} {
			if err := CheckStartup(r.String(), r.String(), env); err != nil {
				t.Errorf("区域 %s 环境 %s 应通过启动自检：%v", r, env, err)
			}
		}
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
		{"非法 INFRA_REGION", "cn", "us", "dev"},
		{"跨区不一致", "cn", "eu", "dev"},
		{"非法 SERVICE_ENV", "cn", "cn", "qa"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := CheckStartup(tc.data, tc.infra, tc.env); err == nil {
				t.Fatal("必须拒绝启动（fail-closed）")
			}
		})
	}
}

// 正常路径：同区请求放行。
func TestRouteAllowed(t *testing.T) {
	for _, r := range AllRegions {
		d, err := Route(r.String(), r.String())
		if err != nil {
			t.Fatalf("区域 %s 同区路由不应报错：%v", r, err)
		}
		if !d.Allowed || d.Reason != "" {
			t.Fatalf("区域 %s 同区路由应放行：%+v", r, d)
		}
	}
}

// 异常路径：跨区请求拒绝并给出 region_mismatch 原因，不自动转发。
func TestRouteMismatchRejected(t *testing.T) {
	cases := [][2]string{{"cn", "eu"}, {"eu", "intl"}, {"intl", "cn"}}
	for _, c := range cases {
		d, err := Route(c[0], c[1])
		if err != nil {
			t.Fatalf("区域不匹配应返回决策而非错误：%v", err)
		}
		if d.Allowed || d.Reason != MismatchReason {
			t.Fatalf("区域 %s -> %s 应拒绝并标记 region_mismatch：%+v", c[0], c[1], d)
		}
	}
}

// 异常路径：非法区域输入必须报错。
func TestRouteInvalidRejected(t *testing.T) {
	for _, c := range [][2]string{{"", "cn"}, {"cn", "us"}, {"CN", "cn"}} {
		if _, err := Route(c[0], c[1]); err == nil {
			t.Fatalf("区域 %q -> %q 非法输入必须报错", c[0], c[1])
		}
	}
}

// 幂等性：同一输入重复调用结果一致（DoD 第 3 条）。
func TestRouteIdempotent(t *testing.T) {
	first, err := Route("cn", "eu")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		next, err := Route("cn", "eu")
		if err != nil {
			t.Fatal(err)
		}
		if next != first {
			t.Fatalf("路由决策必须幂等：%+v != %+v", next, first)
		}
	}
}
