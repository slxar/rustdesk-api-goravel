package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/slxar/rustdesk-api-goravel/v3/global"
	"github.com/slxar/rustdesk-api-goravel/v3/http/middleware"
	"github.com/slxar/rustdesk-api-goravel/v3/lib/jwt"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIntegrationTokenCanOnlyMintAllowedTargetLinks(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserToken{}, &model.IntegrationToken{}); err != nil {
		t.Fatal(err)
	}
	oldDB, oldServices, oldConfig, oldJWT, oldStore := service.DB, service.AllService, global.Config, global.Jwt, service.WebAccesses
	service.DB, service.AllService, global.Jwt, service.WebAccesses = db, &service.Service{}, &jwt.Jwt{}, &service.WebAccessStore{}
	global.Config.Rustdesk.ApiServer = "https://example.test"
	global.Config.Rustdesk.Key = "test-public-key"
	t.Cleanup(func() {
		service.DB, service.AllService, global.Config, global.Jwt, service.WebAccesses = oldDB, oldServices, oldConfig, oldJWT, oldStore
	})
	yes := true
	u := &model.User{Username: "isolated-integration-test", IsAdmin: &yes, Status: model.COMMON_STATUS_ENABLE}
	if err := db.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	token, raw, err := service.CreateIntegrationToken(u, "Support portal", "123456789", 90)
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.POST("/api/webclient/access-tokens", middleware.WebAccessAuth(), MintWebAccess)
	r.GET("/api/currentUser", middleware.RustAuth(), func(c *gin.Context) { c.Status(200) })
	request := func(path, body, auth string) *httptest.ResponseRecorder {
		method := "GET"
		if body != "" {
			method = "POST"
		}
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", auth)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	auth := "Bearer " + raw
	if w := request("/api/currentUser", "", auth); w.Code != 401 {
		t.Fatal("integration token authorized general API")
	}
	if w := request("/api/webclient/access-tokens", `{"id":"987654321"}`, auth); w.Code != 403 {
		t.Fatal("different target was authorized")
	}
	if w := request("/api/webclient/access-tokens", `{"id":"123456789"}`, "NotBearer "+raw); w.Code != 401 {
		t.Fatal("invalid authorization scheme accepted")
	}
	w := request("/api/webclient/access-tokens", `{"id":"123456789"}`, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("mint status %d: %s", w.Code, w.Body)
	}
	var output struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	_, access, err := service.WebAccesses.Redeem(strings.Split(output.URL, "#")[1])
	if err != nil {
		t.Fatal(err)
	}
	if access.IntegrationTokenID != token.Id || !service.WebAccessValid(access) {
		t.Fatal("derived token authority was lost")
	}
	if err := service.RevokeIntegrationToken(token.Id); err != nil {
		t.Fatal(err)
	}
	if service.WebAccessValid(access) {
		t.Fatal("revoked integration token left browser session valid")
	}
	if w := request("/api/webclient/access-tokens", `{"id":"123456789"}`, auth); w.Code != 401 {
		t.Fatal("revoked integration token minted a link")
	}
}
