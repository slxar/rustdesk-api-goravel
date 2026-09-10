package legacy

import (
	"github.com/gin-gonic/gin"
	contractshttp "github.com/goravel/framework/contracts/http"
	goravelgin "github.com/goravel/gin"
)

// Handler lets Goravel's Gin driver dispatch to an unchanged Gin router.
func Handler(router *gin.Engine) contractshttp.HandlerFunc {
	return func(ctx contractshttp.Context) contractshttp.Response {
		instance := ctx.(*goravelgin.Context).Instance()
		// Dispatch through net/http so Gin starts with a fresh response status
		// and preserves Goravel's response-writer middleware (including timeout).
		router.ServeHTTP(instance.Writer, instance.Request)
		return nil
	}
}

// Middleware lets Goravel route groups retain existing Gin middleware.
func Middleware(name string, middleware gin.HandlerFunc) contractshttp.Middleware {
	return ginMiddleware{name: name, middleware: middleware}
}

type ginMiddleware struct {
	name       string
	middleware gin.HandlerFunc
}

func (m ginMiddleware) Handle(ctx contractshttp.Context) {
	m.middleware(ctx.(*goravelgin.Context).Instance())
}

func (m ginMiddleware) Signature() string { return m.name }
