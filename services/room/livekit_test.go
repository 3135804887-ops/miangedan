package room

import (
	"context"
	"strings"
	"testing"
)

func TestNewLiveKitRoomProviderRejectsEmptyURL(t *testing.T) {
	if _, err := NewLiveKitRoomProvider("  "); err == nil {
		t.Fatal("expected error for empty base url")
	}
}

func TestLiveKitRoomProviderCreateRoom(t *testing.T) {
	p, err := NewLiveKitRoomProvider("ws://localhost:7880/")
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	ref, err := p.CreateRoom(context.Background(), CreateRoomInput{
		SessionID:  "s-123",
		DataRegion: "cn",
		ProjectID:  "p-1",
	})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	if ref.Provider != "livekit_selfhost" {
		t.Fatalf("unexpected provider: %s", ref.Provider)
	}
	if ref.RoomID != "room-s-123" {
		t.Fatalf("unexpected room id: %s", ref.RoomID)
	}
	if ref.URL != "ws://localhost:7880" {
		t.Fatalf("trailing slash not trimmed: %s", ref.URL)
	}
	if strings.Contains(ref.URL, "/") == false {
		t.Fatal("expected ws url")
	}
}

func TestLiveKitRoomProviderRequiresSession(t *testing.T) {
	p, err := NewLiveKitRoomProvider("ws://localhost:7880")
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	if _, err := p.CreateRoom(context.Background(), CreateRoomInput{}); err == nil {
		t.Fatal("expected error for empty session id")
	}
}
