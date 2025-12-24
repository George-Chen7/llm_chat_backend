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
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "failed to read audio", ErrCode: 400})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "failed to read audio data", ErrCode: 400})
		return
	}

	fmt.Printf("received audio file: %s, size: %d bytes\n", header.Filename, len(data))
	result := STTResult{
		AudioText:   "hello, nice to talk with you.",
		AudioTokens: 128,
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"result":   result,
	})
}
