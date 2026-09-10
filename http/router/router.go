package router

import (
	"github.com/gin-gonic/gin"
	"github.com/slxar/rustdesk-api-goravel/v3/http/controller/web"
)

func WebInit(g *gin.Engine) {
	i := &web.Index{}
	g.GET("/", i.Index)
	g.Any("/webclient/*path", web.WebClient)
}
