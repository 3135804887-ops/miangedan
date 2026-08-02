// TASK-027 输入模式与便利设置会前冻结（FR-015、FR-016）。
// 追踪：SCREEN-SPEC SCR-07（会前检查页）；realtime-events 7.1（session.pre_check_passed）；
// INTERVIEW-STATE-MACHINE（PRE_CHECK → AVATAR_CONNECTING）；SCORING-SPEC（便利设置不视为弱点）。
package room

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"miangedan/services/project"
)

// InputMode 为会话输入模式（openapi input_modes_used 枚举）。
type InputMode string

// 输入模式枚举：语音/文字/摄像头/岗位工具。
const (
	ModeVoice   InputMode = "voice"
	ModeText    InputMode = "text"
	ModeCamera  InputMode = "camera"
	ModeJobTool InputMode = "job_tool"
)

// 摄像头/麦克风可关（FR-016）；数字人音视频始终开启，无"关闭数字人"模式。
var validInputModes = map[InputMode]bool{
	ModeVoice: true, ModeText: true, ModeCamera: true, ModeJobTool: true,
}

// DeviceReport 为会前设备检测报告（session.pre_check_passed）。
type DeviceReport struct {
	CameraOK     bool   `json:"camera_ok"`
	MicOK        bool   `json:"mic_ok"`
	NetworkRated string `json:"network_rated"` // good / fair / poor
}

// PreCheck 为会前检查冻结结果（SCR-07；确认后不可修改）。
type PreCheck struct {
	SessionID      string
	DataRegion     string
	InputModes     []InputMode
	Accommodations []string
	DeviceReport   DeviceReport
	Frozen         bool
	FrozenAt       *time.Time
}

// FreezePreCheckInput 为会前冻结入参。
type FreezePreCheckInput struct {
	InputModes     []InputMode
	Accommodations []string
	DeviceReport   DeviceReport
}

// 会前冻结错误。
var (
	ErrPreCheckInvalid = errors.New("invalid precheck")
	ErrPreCheckFrozen  = errors.New("precheck already frozen")
)

func normalizeModes(modes []InputMode) ([]InputMode, error) {
	if len(modes) == 0 {
		return nil, fmt.Errorf("%w: 至少启用一种输入模式", ErrPreCheckInvalid)
	}
	seen := make(map[InputMode]bool)
	out := make([]InputMode, 0, len(modes))
	for _, m := range modes {
		if !validInputModes[m] {
			return nil, fmt.Errorf("%w: 未知输入模式 %q", ErrPreCheckInvalid, m)
		}
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out, nil
}

// FreezePreCheck 冻结会前配置（session.pre_check_passed）：PRE_CHECK/ROOM_CREATED →
// AVATAR_CONNECTING。数字人音视频始终开启（无关闭选项）；便利设置来自计划冻结范围。
func (s *Service) FreezePreCheck(ctx context.Context, actor project.Actor, sessionID string, in FreezePreCheckInput, idemKey string) (PreCheck, error) {
	if err := s.validateActor(actor); err != nil {
		return PreCheck{}, err
	}
	modes, err := normalizeModes(in.InputModes)
	if err != nil {
		return PreCheck{}, err
	}
	accom := make([]string, 0, len(in.Accommodations))
	accomSet := make(map[string]bool)
	for _, a := range in.Accommodations {
		ok := false
		for _, allowed := range project.Accommodations {
			if a == allowed {
				ok = true
				break
			}
		}
		if !ok {
			return PreCheck{}, fmt.Errorf("%w: 未知便利设置 %q", ErrPreCheckInvalid, a)
		}
		if !accomSet[a] {
			accomSet[a] = true
			accom = append(accom, a)
		}
	}
	switch in.DeviceReport.NetworkRated {
	case "good", "fair", "poor":
	default:
		return PreCheck{}, fmt.Errorf("%w: network_rated 必须为 good|fair|poor", ErrPreCheckInvalid)
	}
	return idempotent(s, "precheck|"+actor.UserID+"|"+actor.DataRegion+"|", idemKey, func() (PreCheck, error) {
		sess, err := s.ownSession(actor, sessionID)
		if err != nil {
			return PreCheck{}, err
		}
		if existing, err := s.store.GetPreCheck(actor.DataRegion, sessionID); err == nil && existing.Frozen {
			return PreCheck{}, ErrPreCheckFrozen
		}
		if sess.RoomStatus != StatusRoomCreated && sess.RoomStatus != StatusPreCheck {
			return PreCheck{}, fmt.Errorf("%w: 仅 ROOM_CREATED/PRE_CHECK 可冻结会前配置", ErrStateConflict)
		}
		now := s.now()
		pc := PreCheck{
			SessionID:      sessionID,
			DataRegion:     actor.DataRegion,
			InputModes:     modes,
			Accommodations: accom,
			DeviceReport:   in.DeviceReport,
			Frozen:         true,
			FrozenAt:       &now,
		}
		if err := s.store.SavePreCheck(pc); err != nil {
			return PreCheck{}, err
		}
		sess.RoomStatus = StatusAvatarConnecting
		sess.LastActivityAt = now
		if err := s.store.UpdateSession(sess); err != nil {
			return PreCheck{}, err
		}
		return pc, nil
	})
}

// GetPreCheck 读取会前冻结配置。
func (s *Service) GetPreCheck(_ context.Context, actor project.Actor, sessionID string) (PreCheck, error) {
	if err := s.validateActor(actor); err != nil {
		return PreCheck{}, err
	}
	if _, err := s.ownSession(actor, sessionID); err != nil {
		return PreCheck{}, err
	}
	return s.store.GetPreCheck(actor.DataRegion, sessionID)
}

// MarshalPreCheckJSON 为前端/实时事件输出（开放 API payload）。
func MarshalPreCheckJSON(pc PreCheck) (json.RawMessage, error) {
	return json.Marshal(pc)
}
