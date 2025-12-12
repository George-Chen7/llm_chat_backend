package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

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

	// 准备转发到 LLM 的请求
	llmURL := c.GetString("llm_base_url")
	if llmURL == "" {
		llmURL = "https://example-llm.api/chat/stream" // 占位：请替换为真实 LLM 流式接口地址
	}
	apiKey := c.GetString("llm_api_key") // 占位：可由中间件或配置注入

	// 组装 messages：简单将历史与当前消息串联
	messages := make([]map[string]string, 0, len(req.History)+1)
	for _, h := range req.History {
		messages = append(messages, map[string]string{"role": "user", "content": h})
	}
	messages = append(messages, map[string]string{"role": "user", "content": req.Message})

	llmPayload := map[string]any{
		"messages": messages,
		"stream":   true,
	}
	payloadBytes, err := json.Marshal(llmPayload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal llm payload"})
		return
	}

	ctx := c.Request.Context()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, llmURL, bytes.NewReader(payloadBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to build llm request"})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{} // 可根据需要设置超时或连接池
	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to call llm service", "detail": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		c.JSON(http.StatusBadGateway, gin.H{
			"error":  "llm service returned non-200",
			"status": resp.StatusCode,
			"body":   string(body),
		})
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	// 增大 buffer 以防止行过长
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

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
			_, _ = c.Writer.WriteString("data: [DONE]\n\n")
			flusher.Flush()
			// [TODO B 模块集成]: 3. 在 LLM 成功返回数据后（即在退出 SSE 循环之前），调用 Yang 的服务，根据实际生成的 tokens 数量，进行最终的配额扣减和历史记录保存。
			return
		}

		// 将 LLM 返回的 chunk 原样再包装为 SSE
		_, _ = c.Writer.WriteString("data: " + data + "\n\n")
		flusher.Flush()
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		// 仅记录错误，不再写入客户端
		return
	}
}
