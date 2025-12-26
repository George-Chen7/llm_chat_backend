package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/config"
	"backend/internal/middlewares"
	"backend/internal/controller"
)

func NewRouter(cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middlewares.CORSMiddleware())
	_ = cfg
	SetupRouter(r)
	return r
}

func SetupRouter(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.Static("/uploads", "./uploads")

	auth := r.Group("/auth")
	{
		auth.POST("/login", controller.HandleLogin)
		auth.POST("/reset-password", controller.HandleSetPassword)
		auth.POST("/refresh-token", middlewares.AuthMiddleware(), controller.HandleRefreshToken)
	}

	chat := r.Group("/chat")
	chat.Use(middlewares.AuthMiddleware())
	{
		chat.POST("/send-message/:conversation_id", controller.HandleChatSend)
		chat.GET("/history/:conversation_id", controller.HandleGetChatHistory)
		chat.POST("/new-conversation", controller.HandleNewChat)
		chat.PUT("/rename-conversation/:conversation_id", controller.HandleRenameChat)
		chat.DELETE("/delete-conversation/:conversation_id", controller.HandleDeleteChat)
		chat.POST("/upload-file", controller.HandleUploadFile)
		chat.GET("/prompt-preset", controller.HandleGetPromptPreset)
	}

	r.POST("/stt/request-stt", middlewares.AuthMiddleware(), controller.HandleSTTUpload)
	r.POST("/tts/request/:message_id", middlewares.AuthMiddleware(), controller.HandleTTSConvert)

	admin := r.Group("/admin")
	admin.Use(middlewares.AuthMiddleware())
	{
		admin.POST("/new-user", controller.HandleAddUser)
		admin.GET("/users", controller.HandleGetUserList)
		admin.DELETE("/delete-user/:user_id", controller.HandleDeleteUser)
		admin.POST("/set-quota/:user_id", controller.HandleSetQuota)
		admin.GET("/prompt-preset", controller.HandleAdminGetPromptPresets)
		admin.POST("/prompt-preset", controller.HandleAdminCreatePromptPreset)
		admin.DELETE("/prompt-preset/:prompt_preset_id", controller.HandleAdminDeletePromptPreset)
	}

	me := r.Group("/me")
	me.Use(middlewares.AuthMiddleware())
	{
		me.GET("/info", controller.HandleGetMeInfo)
		me.GET("/conversations", controller.HandleGetMeConversations)
	}
}
