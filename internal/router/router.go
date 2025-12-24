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

	public := r.Group("")
	{
		public.POST("/login", handler.HandleLogin)
		public.POST("/setPassword", handler.HandleSetPassword)
		public.POST("/refreshToken", handler.HandleRefreshToken)
	}

	chat := r.Group("")
	chat.Use(middlewares.AuthMiddleware())
	{
		chat.POST("/chat/send-message/:conversation_id", handler.HandleChatStream)
		chat.GET("/getChatHistory", handler.HandleGetChatHistory)
		chat.POST("/newChat", handler.HandleNewChat)
		chat.PUT("/renameChat", handler.HandleRenameChat)
		chat.DELETE("/deleteChat", handler.HandleDeleteChat)
		chat.GET("/getQuota", handler.HandleGetQuota)
	}

	voice := r.Group("")
	voice.Use(middlewares.AuthMiddleware())
	{
		voice.POST("/stt/request-stt", handler.HandleSTTUpload)
		voice.POST("/tts/request-tts/:message_id", handler.HandleTTSConvert)
	}

	admin := r.Group("")
	admin.Use(middlewares.AuthMiddleware())
	{
		admin.POST("/addUser", handler.HandleAddUser)
		admin.DELETE("/deleteUser", handler.HandleDeleteUser)
		admin.POST("/setQuota", handler.HandleSetQuota)
		admin.GET("/getUser", handler.HandleGetUserList)
	}
}
