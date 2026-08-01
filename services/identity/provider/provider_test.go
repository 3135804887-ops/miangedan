package provider

import (
	"testing"

	"miangedan/services/region"
)

// 正常路径：三区开放矩阵符合 PRD（email 全开放；wechat 仅 cn；google/apple 仅 eu/intl）。
func TestRegionProvidersValid(t *testing.T) {
	want := map[string][]string{
		"cn":   {Email, WeChat},
		"eu":   {Email, Google, Apple},
		"intl": {Email, Google, Apple},
	}
	for _, r := range region.AllRegions {
		got, err := RegionProviders(r.String())
		if err != nil {
			t.Fatalf("区域 %s 应返回开放列表: %v", r, err)
		}
		if len(got) != len(want[r.String()]) {
			t.Fatalf("区域 %s 开放列表不符：%v", r, got)
		}
		for i := range got {
			if got[i] != want[r.String()][i] {
				t.Fatalf("区域 %s 开放列表不符：%v", r, got)
			}
		}
		if err := ValidateProviders(r.String(), got); err != nil {
			t.Fatalf("区域 %s 完整开放列表应通过: %v", r, err)
		}
	}
}

// 异常路径：非法区域、跨区提供商、未知提供商、缺 email、重复必须拒绝。
func TestValidateProvidersRejected(t *testing.T) {
	cases := map[string][2]any{
		"非法区域":      {"us", []string{Email}},
		"cn带google": {"cn", []string{Email, Google}},
		"eu带wechat": {"eu", []string{Email, WeChat}},
		"未知提供商":     {"cn", []string{Email, "github"}},
		"缺email":    {"cn", []string{WeChat}},
		"重复email":   {"cn", []string{Email, Email}},
	}
	for name, c := range cases {
		rc := c[0].(string)
		list := c[1].([]string)
		if err := ValidateProviders(rc, list); err == nil {
			t.Fatalf("用例 %q 必须拒绝", name)
		}
	}
}

// 幂等性：开放列表与校验结果确定可重复（DoD 第 3 条）。
func TestProviderIdempotent(t *testing.T) {
	for i := 0; i < 3; i++ {
		got, err := RegionProviders("cn")
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0] != Email || got[1] != WeChat {
			t.Fatalf("开放列表必须确定: %v", got)
		}
		if err := ValidateProviders("cn", got); err != nil {
			t.Fatalf("校验必须幂等通过: %v", err)
		}
	}
}
