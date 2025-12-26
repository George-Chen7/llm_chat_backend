package service

import (
	"context"

	"backend/internal/store"
)

// GetMeInfo 获取用户信息。
func GetMeInfo(ctx context.Context, username string) (store.User, error) {
	return store.GetUserByUsername(ctx, username)
}

// ListMyConversations 获取用户会话列表。
func ListMyConversations(ctx context.Context, userID int) ([]store.ConversationInfo, error) {
	return store.ListConversationsByUser(ctx, userID)
}
