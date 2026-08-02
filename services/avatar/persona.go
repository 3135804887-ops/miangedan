package avatar

import "fmt"

// 人格参数枚举（对齐 config/interview-flows style_parameters 边界）。
var (
	validTones     = []string{"professional", "friendly", "neutral"}
	validPaces     = []string{"slow", "normal", "fast"}
	validIntensity = []string{"low", "medium", "high"}
	validHint      = []string{"none", "low", "high"}
	validPressure  = []string{"low", "standard", "challenge"}
)

// ValidatePersona 校验动态面试官人格：参数有界（style_parameters 枚举）。
// 人格字段为封闭枚举、无自由文本，结构上不可能携带候选保护属性/外貌/情绪（安全红线）。
func ValidatePersona(p Persona) error {
	if !contains(validTones, p.Tone) || !contains(validPaces, p.Pace) ||
		!contains(validIntensity, p.FollowupIntensity) ||
		!contains(validHint, p.HintLevel) || !contains(validPressure, p.PressureLevel) {
		return fmt.Errorf("%w: 人格参数越界（tone/pace/followup/hint/pressure）", ErrInvalidPersona)
	}
	return nil
}

// DefaultPersona 返回合成默认人格（开发/测试）。
func DefaultPersona() Persona {
	return Persona{
		Tone: "professional", Pace: "normal", FollowupIntensity: "medium",
		PoliteInterruptionAllowed: true, HintLevel: "low", PressureLevel: "standard",
	}
}

// ValidateVideoProfile 校验视频档位（NFR-012：默认 ≥720p/24fps；弱网可降码率但音频连续）。
func ValidateVideoProfile(p VideoProfile) error {
	if p.Width <= 0 || p.Height <= 0 || p.FPS <= 0 {
		return ErrInvalidVideoProfile
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
