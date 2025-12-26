package store

import "context"

// ListConversationsByUser 获取用户会话列表。
func ListConversationsByUser(ctx context.Context, userID int) ([]ConversationInfo, error) {
	dbx, err := GetDB()
	if err != nil {
		return nil, err
	}
	rows, err := dbx.QueryContext(ctx, `
		SELECT conversation_id, title, status, llm_model
		FROM conversations
		WHERE user_id = ?
		ORDER BY conversation_id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	conversations := make([]ConversationInfo, 0)
	for rows.Next() {
		var info ConversationInfo
		if err := rows.Scan(&info.ConversationID, &info.Title, &info.Status, &info.LLMModel); err != nil {
			return nil, err
		}
		conversations = append(conversations, info)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return conversations, nil
}
