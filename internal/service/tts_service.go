package service

import (
	"context"
	"crypto/rand"
)

// TextToSpeech 占位 TTS 实现，返回随机音频数据。
func TextToSpeech(_ context.Context, _ string) ([]byte, error) {
	audio := make([]byte, 512)
	if _, err := rand.Read(audio); err != nil {
		return nil, err
	}
	return audio, nil
}
