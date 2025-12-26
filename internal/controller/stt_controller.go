package controller

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
)

// HandleSTTUpload 语音转文本（占位）。
func HandleSTTUpload(c *gin.Context) {
	file, _, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"err_msg":  "missing audio file",
			"err_code": 400,
		})
		return
	}
	defer file.Close()

	result, err := service.SpeechToText(c.Request.Context(), file)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"err_msg":  "read file error",
			"err_code": 500,
			"result":   nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"result": gin.H{
			"audio_text":   result.AudioText,
			"audio_tokens": result.AudioTokens,
		},
	})
}
