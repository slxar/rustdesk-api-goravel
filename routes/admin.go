package routes

import (
	"github.com/goravel/framework/http/middleware"
	sessionmiddleware "github.com/goravel/framework/session/middleware"
	"github.com/slxar/rustdesk-api-goravel/v3/app/facades"
	"github.com/slxar/rustdesk-api-goravel/v3/app/http/controllers/adminui"
	adminmiddleware "github.com/slxar/rustdesk-api-goravel/v3/app/http/middleware/adminui"
)

// Admin registers the server-rendered administrator UI. The API remains at /api/admin.
func Admin() {
	controller := adminui.Controller{}
	root := facades.Route().Prefix("/_admin").Middleware(adminmiddleware.SecurityHeaders{}, sessionmiddleware.StartSession(), middleware.VerifyCsrfToken())
	root.Get("/login", controller.Login)
	root.Post("/login", controller.Login)
	secure := root.Middleware(adminmiddleware.Auth{})
	secure.Get("/", controller.Dashboard)
	secure.Post("/logout", controller.Logout)
	secure.Get("/webclient", controller.WebClient)
	secure.Post("/webclient", controller.WebClient)
	secure.Get("/integration-tokens", controller.IntegrationTokens)
	secure.Post("/integration-tokens", controller.IntegrationTokens)
	secure.Get("/manage/{resource}", controller.Manage)
	secure.Post("/manage/{action}", controller.Mutate)
	secure.Get("/{resource}", controller.List)
}
