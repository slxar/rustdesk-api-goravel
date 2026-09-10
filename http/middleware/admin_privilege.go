package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/slxar/rustdesk-api-goravel/v3/http/response"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
)

// AdminPrivilege ...
func AdminPrivilege() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := service.AllService.UserService.CurUser(c)

		if !service.AllService.UserService.IsAdmin(u) {
			response.Fail(c, 403, response.TranslateMsg(c, "NoAccess"))
			c.Abort()
			return
		}

		c.Next()
	}
}
