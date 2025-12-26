package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/llm"
	"backend/internal/store"

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
	if _, err := store.GetConversation(c.Request.Context(), convID, userID); err != nil {
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

	userMsgID, err := store.InsertMessage(c.Request.Context(), convID, store.SenderUser, req.Message.ContentType, req.Message.Content, len(req.Message.Content))
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to save message", ErrCode: 500})
		return
	}

	attachments := make([]gin.H, 0)
	if len(req.AttachmentIDs) > 0 {
		if err := store.AttachFilesToMessage(c.Request.Context(), userID, userMsgID, req.AttachmentIDs); err != nil {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to attach files", ErrCode: 500})
			return
		}
		attachmentsMap, err := store.LoadAttachmentsMap(c.Request.Context(), []int{userMsgID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to load attachments", ErrCode: 500})
			return
		}
		for _, a := range attachmentsMap[userMsgID] {
			attachments = append(attachments, gin.H{
				"attachment_id":   a.AttachmentID,
				"attachment_type": a.AttachmentType,
				"mime_type":       a.MimeType,
				"url_or_path":     a.URLOrPath,
				"duration_ms":     a.DurationMS,
			})
		}
	}

	modelMsgID, err := store.InsertMessage(c.Request.Context(), convID, store.SenderAssistant, "TEXT", reply, len(reply))
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
