package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/config"
	"backend/internal/handler"
	"backend/internal/middlewares"
)

func NewRouter(cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	_ = cfg

	SetupRouter(r)

	return r
}

func SetupRouter(r *gin.Engine) {
	// health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// auth module
	auth := r.Group("/auth")
	{
		auth.POST("/login", handler.HandleLogin)
		auth.POST("/reset-password", handler.HandleSetPassword)
		auth.POST("/refresh-token", handler.HandleRefreshToken)
	}

	// chat module
	chat := r.Group("/chat")
	chat.Use(middlewares.AuthMiddleware())
	{
		chat.POST("/send-message/:conversation_id", handler.HandleChatStream)
		chat.GET("/history", handler.HandleGetChatHistory)
		chat.POST("/new-conversation", handler.HandleNewChat)
		chat.PUT("/rename", handler.HandleRenameChat)
		chat.DELETE("/delete", handler.HandleDeleteChat)
		chat.GET("/quota", handler.HandleGetQuota)
	}

	// voice module
	voice := r.Group("")
	voice.Use(middlewares.AuthMiddleware())
	{
		voice.POST("/stt/request-stt", handler.HandleSTTUpload)
		voice.POST("/tts/request-tts/:message_id", handler.HandleTTSConvert)
	}

	// admin module
	admin := r.Group("/admin")
	admin.Use(middlewares.AuthMiddleware())
	{
		admin.POST("/user/add", handler.HandleAddUser)
		admin.DELETE("/user/delete", handler.HandleDeleteUser)
		admin.POST("/user/set-quota", handler.HandleSetQuota)
		admin.GET("/user/list", handler.HandleGetUserList)
	}
}
