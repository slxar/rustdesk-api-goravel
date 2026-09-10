package adminui

import (
	"errors"
	"fmt"
	stdhttp "net/http"
	"strconv"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
)

const managementPrefix = "/_admin/manage/"

func (Controller) Manage(ctx http.Context) http.Response {
	resource := ctx.Request().Route("resource")
	title, columns, items, err := list(resource, strings.TrimSpace(ctx.Request().Query("q")))
	if err != nil || !manageable(resource) {
		return ctx.Response().View().Make("management.tmpl", managementPage(ctx, "Not found", "", nil, nil, ""))
	}
	notice := ""
	if ctx.Request().Query("result") == "error" {
		notice = "Unable to complete that request."
	}
	if ctx.Request().Query("result") == "ok" {
		notice = "Saved."
	}
	return ctx.Response().View().Make("management.tmpl", managementPage(ctx, title, resource, columns, items, notice))
}

func (Controller) Mutate(ctx http.Context) http.Response {
	if !managementAdmin(ctx) {
		return mutationResponse(ctx, "", errors.New("Unauthorized"))
	}
	action := ctx.Request().Route("action")
	if err := mutate(action, ctx); err != nil {
		return mutationResponse(ctx, action, err)
	}
	return mutationResponse(ctx, action, nil)
}

func managementAdmin(ctx http.Context) bool {
	token, _ := ctx.Request().Session().Get(sessionTokenKey).(string)
	u, _ := service.AllService.UserService.InfoByAccessToken(token)
	return token != "" && u.Id != 0 && u.IsAdmin != nil && service.AllService.UserService.CheckUserEnable(u) && service.AllService.UserService.IsAdmin(u)
}

func manageable(resource string) bool {
	switch resource {
	case "users", "groups", "tokens", "oauth":
		return true
	}
	return false
}

func managementPage(ctx http.Context, title, resource string, columns []string, items [][]string, notice string) map[string]any {
	return map[string]any{"Title": title, "Resource": resource, "Columns": columns, "Items": items, "Notice": notice, "CsrfToken": ctx.Request().Session().Token()}
}

func mutationResponse(ctx http.Context, action string, err error) http.Response {
	data := map[string]any{"OK": err == nil, "Message": "Saved."}
	if err != nil {
		data["Message"] = "Unable to complete that request."
	}
	if ctx.Request().Header("HX-Request") == "true" {
		return ctx.Response().View().Make("management_result.tmpl", data)
	}
	result := "ok"
	if err != nil {
		result = "error"
	}
	return ctx.Response().Redirect(stdhttp.StatusFound, managementPrefix+managementResource(action)+"?result="+result)
}

func managementResource(action string) string {
	resource := strings.Split(action, "-")[0]
	if resource == "user" {
		resource = "users"
	}
	if resource == "group" {
		resource = "groups"
	}
	if resource == "token" {
		resource = "tokens"
	}
	return resource
}

func validateManagementInput(action, id, confirm string) error {
	switch action {
	case "user-create", "group-create", "oauth-create":
		return nil
	case "user-update", "user-password", "group-update", "oauth-update":
		if _, err := positiveID(id); err != nil {
			return err
		}
		return nil
	case "user-delete", "group-delete", "oauth-delete", "token-revoke":
		if _, err := positiveID(id); err != nil {
			return err
		}
		if confirm != "delete" {
			return errors.New("confirmation required")
		}
		return nil
	default:
		return errors.New("unknown action")
	}
}

func positiveID(value string) (uint, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 0)
	if err != nil || id == 0 {
		return 0, errors.New("invalid ID")
	}
	return uint(id), nil
}

