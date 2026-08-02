package room

import (
	"context"
	"errors"
)

// ProviderRef 为房间提供方引用（供应商中立，ADR-0003；LiveKit 为技术基线）。
type ProviderRef struct {
	Provider string
	RoomID   string
	URL      string
}

// CreateRoomInput 为建房间入参（不含任何业务正文，只含匿名技术标识）。
type CreateRoomInput struct {
	SessionID  string
	DataRegion string
	ProjectID  string
}

// Provider 为房间提供方适配层契约（TASK-020 阶段以桩实现，真实接入随供应商选型）。
type Provider interface {
	CreateRoom(context.Context, CreateRoomInput) (ProviderRef, error)
}

// StubRoomProvider 为合成桩：返回稳定房间引用，不产生真实媒体面（TASK-020 验收为令牌与设备绑定）。
type StubRoomProvider struct{}

// CreateRoom 返回确定性房间引用（合成，仅测试/开发）。
func (StubRoomProvider) CreateRoom(_ context.Context, in CreateRoomInput) (ProviderRef, error) {
	if in.SessionID == "" {
		return ProviderRef{}, errors.New("session id required")
	}
	return ProviderRef{
		Provider: "stub",
		RoomID:   "room-" + in.SessionID,
		URL:      "wss://stub.miangedan.example/room/" + in.SessionID,
	}, nil
}
