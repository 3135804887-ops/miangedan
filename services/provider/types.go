// Package provider 提供供应商中立的能力适配层（TASK-030，EPIC-04）。
// 追踪：IMPLEMENTATION_PLAN.md TASK-030；docs/ai/PROVIDER-ADAPTERS.md；ADR-0003；
// PRD 架构决策 4；FR-037 部分；NFR-007 ~ NFR-012。
package provider

import (
	"errors"
	"fmt"
	"strings"

	"miangedan/services/region"
)

// Capability 为外部能力类别（PROVIDER-ADAPTERS §4 五类核心能力）。
type Capability string

// 能力枚举。
const (
	CapLLM    Capability = "llm"
	CapASR    Capability = "asr"
	CapTTS    Capability = "tts"
	CapAvatar Capability = "avatar"
	CapSearch Capability = "search"
)

// AllCapabilities 返回五类核心能力（稳定顺序）。
func AllCapabilities() []Capability {
	return []Capability{CapLLM, CapASR, CapTTS, CapAvatar, CapSearch}
}

// Role 为供应商角色（PROVIDER-ADAPTERS §5：primary / secondary / disabled）。
type Role string

// 角色枚举。
const (
	RolePrimary   Role = "primary"
	RoleSecondary Role = "secondary"
	RoleDisabled  Role = "disabled"
)

// Info 为注册表条目（按数据区隔离，§5）。
type Info struct {
	ProviderID string
	Capability Capability
	DataRegion string
	Languages  []string
	Role       Role
	Version    string
}

// HealthFunc 为供应商主动健康探测（低频合成探针，§6；禁用真实用户内容）。
type HealthFunc func() error

// 适配层错误（业务代码只依赖本层语义，§10 红线 1）。
var (
	ErrNoVerifiedProvider  = errors.New("no verified provider for capability in region")
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrPinnedUnavailable   = errors.New("pinned provider unavailable; active formal session must not silently switch")
	ErrInvalidProvider     = errors.New("invalid provider")
)

// AllLanguages 为已支持语言。
var AllLanguages = []string{"zh-CN", "en-US"}

// ValidateProviderID 校验供应商 ID 形态：{capability}_{region}_{role}（如 llm_cn_primary）。
func ValidateProviderID(id string) error {
	parts := strings.Split(id, "_")
	if len(parts) != 3 {
		return fmt.Errorf("%w: 供应商 ID %q 必须为 {capability}_{region}_{role}", ErrInvalidProvider, id)
	}
	capability, dataRegion, role := Capability(parts[0]), parts[1], Role(parts[2])
	if !validCapability(capability) {
		return fmt.Errorf("%w: 未知能力类别 %q", ErrInvalidProvider, capability)
	}
	if err := region.ValidateDataRegion(dataRegion); err != nil {
		return err
	}
	if role != RolePrimary && role != RoleSecondary && role != RoleDisabled {
		return fmt.Errorf("%w: 未知角色 %q", ErrInvalidProvider, role)
	}
	return nil
}

func validCapability(c Capability) bool {
	for _, a := range AllCapabilities() {
		if a == c {
			return true
		}
	}
	return false
}

// ValidateInfo 校验注册条目。
func ValidateInfo(info Info) error {
	if err := ValidateProviderID(info.ProviderID); err != nil {
		return err
	}
	if info.Capability != Capability(strings.Split(info.ProviderID, "_")[0]) {
		return fmt.Errorf("%w: 能力与 ID 不一致", ErrInvalidProvider)
	}
	if info.DataRegion != strings.Split(info.ProviderID, "_")[1] {
		return fmt.Errorf("%w: 区域与 ID 不一致", ErrInvalidProvider)
	}
	if info.Role != Role(strings.Split(info.ProviderID, "_")[2]) {
		return fmt.Errorf("%w: 角色与 ID 不一致", ErrInvalidProvider)
	}
	if len(info.Languages) == 0 {
		return fmt.Errorf("%w: 至少声明一种语言", ErrInvalidProvider)
	}
	for _, lang := range info.Languages {
		if !contains(AllLanguages, lang) {
			return fmt.Errorf("%w: 未知语言 %q", ErrInvalidProvider, lang)
		}
	}
	if strings.TrimSpace(info.Version) == "" {
		return fmt.Errorf("%w: 必须声明固定版本（§9 版本 pin）", ErrInvalidProvider)
	}
	return nil
}

func contains(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}
