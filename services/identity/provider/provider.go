// Package provider 提供身份提供商区域注册表（TASK-007）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-007；PRD FR-027（邮箱验证码先行，Google/Apple/微信随区域开放）。
package provider

import (
	"errors"
	"fmt"
	"strings"

	"miangedan/services/region"
)

// 身份提供商标识（OD-10：稳定英文命名）。
const (
	Email  = "email"
	Google = "google"
	Apple  = "apple"
	WeChat = "wechat"
)

// regionProviders 为 PRD 区域开放矩阵：邮箱验证码所有区；微信仅 cn；Google/Apple 仅 eu/intl。
var regionProviders = map[string][]string{
	region.CN.String():   {Email, WeChat},
	region.EU.String():   {Email, Google, Apple},
	region.INTL.String(): {Email, Google, Apple},
}

// RegionProviders 返回某数据区允许的身份提供商（顺序稳定、副本安全）。
func RegionProviders(regionCode string) ([]string, error) {
	if err := region.ValidateDataRegion(regionCode); err != nil {
		return nil, err
	}
	out := make([]string, len(regionProviders[regionCode]))
	copy(out, regionProviders[regionCode])
	return out, nil
}

// ValidateProviders 校验区域身份提供商配置：区域合法、必含 email、无未知/重复/跨区提供商（fail-closed）。
func ValidateProviders(regionCode string, providers []string) error {
	allowed, err := RegionProviders(regionCode)
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		return fmt.Errorf("区域 %s 身份提供商列表为空：必须至少含 email（FR-027）", regionCode)
	}
	allowedSet := make(map[string]bool, len(allowed))
	for _, p := range allowed {
		allowedSet[p] = true
	}
	seen := make(map[string]bool, len(providers))
	hasEmail := false
	for _, p := range providers {
		if !allowedSet[p] {
			return fmt.Errorf("身份提供商 %q 不在区域 %s 开放范围（允许：%s）", p, regionCode, strings.Join(allowed, ", "))
		}
		if seen[p] {
			return fmt.Errorf("身份提供商重复 %q", p)
		}
		seen[p] = true
		if p == Email {
			hasEmail = true
		}
	}
	if !hasEmail {
		return errors.New("身份提供商必须包含 email（邮箱验证码先行，FR-027）")
	}
	return nil
}
