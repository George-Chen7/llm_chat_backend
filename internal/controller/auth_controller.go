package controller

import (
	"database/sql"
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
)

// HandleLogin 账号密码登录。
func HandleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}

	token, err := service.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "invalid credentials", ErrCode: 401})
			return
		}
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"err_msg":   "success",
		"err_code":  0,
		"jwt_token": token,
	})
}

// HandleSetPassword 修改密码。
func HandleSetPassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, BaseResponse{ErrMsg: "invalid params", ErrCode: 400})
		return
	}
	username, err := getUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	if err := service.ResetPassword(c.Request.Context(), username, req.OldPassword, req.NewPassword); err != nil {
		if err == service.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "old password incorrect", ErrCode: 401})
			return
		}
		if err == sql.ErrNoRows || err == service.ErrUserNotFound {
			c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "user not found", ErrCode: 401})
			return
		}
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "db error", ErrCode: 500})
		return
	}
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

// HandleRefreshToken 刷新 JWT。
func HandleRefreshToken(c *gin.Context) {
	username, err := getUsername(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, BaseResponse{ErrMsg: "unauthorized", ErrCode: 401})
		return
	}
	token, err := service.RefreshToken(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, BaseResponse{ErrMsg: "token error", ErrCode: 500})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"err_msg":   "success",
		"err_code":  0,
		"jwt_token": token,
	})
}
