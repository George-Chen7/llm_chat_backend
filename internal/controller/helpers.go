package controller

import (
	"errors"

	"backend/internal/store"

	"github.com/gin-gonic/gin"
)

// getUsername 从上下文获取用户名。
func getUsername(c *gin.Context) (string, error) {
	username, ok := c.Get("username")
	if !ok {
		return "", errors.New("missing username in context")
	}
	u, ok := username.(string)
	if !ok || u == "" {
		return "", errors.New("invalid username in context")
	}
	return u, nil
}

// getUserIDFromContext 通过用户名反查用户ID。
func getUserIDFromContext(c *gin.Context) (int, error) {
	username, err := getUsername(c)
	if err != nil {
		return 0, err
	}
	u, err := store.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		return 0, err
	}
	return u.UserID, nil
}

// senderTypeToAPI 转换为 API 字段值。
func senderTypeToAPI(i int) string {
	switch i {
	case store.SenderAssistant:
		return "ASSISTANT"
	case store.SenderSystem:
		return "SYSTEM"
	default:
		return "USER"
	}
}
