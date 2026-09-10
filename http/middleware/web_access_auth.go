package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
)

// Integration tokens authorize only access-link issuance, never general APIs.
func WebAccessAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if !strings.HasPrefix(authorization, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		raw := strings.TrimPrefix(authorization, "Bearer ")
		if !strings.HasPrefix(raw, "rdapi_") {
			RustAuth()(c)
			return
		}
		token, user, err := service.AuthenticateIntegrationToken(raw)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		c.Set("curUser", user)
		c.Set("integrationToken", token)
		c.Next()
	}
}
