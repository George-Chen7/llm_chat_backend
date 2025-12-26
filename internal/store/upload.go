package store

import (
	"context"
)

// GetOrCreateUploadConversation 获取或创建上传用会话。
func GetOrCreateUploadConversation(ctx context.Context, userID int, llmModel string) (int, error) {
	dbx, err := GetDB()
	if err != nil {
		return 0, err
	}
	var convID int
	row := dbx.QueryRowContext(ctx, `
		SELECT conversation_id
		FROM conversations
		WHERE user_id = ? AND title = 'Uploads' AND status = 'ACTIVE'
		ORDER BY conversation_id DESC
		LIMIT 1
	`, userID)
	if err := row.Scan(&convID); err == nil {
		return convID, nil
	}
	res, err := dbx.ExecContext(ctx, `
		INSERT INTO conversations (user_id, title, status, llm_model, system_prompt)
		VALUES (?, 'Uploads', 'ACTIVE', ?, NULL)
	`, userID, llmModel)
	if err != nil {
		return 0, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(newID), nil
}

// CreateUploadMessage 创建上传占位消息。
func CreateUploadMessage(ctx context.Context, conversationID int) (int, error) {
	dbx, err := GetDB()
	if err != nil {
		return 0, err
	}
	res, err := dbx.ExecContext(ctx, `
		INSERT INTO messages (conversation_id, sender_type, content_type, content, token_total)
		VALUES (?, ?, 'FILE', 'UPLOAD', 0)
	`, conversationID, SenderSystem)
	if err != nil {
		return 0, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(newID), nil
}

// CreateAttachment 记录附件并返回 ID。
func CreateAttachment(ctx context.Context, messageID int, attachmentType, mimeType, storageType, urlOrPath string, duration *float64) (int, error) {
	dbx, err := GetDB()
	if err != nil {
		return 0, err
	}
	var durationVal interface{}
	if duration != nil {
		durationVal = *duration
	} else {
		durationVal = nil
	}
	res, err := dbx.ExecContext(ctx, `
		INSERT INTO message_attachments (message_id, attachment_type, mime_type, storage_type, url_or_path, duration_ms)
		VALUES (?, ?, ?, ?, ?, ?)
	`, messageID, attachmentType, mimeType, storageType, urlOrPath, durationVal)
	if err != nil {
		return 0, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(newID), nil
}
