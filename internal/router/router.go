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
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// --- 1. 认证模块 (对齐 /auth/...) ---
	auth := r.Group("/auth")
	{
		auth.POST("/login", handler.HandleLogin)             // 对齐 OpenAPI: /auth/login
		auth.POST("/reset-password", handler.HandleSetPassword) // 对齐 OpenAPI: /auth/reset-password
		auth.POST("/refresh-token", handler.HandleRefreshToken) // 对齐 OpenAPI: /auth/refresh-token
	}

	// --- 2. 聊天模块 (对齐 /chat/...) ---
	chat := r.Group("/chat")
	chat.Use(middlewares.AuthMiddleware())
	{
		chat.POST("/send-message/:conversation_id", handler.HandleChatStream) // 已对齐
		chat.GET("/history", handler.HandleGetChatHistory)                    // 对齐 OpenAPI: /chat/history
		chat.POST("/new-conversation", handler.HandleNewChat)                // 对齐 OpenAPI: /chat/new-conversation
		chat.PUT("/rename", handler.HandleRenameChat)                        // 对齐 OpenAPI: /chat/rename
		chat.DELETE("/delete", handler.HandleDeleteChat)                     // 对齐 OpenAPI: /chat/delete
		chat.GET("/quota", handler.HandleGetQuota)                           // 对齐 OpenAPI: /chat/quota
	}

	// --- 3. 语音模块 (对齐 /stt 和 /tts) ---
	voice := r.Group("")
	voice.Use(middlewares.AuthMiddleware())
	{
		voice.POST("/stt/request-stt", handler.HandleSTTUpload)       // 对齐 OpenAPI: /stt/request-stt
		voice.POST("/tts/request-tts/:message_id", handler.HandleTTSConvert) // 对齐 OpenAPI: /tts/request-tts/:message_id
	}

	// --- 4. 管理员模块 (对齐 /admin/...) ---
	admin := r.Group("/admin")
	admin.Use(middlewares.AuthMiddleware())
	{
		admin.POST("/user/add", handler.HandleAddUser)       // 对齐 OpenAPI: /admin/user/add
		admin.DELETE("/user/delete", handler.HandleDeleteUser) // 对齐 OpenAPI: /admin/user/delete
		admin.POST("/user/set-quota", handler.HandleSetQuota)  // 对齐 OpenAPI: /admin/user/set-quota
		admin.GET("/user/list", handler.HandleGetUserList)    // 对齐 OpenAPI: /admin/user/list
	}
}