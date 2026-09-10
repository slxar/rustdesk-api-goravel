package api

import (
	"github.com/gin-gonic/gin"
	apiResp "github.com/slxar/rustdesk-api-goravel/v3/http/response/api"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
	"net/http"
)

type User struct {
}

// Info 用户信息
// @Tags 用户
// @Summary 用户信息
// @Description 用户信息
// @Accept  json
// @Produce  json
// @Success 200 {object} apiResp.UserPayload
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /currentUser [post]
// @Router /user/info [get]
// @Security BearerAuth
func (u *User) Info(c *gin.Context) {
	user := service.AllService.UserService.CurUser(c)
	up := (&apiResp.UserPayload{}).FromUser(user)
	c.JSON(http.StatusOK, up)
}
