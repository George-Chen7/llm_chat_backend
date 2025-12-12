package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ChatRequest 定义聊天请求体。
type ChatRequest struct {
	Message string   `json:"message" binding:"required"`
	History []string `json:"history"`
}

// HandleChatStream 处理聊天流式 SSE。
func HandleChatStream(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// [TODO B 模块集成]: 1. 调用服务解析并验证用户 JWT Token。如果验证失败，返回 401 Unauthorized。
	// [TODO B 模块集成]: 2. 调用服务检查用户当前剩余配额是否足够开始对话。如果配额不足，返回 403 Forbidden。

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	// 模拟 LLM 流式输出的 channel
	stream := make(chan string)

	go func() {
		defer close(stream)
		chunks := []string{"Hello", " world", " from", " LLM!"}
		for _, chunk := range chunks {
			time.Sleep(100 * time.Millisecond)
			stream <- chunk
		}
		stream <- "[DONE]"
	}()

	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-stream:
			if !ok {
				return
			}

			if chunk == "[DONE]" {
				_, _ = c.Writer.WriteString("data: [DONE]\n\n")
				flusher.Flush()
				// [TODO B 模块集成]: 3. 在 LLM 成功返回数据后（即在退出 SSE 循环之前），调用 Yang 的服务，根据实际生成的 tokens 数量，进行最终的配额扣减和历史记录保存。
				return
			}

			payload, err := json.Marshal(gin.H{"text": chunk})
			if err != nil {
				// 序列化失败终止流
				return
			}

			_, _ = c.Writer.WriteString(fmt.Sprintf("data: %s\n\n", payload))
			flusher.Flush()
		}
	}
}
