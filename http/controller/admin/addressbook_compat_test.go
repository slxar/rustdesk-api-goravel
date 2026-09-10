package admin

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

func TestAddressBookListFiltersByAlias(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AddressBook{}, &model.AddressBookCollection{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AddressBook{Id: "keep", Alias: "office"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AddressBook{Id: "hide", Alias: "home"}).Error; err != nil {
		t.Fatal(err)
	}
	previousDB, previousServices := service.DB, service.AllService
	service.DB = db
	service.AllService = &service.Service{AddressBookService: &service.AddressBookService{}}
	t.Cleanup(func() { service.DB, service.AllService = previousDB, previousServices })
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/?alias=office", nil)
	(&AddressBook{}).List(c)
	var response struct {
		Data struct {
			Items []struct {
				Alias string `json:"alias"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data.Items) != 1 || response.Data.Items[0].Alias != "office" {
		t.Fatalf("alias filter returned %#v", response.Data.Items)
	}
}
