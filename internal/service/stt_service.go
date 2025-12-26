package service

import (
	"context"
	"errors"
	"io"

	"backend/internal/store"
)

// STTResult 语音识别结果。
type STTResult struct {
	AudioText   string
	AudioTokens int
}

// SpeechToText 调用 Dashscope ASR。
func SpeechToText(ctx context.Context, userID int, filename, mimeType string, reader io.Reader) (STTResult, error) {
	if reader == nil {
		return STTResult{}, errors.New("missing audio")
	}

	totalQuota, usedQuota, err := store.GetUserQuotaUsage(ctx, userID)
	if err != nil {
		return STTResult{}, err
	}
	if usedQuota >= totalQuota {
		return STTResult{}, ErrQuotaExceeded
	}

	if ossClient == nil {
		return STTResult{}, ErrOSSNotReady
	}

	objectKey := BuildOSSObjectKey("stt", filename)
	if err := PutObjectToOSS(ctx, objectKey, reader, mimeType); err != nil {
		return STTResult{}, err
	}
	audioURL, err := PresignGetURL(ctx, objectKey, 0)
	if err != nil {
		return STTResult{}, err
	}

	text, totalTokens, err := DashscopeAudioASR(ctx, audioURL)
	if err != nil {
		return STTResult{}, err
	}
	if totalTokens > 0 {
		if err := store.IncreaseUserUsedQuota(ctx, userID, totalTokens); err != nil {
			return STTResult{}, err
		}
	}

	return STTResult{
		AudioText:   text,
		AudioTokens: totalTokens,
	}, nil
}
