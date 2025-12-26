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
		authHeader := c.GetHeader("Authorization")
		tokenString := ""

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if !(len(parts) == 2 && parts[0] == "Bearer") {
				c.JSON(http.StatusUnauthorized, gin.H{"err_msg": "invalid token format", "err_code": 401})
				c.Abort()
				return
			}
			tokenString = parts[1]
		} else {
			if cookieToken, err := c.Cookie("jwt_token"); err == nil && cookieToken != "" {
				tokenString = cookieToken
			}
		}
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"err_msg": "missing token", "err_code": 401})
			c.Abort()
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"err_msg": "invalid or expired token", "err_code": 401})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*MyClaims); ok {
			c.Set("username", claims.Username)
		}

		c.Next()
	}
}
