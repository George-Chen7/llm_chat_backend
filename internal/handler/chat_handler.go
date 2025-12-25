package handler

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/llm"

	"github.com/gin-gonic/gin"
)

type SendMessageRequest struct {
	Message struct {
		ContentType string `json:"content_type" binding:"required"`
		Content     string `json:"content" binding:"required"`
	} `json:"message" binding:"required"`
	AttachmentIDs []int `json:"attachment_ids"`
}

func HandleChatStream(c *gin.Context) {
	conversationID := c.Param("conversation_id")
	if strings.TrimSpace(conversationID) == "" {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing conversation_id", ErrCode: 400})
		return
	}
	convID, err := strconv.Atoi(conversationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid conversation_id", ErrCode: 400})
		return
	}
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: err.Error(), ErrCode: 400})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	if _, err := getConversation(c.Request.Context(), convID, userID); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, BaseResponse{ErrMsg: "conversation not found", ErrCode: 404})
			return
		}
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	llmClient := llm.Get()
	if llmClient == nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "llm client not initialized", ErrCode: 500})
		return
	}

	reply, err := llmClient.ChatCompletion(c.Request.Context(), req.Message.Content)
	if err != nil {
		c.JSON(http.StatusBadGateway, BaseResponse{ErrMsg: err.Error(), ErrCode: 502})
		return
	}

	dbx, err := getDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db not initialized", ErrCode: 500})
		return
	}

	userMsgID, err := insertMessage(c.Request.Context(), dbx, convID, senderUser, req.Message.ContentType, req.Message.Content, len(req.Message.Content))
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to save message", ErrCode: 500})
		return
	}

	attachments := make([]gin.H, 0)
	if len(req.AttachmentIDs) > 0 {
		if err := attachFilesToMessage(c.Request.Context(), dbx, userID, userMsgID, req.AttachmentIDs); err != nil {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to attach files", ErrCode: 500})
			return
		}
		attachmentsMap, err := loadAttachmentsMap(c.Request.Context(), dbx, []int{userMsgID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to load attachments", ErrCode: 500})
			return
		}
		attachments = attachmentsMap[userMsgID]
	}

	modelMsgID, err := insertMessage(c.Request.Context(), dbx, convID, senderAssistant, "TEXT", reply, len(reply))
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to save reply", ErrCode: 500})
		return
	}

	userMsg := gin.H{
		"message_id":   userMsgID,
		"sender_type":  "USER",
		"content_type": req.Message.ContentType,
		"content":      req.Message.Content,
		"token_total":  len(req.Message.Content),
		"attachments":  attachments,
	}
	modelMsg := gin.H{
		"message_id":   modelMsgID,
		"sender_type":  "ASSISTANT",
		"content_type": "TEXT",
		"content":      reply,
		"token_total":  len(reply),
		"attachments":  []any{},
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":       "success",
		"err_code":      0,
		"user_message":  userMsg,
		"model_message": modelMsg,
	})
}

func insertMessage(ctx context.Context, dbx *sql.DB, conversationID int, senderType int, contentType, content string, tokenTotal int) (int, error) {
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

func attachFilesToMessage(ctx context.Context, dbx *sql.DB, userID, messageID int, attachmentIDs []int) error {
	inClause, args := buildInClause(attachmentIDs)
	if inClause == "" {
		return nil
	}
	args = append([]any{messageID, userID}, args...)
	_, err := dbx.ExecContext(ctx, `
		UPDATE message_attachments ma
		JOIN messages m ON ma.message_id = m.message_id
		JOIN conversations c ON m.conversation_id = c.conversation_id
		SET ma.message_id = ?
		WHERE c.user_id = ? AND ma.attachment_id IN `+inClause, args...)
	return err
}

func loadAttachmentsMap(ctx context.Context, dbx *sql.DB, messageIDs []int) (map[int][]gin.H, error) {
	inClause, args := buildInClause(messageIDs)
	if inClause == "" {
		return map[int][]gin.H{}, nil
	}
	rows, err := dbx.QueryContext(ctx, `
		SELECT attachment_id, message_id, attachment_type, mime_type, url_or_path, duration_ms
		FROM message_attachments
		WHERE message_id IN `+inClause, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int][]gin.H)
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
		item := gin.H{
			"attachment_id":   attachmentID,
			"attachment_type": atype,
			"mime_type":       mimeType,
			"url_or_path":     urlOrPath,
		}
		if duration.Valid {
			item["duration_ms"] = duration.Float64
		}
		out[messageID] = append(out[messageID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
