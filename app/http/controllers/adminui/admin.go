package adminui

import (
	"fmt"
	"html/template"
	stdhttp "net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/goravel/framework/contracts/http"
	frameworksession "github.com/goravel/framework/session"
	"github.com/slxar/rustdesk-api-goravel/v3/global"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
	"gorm.io/gorm"
)

type Controller struct{}

const sessionTokenKey = "admin_ui_token"

func (Controller) Login(ctx http.Context) http.Response {
	if ctx.Request().Method() == stdhttp.MethodGet {
		return loginView(ctx, "")
	}
	username, password := strings.TrimSpace(ctx.Request().Input("username")), ctx.Request().Input("password")
	ip := ctx.Request().Ip()
	needCaptcha := false
	if global.LoginLimiter != nil {
		banned, required := global.LoginLimiter.CheckSecurityStatus(ip)
		needCaptcha = required
		if banned {
			return invalidLogin(ctx)
		}
	}
	if username == "" || len(username) > 64 || password == "" || len(password) > 256 || global.Config.App.DisablePwdLogin {
		if global.LoginLimiter != nil {
			global.LoginLimiter.RecordFailedAttempt(ip)
		}
		return invalidLogin(ctx)
	}
	if needCaptcha && !global.LoginLimiter.VerifyCaptcha(ctx.Request().Input("captcha_id"), ctx.Request().Input("captcha")) {
		return invalidLogin(ctx)
	}
	user := service.AllService.UserService.InfoByUsernamePassword(username, password)
	if user.Id == 0 || !service.AllService.UserService.CheckUserEnable(user) || !service.AllService.UserService.IsAdmin(user) {
		if global.LoginLimiter != nil {
			global.LoginLimiter.RecordFailedAttempt(ip)
		}
		return invalidLogin(ctx)
	}
	if err := ctx.Request().Session().Regenerate(true); err != nil {
		return loginView(ctx, "Sign-in is temporarily unavailable.")
	}
	token := service.AllService.UserService.Login(user, &model.LoginLog{UserId: user.Id, Client: model.LoginLogClientWebAdmin, Ip: ip, Type: model.LoginLogTypeAccount, Platform: "admin-ui"})
	ctx.Request().Session().Put(sessionTokenKey, token.Token)
	frameworksession.WriteCookie(ctx, ctx.Request().Session())
	if global.LoginLimiter != nil {
		global.LoginLimiter.RemoveAttempts(ip)
	}
	return ctx.Response().Redirect(stdhttp.StatusFound, "/_admin")
}

func invalidLogin(ctx http.Context) http.Response {
	return loginView(ctx, "Invalid sign-in details.")
}

func loginView(ctx http.Context, message string) http.Response {
	data := map[string]any{"CsrfToken": ctx.Request().Session().Token(), "Error": message}
	// The proven OIDC flow creates per-request state, verifier and nonce server-side;
	// a static browser link cannot safely replace it, so do not expose one here.
	data["OAuthProviders"] = oauthProviders()
	if global.LoginLimiter != nil {
		_, required := global.LoginLimiter.CheckSecurityStatus(ctx.Request().Ip())
		if required {
			if err, captcha := global.LoginLimiter.RequireCaptcha(); err == nil {
				if err, image := global.LoginLimiter.DrawCaptcha(captcha.Content); err == nil {
					data["CaptchaId"], data["CaptchaImage"] = captcha.Id, template.URL(image)
				}
			}
		}
	}
	return ctx.Response().View().Make("login.tmpl", data)
}

func oauthProviders() []string {
	if service.AllService == nil || service.AllService.OauthService == nil {
		return nil
	}
	return service.AllService.OauthService.GetOauthProviders()
}

func (Controller) Logout(ctx http.Context) http.Response {
	token, _ := ctx.Request().Session().Get(sessionTokenKey).(string)
	if user, _ := service.AllService.UserService.InfoByAccessToken(token); user.Id != 0 {
		service.AllService.UserService.Logout(user, token)
	}
	_ = ctx.Request().Session().Invalidate()
	frameworksession.WriteCookie(ctx, ctx.Request().Session())
	return ctx.Response().Redirect(stdhttp.StatusFound, "/_admin/login")
}