func mutate(action string, ctx http.Context) error {
	idText := ctx.Request().Input("id")
	if err := validateManagementInput(action, idText, ctx.Request().Input("confirm")); err != nil {
		return err
	}
	switch action {
	case "user-create":
		u, err := userFromRequest(ctx, 0)
		if err != nil {
			return err
		}
		password := ctx.Request().Input("password")
		if len(password) < 4 || len(password) > 32 {
			return errors.New("invalid password")
		}
		u.Password = password
		return service.AllService.UserService.Create(u)
	case "user-update":
		id, _ := positiveID(idText)
		existing := service.AllService.UserService.InfoById(id)
		if existing.Id == 0 {
			return errors.New("not found")
		}
		u, err := userFromRequest(ctx, id)
		if err != nil {
			return err
		}
		u.Password, u.Avatar = existing.Password, existing.Avatar
		return service.AllService.UserService.Update(u)
	case "user-password":
		id, _ := positiveID(idText)
		u := service.AllService.UserService.InfoById(id)
		password := ctx.Request().Input("password")
		if u.Id == 0 || len(password) < 4 || len(password) > 32 {
			return errors.New("invalid request")
		}
		return service.AllService.UserService.UpdatePassword(u, password)
	case "user-delete":
		id, _ := positiveID(idText)
		u := service.AllService.UserService.InfoById(id)
		if u.Id == 0 {
			return errors.New("not found")
		}
		return service.AllService.UserService.Delete(u)
	case "group-create", "group-update":
		name := strings.TrimSpace(ctx.Request().Input("name"))
		if len(name) == 0 || len(name) > 128 {
			return errors.New("invalid group")
		}
		typeID, err := strconv.Atoi(ctx.Request().Input("type"))
		if err != nil || (typeID != model.GroupTypeDefault && typeID != model.GroupTypeShare) {
			return errors.New("invalid group")
		}
		g := &model.Group{Name: name, Type: typeID}
		if action == "group-update" {
			g.Id, _ = positiveID(idText)
			if service.AllService.GroupService.InfoById(g.Id).Id == 0 {
				return errors.New("not found")
			}
			return service.AllService.GroupService.Update(g)
		}
		return service.AllService.GroupService.Create(g)
	case "group-delete":
		id, _ := positiveID(idText)
		g := service.AllService.GroupService.InfoById(id)
		if g.Id == 0 {
			return errors.New("not found")
		}
		return service.AllService.GroupService.Delete(g)
	case "token-revoke":
		id, _ := positiveID(idText)
		token := service.AllService.UserService.TokenInfoById(id)
		if token.Id == 0 {
			return errors.New("not found")
		}
		return service.AllService.UserService.DeleteToken(token)
	case "oauth-create", "oauth-update":
		o, err := oauthFromRequest(ctx)
		if err != nil {
			return err
		}
		if action == "oauth-update" {
			o.Id, _ = positiveID(idText)
			if service.AllService.OauthService.InfoById(o.Id).Id == 0 {
				return errors.New("not found")
			}
			return service.AllService.OauthService.Update(o)
		}
		if service.AllService.OauthService.InfoByOp(o.Op).Id != 0 {
			return errors.New("already exists")
		}
		return service.AllService.OauthService.Create(o)
	case "oauth-delete":
		id, _ := positiveID(idText)
		o := service.AllService.OauthService.InfoById(id)
		if o.Id == 0 {
			return errors.New("not found")
		}
		return service.AllService.OauthService.Delete(o)
	}
	return errors.New("unknown action")
}

func userFromRequest(ctx http.Context, id uint) (*model.User, error) {
	username := strings.TrimSpace(ctx.Request().Input("username"))
	if len(username) < 2 || len(username) > 64 {
		return nil, errors.New("invalid user")
	}
	groupID, err := positiveID(ctx.Request().Input("group_id"))
	if err != nil {
		return nil, err
	}
	status, err := strconv.Atoi(ctx.Request().Input("status"))
	if err != nil || (status != int(model.COMMON_STATUS_DISABLED) && status != int(model.COMMON_STATUS_ENABLE)) {
		return nil, errors.New("invalid user")
	}
	for _, value := range []string{ctx.Request().Input("email"), ctx.Request().Input("nickname"), ctx.Request().Input("remark")} {
		if len(value) > 256 {
			return nil, errors.New("invalid user")
		}
	}
	admin := ctx.Request().Input("is_admin") == "on"
	return &model.User{IdModel: model.IdModel{Id: id}, Username: username, Email: strings.TrimSpace(ctx.Request().Input("email")), Nickname: strings.TrimSpace(ctx.Request().Input("nickname")), GroupId: groupID, IsAdmin: &admin, Status: model.StatusCode(status), Remark: strings.TrimSpace(ctx.Request().Input("remark"))}, nil
}

func oauthFromRequest(ctx http.Context) (*model.Oauth, error) {
	typeName, op := strings.TrimSpace(ctx.Request().Input("oauth_type")), strings.TrimSpace(ctx.Request().Input("op"))
	if len(op) == 0 || len(op) > 64 || len(ctx.Request().Input("client_id")) == 0 || len(ctx.Request().Input("client_id")) > 512 || len(ctx.Request().Input("client_secret")) == 0 || len(ctx.Request().Input("client_secret")) > 512 || len(ctx.Request().Input("issuer")) > 2048 || len(ctx.Request().Input("scopes")) > 512 || model.ValidateOauthType(typeName) != nil {
		return nil, errors.New("invalid OAuth provider")
	}
	auto, pkce := ctx.Request().Input("auto_register") == "on", ctx.Request().Input("pkce_enable") == "on"
	o := &model.Oauth{Op: op, OauthType: typeName, ClientId: ctx.Request().Input("client_id"), ClientSecret: ctx.Request().Input("client_secret"), Issuer: strings.TrimSpace(ctx.Request().Input("issuer")), Scopes: strings.TrimSpace(ctx.Request().Input("scopes")), AutoRegister: &auto, PkceEnable: &pkce, PkceMethod: model.PKCEMethodS256}
	if err := o.FormatOauthInfo(); err != nil {
		return nil, fmt.Errorf("invalid OAuth provider")
	}
	return o, nil
}
