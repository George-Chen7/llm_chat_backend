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
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	auth := r.Group("/auth")
	{
		auth.POST("/login", handler.HandleLogin)
		auth.POST("/reset-password", handler.HandleSetPassword)
		auth.POST("/refresh-token", handler.HandleRefreshToken)
	}

	chat := r.Group("/chat")
	chat.Use(middlewares.AuthMiddleware())
	{
		chat.POST("/send-message/:conversation_id", handler.HandleChatStream)
		chat.GET("/history/:conversation_id", handler.HandleGetChatHistory)
		chat.POST("/new-conversation", handler.HandleNewChat)
		chat.PUT("/rename-conversation/:conversation_id", handler.HandleRenameChat)
		chat.DELETE("/delete-conversation/:conversation_id", handler.HandleDeleteChat)
		chat.POST("/upload-file", handler.HandleUploadFile)
		chat.GET("/prompt-preset", handler.HandleGetPromptPreset)
	}

	r.POST("/stt/request-stt", middlewares.AuthMiddleware(), handler.HandleSTTUpload)
	r.POST("/tts/request/:message_id", middlewares.AuthMiddleware(), handler.HandleTTSConvert)

	admin := r.Group("/admin")
	admin.Use(middlewares.AuthMiddleware())
	{
		admin.POST("/new-user", handler.HandleAddUser)
		admin.GET("/users", handler.HandleGetUserList)
		admin.DELETE("/delete-user/:user_id", handler.HandleDeleteUser)
		admin.POST("/set-quota/:user_id", handler.HandleSetQuota)
		admin.GET("/prompt-preset", handler.HandleAdminGetPromptPresets)
		admin.POST("/prompt-preset", handler.HandleAdminCreatePromptPreset)
		admin.DELETE("/prompt-preset/:prompt_preset_id", handler.HandleAdminDeletePromptPreset)
	}

	me := r.Group("/me")
	me.Use(middlewares.AuthMiddleware())
	{
		me.GET("/info", handler.HandleGetMeInfo)
		me.GET("/conversations", handler.HandleGetMeConversations)
	}
}
