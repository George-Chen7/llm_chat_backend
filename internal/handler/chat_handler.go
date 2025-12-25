package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: err.Error(), ErrCode: 400})
		return
	}

	llmURL := c.GetString("llm_base_url")
	if llmURL == "" {
		llmURL = "https://example-llm.api/chat/stream"
	}
	apiKey := c.GetString("llm_api_key")

	useMock := llmURL == "" || strings.Contains(llmURL, "example-llm.api")

	if useMock {
		userMsg := gin.H{
			"message_id":   1001,
			"sender_type":  "USER",
			"content_type": req.Message.ContentType,
			"content":      req.Message.Content,
			"token_total":  len(req.Message.Content),
			"attachments":  []any{},
		}
		modelMsg := gin.H{
			"message_id":   1002,
			"sender_type":  "ASSISTANT",
			"content_type": "TEXT",
			"content":      "This is a mock response from the assistant.",
			"token_total":  42,
			"attachments":  []any{},
		}

		c.JSON(http.StatusOK, gin.H{
			"err_msg":       "success",
			"err_code":      0,
			"user_message":  userMsg,
			"model_message": modelMsg,
		})
		return
	}

	messages := []map[string]string{
		{"role": "user", "content": req.Message.Content},
	}

	llmPayload := map[string]any{
		"messages":        messages,
		"stream":          false,
		"conversation_id": conversationID,
	}
	payloadBytes, err := json.Marshal(llmPayload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to marshal llm payload", ErrCode: 500})
		return
	}

	ctx := c.Request.Context()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, llmURL, bytes.NewReader(payloadBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "failed to build llm request", ErrCode: 400})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		userMsg := gin.H{
			"message_id":   1001,
			"sender_type":  "USER",
			"content_type": req.Message.ContentType,
			"content":      req.Message.Content,
			"token_total":  len(req.Message.Content),
			"attachments":  []any{},
		}
		modelMsg := gin.H{
			"message_id":   1002,
			"sender_type":  "ASSISTANT",
			"content_type": "TEXT",
			"content":      "Assistant mock response due to connection error.",
			"token_total":  42,
			"attachments":  []any{},
		}
		c.JSON(http.StatusOK, gin.H{
			"err_msg":       "success",
			"err_code":      0,
			"user_message":  userMsg,
			"model_message": modelMsg,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		userMsg := gin.H{
			"message_id":   1001,
			"sender_type":  "USER",
			"content_type": req.Message.ContentType,
			"content":      req.Message.Content,
			"token_total":  len(req.Message.Content),
			"attachments":  []any{},
		}
		modelMsg := gin.H{
			"message_id":   1002,
			"sender_type":  "ASSISTANT",
			"content_type": "TEXT",
			"content":      fmt.Sprintf("Assistant mock response due to status %d.", resp.StatusCode),
			"token_total":  42,
			"attachments":  []any{},
		}
		c.JSON(http.StatusOK, gin.H{
			"err_msg":       "success",
			"err_code":      0,
			"user_message":  userMsg,
			"model_message": modelMsg,
		})
		return
	}

	var llmResp struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to decode llm response", ErrCode: 500})
		return
	}

	userMsg := gin.H{
		"message_id":   2001,
		"sender_type":  "USER",
		"content_type": req.Message.ContentType,
		"content":      req.Message.Content,
		"token_total":  len(req.Message.Content),
		"attachments":  []any{},
	}
	modelMsg := gin.H{
		"message_id":   2002,
		"sender_type":  "ASSISTANT",
		"content_type": "TEXT",
		"content":      llmResp.Content,
		"token_total":  len(llmResp.Content),
		"attachments":  []any{},
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":       "success",
		"err_code":      0,
		"user_message":  userMsg,
		"model_message": modelMsg,
	})
}
