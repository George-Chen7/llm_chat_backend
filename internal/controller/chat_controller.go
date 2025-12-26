package controller

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/llm"
	"backend/internal/service"

	"github.com/gin-gonic/gin"
)

// SendMessageRequest 发送消息请求体。
type SendMessageRequest struct {
	Message struct {
		ContentType string `json:"content_type" binding:"required"`
		Content     string `json:"content" binding:"required"`
	} `json:"message" binding:"required"`
	AttachmentIDs []int `json:"attachment_ids"`
}

// HandleChatSend 发送消息。
func HandleChatSend(c *gin.Context) {
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

	userMsgID, modelMsgID, attachments, reply, err := service.SendMessage(
		c.Request.Context(),
		userID,
		convID,
		req.Message.ContentType,
		req.Message.Content,
		req.AttachmentIDs,
	)
	if err != nil {
		switch err {
		case service.ErrConversationNotFound:
			c.JSON(http.StatusNotFound, BaseResponse{ErrMsg: "conversation not found", ErrCode: 404})
		case service.ErrLLMNotReady:
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "llm client not initialized", ErrCode: 500})
		default:
			c.JSON(http.StatusBadGateway, BaseResponse{ErrMsg: err.Error(), ErrCode: 502})
		}
		return
	}

	attachList := make([]gin.H, 0, len(attachments))
	for _, a := range attachments {
		attachList = append(attachList, gin.H{
			"attachment_id":   a.AttachmentID,
			"attachment_type": a.AttachmentType,
			"mime_type":       a.MimeType,
			"url_or_path":     a.URLOrPath,
			"duration_ms":     a.DurationMS,
		})
	}

	userMsg := gin.H{
		"message_id":   userMsgID,
		"sender_type":  "USER",
		"content_type": req.Message.ContentType,
		"content":      req.Message.Content,
		"token_total":  len(req.Message.Content),
		"attachments":  attachList,
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

// HandleNewChat 新建对话。
func HandleNewChat(c *gin.Context) {
	var req struct {
		Title        string `json:"title" binding:"required"`
		SystemPrompt string `json:"system_prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}

	llmModel := "unknown"
	if client := llm.Get(); client != nil && client.Model() != "" {
		llmModel = client.Model()
	}

	var systemPrompt sql.NullInt64
	if req.SystemPrompt != "" {
		if v, err := strconv.Atoi(req.SystemPrompt); err == nil {
			systemPrompt = sql.NullInt64{Int64: int64(v), Valid: true}
		}
	}

	convInfo, err := service.NewConversation(c.Request.Context(), userID, req.Title, systemPrompt, llmModel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"conversation": gin.H{
			"conversation_id": convInfo.ConversationID,
			"title":           convInfo.Title,
			"status":          convInfo.Status,
			"llm_model":       convInfo.LLMModel,
		},
	})
}

// HandleRenameChat 重命名对话。
func HandleRenameChat(c *gin.Context) {
	conversationID := c.Param("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing conversation_id", ErrCode: 400})
		return
	}
	convID, err := strconv.Atoi(conversationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid conversation_id", ErrCode: 400})
		return
	}
	var req struct {
		Title string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	updated, err := service.RenameConversation(c.Request.Context(), userID, convID, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	if !updated {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "not found", ErrCode: 404})
		return
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

// HandleDeleteChat 删除对话。
func HandleDeleteChat(c *gin.Context) {
	conversationID := c.Param("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing conversation_id", ErrCode: 400})
		return
	}
	convID, err := strconv.Atoi(conversationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid conversation_id", ErrCode: 400})
		return
	}
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	deleted, err := service.DeleteConversation(c.Request.Context(), userID, convID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	if !deleted {
		c.JSON(http.StatusOK, BaseResponse{ErrMsg: "not found", ErrCode: 404})
		return
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

// HandleGetChatHistory 获取对话历史。
func HandleGetChatHistory(c *gin.Context) {
	conversationID := c.Param("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing conversation_id", ErrCode: 400})
		return
	}
	convID, err := strconv.Atoi(conversationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid conversation_id", ErrCode: 400})
		return
	}
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("current_page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	items, attachmentsMap, totalCount, err := service.GetHistory(c.Request.Context(), userID, convID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	messages := make([]gin.H, 0, len(items))
	for _, m := range items {
		attachments := make([]gin.H, 0)
		for _, a := range attachmentsMap[m.MessageID] {
			attachments = append(attachments, gin.H{
				"attachment_id":   a.AttachmentID,
				"attachment_type": a.AttachmentType,
				"mime_type":       a.MimeType,
				"url_or_path":     a.URLOrPath,
				"duration_ms":     a.DurationMS,
			})
		}
		if attachments == nil {
			attachments = []gin.H{}
		}
		messages = append(messages, gin.H{
			"message_id":   m.MessageID,
			"sender_type":  senderTypeToAPI(m.SenderType),
			"content_type": m.ContentType,
			"content":      m.Content,
			"token_total":  m.TokenTotal,
			"attachments":  attachments,
		})
	}

	totalPage := (totalCount + pageSize - 1) / pageSize
	c.JSON(http.StatusOK, gin.H{
		"err_msg":      "success",
		"err_code":     0,
		"total_page":   totalPage,
		"total_count":  totalCount,
		"current_page": page,
		"page_size":    pageSize,
		"messages":     messages,
	})
}
