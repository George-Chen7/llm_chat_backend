package router

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"backend/internal/config"
)

// NewRouter 构建 Gin 路由。
func NewRouter(cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// 示例 SSE 流式接口
	r.GET("/v1/chat/stream", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")

		messages := []string{
			"欢迎使用 SSE 流式接口",
			"这是一条模拟的模型回复",
			"可以在这里替换为真实的 LLM 推理输出",
		}

		c.Stream(func(w io.Writer) bool {
			if len(messages) == 0 {
				return false
			}
			msg := messages[0]
			messages = messages[1:]
			c.SSEvent("message", msg)
			time.Sleep(500 * time.Millisecond)
			return len(messages) > 0
		})
	})

	return r
}

