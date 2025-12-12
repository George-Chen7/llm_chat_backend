package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 以下为占位 Handler，后续可替换为真实实现。

func HandleLogin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "login placeholder"})
}

func HandleSetPassword(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "setPassword placeholder"})
}

func HandleRefreshToken(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "refreshToken placeholder"})
}

func HandleGetChatHistory(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "getChatHistory placeholder"})
}

func HandleNewChat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "newChat placeholder"})
}

func HandleRenameChat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "renameChat placeholder"})
}

func HandleDeleteChat(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "deleteChat placeholder"})
}

func HandleGetQuota(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "getQuota placeholder"})
}

func HandleAddUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "addUser placeholder"})
}

func HandleDeleteUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "deleteUser placeholder"})
}

func HandleSetQuota(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "setQuota placeholder"})
}

func HandleGetUserList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "getUser placeholder"})
}
