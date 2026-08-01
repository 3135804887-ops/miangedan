package main

import "testing"

// 正常路径：三个批准区域全部放行。
func TestRequireDataRegionValid(t *testing.T) {
	for _, region := range []string{"cn", "eu", "intl"} {
		if err := requireDataRegion(region); err != nil {
			t.Errorf("区域 %q 应被接受，得到错误：%v", region, err)
		}
	}
}

// 异常路径：空值、大小写变体、非法区域、拼接串必须 fail-closed 拒绝。
func TestRequireDataRegionInvalid(t *testing.T) {
	for _, region := range []string{"", "CN", "us", "cn,eu", " intl", "eu "} {
		if err := requireDataRegion(region); err == nil {
			t.Errorf("区域 %q 必须被拒绝（fail-closed）", region)
		}
	}
}
