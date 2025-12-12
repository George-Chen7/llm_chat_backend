package middlewares

import "github.com/gin-gonic/gin"

// AuthMiddleware 占位鉴权中间件，后续可接入 JWT 校验。
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现 JWT 校验逻辑
		c.Next()
	}
}

