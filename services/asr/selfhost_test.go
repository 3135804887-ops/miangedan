package asr

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSelfHostClientTranscribeWAV(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Fatal("expected multipart content type")
		}
		if !strings.Contains(string(body), "audio.wav") {
			t.Fatal("multipart file field missing")
		}
		if !strings.Contains(string(body), "zh-CN") {
			t.Fatal("language field missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"你好，我是面试官"}`))
	}))
	defer server.Close()

	client, err := NewSelfHostClient(server.URL, "test-key")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	text, err := client.TranscribeWAV(context.Background(), []byte("RIFF...."), "zh-CN")
	if err != nil {
		t.Fatalf("transcribe: %v", err)
	}
	if text != "你好，我是面试官" {
		t.Fatalf("unexpected text: %s", text)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("unexpected auth: %s", gotAuth)
	}
}

func TestSelfHostClientRejectsEmptyWAV(t *testing.T) {
	client, err := NewSelfHostClient("http://127.0.0.1:8000", "")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.TranscribeWAV(context.Background(), nil, "zh-CN"); err == nil {
		t.Fatal("expected error for empty wav")
	}
}

func TestSelfHostClientStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()
	client, err := NewSelfHostClient(server.URL, "")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.TranscribeWAV(context.Background(), []byte("RIFF"), "zh-CN"); err == nil {
		t.Fatal("expected error for bad gateway")
	}
}

func TestNewSelfHostClientRejectsEmptyURL(t *testing.T) {
	if _, err := NewSelfHostClient("", ""); err == nil {
		t.Fatal("expected error for empty base url")
	}
}
