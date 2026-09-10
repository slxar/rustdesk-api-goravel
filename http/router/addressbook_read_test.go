package router

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/slxar/rustdesk-api-goravel/v3/global"
	"github.com/slxar/rustdesk-api-goravel/v3/lib/jwt"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Exercise the actual authenticated router with native POSTs and official
// res/ab.py GETs. Every record lives in an isolated in-memory database.
func TestAddressBookReadsSupportOfficialScriptAndNativePagination(t *testing.T) {
	t.Chdir("../..")
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}, &model.UserToken{}, &model.AddressBook{}, &model.AddressBookCollection{}, &model.AddressBookCollectionRule{}, &model.Tag{}); err != nil {
		t.Fatal(err)
	}
	oldDB, oldServices, oldConfig, oldServiceConfig, oldJWT, oldLocalizer := service.DB, service.AllService, global.Config, service.Config, global.Jwt, global.Localizer
	service.DB, service.AllService, service.Config, global.Jwt = db, &service.Service{}, &global.Config, &jwt.Jwt{}
	global.Config.Rustdesk.Personal = 1
	global.Config.Gin.ResourcesPath = "resources"
	global.InitI18n()
	t.Cleanup(func() {
		service.DB, service.AllService, global.Config, service.Config, global.Jwt, global.Localizer = oldDB, oldServices, oldConfig, oldServiceConfig, oldJWT, oldLocalizer
	})
	create := func(value any) {
		t.Helper()
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	reader := &model.User{Username: "read-test", GroupId: 1, Status: model.COMMON_STATUS_ENABLE}
	owner := &model.User{Username: "owner-test", GroupId: 2, Status: model.COMMON_STATUS_ENABLE}
	create(reader)
	create(owner)
	create(&model.UserToken{UserId: reader.Id, Token: "isolated-read-token", ExpiredAt: time.Now().Add(time.Hour).Unix()})
	for i := 1; i <= 105; i++ {
		create(&model.AddressBookCollection{UserId: reader.Id, Name: fmt.Sprintf("Book %03d", i)})
	}
	shared := &model.AddressBookCollection{UserId: owner.Id, Name: "Shared office"}
	create(shared)
	create(&model.AddressBookCollection{UserId: owner.Id, Name: "Hidden"})
	orphan := &model.AddressBookCollection{UserId: 999, Name: "Orphan"}
	create(orphan)
	for _, rule := range []*model.AddressBookCollectionRule{
		{UserId: owner.Id, CollectionId: shared.Id, Type: model.ShareAddressBookRuleTypePersonal, ToId: reader.Id, Rule: 1},
		{UserId: owner.Id, CollectionId: shared.Id, Type: model.ShareAddressBookRuleTypeGroup, ToId: reader.GroupId, Rule: 2},
		{UserId: reader.Id, CollectionId: 1, Type: model.ShareAddressBookRuleTypeGroup, ToId: reader.GroupId, Rule: 1},
		{UserId: 999, CollectionId: orphan.Id, Type: model.ShareAddressBookRuleTypePersonal, ToId: reader.Id, Rule: 1},
	} {
		create(rule)
	}
	for i, alias := range []string{"office-one", "home", "office-two", "office-three"} {
		create(&model.AddressBook{Id: fmt.Sprintf("10000%d", i), UserId: reader.Id, Alias: alias})
	}
	bulkPeers := make([]model.AddressBook, 1050)
	for i := range bulkPeers {
		bulkPeers[i] = model.AddressBook{Id: fmt.Sprintf("bulk-%04d", i), UserId: reader.Id, CollectionId: 1}
	}
	if err := db.CreateInBatches(bulkPeers, 100).Error; err != nil {
		t.Fatal(err)
	}
	create(&model.AddressBook{Id: "200000", UserId: owner.Id, CollectionId: shared.Id, Alias: "shared"})
	create(&model.AddressBook{Id: "300000", UserId: owner.Id, Alias: "private"})
	create(&model.Tag{UserId: reader.Id, Name: "office", Color: 4281558681})
	r := gin.New()
	ApiInit(r)
	request := func(method, path, token string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	read := func(method, path string) string {
		t.Helper()
		w := request(method, path, "isolated-read-token")
		if w.Code != 200 || strings.Contains(w.Body.String(), `"error"`) {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body)
		}
		return w.Body.String()
	}
	personal := fmt.Sprintf("1-%d-0", reader.Id)
	sharedGUID := fmt.Sprintf("2-%d-%d", owner.Id, shared.Id)
	for _, path := range []string{"/api/ab/personal", "/api/ab/shared/profiles?current=2&pageSize=100", "/api/ab/peers?ab=" + personal, "/api/ab/tags/" + personal} {
		if w := request("GET", path, ""); w.Code != 401 {
			t.Fatalf("unauthenticated GET %s returned %d", path, w.Code)
		}
		if get, post := read("GET", path), read("POST", path); get != post {
			t.Fatalf("GET/POST mismatch for %s: %s / %s", path, get, post)
		}
	}
	var profiles struct {
		Total int `json:"total"`
		Data  []struct {
			Guid, Name string
			Rule       int
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(read("GET", "/api/ab/shared/profiles?current=2&pageSize=100")), &profiles); err != nil {
		t.Fatal(err)
	}
	if profiles.Total != 106 || len(profiles.Data) != 6 || profiles.Data[5].Guid != sharedGUID || profiles.Data[5].Rule != 2 {
		t.Fatalf("incomplete, duplicated or unauthorized profiles: %+v", profiles)
	}
	if body := read("GET", "/api/ab/shared/profiles?name=Shared%20office"); !strings.Contains(body, `"total":1`) || !strings.Contains(body, sharedGUID) {
		t.Fatal(body)
	}
	var peers struct {
		Total int `json:"total"`
		Data  []struct {
			Id string `json:"id"`
		} `json:"data"`
	}
	path := "/api/ab/peers?ab=" + personal + "&alias=%25office%25&current=2&pageSize=2"
	if err := json.Unmarshal([]byte(read("GET", path)), &peers); err != nil {
		t.Fatal(err)
	}
	if peers.Total != 3 || len(peers.Data) != 1 || peers.Data[0].Id != "100003" {
		t.Fatalf("incorrect filtered page: %+v", peers)
	}
	bulkPath := fmt.Sprintf("/api/ab/peers?ab=1-%d-1&current=11&pageSize=100", reader.Id)
	for _, method := range []string{"GET", "POST"} {
		if err := json.Unmarshal([]byte(read(method, bulkPath)), &peers); err != nil {
			t.Fatal(err)
		}
		if peers.Total != 1050 || len(peers.Data) != 50 || peers.Data[0].Id != "bulk-1000" || peers.Data[49].Id != "bulk-1049" {
			t.Fatalf("peers after the old 1000-row cutoff were lost: %+v", peers)
		}
	}
	for _, query := range []string{"id=missing", "alias=" + url.QueryEscape("' OR 1=1 --"), "current=9&pageSize=100"} {
		if body := read("GET", "/api/ab/peers?ab="+personal+"&"+query); !strings.Contains(body, `"data":[]`) {
			t.Fatal(body)
		}
	}
	if body := read("GET", "/api/ab/peers?ab="+sharedGUID); !strings.Contains(body, "200000") || strings.Contains(body, "300000") {
		t.Fatal(body)
	}
	for _, path := range []string{
		fmt.Sprintf("/api/ab/peers?ab=2-%d-0", owner.Id),
		fmt.Sprintf("/api/ab/tags/2-%d-0", owner.Id),
		fmt.Sprintf("/api/ab/peers?ab=2-%d-%d", owner.Id, shared.Id+1),
		"/api/ab/peers?ab=" + personal + "&current=-1",
		"/api/ab/peers?ab=" + personal + "&pageSize=1001",
		"/api/ab/shared/profiles?current=18446744073709551615&pageSize=100",
	} {
		w := request("GET", path, "isolated-read-token")
		if !strings.Contains(w.Body.String(), `"error"`) {
			t.Fatalf("unauthorized or malformed read accepted: %s: %s", path, w.Body)
		}
	}
}
