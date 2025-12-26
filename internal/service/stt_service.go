package service

import (
	"context"
	"io"
)

// STTResult 语音识别结果。
type STTResult struct {
	AudioText   string
	AudioTokens int
}

// SpeechToText 占位 STT 实现。
func SpeechToText(_ context.Context, reader io.Reader) (STTResult, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return STTResult{}, err
	}
	return STTResult{
		AudioText:   "hello, this is an example text recognized from audio.",
		AudioTokens: len(data) / 10,
	}, nil
}
