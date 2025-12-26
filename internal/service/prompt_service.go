package service

import (
	"context"

	"backend/internal/store"
)

// ListPromptPresets 获取提示词列表。
func ListPromptPresets(ctx context.Context) ([]store.PromptPreset, error) {
	return store.ListPromptPresets(ctx)
}

// CreatePromptPreset 新增提示词。
func CreatePromptPreset(ctx context.Context, name, description, content string) error {
	return store.CreatePromptPreset(ctx, name, description, content)
}

// DeletePromptPreset 删除提示词。
func DeletePromptPreset(ctx context.Context, id int) (bool, error) {
	return store.DeletePromptPreset(ctx, id)
}
