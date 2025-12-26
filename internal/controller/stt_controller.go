package controller

import (
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
)

// HandleSTTUpload 语音转文本（占位）。
func HandleSTTUpload(c *gin.Context) {
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"err_msg":  "missing audio file",
			"err_code": 400,
		})
		return
	}
	defer file.Close()

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}

	filename := ""
	mimeType := ""
	if header != nil {
		filename = header.Filename
		if header.Header != nil {
			mimeType = header.Header.Get("Content-Type")
		}
	}

	result, err := service.SpeechToText(c.Request.Context(), userID, filename, mimeType, file)
	if err != nil {
		if err == service.ErrQuotaExceeded {
			c.JSON(http.StatusForbidden, BaseResponse{ErrMsg: "quota exhausted", ErrCode: 403})
			return
		}
		if err == service.ErrDashscopeNotReady {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "dashscope not configured", ErrCode: 500})
			return
		}
		if err == service.ErrOSSNotReady {
			c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "oss not configured", ErrCode: 500})
			return
		}
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
