package room

import (
	"context"
	"errors"
	"strings"
)

// LiveKitRoomProvider 为自建 LiveKit SFU 的房间提供方（OD-01 自建矩阵；ADR-0003）。
// 房间由媒体令牌首次入会时隐式创建（LiveKit 语义），本提供方只返回确定性房间引用。
type LiveKitRoomProvider struct {
	baseURL string
}

// NewLiveKitRoomProvider 构造 LiveKit 提供方；baseURL 形如 ws://localhost:7880 或 wss://<domain>。
func NewLiveKitRoomProvider(baseURL string) (*LiveKitRoomProvider, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("livekit base url required")
	}
	return &LiveKitRoomProvider{baseURL: strings.TrimRight(baseURL, "/")}, nil
}

// CreateRoom 返回 LiveKit 房间引用（provider 标识与配置侧对齐：livekit_selfhost）。
func (p *LiveKitRoomProvider) CreateRoom(_ context.Context, in CreateRoomInput) (ProviderRef, error) {
	if in.SessionID == "" {
		return ProviderRef{}, errors.New("session id required")
	}
	return ProviderRef{
		Provider: "livekit_selfhost",
		RoomID:   "room-" + in.SessionID,
		URL:      p.baseURL,
	}, nil
}
