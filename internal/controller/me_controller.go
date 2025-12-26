package controller

import (
	"database/sql"
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
)

// HandleGetMeInfo 获取当前用户信息。
func HandleGetMeInfo(c *gin.Context) {
	username, err := getUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	user, err := service.GetMeInfo(c.Request.Context(), username)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "user not found", ErrCode: 401})
			return
		}
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"err_msg":  "success",
		"err_code": 0,
		"user": gin.H{
			"user_id":     user.UserID,
			"username":    user.Username,
			"nickname":    user.Nickname,
			"role":        user.Role,
			"total_quota": user.TotalQuota,
			"used_quota":  user.UsedQuota,
		},
	})
}

// HandleGetMeConversations 获取当前用户会话列表。
func HandleGetMeConversations(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	conversations, err := service.ListMyConversations(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"err_msg":       "success",
		"err_code":      0,
		"conversations": conversations,
	})
}
