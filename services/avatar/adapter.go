package avatar

import (
	"miangedan/services/provider"
	"miangedan/services/region"
)

// RegisterDriver 将数字人驱动注册为 TASK-030 注册表中的 CapAvatar 供应商
// （provider_id = avatar_{region}_{role}；版本固定）。
func RegisterDriver(reg *provider.Registry, driver Driver, dataRegion string, role provider.Role, version string) error {
	if driver == nil || reg == nil {
		return ErrCharacterNotFound
	}
	if err := region.ValidateDataRegion(dataRegion); err != nil {
		return err
	}
	info := provider.Info{
		ProviderID: "avatar_" + dataRegion + "_" + string(role),
		Capability: provider.CapAvatar,
		DataRegion: dataRegion,
		Languages:  []string{"zh-CN", "en-US"},
		Role:       role,
		Version:    version,
	}
	health := func() error { return nil } // 合成探针；真实探针随供应商接入
	return reg.Register(info, health)
}
