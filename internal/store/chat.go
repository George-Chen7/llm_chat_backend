package store

import (
	"context"
	"database/sql"
)

// GetConversation 获取用户的指定会话。
func GetConversation(ctx context.Context, conversationID int, userID int) (ConversationInfo, error) {
	dbx, err := GetDB()
	if err != nil {
		return ConversationInfo{}, err
	}
	var cinfo ConversationInfo
	row := dbx.QueryRowContext(ctx, `
		SELECT conversation_id, title, status, llm_model
		FROM conversations
		WHERE conversation_id = ? AND user_id = ?
	`, conversationID, userID)
	if err := row.Scan(&cinfo.ConversationID, &cinfo.Title, &cinfo.Status, &cinfo.LLMModel); err != nil {
		return ConversationInfo{}, err
	}
	return cinfo, nil
}

// CreateConversation 创建会话并返回概要信息。
func CreateConversation(ctx context.Context, userID int, title, llmModel string, systemPrompt sql.NullInt64) (ConversationInfo, error) {
	dbx, err := GetDB()
	if err != nil {
		return ConversationInfo{}, err
	}
	res, err := dbx.ExecContext(ctx, `
		INSERT INTO conversations (user_id, title, status, llm_model, system_prompt)
		VALUES (?, ?, 'ACTIVE', ?, ?)
	`, userID, title, llmModel, systemPrompt)
	if err != nil {
		return ConversationInfo{}, err
	}
	convID, err := res.LastInsertId()
	if err != nil {
		return ConversationInfo{}, err
	}
	return ConversationInfo{
		ConversationID: int(convID),
		Title:          title,
		Status:         "ACTIVE",
		LLMModel:       llmModel,
	}, nil
}

// RenameConversation 更新会话标题。
func RenameConversation(ctx context.Context, conversationID, userID int, title string) (bool, error) {
	dbx, err := GetDB()
	if err != nil {
		return false, err
	}
	res, err := dbx.ExecContext(ctx, `
		UPDATE conversations SET title = ?
		WHERE conversation_id = ? AND user_id = ?
	`, title, conversationID, userID)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

// DeleteConversation 逻辑删除会话。
func DeleteConversation(ctx context.Context, conversationID, userID int) (bool, error) {
	dbx, err := GetDB()
	if err != nil {
		return false, err
	}
	res, err := dbx.ExecContext(ctx, `
		UPDATE conversations SET status = 'DELETED'
		WHERE conversation_id = ? AND user_id = ?
	`, conversationID, userID)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

// CountMessages 统计会话消息数量。
func CountMessages(ctx context.Context, userID, conversationID int) (int, error) {
	dbx, err := GetDB()
	if err != nil {
		return 0, err
	}
	var total int
	if err := dbx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM messages m
		JOIN conversations c ON m.conversation_id = c.conversation_id
		WHERE c.user_id = ? AND m.conversation_id = ?
	`, userID, conversationID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// ListMessages 获取会话消息列表与消息ID。
func ListMessages(ctx context.Context, userID, conversationID, page, pageSize int) ([]MessageRow, []int, error) {
	dbx, err := GetDB()
	if err != nil {
		return nil, nil, err
	}
	offset := (page - 1) * pageSize
	rows, err := dbx.QueryContext(ctx, `
		SELECT m.message_id, m.sender_type, m.content_type, m.content, m.token_total
		FROM messages m
		JOIN conversations c ON m.conversation_id = c.conversation_id
		WHERE c.user_id = ? AND m.conversation_id = ?
		ORDER BY m.created_at ASC
		LIMIT ? OFFSET ?
	`, userID, conversationID, pageSize, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	items := make([]MessageRow, 0)
	ids := make([]int, 0)
	for rows.Next() {
		var m MessageRow
		if err := rows.Scan(&m.MessageID, &m.SenderType, &m.ContentType, &m.Content, &m.TokenTotal); err != nil {
			return nil, nil, err
		}
		items = append(items, m)
		ids = append(ids, m.MessageID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return items, ids, nil
}

// InsertMessage 创建消息并返回 ID。
func InsertMessage(ctx context.Context, conversationID int, senderType int, contentType, content string, tokenTotal int) (int, error) {
	dbx, err := GetDB()
	if err != nil {
		return 0, err
	}
	res, err := dbx.ExecContext(ctx, `
		INSERT INTO messages (conversation_id, sender_type, content_type, content, token_total)
		VALUES (?, ?, ?, ?, ?)
	`, conversationID, senderType, contentType, content, tokenTotal)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// AttachFilesToMessage 绑定附件到消息。
func AttachFilesToMessage(ctx context.Context, userID, messageID int, attachmentIDs []int) error {
	dbx, err := GetDB()
	if err != nil {
		return err
	}
	inClause, args := BuildInClause(attachmentIDs)
	if inClause == "" {
		return nil
	}
	args = append([]any{messageID, userID}, args...)
	_, err = dbx.ExecContext(ctx, `
		UPDATE message_attachments ma
		JOIN messages m ON ma.message_id = m.message_id
		JOIN conversations c ON m.conversation_id = c.conversation_id
		SET ma.message_id = ?
		WHERE c.user_id = ? AND ma.attachment_id IN `+inClause, args...)
	return err
}

// LoadAttachmentsMap 按 message_id 返回附件列表。
func LoadAttachmentsMap(ctx context.Context, messageIDs []int) (map[int][]AttachmentInfo, error) {
	dbx, err := GetDB()
	if err != nil {
		return nil, err
	}
	inClause, args := BuildInClause(messageIDs)
	if inClause == "" {
		return map[int][]AttachmentInfo{}, nil
	}
	rows, err := dbx.QueryContext(ctx, `
		SELECT attachment_id, message_id, attachment_type, mime_type, url_or_path, duration_ms
		FROM message_attachments
		WHERE message_id IN `+inClause, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int][]AttachmentInfo)
	for rows.Next() {
		var (
			attachmentID int
			messageID    int
			atype        string
			mimeType     string
			urlOrPath    string
			duration     sql.NullFloat64
		)
		if err := rows.Scan(&attachmentID, &messageID, &atype, &mimeType, &urlOrPath, &duration); err != nil {
			return nil, err
		}
		item := AttachmentInfo{
			AttachmentID:   attachmentID,
			AttachmentType: atype,
			MimeType:       mimeType,
			URLOrPath:      urlOrPath,
		}
		if duration.Valid {
			val := duration.Float64
			item.DurationMS = &val
		}
		out[messageID] = append(out[messageID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
