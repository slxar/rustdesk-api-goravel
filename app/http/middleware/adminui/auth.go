package adminui

import (
	stdhttp "net/http"

	"github.com/goravel/framework/contracts/http"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
)

const sessionTokenKey = "admin_ui_token"

type Auth struct{}

func (Auth) Signature() string { return "admin_ui.auth" }

func (Auth) Handle(ctx http.Context) {
	token, _ := ctx.Request().Session().Get(sessionTokenKey).(string)
	if token == "" {
		_ = ctx.Response().Redirect(stdhttp.StatusFound, "/_admin/login").Abort()
		return
	}
	user, _ := service.AllService.UserService.InfoByAccessToken(token)
	if user.Id == 0 || !service.AllService.UserService.CheckUserEnable(user) || !service.AllService.UserService.IsAdmin(user) {
		_ = ctx.Response().Redirect(stdhttp.StatusFound, "/_admin/login").Abort()
		return
	}
	ctx.WithValue("admin_ui_user", user)
	ctx.Request().Next()
}

type SecurityHeaders struct{}

func (SecurityHeaders) Signature() string { return "admin_ui.security_headers" }

func (SecurityHeaders) Handle(ctx http.Context) {
	ctx.Response().Header("Content-Security-Policy", "default-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; object-src 'none'")
	ctx.Response().Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	ctx.Response().Header("Referrer-Policy", "no-referrer")
	ctx.Response().Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	ctx.Response().Header("X-Content-Type-Options", "nosniff")
	ctx.Response().Header("X-Frame-Options", "DENY")
	ctx.Request().Next()
}
