package handler

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type TTSRequest struct {
	Text string `json:"text"`
}

func HandleTTSConvert(c *gin.Context) {
	messageID := c.Param("message_id")
	if strings.TrimSpace(messageID) == "" {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing message_id", ErrCode: 400})
		return
	}
	var req TTSRequest
	_ = c.ShouldBindJSON(&req)
	fmt.Printf("TTS request received: message_id=%s, text=%s\n", messageID, req.Text)

	audio := make([]byte, 512)
	if _, err := rand.Read(audio); err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to generate audio data", ErrCode: 500})
		return
	}

	c.Header("Content-Type", "audio/wav")
	c.Header("Content-Disposition", `attachment; filename="tts_output.wav"`)
	c.Header("Content-Length", strconv.Itoa(len(audio)))

	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(audio)
}
