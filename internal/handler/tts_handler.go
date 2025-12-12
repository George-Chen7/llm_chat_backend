package handler

import (
	"crypto/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TTSRequest 定义文本转语音请求体。
type TTSRequest struct {
	Text string `json:"text" binding:"required"`
}

// HandleTTSConvert 处理文本转语音请求（占位实现）。
func HandleTTSConvert(c *gin.Context) {
	var req TTSRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: text is required"})
		return
	}

	// [TODO TTS 云 API 对接]: 构造 HTTP 请求并将文本发送给云 TTS 服务的 API。

	// 模拟云端返回的音频数据（长度 512 的随机字节）
	audio := make([]byte, 512)
	if _, err := rand.Read(audio); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate audio data"})
		return
	}

	// 设置音频流响应头
	c.Header("Content-Type", "audio/mp3")
	c.Header("Content-Disposition", `attachment; filename="tts_output.mp3"`)
	c.Header("Content-Length", string(len(audio)))

	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(audio)
}