func (Controller) Dashboard(ctx http.Context) http.Response {
	return ctx.Response().View().Make("page.tmpl", page(ctx, "Dashboard", "", "", []string{"Users", "Peers", "Groups"}, [][]string{{fmt.Sprint(service.AllService.UserService.List(1, 1, nil).Total), fmt.Sprint(service.AllService.PeerService.List(1, 1, nil).Total), fmt.Sprint(service.AllService.GroupService.List(1, 1, nil).Total)}}))
}

func (Controller) List(ctx http.Context) http.Response {
	name, q := ctx.Request().Route("resource"), strings.TrimSpace(ctx.Request().Query("q"))
	currentPage, pageSize := pagination(ctx.Request().Query("page"), ctx.Request().Query("page_size"))
	title, columns, items, total, err := listPage(name, q, currentPage, pageSize)
	if err != nil {
		return ctx.Response().View().Make("page.tmpl", page(ctx, "Not found", "", "", nil, nil))
	}
	data := pageData(title, "/_admin/"+name, q, columns, items, currentPage, pageSize, total)
	data["CsrfToken"] = ctx.Request().Session().Token()
	if manageable(name) {
		data["ManageURL"] = managementPrefix + name
	}
	if ctx.Request().Header("HX-Request") == "true" {
		return ctx.Response().View().Make("table.tmpl", data)
	}
	return ctx.Response().View().Make("page.tmpl", data)
}

func page(ctx http.Context, title, path, query string, columns []string, items [][]string) map[string]any {
	data := pageData(title, path, query, columns, items, 1, 50, int64(len(items)))
	data["CsrfToken"] = ctx.Request().Session().Token()
	return data
}

func pagination(pageText, sizeText string) (uint, uint) {
	page, _ := strconv.Atoi(pageText)
	size, _ := strconv.Atoi(sizeText)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 25
	}
	if size > 100 {
		size = 100
	}
	return uint(page), uint(size)
}

func pageData(title, path, query string, columns []string, items [][]string, currentPage, pageSize uint, total int64) map[string]any {
	data := map[string]any{"Title": title, "Path": path, "Query": query, "Columns": columns, "Items": items, "Page": currentPage, "PageSize": pageSize, "Total": total}
	if path == "" {
		return data
	}
	link := func(page uint) string {
		values := url.Values{"page": {strconv.FormatUint(uint64(page), 10)}, "page_size": {strconv.FormatUint(uint64(pageSize), 10)}}
		if query != "" {
			values.Set("q", query)
		}
		return path + "?" + values.Encode()
	}
	if currentPage > 1 {
		data["PreviousURL"] = link(currentPage - 1)
	}
	if int64(currentPage)*int64(pageSize) < total {
		data["NextURL"] = link(currentPage + 1)
	}
	return data
}

func list(name, q string) (string, []string, [][]string, error) {
	title, columns, items, _, err := listPage(name, q, 1, 50)
	return title, columns, items, err
}

