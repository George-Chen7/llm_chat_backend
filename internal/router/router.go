package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/config"
	"backend/internal/handler"
	"backend/internal/middlewares"
)

// NewRouter 构建 Gin 路由。
func NewRouter(cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// cfg 当前暂未使用，保留以便后续扩展
	_ = cfg

	SetupRouter(r)

	return r
}

// SetupRouter 配置所有路由。
func SetupRouter(r *gin.Engine) {
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 登录模块（无鉴权）
	public := r.Group("")
	{
		public.POST("/login", handler.HandleLogin)
		public.POST("/setPassword", handler.HandleSetPassword)
		public.POST("/refreshToken", handler.HandleRefreshToken)
	}

	// 聊天模块（需鉴权）
	chat := r.Group("")
	chat.Use(middlewares.AuthMiddleware())
	{
		chat.POST("/sendMessage", handler.HandleChatStream)
		chat.GET("/getChatHistory", handler.HandleGetChatHistory)
		chat.POST("/newChat", handler.HandleNewChat)
		chat.PUT("/renameChat", handler.HandleRenameChat)
		chat.DELETE("/deleteChat", handler.HandleDeleteChat)
		chat.GET("/getQuota", handler.HandleGetQuota)
	}

	// STT/TTS 模块（需鉴权）
	voice := r.Group("")
	voice.Use(middlewares.AuthMiddleware())
	{
		voice.POST("/upload", handler.HandleSTTUpload)
		voice.POST("/convert", handler.HandleTTSConvert)
	}

	// 管理模块（需鉴权）
	admin := r.Group("")
	admin.Use(middlewares.AuthMiddleware())
	{
		admin.POST("/addUser", handler.HandleAddUser)
		admin.DELETE("/deleteUser", handler.HandleDeleteUser)
		admin.POST("/setQuota", handler.HandleSetQuota)
		admin.GET("/getUser", handler.HandleGetUserList)
	}
}
