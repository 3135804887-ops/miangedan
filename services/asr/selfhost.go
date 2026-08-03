package asr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// SelfHostClient 为自建 ASR 服务（mgd-selfhost）的 HTTP 客户端（OD-01 自建矩阵）。
// 当前提供整段 WAV 转写；流式 OpenStream 适配随媒体链路演进接入。
type SelfHostClient struct {
	baseURL string
	client  *http.Client
	apiKey  string
}

// NewSelfHostClient 构造自建 ASR 客户端；baseURL 形如 http://127.0.0.1:8000。
func NewSelfHostClient(baseURL string, apiKey string) (*SelfHostClient, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("%w: 自建 ASR base url 必填", ErrInvalidConfig)
	}
	return &SelfHostClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
		apiKey:  apiKey,
	}, nil
}

// Transcription 为转写结果。
type Transcription struct {
	Text string `json:"text"`
}

// TranscribeWAV 上传 WAV 音频并返回最终文本（开发期闭环；只接受合成素材）。
func (c *SelfHostClient) TranscribeWAV(ctx context.Context, wav []byte, language string) (string, error) {
	if len(wav) == 0 {
		return "", errors.New("empty wav")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", err
	}
	if _, err := fileWriter.Write(wav); err != nil {
		return "", err
	}
	if err := writer.WriteField("language", language); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+"/v1/asr/transcribe", &body,
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return "", fmt.Errorf(
			"asr service status %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(data)),
		)
	}
	var result Transcription
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Text, nil
}
