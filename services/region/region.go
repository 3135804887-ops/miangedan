// Package region 提供三数据区（cn / eu / intl）的枚举、fail-closed 启动自检与区域路由决策。
// 追踪：IMPLEMENTATION_PLAN.md TASK-002；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 5 节；
// docs/architecture/DEPLOYMENT.md 第 9 节；ADR-0005；OD-09。
package region

import "fmt"

// Region 为 ADR-0005 批准的数据区代码（OD-09）。
type Region string

// 已批准数据区枚举。
const (
	CN   Region = "cn"
	EU   Region = "eu"
	INTL Region = "intl"
)

// AllRegions 返回全部批准数据区。
var AllRegions = []Region{CN, EU, INTL}

// validEnvironments 为部署环境白名单。
var validEnvironments = map[string]bool{"dev": true, "staging": true, "production": true}

// MismatchReason 为区域不匹配时的稳定原因码（与 openapi.yaml 的 region_mismatch 一致）。
const MismatchReason = "region_mismatch"

// String 返回区域代码原文。
func (r Region) String() string { return string(r) }

// Valid 判断字符串是否为批准数据区。
func Valid(value string) bool {
	for _, r := range AllRegions {
		if r == Region(value) {
			return true
		}
	}
	return false
}

// ValidateDataRegion 校验 DATA_REGION，非法即报错（fail-closed，ADR-0005）。
func ValidateDataRegion(value string) error {
	if !Valid(value) {
		return fmt.Errorf("DATA_REGION %q 非法：必须为 cn | eu | intl（fail-closed，ADR-0005）", value)
	}
	return nil
}

// ValidateEnvironment 校验 SERVICE_ENV。
func ValidateEnvironment(value string) error {
	if !validEnvironments[value] {
		return fmt.Errorf("SERVICE_ENV %q 非法：必须为 dev | staging | production", value)
	}
	return nil
}

// CheckStartup 执行服务启动自检：DATA_REGION、INFRA_REGION 与 SERVICE_ENV 全部合法，
// 且 DATA_REGION 必须与所连基础设施区域 INFRA_REGION 一致（TASK-002 fail-closed）。
func CheckStartup(dataRegion, infraRegion, serviceEnv string) error {
	if err := ValidateDataRegion(dataRegion); err != nil {
		return err
	}
	if err := ValidateEnvironment(serviceEnv); err != nil {
		return err
	}
	if !Valid(infraRegion) {
		return fmt.Errorf("INFRA_REGION %q 非法：必须为 cn | eu | intl（fail-closed，ADR-0005）", infraRegion)
	}
	if dataRegion != infraRegion {
		return fmt.Errorf(
			"DATA_REGION=%q 与 INFRA_REGION=%q 不一致：拒绝启动（防止静默跨区，ADR-0005）",
			dataRegion, infraRegion,
		)
	}
	return nil
}

// RouteDecision 为区域路由决策结果。
type RouteDecision struct {
	// Allowed 表示请求是否放行。
	Allowed bool
	// AccountRegion 为账户硬归属数据区。
	AccountRegion Region
	// RequestRegion 为请求入口数据区。
	RequestRegion Region
	// Reason 为拒绝原因；放行时为空，拒绝时为 region_mismatch。
	Reason string
}

// Route 计算请求路由决策：账户归属区域与请求入口区域必须一致，否则拒绝并给出
// region_mismatch 原因（SEC-051，docs/api/openapi.yaml 错误码）。
func Route(accountRegion, requestRegion string) (RouteDecision, error) {
	account, err := Parse(accountRegion)
	if err != nil {
		return RouteDecision{}, err
	}
	request, err := Parse(requestRegion)
	if err != nil {
		return RouteDecision{}, err
	}
	if account != request {
		return RouteDecision{
			Allowed:       false,
			AccountRegion: account,
			RequestRegion: request,
			Reason:        MismatchReason,
		}, nil
	}
	return RouteDecision{Allowed: true, AccountRegion: account, RequestRegion: request}, nil
}

// Parse 严格解析区域代码（大小写与空白均不宽容）。
func Parse(value string) (Region, error) {
	if !Valid(value) {
		return "", fmt.Errorf("区域 %q 非法：必须为 cn | eu | intl", value)
	}
	return Region(value), nil
}
