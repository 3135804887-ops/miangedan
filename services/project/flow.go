package project

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FlowConfig 为 config/interview-flows/v1/default.yaml 的子集（TASK-016 所需边界与轮次注册）。
type FlowConfig struct {
	Bounds struct {
		Rounds struct {
			UserConfigurable struct {
				Min int `yaml:"min"`
				Max int `yaml:"max"`
			} `yaml:"user_configurable"`
			DefaultRecommended int `yaml:"default_recommended"`
		} `yaml:"rounds"`
		DurationMinutes struct {
			UserConfigurable struct {
				Min int `yaml:"min"`
				Max int `yaml:"max"`
			} `yaml:"user_configurable"`
		} `yaml:"duration_minutes"`
	} `yaml:"bounds"`
	RoundTypes []struct {
		Key                       string   `yaml:"key"`
		DefaultCriticalDimensions []string `yaml:"default_critical_dimensions"`
		ToolCapabilities          []string `yaml:"tool_capabilities"`
	} `yaml:"round_types"`
}

// LoadFlowConfig 读取面试流程配置；默认仓库内默认流程文件，可用 MGD_FLOW_CONFIG 覆盖。
func LoadFlowConfig(path string) (*FlowConfig, error) {
	if path == "" {
		path = os.Getenv("MGD_FLOW_CONFIG")
	}
	if path == "" {
		candidates := []string{
			"../../config/interview-flows/v1/default.yaml",    // services/project
			"../../../config/interview-flows/v1/default.yaml", // services/project/httpapi
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				path = c
				break
			}
		}
	}
	// #nosec G304,G703 -- 流程配置文件路径为部署显式配置，非不可信输入
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取面试流程配置失败: %w", err)
	}
	var fc FlowConfig
	if err := yaml.Unmarshal(raw, &fc); err != nil {
		return nil, fmt.Errorf("解析面试流程配置失败: %w", err)
	}
	if fc.Bounds.Rounds.UserConfigurable.Min < 1 || fc.Bounds.Rounds.UserConfigurable.Max < fc.Bounds.Rounds.UserConfigurable.Min {
		return nil, fmt.Errorf("流程配置轮次边界非法: %+v", fc.Bounds.Rounds)
	}
	if fc.Bounds.DurationMinutes.UserConfigurable.Min < 1 || fc.Bounds.DurationMinutes.UserConfigurable.Max < fc.Bounds.DurationMinutes.UserConfigurable.Min {
		return nil, fmt.Errorf("流程配置时长边界非法: %+v", fc.Bounds.DurationMinutes)
	}
	if len(fc.RoundTypes) == 0 {
		return nil, fmt.Errorf("流程配置缺少轮次类型注册")
	}
	return &fc, nil
}
