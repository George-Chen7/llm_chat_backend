package service

import (
	"context"
	"database/sql"
	"errors"

	"backend/internal/llm"
	"backend/internal/store"
)

var (
	// ErrConversationNotFound 对话不存在。
	ErrConversationNotFound = errors.New("conversation not found")
	// ErrLLMNotReady 模型服务不可用。
	ErrLLMNotReady = errors.New("llm not initialized")
)

// SendMessage 发送消息并写入用户消息与模型回复。
func SendMessage(ctx context.Context, userID, conversationID int, contentType, content string, attachmentIDs []int) (int, int, []store.AttachmentInfo, string, error) {
	if _, err := store.GetConversation(ctx, conversationID, userID); err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, nil, "", ErrConversationNotFound
		}
		return 0, 0, nil, "", err
	}

	client := llm.Get()
	if client == nil {
		return 0, 0, nil, "", ErrLLMNotReady
	}
	reply, err := client.ChatCompletion(ctx, content)
	if err != nil {
		return 0, 0, nil, "", err
	}

	userMsgID, err := store.InsertMessage(ctx, conversationID, store.SenderUser, contentType, content, len(content))
	if err != nil {
		return 0, 0, nil, "", err
	}

	if err := store.AttachFilesToMessage(ctx, userID, userMsgID, attachmentIDs); err != nil {
		return 0, 0, nil, "", err
	}

	attachments := make([]store.AttachmentInfo, 0)
	if len(attachmentIDs) > 0 {
		attachmentsMap, err := store.LoadAttachmentsMap(ctx, []int{userMsgID})
		if err != nil {
			return 0, 0, nil, "", err
		}
		attachments = attachmentsMap[userMsgID]
	}

	modelMsgID, err := store.InsertMessage(ctx, conversationID, store.SenderAssistant, "TEXT", reply, len(reply))
	if err != nil {
		return 0, 0, nil, "", err
	}

	return userMsgID, modelMsgID, attachments, reply, nil
}

// NewConversation 创建会话。
func NewConversation(ctx context.Context, userID int, title string, systemPrompt sql.NullInt64, llmModel string) (store.ConversationInfo, error) {
	return store.CreateConversation(ctx, userID, title, llmModel, systemPrompt)
}

// RenameConversation 重命名会话。
func RenameConversation(ctx context.Context, userID, conversationID int, title string) (bool, error) {
	return store.RenameConversation(ctx, conversationID, userID, title)
}

// DeleteConversation 删除会话。
func DeleteConversation(ctx context.Context, userID, conversationID int) (bool, error) {
	return store.DeleteConversation(ctx, conversationID, userID)
}

// GetHistory 获取会话消息历史。
func GetHistory(ctx context.Context, userID, conversationID, page, pageSize int) ([]store.MessageRow, map[int][]store.AttachmentInfo, int, error) {
	totalCount, err := store.CountMessages(ctx, userID, conversationID)
	if err != nil {
		return nil, nil, 0, err
	}
	items, messageIDs, err := store.ListMessages(ctx, userID, conversationID, page, pageSize)
	if err != nil {
		return nil, nil, 0, err
	}
	attachmentsMap, err := store.LoadAttachmentsMap(ctx, messageIDs)
	if err != nil {
		return nil, nil, 0, err
	}
	return items, attachmentsMap, totalCount, nil
}
