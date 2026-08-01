// Package secretref 提供密钥引用契约与展示脱敏（TASK-006）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-006；docs/architecture/EPIC-01-INFRA-DESIGN.md 第 6 节；
// docs/security/SECURITY-REQUIREMENTS.md SEC-012、4.7 密钥轮换周期表。
package secretref

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// RefSuffix 为密钥管理系统引用变量的统一后缀（SECURITY-REQUIREMENTS 4.7 引用模式）。
const RefSuffix = "_REF"

var envVarNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// inlineSecretPatterns 识别"引用值里塞了真实密钥"的常见形态（fail-closed）。
var inlineSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`),
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/\-=]{16,}`),
	regexp.MustCompile(`sk-[A-Za-z0-9_-]{16,}`),
	regexp.MustCompile(`[A-Za-z0-9]{32,}`),
}

// IsRefName 判断变量名是否为密钥引用（以 _REF 结尾）。
func IsRefName(name string) bool {
	return strings.HasSuffix(name, RefSuffix)
}

// ValidateRefName 校验密钥引用变量名：以 _REF 结尾且为合法环境变量名。
func ValidateRefName(name string) error {
	if !IsRefName(name) {
		return fmt.Errorf("密钥引用名 %q 必须以 %s 结尾（SECURITY-REQUIREMENTS 4.7）", name, RefSuffix)
	}
	if !envVarNamePattern.MatchString(name) {
		return fmt.Errorf("密钥引用名 %q 非法：必须为 ^[A-Z][A-Z0-9_]*$", name)
	}
	return nil
}

// ValidateEnvVarName 校验环境变量名合法性（引用值/普通凭证名共用）。
func ValidateEnvVarName(name string) error {
	if !envVarNamePattern.MatchString(name) {
		return fmt.Errorf("环境变量名 %q 非法：必须为 ^[A-Z][A-Z0-9_]*$", name)
	}
	return nil
}

// ValidateRefs 校验区域密钥引用表（regions 拓扑 secrets.refs）：
// 值必须是合法环境变量名、非空且不得内联真实密钥（fail-closed，SEC-012）。
func ValidateRefs(refs map[string]string) error {
	if len(refs) == 0 {
		return errors.New("密钥引用表为空：必须至少含一项引用")
	}
	for name, value := range refs {
		if strings.TrimSpace(name) == "" {
			return errors.New("密钥引用键名为空")
		}
		if err := ValidateEnvVarName(value); err != nil {
			return fmt.Errorf("密钥引用 %q 的值非法: %w", name, err)
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("密钥引用 %q 的值为空", name)
		}
		if looksLikeInlineSecret(value) {
			return fmt.Errorf("密钥引用 %q 的值疑似内联真实密钥：仓库/配置只允许引用名，禁止明文（SEC-012）", name)
		}
	}
	return nil
}

func looksLikeInlineSecret(value string) bool {
	for _, re := range inlineSecretPatterns {
		if re.MatchString(value) {
			return true
		}
	}
	return false
}

// MaskSecret 生成后台/日志展示用的掩码：只保留末 4 位，其余以星号代替（SEC-012）。
func MaskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}
