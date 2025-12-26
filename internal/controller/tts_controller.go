package controller

import (
	"net/http"
	"strconv"
	"strings"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
)

// HandleTTSConvert 文本转语音（占位）。
func HandleTTSConvert(c *gin.Context) {
	messageID := c.Param("message_id")
	if strings.TrimSpace(messageID) == "" {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing message_id", ErrCode: 400})
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	_ = c.ShouldBindJSON(&req)

	audio, err := service.TextToSpeech(c.Request.Context(), req.Text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to generate audio data", ErrCode: 500})
		return
	}

	c.Header("Content-Type", "audio/wav")
	c.Header("Content-Disposition", `attachment; filename="tts_`+messageID+`.wav"`)
	c.Header("Content-Length", strconv.Itoa(len(audio)))

	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write(audio)
}
