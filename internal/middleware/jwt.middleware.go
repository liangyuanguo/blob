package middleware

import (
	"errors"
	"liangyuanguo/aw/blob/internal/config"
	"liangyuanguo/aw/blob/internal/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeader = "Authorization"
	BearerSchema        = "Bearer "
)

// JWTAuthMiddleware JWT认证中间件
func JWTAuthMiddleware(jwtUtil *utils.JWTUtil) gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.Config.Jwt.Secret == "" {
			c.Set("uid", "")
			c.Next()
			return
		}

		// 从请求头中获取token
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "未提供认证令牌",
			})
			return
		}

		// 检查token格式
		if !strings.HasPrefix(authHeader, BearerSchema) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "认证令牌格式不正确",
			})
			return
		}

		// 提取token
		tokenString := strings.TrimPrefix(authHeader, BearerSchema)
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "认证令牌不能为空",
			})
			return
		}

		// 解析和验证token
		claims, err := jwtUtil.ParseToken(tokenString)
		if err != nil {
			status := http.StatusUnauthorized
			if errors.Is(err, utils.ErrExpiredToken) {
				status = http.StatusForbidden
			}
			c.AbortWithStatusJSON(status, gin.H{
				"code":    status,
				"message": err.Error(),
			})
			return
		}

		//// 将用户信息存入上下文
		c.Set("uid", claims.Uid)
		//c.Set("username", claims.Username)

		c.Next()
	}
}
