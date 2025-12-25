package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"backend/internal/db"

	"github.com/gin-gonic/gin"
)

const (
	senderUser      = 1
	senderAssistant = 2
	senderSystem    = 3
)

func getDB() (*sql.DB, error) {
	dbx := db.Get()
	if dbx == nil {
		return nil, errors.New("db not initialized")
	}
	return dbx, nil
}

func getUsername(c *gin.Context) (string, error) {
	username, ok := c.Get("username")
	if !ok {
		return "", errors.New("missing username in context")
	}
	u, ok := username.(string)
	if !ok || u == "" {
		return "", errors.New("invalid username in context")
	}
	return u, nil
}

func getUserByUsername(ctx context.Context, username string) (User, error) {
	dbx, err := getDB()
	if err != nil {
		return User{}, err
	}

	var u User
	row := dbx.QueryRowContext(ctx, `
		SELECT user_id, username, nickname, role, total_quota, remaining_quota
		FROM users
		WHERE username = ? AND status = 1
	`, username)
	if err := row.Scan(&u.UserID, &u.Username, &u.Nickname, &u.Role, &u.TotalQuota, &u.RemainingQuota); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, err
		}
		return User{}, err
	}
	return u, nil
}

func getConversation(ctx context.Context, conversationID int, userID int) (ConversationInfo, error) {
	dbx, err := getDB()
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

func getOrCreateUploadConversation(ctx context.Context, userID int, llmModel string) (int, error) {
	dbx, err := getDB()
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

func createUploadMessage(ctx context.Context, conversationID int) (int, error) {
	dbx, err := getDB()
	if err != nil {
		return 0, err
	}
	res, err := dbx.ExecContext(ctx, `
		INSERT INTO messages (conversation_id, sender_type, content_type, content, token_total)
		VALUES (?, ?, 'FILE', 'UPLOAD', 0)
	`, conversationID, senderSystem)
	if err != nil {
		return 0, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(newID), nil
}

func getUserIDFromContext(c *gin.Context) (int, error) {
	username, err := getUsername(c)
	if err != nil {
		return 0, err
	}
	u, err := getUserByUsername(c.Request.Context(), username)
	if err != nil {
		return 0, err
	}
	return u.UserID, nil
}

func senderTypeToDB(s string) int {
	switch s {
	case "USER":
		return senderUser
	case "ASSISTANT":
		return senderAssistant
	case "SYSTEM":
		return senderSystem
	default:
		return senderUser
	}
}

func senderTypeToAPI(i int) string {
	switch i {
	case senderAssistant:
		return "ASSISTANT"
	case senderSystem:
		return "SYSTEM"
	default:
		return "USER"
	}
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

func buildInClause(ids []int) (string, []any) {
	if len(ids) == 0 {
		return "", nil
	}
	placeholders := make([]byte, 0, len(ids)*2)
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders = append(placeholders, ',')
		}
		placeholders = append(placeholders, '?')
		args = append(args, id)
	}
	return fmt.Sprintf("(%s)", string(placeholders)), args
}
