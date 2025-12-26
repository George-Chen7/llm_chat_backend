package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"backend/internal/store"
)

// RequestTTSURL ?? Dashscope TTS ????? URL?
func RequestTTSURL(ctx context.Context, userID, messageID int) (string, error) {
	totalQuota, usedQuota, err := store.GetUserQuotaUsage(ctx, userID)
	if err != nil {
		return "", err
	}
	if usedQuota >= totalQuota {
		return "", ErrQuotaExceeded
	}
	text, err := store.GetMessageContent(ctx, userID, messageID)
	if err != nil {
		return "", err
	}
	text = sanitizeTTSText(text)
	if text == "" {
		return "", errors.New("missing text")
	}
	audioURL, tokens, err := DashscopeTTS(ctx, text, "", "")
	if err != nil {
		return "", err
	}
	if tokens > 0 {
		if err := store.IncreaseUserUsedQuota(ctx, userID, tokens); err != nil {
			return "", err
		}
	}
	return audioURL, nil
}

// TextToSpeech 调用 Dashscope TTS 并返回音频数据。
func TextToSpeech(ctx context.Context, userID, messageID int) ([]byte, error) {
	audioURL, err := RequestTTSURL(ctx, userID, messageID)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, audioURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("dashscope audio download failed")
	}
	return io.ReadAll(resp.Body)
}

// StreamTextToSpeech streams Dashscope TTS audio chunks to writer.
func StreamTextToSpeech(ctx context.Context, userID, messageID int, writer io.Writer, flush func()) error {
	totalQuota, usedQuota, err := store.GetUserQuotaUsage(ctx, userID)
	if err != nil {
		return err
	}
	if usedQuota >= totalQuota {
		return ErrQuotaExceeded
	}
	text, err := store.GetMessageContent(ctx, userID, messageID)
	if err != nil {
		return err
	}
	text = sanitizeTTSText(text)
	if text == "" {
		return errors.New("missing text")
	}

	lastTokens := 0
	onChunk := func(chunk []byte) {
		if len(chunk) == 0 {
			return
		}
		_, _ = writer.Write(chunk)
		if flush != nil {
			flush()
		}
	}
	onUsage := func(tokens int) {
		if tokens > 0 {
			lastTokens = tokens
		}
	}
	if err := DashscopeTTSStream(ctx, text, "", "", onChunk, onUsage); err != nil {
		return err
	}
	if lastTokens > 0 {
		if err := store.IncreaseUserUsedQuota(ctx, userID, lastTokens); err != nil {
			return err
		}
	}
	return nil
}

func sanitizeTTSText(text string) string {
	if text == "" {
		return ""
	}
	text = strings.ReplaceAll(text, "\r\n", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	return strings.Join(strings.Fields(text), " ")
}
