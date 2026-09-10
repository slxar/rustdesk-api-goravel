package my

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPeerListFiltersByAlias(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Peer{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Peer{Id: "keep", Alias: "office", UserId: 7}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Peer{Id: "hide", Alias: "home", UserId: 7}).Error; err != nil {
		t.Fatal(err)
	}
	previousDB, previousServices := service.DB, service.AllService
	service.DB = db
	service.AllService = &service.Service{UserService: &service.UserService{}, PeerService: &service.PeerService{}}
	t.Cleanup(func() { service.DB, service.AllService = previousDB, previousServices })

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?alias=office", nil)
	c.Set("curUser", &model.User{IdModel: model.IdModel{Id: 7}})
	(&Peer{}).List(c)

	var response struct {
		Data struct {
			Peers []struct {
				Alias string `json:"alias"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data.Peers) != 1 || response.Data.Peers[0].Alias != "office" {
		t.Fatalf("alias filter returned %#v", response.Data.Peers)
	}
}
