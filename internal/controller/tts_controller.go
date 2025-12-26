package controller

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
)

// HandleTTSConvert text-to-speech.
func HandleTTSConvert(c *gin.Context) {
	messageID := c.Param("message_id")
	if strings.TrimSpace(messageID) == "" {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing message_id", ErrCode: 400})
		return
	}
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}

	msgID, err := strconv.Atoi(messageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid message_id", ErrCode: 400})
		return
	}

	audioURL, err := service.RequestTTSURL(c.Request.Context(), userID, msgID)
	if err != nil {
		if err == service.ErrQuotaExceeded {
			c.JSON(http.StatusForbidden, BaseResponse{ErrMsg: "quota exhausted", ErrCode: 403})
			return
		}
		if err == service.ErrDashscopeNotReady {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "dashscope tts not configured", ErrCode: 500})
			return
		}
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to prepare audio url", ErrCode: 500})
		return
	}

	rangeHeader := c.GetHeader("Range")
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, audioURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to proxy audio", ErrCode: 500})
		return
	}
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "failed to proxy audio", ErrCode: 500})
		return
	}
	defer resp.Body.Close()

	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Type", "audio/wav")
	if resp.Header.Get("Content-Range") != "" {
		c.Header("Content-Range", resp.Header.Get("Content-Range"))
	}
	if resp.Header.Get("Content-Length") != "" {
		c.Header("Content-Length", resp.Header.Get("Content-Length"))
	}

	status := resp.StatusCode
	if status == 0 {
		status = http.StatusOK
	}
	c.Status(status)
	_, _ = io.Copy(c.Writer, resp.Body)
	return
}
