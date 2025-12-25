package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type STTResult struct {
	AudioText   string `json:"audio_text"`
	AudioTokens int    `json:"audio_tokens"`
}

func HandleSTTUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("audio")

	if err != nil {
		fmt.Printf("STT Warning: cannot get 'audio' form file, mock mode. err=%v\n", err)

		mockResult := STTResult{
			AudioText:   "mocked speech-to-text result (no file detected)",
			AudioTokens: 0,
		}

		c.JSON(http.StatusOK, gin.H{
			"err_msg":  "success",
			"err_code": 0,
			"result":   mockResult,
		})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"err_msg":  "read file error",
			"err_code": 500,
			"result":   nil,
		})
		return
	}

	fmt.Printf("STT request received: filename=%s, size=%d bytes\n", header.Filename, len(data))

	result := STTResult{
		AudioText:   "hello, this is an example text recognized from audio.",
		AudioTokens: len(data) / 10,
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"result":   result,
	})
}
