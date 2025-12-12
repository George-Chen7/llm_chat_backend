package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// STTResponse 定义语音转文本的响应。
type STTResponse struct {
	Text string `json:"text"`
}

// HandleSTTUpload 处理上传音频并转发到云 STT 服务（占位实现）。
func HandleSTTUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("audio_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read audio_file: " + err.Error()})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read audio data: " + err.Error()})
		return
	}

	fmt.Printf("收到音频文件，文件名: %s, 大小: %d bytes\n", header.Filename, len(data))

	// [TODO STT 云 API 对接]: 构造 HTTP 请求并将音频数据发送给云 STT 服务的 API。

	// 模拟云端返回的识别结果
	resultText := "用户说：很高兴能与你对话。"

	c.JSON(http.StatusOK, STTResponse{Text: resultText})
}

