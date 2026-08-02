package asr

import (
	"miangedan/services/provider"
	"miangedan/services/region"
)

// RegisterProvider 将 ASR 提供方注册为 TASK-030 注册表的 asr_{region}_{role} 供应商（版本固定）。
func RegisterProvider(reg *provider.Registry, p Provider, dataRegion string, role provider.Role, version string) error {
	if p == nil || reg == nil {
		return ErrInvalidConfig
	}
	if err := region.ValidateDataRegion(dataRegion); err != nil {
		return err
	}
	info := provider.Info{
		ProviderID: "asr_" + dataRegion + "_" + string(role),
		Capability: provider.CapASR,
		DataRegion: dataRegion,
		Languages:  []string{"zh-CN", "en-US"},
		Role:       role,
		Version:    version,
	}
	health := func() error { return nil } // 合成探针；真实探针随供应商接入
	return reg.Register(info, health)
}
