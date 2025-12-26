package controller

import (
	"net/http"
	"time"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
)

// HandleUploadFile 上传附件并落库。
func HandleUploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "missing file", ErrCode: 400})
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

	result, err := service.UploadAndRecord(c.Request.Context(), userID, filename, mimeType, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "upload failed", ErrCode: 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"attachment": gin.H{
			"attachment_id":   result.AttachmentID,
			"attachment_type": "FILE",
			"mime_type":       result.MimeType,
			"url_or_path":     result.URLOrPath,
			"created_at":      time.Now().Format(time.RFC3339),
		},
	})
}
