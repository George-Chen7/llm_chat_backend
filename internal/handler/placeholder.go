package handler

import (
	"net/http"
	"time"

	"backend/internal/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// --- 认证相关 ---

// HandleLogin 账号密码登录 - 必须返回 jwt_token
func HandleLogin(c *gin.Context) {
    // 模拟登录校验逻辑（假设校验通过）
    username := "admin"

    // 生成 JWT Token
    claims := middlewares.MyClaims{
        Username: username,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)), // 24小时过期
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, _ := token.SignedString(middlewares.JWTSecret)

    c.JSON(http.StatusOK, gin.H{
        "err_msg":     "success",
        "err_code":    0,
        "jwt_token":   tokenString, // 返回真实的 JWT
        "expire_time": claims.ExpiresAt.Unix(),
        "user_info": gin.H{
            "id":       1,
            "username": username,
            "avatar":   "",
        },
    })
}

func HandleSetPassword(c *gin.Context) {
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

func HandleRefreshToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"err_msg":   "success",
		"err_code":  0,
		"jwt_token": "refreshed_mock_token_7890",
	})
}

// --- 会话管理 ---

// HandleGetChatHistory 获取历史记录 - 必须包含分页字段和 list
func HandleGetChatHistory(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"err_msg":      "success",
		"err_code":     0,
		"total_page":   1,
		"total_count":  1,
		"current_page": 1,
		"page_size":    10,
		"list": []gin.H{
			{
				"conversation_id": "conv_mock_001",
				"title":           "模拟历史对话",
				"update_time":     time.Now().Unix(),
			},
		},
	})
}

func HandleNewChat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"err_msg":         "success",
		"err_code":        0,
		"conversation_id": "new_conv_" + time.Now().Format("20060102150405"),
	})
}

func HandleRenameChat(c *gin.Context) {
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

func HandleDeleteChat(c *gin.Context) {
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

// --- 额度与用户管理 (修复编译错误的关键) ---

func HandleGetQuota(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"err_msg":      "success",
		"err_code":     0,
		"total_quota":  1000,
		"used_quota":   150,
		"remain_quota": 850,
	})
}

// HandleAddUser 添加用户占位
func HandleAddUser(c *gin.Context) {
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

// HandleDeleteUser 删除用户占位
func HandleDeleteUser(c *gin.Context) {
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

// HandleSetQuota 设置额度占位
func HandleSetQuota(c *gin.Context) {
	c.JSON(http.StatusOK, BaseResponse{ErrMsg: "success", ErrCode: 0})
}

// HandleGetUserList 获取用户列表占位
func HandleGetUserList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"err_msg":      "success",
		"err_code":     0,
		"total_page":   1,
		"total_count":  1,
		"current_page": 1,
		"page_size":    10,
		"list": []gin.H{
			{
				"id":       1,
				"username": "test_user",
				"quota":    500,
			},
		},
	})
}
