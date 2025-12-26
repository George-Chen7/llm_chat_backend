package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"backend/internal/llm"
	"backend/internal/store"

	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

var (
	// ErrConversationNotFound 对话不存在。
	ErrConversationNotFound = errors.New("conversation not found")
	// ErrLLMNotReady 模型服务不可用。
	ErrLLMNotReady = errors.New("llm not initialized")
	// ErrQuotaExceeded ?????
	ErrQuotaExceeded = errors.New("quota exceeded")
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

	totalQuota, usedQuota, err := store.GetUserQuotaUsage(ctx, userID)
	if err != nil {
		return 0, 0, nil, "", err
	}
	if usedQuota >= totalQuota {
		return 0, 0, nil, "", ErrQuotaExceeded
	}
	attachmentsForLLM, err := store.LoadAttachmentsByIDs(ctx, userID, attachmentIDs)
	if err != nil {
		return 0, 0, nil, "", err
	}
	historyItems, historyIDs, err := store.ListAllMessages(ctx, userID, conversationID)
	if err != nil {
		return 0, 0, nil, "", err
	}
	historyAttachments, err := store.LoadAttachmentsMap(ctx, historyIDs)
	if err != nil {
		return 0, 0, nil, "", err
	}
	messages, err := buildLLMMessages(ctx, historyItems, historyAttachments, content, attachmentsForLLM)
	if err != nil {
		return 0, 0, nil, "", err
	}
	reply, usage, err := client.ChatCompletion(ctx, messages)
	if err != nil {
		return 0, 0, nil, "", err
	}
	totalTokens := usage.TotalTokens
	if totalTokens == 0 {
		totalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	if totalTokens > 0 {
		if err := store.IncreaseUserUsedQuota(ctx, userID, totalTokens); err != nil {
			return 0, 0, nil, "", err
		}
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

func buildLLMMessages(
	ctx context.Context,
	history []store.MessageRow,
	historyAttachments map[int][]store.AttachmentInfo,
	content string,
	currentAttachments []store.AttachmentInfo,
) ([]*arkmodel.ChatCompletionMessage, error) {
	messages := make([]*arkmodel.ChatCompletionMessage, 0, len(history)+1)

	for _, msg := range history {
		role := senderTypeToRole(msg.SenderType)
		attachments := historyAttachments[msg.MessageID]
		if len(attachments) == 0 {
			text := msg.Content
			messages = append(messages, &arkmodel.ChatCompletionMessage{
				Role: role,
				Content: &arkmodel.ChatCompletionMessageContent{
					StringValue: &text,
				},
			})
			continue
		}

		parts, err := buildContentParts(ctx, msg.Content, attachments)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &arkmodel.ChatCompletionMessage{
			Role: role,
			Content: &arkmodel.ChatCompletionMessageContent{
				ListValue: parts,
			},
		})
	}

	if len(currentAttachments) == 0 {
		messages = append(messages, &arkmodel.ChatCompletionMessage{
			Role: arkmodel.ChatMessageRoleUser,
			Content: &arkmodel.ChatCompletionMessageContent{
				StringValue: &content,
			},
		})
		return messages, nil
	}

	parts, err := buildContentParts(ctx, content, currentAttachments)
	if err != nil {
		return nil, err
	}
	messages = append(messages, &arkmodel.ChatCompletionMessage{
		Role: arkmodel.ChatMessageRoleUser,
		Content: &arkmodel.ChatCompletionMessageContent{
			ListValue: parts,
		},
	})
	return messages, nil
}

func buildContentParts(ctx context.Context, text string, attachments []store.AttachmentInfo) ([]*arkmodel.ChatCompletionMessageContentPart, error) {
	parts := make([]*arkmodel.ChatCompletionMessageContentPart, 0, len(attachments)+1)
	for _, attachment := range attachments {
		url, err := ResolveAttachmentURL(ctx, attachment)
		if err != nil {
			return nil, err
		}

		if strings.EqualFold(attachment.AttachmentType, "IMAGE") || strings.HasPrefix(strings.ToLower(attachment.MimeType), "image/") {
			parts = append(parts, &arkmodel.ChatCompletionMessageContentPart{
				Type: arkmodel.ChatCompletionMessageContentPartTypeImageURL,
				ImageURL: &arkmodel.ChatMessageImageURL{
					URL: url,
				},
			})
			continue
		}
		if strings.HasPrefix(strings.ToLower(attachment.MimeType), "video/") {
			parts = append(parts, &arkmodel.ChatCompletionMessageContentPart{
				Type: arkmodel.ChatCompletionMessageContentPartTypeVideoURL,
				VideoURL: &arkmodel.ChatMessageVideoURL{
					URL: url,
				},
			})
			continue
		}

		parts = append(parts, &arkmodel.ChatCompletionMessageContentPart{
			Type: arkmodel.ChatCompletionMessageContentPartTypeText,
			Text: url,
		})
	}

	parts = append(parts, &arkmodel.ChatCompletionMessageContentPart{
		Type: arkmodel.ChatCompletionMessageContentPartTypeText,
		Text: text,
	})
	return parts, nil
}

func senderTypeToRole(sender int) string {
	switch sender {
	case store.SenderAssistant:
		return arkmodel.ChatMessageRoleAssistant
	case store.SenderSystem:
		return arkmodel.ChatMessageRoleSystem
	default:
		return arkmodel.ChatMessageRoleUser
	}
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
