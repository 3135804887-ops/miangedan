package asr

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSelfHostProviderTurnLevelTranscribe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !bytes.Contains(body, []byte("WAVE")) {
			t.Fatal("expected WAV payload")
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Fatal("expected multipart")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"你好，我是面试官"}`))
	}))
	defer server.Close()

	client, err := NewSelfHostClient(server.URL, "")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	provider, err := NewSelfHostProvider(client)
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	stream, err := provider.OpenStream(context.Background(), StreamConfig{
		Language:          "zh-CN",
		SilenceEndpointMs: 800,
	})
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	pcm := make([]byte, 3200) // 100ms @16k mono PCM16
	for i := range pcm {
		pcm[i] = byte(i % 251)
	}
	if err := stream.SendAudio(AudioFrame{Seq: 1, Speech: true, PCM: pcm}); err != nil {
		t.Fatalf("send audio: %v", err)
	}
	if err := stream.SendAudio(AudioFrame{Seq: 2, Speech: true, PCM: pcm}); err != nil {
		t.Fatalf("send audio: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	if event.Kind != EventFinal {
		t.Fatalf("unexpected kind: %s", event.Kind)
	}
	if event.FinalText != "你好，我是面试官" {
		t.Fatalf("unexpected text: %s", event.FinalText)
	}
}

func TestBuildWAV16kMonoHeader(t *testing.T) {
	wav, err := buildWAV16kMono(make([]byte, 3200))
	if err != nil {
		t.Fatalf("build wav: %v", err)
	}
	if len(wav) != 44+3200 {
		t.Fatalf("unexpected length: %d", len(wav))
	}
	if string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatal("invalid RIFF header")
	}
	if got := binary.LittleEndian.Uint32(wav[40:44]); got != 3200 {
		t.Fatalf("unexpected data chunk size: %d", got)
	}
}

func TestBuildWAVRejectsOddPCM(t *testing.T) {
	if _, err := buildWAV16kMono([]byte{0x00}); err == nil {
		t.Fatal("expected error for odd pcm length")
	}
}

func TestNewSelfHostProviderRejectsNilClient(t *testing.T) {
	if _, err := NewSelfHostProvider(nil); err == nil {
		t.Fatal("expected error for nil client")
	}
}
