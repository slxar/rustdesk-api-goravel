package adminui

import (
	"net/http"
	"strconv"
	"time"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
)

func (Controller) IntegrationTokens(ctx contractshttp.Context) contractshttp.Response {
	ctx.Response().Header("Cache-Control", "no-store")
	data := map[string]any{"CsrfToken": ctx.Request().Session().Token(), "Path": "/_admin/integration-tokens"}
	if ctx.Request().Method() == http.MethodPost {
		switch ctx.Request().Input("action") {
		case "create":
			user, _ := ctx.Value("admin_ui_user").(*model.User)
			days, err := strconv.Atoi(ctx.Request().Input("days"))
			if err != nil {
				data["Error"] = "Expiry must be 1–365 days."
				break
			}
			_, raw, err := service.CreateIntegrationToken(user, ctx.Request().Input("name"), ctx.Request().Input("targets"), days)
			if err != nil {
				data["Error"] = "Cannot create token. Use a name, valid target IDs (or *), and an expiry of 1–365 days."
			} else {
				data["Token"] = raw
			}
		case "revoke":
			id, err := strconv.ParseUint(ctx.Request().Input("id"), 10, 32)
			if err != nil || id == 0 || service.RevokeIntegrationToken(uint(id)) != nil {
				data["Error"] = "Cannot revoke this token."
			} else {
				data["Message"] = "Token revoked. Its access links and browser sessions are also revoked."
			}
		default:
			data["Error"] = "Invalid action."
		}
	}
	tokens, err := service.ListIntegrationTokens()
	if err != nil {
		data["Error"] = "Token list is temporarily unavailable."
	}
	type row struct {
		ID                                               uint
		Name, Prefix, Targets, Expires, LastUsed, Status string
		Active                                           bool
	}
	rows := make([]row, 0, len(tokens))
	for _, token := range tokens {
		status := "Active"
		if token.RevokedAt > 0 {
			status = "Revoked"
		} else if token.ExpiresAt <= time.Now().Unix() {
			status = "Expired"
		}
		last := "Never"
		if token.LastUsedAt > 0 {
			last = time.Unix(token.LastUsedAt, 0).UTC().Format(time.RFC3339)
		}
		rows = append(rows, row{token.Id, token.Name, token.TokenPrefix, token.TargetAllowlist, time.Unix(token.ExpiresAt, 0).UTC().Format(time.RFC3339), last, status, status == "Active"})
	}
	data["Tokens"] = rows
	return ctx.Response().View().Make("integration_tokens.tmpl", data)
}
