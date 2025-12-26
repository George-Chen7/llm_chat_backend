package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"backend/internal/config"
)

var (
	dashscopeConfig config.DashscopeConfig
	dashscopeClient = &http.Client{Timeout: 60 * time.Second}

	// ErrDashscopeNotReady Dashscope config not initialized.
	ErrDashscopeNotReady = errors.New("dashscope not initialized")
)

// InitDashscope stores Dashscope config for later use.
func InitDashscope(cfg config.DashscopeConfig) {
	dashscopeConfig = cfg
}

type dashscopeRequest struct {
	Model string `json:"model"`
	Input struct {
		Messages []dashscopeMessage `json:"messages"`
	} `json:"input"`
}

type dashscopeMessage struct {
	Role    string               `json:"role"`
	Content []dashscopeContentIn `json:"content"`
}

type dashscopeContentIn struct {
	Audio string `json:"audio,omitempty"`
	Text  string `json:"text,omitempty"`
}

type dashscopeResponse struct {
	StatusCode int    `json:"status_code"`
	RequestID  string `json:"request_id"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Output     struct {
		Choices []struct {
			Message struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		AudioTokens  int `json:"audio_tokens"`
	} `json:"usage"`
}

// DashscopeAudioASR calls Dashscope ASR with audio URL.
func DashscopeAudioASR(ctx context.Context, audioURL string) (string, int, error) {
	if dashscopeConfig.APIKey == "" {
		return "", 0, ErrDashscopeNotReady
	}
	endpoint := dashscopeConfig.STT.Endpoint
	if endpoint == "" {
		endpoint = "https://dashscope.aliyuncs.com/api/v1/services/aigc/multimodal-generation/generation"
	}
	model := dashscopeConfig.STT.Model
	if model == "" {
		model = "qwen-audio-asr"
	}

	var req dashscopeRequest
	req.Model = model
	req.Input.Messages = []dashscopeMessage{
		{
			Role: "user",
			Content: []dashscopeContentIn{
				{Audio: audioURL},
			},
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", 0, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+dashscopeConfig.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := dashscopeClient.Do(httpReq)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	var result dashscopeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if result.Message != "" {
			return "", 0, errors.New(result.Message)
		}
		return "", 0, errors.New("dashscope request failed")
	}

	text := ""
	if len(result.Output.Choices) > 0 && len(result.Output.Choices[0].Message.Content) > 0 {
		text = result.Output.Choices[0].Message.Content[0].Text
	}
	totalTokens := result.Usage.InputTokens + result.Usage.OutputTokens + result.Usage.AudioTokens
	return text, totalTokens, nil
}

type dashscopeTTSRequest struct {
	Model string `json:"model"`
	Input struct {
		Text         string `json:"text"`
		Voice        string `json:"voice,omitempty"`
		LanguageType string `json:"language_type,omitempty"`
	} `json:"input"`
}

type dashscopeTTSResponse struct {
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Output    struct {
		FinishReason string `json:"finish_reason"`
		Audio        struct {
			Data      string `json:"data"`
			URL       string `json:"url"`
			ID        string `json:"id"`
			ExpiresAt int64  `json:"expires_at"`
		} `json:"audio"`
	} `json:"output"`
	Usage struct {
		Characters   int `json:"characters"`
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// DashscopeTTS calls Dashscope TTS and returns audio URL and tokens.
func DashscopeTTS(ctx context.Context, text, voice, language string) (string, int, error) {
	if dashscopeConfig.APIKey == "" {
		return "", 0, ErrDashscopeNotReady
	}
	endpoint := dashscopeConfig.TTS.Endpoint
	model := dashscopeConfig.TTS.Model
	if endpoint == "" || model == "" {
		return "", 0, ErrDashscopeNotReady
	}
	if voice == "" {
		voice = dashscopeConfig.TTS.Voice
	}
	if voice == "" {
		return "", 0, ErrDashscopeNotReady
	}
	if language == "" {
		language = "Auto"
	}

	var req dashscopeTTSRequest
	req.Model = model
	req.Input.Text = text
	req.Input.Voice = voice
	req.Input.LanguageType = language
	body, err := json.Marshal(req)
	if err != nil {
		return "", 0, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+dashscopeConfig.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := dashscopeClient.Do(httpReq)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	var result dashscopeTTSResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || result.Code != "" {
		if result.Message != "" {
			return "", 0, errors.New(result.Message)
		}
		return "", 0, errors.New("dashscope request failed")
	}
	if result.Output.Audio.URL == "" {
		return "", 0, errors.New("dashscope audio url missing")
	}

	totalTokens := result.Usage.TotalTokens
	if totalTokens == 0 {
		totalTokens = result.Usage.InputTokens + result.Usage.OutputTokens
	}
	if totalTokens == 0 {
		totalTokens = result.Usage.Characters
	}
	return result.Output.Audio.URL, totalTokens, nil
}

// DashscopeTTSStream calls Dashscope TTS with SSE enabled and streams audio chunks.
func DashscopeTTSStream(ctx context.Context, text, voice, language string, onChunk func([]byte), onUsage func(int)) error {
	if dashscopeConfig.APIKey == "" {
		return ErrDashscopeNotReady
	}
	endpoint := dashscopeConfig.TTS.Endpoint
	model := dashscopeConfig.TTS.Model
	if endpoint == "" || model == "" {
		return ErrDashscopeNotReady
	}
	if voice == "" {
		voice = dashscopeConfig.TTS.Voice
	}
	if voice == "" {
		return ErrDashscopeNotReady
	}
	if language == "" {
		language = "Auto"
	}

	var req dashscopeTTSRequest
	req.Model = model
	req.Input.Text = text
	req.Input.Voice = voice
	req.Input.LanguageType = language
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+dashscopeConfig.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-DashScope-SSE", "enable")

	resp, err := dashscopeClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dashscope request failed: %s", string(b))
	}

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 5*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var result dashscopeTTSResponse
		if err := json.Unmarshal([]byte(payload), &result); err != nil {
			return err
		}
		if result.Code != "" {
			if result.Message != "" {
				return errors.New(result.Message)
			}
			return errors.New("dashscope request failed")
		}

		if result.Output.Audio.Data != "" {
			chunk, err := base64.StdEncoding.DecodeString(result.Output.Audio.Data)
			if err != nil {
				return err
			}
			if len(chunk) > 0 && onChunk != nil {
				onChunk(chunk)
			}
		}
		if onUsage != nil {
			totalTokens := result.Usage.TotalTokens
			if totalTokens == 0 {
				totalTokens = result.Usage.InputTokens + result.Usage.OutputTokens
			}
			if totalTokens == 0 {
				totalTokens = result.Usage.Characters
			}
			if totalTokens > 0 {
				onUsage(totalTokens)
			}
		}
		if result.Output.FinishReason != "null" {
			if result.Output.FinishReason != "stop" {
				return errors.New("dashscope tts finished: " + result.Output.FinishReason)
			}
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
