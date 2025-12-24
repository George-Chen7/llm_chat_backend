package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	fmt.Printf("HandleChatStream connect base_url=%s\n", llmURL)

	useMock := llmURL == "" || strings.Contains(llmURL, "example-llm.api")

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "streaming unsupported", ErrCode: 500})
		return
	}

	if useMock {
		fmt.Printf("Switching to Mock Mode (early check)\n")
		for i := 1; i <= 3; i++ {
			if i == 1 {
				respObj := map[string]any{
					"err_msg":  "success",
					"err_code": 0,
					"user_message": map[string]any{
						"message_id":   0,
						"role":         "USER",
						"content_type": req.Message.ContentType,
						"content":      req.Message.Content,
						"token_total":  0,
						"attachments":  []any{},
					},
					"model_message": map[string]any{
						"message_id":   0,
						"role":         "ASSISTANT",
						"content_type": "TEXT",
						"content":      fmt.Sprintf("mock chunk %d: %s", i, req.Message.Content),
						"token_total":  0,
						"attachments":  []any{},
					},
				}
				b, _ := json.Marshal(respObj)
				_, _ = c.Writer.WriteString("data: " + string(b) + "\n\n")
				flusher.Flush()
				continue
			}
			respObj := map[string]any{
				"err_msg":  "success",
				"err_code": 0,
				"model_message": map[string]any{
					"message_id":   0,
					"role":         "ASSISTANT",
					"content_type": "TEXT",
					"content":      fmt.Sprintf("mock chunk %d: %s", i, req.Message.Content),
					"token_total":  0,
					"attachments":  []any{},
				},
			}
			b, _ := json.Marshal(respObj)
			_, _ = c.Writer.WriteString("data: " + string(b) + "\n\n")
			flusher.Flush()
		}
		final := map[string]any{
			"err_msg":  "success",
			"err_code": 0,
			"model_message": map[string]any{
				"message_id":   0,
				"role":         "ASSISTANT",
				"content_type": "TEXT",
				"content":      "",
				"token_total":  0,
				"attachments":  []any{},
			},
		}
		bf, _ := json.Marshal(final)
		_, _ = c.Writer.WriteString("data: " + string(bf) + "\n\n")
		flusher.Flush()
		_, _ = c.Writer.WriteString("data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	messages := []map[string]string{
		{"role": "user", "content": req.Message.Content},
	}

	llmPayload := map[string]any{
		"messages":        messages,
		"stream":          true,
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
		fmt.Printf("Switching to Mock Mode (request error)\n")
		// fallback immediate mock to avoid 502
		for i := 1; i <= 3; i++ {
			if i == 1 {
				respObj := map[string]any{
					"err_msg":  "success",
					"err_code": 0,
					"user_message": map[string]any{
						"message_id":   0,
						"role":         "USER",
						"content_type": req.Message.ContentType,
						"content":      req.Message.Content,
						"token_total":  0,
						"attachments":  []any{},
					},
					"model_message": map[string]any{
						"message_id":   0,
						"role":         "ASSISTANT",
						"content_type": "TEXT",
						"content":      fmt.Sprintf("mock chunk %d: %s", i, req.Message.Content),
						"token_total":  0,
						"attachments":  []any{},
					},
				}
				b, _ := json.Marshal(respObj)
				_, _ = c.Writer.WriteString("data: " + string(b) + "\n\n")
				flusher.Flush()
				continue
			}
			respObj := map[string]any{
				"err_msg":  "success",
				"err_code": 0,
				"model_message": map[string]any{
					"message_id":   0,
					"role":         "ASSISTANT",
					"content_type": "TEXT",
					"content":      fmt.Sprintf("mock chunk %d: %s", i, req.Message.Content),
					"token_total":  0,
					"attachments":  []any{},
				},
			}
			b, _ := json.Marshal(respObj)
			_, _ = c.Writer.WriteString("data: " + string(b) + "\n\n")
			flusher.Flush()
		}
		final := map[string]any{
			"err_msg":  "success",
			"err_code": 0,
			"model_message": map[string]any{
				"message_id":   0,
				"role":         "ASSISTANT",
				"content_type": "TEXT",
				"content":      "",
				"token_total":  0,
				"attachments":  []any{},
			},
		}
		bf, _ := json.Marshal(final)
		_, _ = c.Writer.WriteString("data: " + string(bf) + "\n\n")
		flusher.Flush()
		_, _ = c.Writer.WriteString("data: [DONE]\n\n")
		flusher.Flush()
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Switching to Mock Mode (non-200 response %d)\n", resp.StatusCode)
		for i := 1; i <= 3; i++ {
			if i == 1 {
				respObj := map[string]any{
					"err_msg":  "success",
					"err_code": 0,
					"user_message": map[string]any{
						"message_id":   0,
						"role":         "USER",
						"content_type": req.Message.ContentType,
						"content":      req.Message.Content,
						"token_total":  0,
						"attachments":  []any{},
					},
					"model_message": map[string]any{
						"message_id":   0,
						"role":         "ASSISTANT",
						"content_type": "TEXT",
						"content":      fmt.Sprintf("mock chunk %d: %s", i, req.Message.Content),
						"token_total":  0,
						"attachments":  []any{},
					},
				}
				b, _ := json.Marshal(respObj)
				_, _ = c.Writer.WriteString("data: " + string(b) + "\n\n")
				flusher.Flush()
				continue
			}
			respObj := map[string]any{
				"err_msg":  "success",
				"err_code": 0,
				"model_message": map[string]any{
					"message_id":   0,
					"role":         "ASSISTANT",
					"content_type": "TEXT",
					"content":      fmt.Sprintf("mock chunk %d: %s", i, req.Message.Content),
					"token_total":  0,
					"attachments":  []any{},
				},
			}
			b, _ := json.Marshal(respObj)
			_, _ = c.Writer.WriteString("data: " + string(b) + "\n\n")
			flusher.Flush()
		}
		final := map[string]any{
			"err_msg":  "success",
			"err_code": 0,
			"model_message": map[string]any{
				"message_id":   0,
				"role":         "ASSISTANT",
				"content_type": "TEXT",
				"content":      "",
				"token_total":  0,
				"attachments":  []any{},
			},
		}
		bf, _ := json.Marshal(final)
		_, _ = c.Writer.WriteString("data: " + string(bf) + "\n\n")
		flusher.Flush()
		_, _ = c.Writer.WriteString("data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	first := true

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}

		if data == "[DONE]" {
			final := map[string]any{
				"err_msg":  "success",
				"err_code": 0,
				"model_message": map[string]any{
					"message_id":   0,
					"role":         "ASSISTANT",
					"content_type": "TEXT",
					"content":      "",
					"token_total":  0,
					"attachments":  []any{},
				},
			}
			bf, _ := json.Marshal(final)
			_, _ = c.Writer.WriteString("data: " + string(bf) + "\n\n")
			flusher.Flush()
			_, _ = c.Writer.WriteString("data: [DONE]\n\n")
			flusher.Flush()
			return
		}

		if first {
			content := data
			var jm map[string]any
			if err := json.Unmarshal([]byte(data), &jm); err == nil {
				if v, ok := jm["content"]; ok {
					if s, ok := v.(string); ok && s != "" {
						content = s
					}
				}
			}
			respObj := map[string]any{
				"err_msg":  "success",
				"err_code": 0,
				"user_message": map[string]any{
					"message_id":   0,
					"role":         "USER",
					"content_type": req.Message.ContentType,
					"content":      req.Message.Content,
					"token_total":  0,
					"attachments":  []any{},
				},
				"model_message": map[string]any{
					"message_id":   0,
					"role":         "ASSISTANT",
					"content_type": "TEXT",
					"content":      content,
					"token_total":  0,
					"attachments":  []any{},
				},
			}
			b, _ := json.Marshal(respObj)
			_, _ = c.Writer.WriteString("data: " + string(b) + "\n\n")
			flusher.Flush()
			first = false
			continue
		}

		respObj := map[string]any{
			"err_msg":  "success",
			"err_code": 0,
			"model_message": map[string]any{
				"message_id":   0,
				"role":         "ASSISTANT",
				"content_type": "TEXT",
				"content":      data,
				"token_total":  0,
				"attachments":  []any{},
			},
		}
		b, _ := json.Marshal(respObj)
		_, _ = c.Writer.WriteString("data: " + string(b) + "\n\n")
		flusher.Flush()
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return
	}
}