func listPage(name, q string, page, pageSize uint) (string, []string, [][]string, int64, error) {
	like := func(column string) func(*gorm.DB) {
		return func(tx *gorm.DB) {
			if q != "" {
				tx.Where(column+" like ?", "%"+q+"%")
			}
		}
	}
	switch name {
	case "users":
		r := service.AllService.UserService.List(page, pageSize, like("username"))
		out := make([][]string, 0, len(r.Users))
		for _, v := range r.Users {
			out = append(out, []string{fmt.Sprint(v.Id), v.Username, v.Email, v.Nickname, fmt.Sprint(v.Status)})
		}
		return "Users", []string{"ID", "Username", "Email", "Name", "Status"}, out, r.Total, nil
	case "peers":
		r := service.AllService.PeerService.List(page, pageSize, func(tx *gorm.DB) {
			if q != "" {
				tx.Where("alias like ? OR id like ? OR hostname like ?", "%"+q+"%", "%"+q+"%", "%"+q+"%")
			}
		})
		out := make([][]string, 0, len(r.Peers))
		for _, v := range r.Peers {
			out = append(out, []string{fmt.Sprint(v.RowId), v.Id, v.Alias, v.Hostname, v.Username})
		}
		return "Peers", []string{"Row", "ID", "Alias", "Host", "User"}, out, r.Total, nil
	case "groups":
		r := service.AllService.GroupService.List(page, pageSize, like("name"))
		out := make([][]string, 0, len(r.Groups))
		for _, v := range r.Groups {
			out = append(out, []string{fmt.Sprint(v.Id), v.Name, fmt.Sprint(v.Type)})
		}
		return "Groups", []string{"ID", "Name", "Type"}, out, r.Total, nil
	case "address-books":
		r := service.AllService.AddressBookService.List(page, pageSize, like("alias"))
		out := make([][]string, 0, len(r.AddressBooks))
		for _, v := range r.AddressBooks {
			out = append(out, []string{fmt.Sprint(v.RowId), v.Id, v.Alias, v.Hostname, fmt.Sprint(v.UserId)})
		}
		return "Address books", []string{"Row", "ID", "Alias", "Host", "Owner"}, out, r.Total, nil
	case "tags":
		r := service.AllService.TagService.List(page, pageSize, like("name"))
		out := make([][]string, 0, len(r.Tags))
		for _, v := range r.Tags {
			out = append(out, []string{fmt.Sprint(v.Id), v.Name, fmt.Sprint(v.UserId), fmt.Sprint(v.CollectionId)})
		}
		return "Tags", []string{"ID", "Name", "Owner", "Collection"}, out, r.Total, nil
	case "audits":
		r := service.AllService.AuditService.AuditConnList(page, pageSize, like("peer_id"))
		out := make([][]string, 0, len(r.AuditConns))
		for _, v := range r.AuditConns {
			out = append(out, []string{fmt.Sprint(v.Id), v.Action, v.PeerId, v.FromPeer, v.Ip})
		}
		return "Audit logs", []string{"ID", "Action", "Peer", "From", "IP"}, out, r.Total, nil
	case "alarms":
		r := service.AllService.AuditService.AuditAlarmList(page, pageSize, like("peer_id"))
		out := make([][]string, 0, len(r.AuditAlarms))
		for _, v := range r.AuditAlarms {
			out = append(out, []string{fmt.Sprint(v.Id), v.PeerId, fmt.Sprint(v.Type), fmt.Sprint(v.ConnId), v.Info})
		}
		return "Alarm logs", []string{"ID", "Peer", "Type", "Connection", "Info"}, out, r.Total, nil
	case "tokens":
		r := service.AllService.UserService.TokenList(page, pageSize, like("device_id"))
		out := make([][]string, 0, len(r.UserTokens))
		for _, v := range r.UserTokens {
			out = append(out, []string{fmt.Sprint(v.Id), fmt.Sprint(v.UserId), v.DeviceId, v.DeviceUuid, fmt.Sprint(v.ExpiredAt)})
		}
		return "Tokens", []string{"ID", "User", "Device", "UUID", "Expires"}, out, r.Total, nil
	case "oauth":
		r := service.AllService.OauthService.List(page, pageSize, like("op"))
		out := make([][]string, 0, len(r.Oauths))
		for _, v := range r.Oauths {
			out = append(out, []string{fmt.Sprint(v.Id), v.Op, v.OauthType, v.Issuer})
		}
		return "OAuth providers", []string{"ID", "Name", "Type", "Issuer"}, out, r.Total, nil
	case "ldap":
		return "LDAP", []string{"Status"}, [][]string{{"Configured: " + fmt.Sprint(global.Config.Ldap.Enable)}}, 1, nil
	case "config":
		return "Configuration", []string{"Setting", "Value"}, [][]string{{"ID server", global.Config.Rustdesk.IdServer}, {"Relay server", global.Config.Rustdesk.RelayServer}, {"API server", global.Config.Rustdesk.ApiServer}}, 3, nil
	default:
		return "", nil, nil, 0, fmt.Errorf("unknown resource")
	}
}
