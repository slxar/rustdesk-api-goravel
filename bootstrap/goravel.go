package bootstrap

import (
	"github.com/goravel/framework/contracts/foundation"
	contractsroute "github.com/goravel/framework/contracts/route"
	framework "github.com/goravel/framework/foundation"
	frameworkhttp "github.com/goravel/framework/http"
	"github.com/goravel/framework/log"
	"github.com/goravel/framework/route"
	"github.com/goravel/framework/session"
	frameworkpath "github.com/goravel/framework/support/path"
	"github.com/goravel/framework/validation"
	"github.com/goravel/framework/view"
	"github.com/goravel/gin"
	ginfacades "github.com/goravel/gin/facades"
	"github.com/slxar/rustdesk-api-goravel/v3/app/http/legacy"
	legacyhttp "github.com/slxar/rustdesk-api-goravel/v3/http"
	appRoutes "github.com/slxar/rustdesk-api-goravel/v3/routes"
	"net"
	"time"
)

// Boot creates only the Goravel services needed to own this application's HTTP lifecycle.
func Boot(addr string) foundation.Application {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		port = addr
	}
	return framework.Setup().
		WithConfig(func() {
			config := framework.App.MakeConfig()
			config.Add("app", map[string]any{"debug": false})
			config.Add("session", map[string]any{
				"default":         "file",
				"drivers":         map[string]any{"file": map[string]any{"driver": "file"}},
				"lifetime":        config.Env("SESSION_LIFETIME", 120),
				"expire_on_close": false,
				"files":           config.Env("SESSION_FILES", frameworkpath.Storage("framework/sessions")),
				"gc_interval":     30,
				"cookie":          "rustdesk_admin_session",
				"path":            "/_admin",
				"domain":          "",
				"secure":          true,
				"http_only":       true,
				"same_site":       "lax",
			})
			config.Add("http", map[string]any{
				"default": "gin",
				"host":    host,
				"port":    port,
				"drivers": map[string]any{"gin": map[string]any{
					"body_limit": 4096, "header_limit": 4096,
					"route": func() (contractsroute.Route, error) { return ginfacades.Route("gin"), nil },
				}},
			})
		}).
		WithProviders(func() []foundation.ServiceProvider {
			return []foundation.ServiceProvider{
				&log.ServiceProvider{}, &frameworkhttp.ServiceProvider{}, &validation.ServiceProvider{},
				&session.ServiceProvider{}, &view.ServiceProvider{}, &route.ServiceProvider{}, &gin.ServiceProvider{},
			}
		}).
		WithRouting(func() {
			router := framework.App.MakeRoute()
			router.Static("public", frameworkpath.Public())
			appRoutes.Admin()
			engine := legacyhttp.NewEngine()
			// WebSockets have their own capability deadline and must outlive the
			// normal buffered HTTP timeout.
			timeout := gin.Timeout(time.Duration(framework.App.MakeConfig().GetInt("http.request_timeout", 3)) * time.Second)
			for _, path := range []string{"/webclient/ws/id", "/webclient/ws/relay"} {
				router.Get(path, legacy.Handler(engine)).WithoutMiddleware(timeout)
			}
			router.Fallback(legacy.Handler(engine))
		}).
		Create()
}
