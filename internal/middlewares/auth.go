package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTSecret 建议实际从 config 中读取，这里为了演示和 Mock 兼容先定义一个常量
// 之后可以改为：cfg.Server.JwtSecret
var JWTSecret = []byte("your-256-bit-secret")

// MyClaims 定义 JWT 中存储的数据
type MyClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// AuthMiddleware 完善后的 JWT 鉴权中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取 Authorization Header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"err_msg": "未携带 Token", "err_code": 401})
			c.Abort()
			return
		}

		// 2. 解析 Bearer 格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"err_msg": "Token 格式错误", "err_code": 401})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 3. 解析并校验 Token
		token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		})

		// 4. 校验失败处理
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"err_msg": "无效或已过期的 Token", "err_code": 401})
			c.Abort()
			return
		}

		// 5. 将解析出的用户信息存入上下文，方便后续 Handler 获取
		if claims, ok := token.Claims.(*MyClaims); ok {
			c.Set("username", claims.Username)
		}

		c.Next()
	}
}
