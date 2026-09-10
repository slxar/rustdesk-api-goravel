package adminui

import (
	"net/http"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
)

func (Controller) WebClient(ctx contractshttp.Context) contractshttp.Response {
	data := map[string]any{"CsrfToken": ctx.Request().Session().Token(), "Path": "/_admin/webclient"}
	if ctx.Request().Method() == http.MethodPost {
		user, _ := ctx.Value("admin_ui_user").(*model.User)
		link, err := service.IssueWebLink(user, ctx.Request().Input("id"), 5*time.Minute)
		if err != nil {
			data["Error"] = "Cannot create access for this host. Check the ID and server configuration."
		} else {
			data["Link"] = link
		}
	}
	return ctx.Response().View().Make("webclient.tmpl", data)
}
