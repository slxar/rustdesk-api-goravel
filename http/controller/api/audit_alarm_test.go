package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuditAlarmStoresOnlySupportedRustDeskEvents(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AuditAlarm{}); err != nil {
		t.Fatal(err)
	}
	previousDB, previousServices := service.DB, service.AllService
	service.DB, service.AllService = db, &service.Service{AuditService: &service.AuditService{}}
	t.Cleanup(func() { service.DB, service.AllService = previousDB, previousServices })

	post := func(body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/api/audit/alarm", strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		(&Audit{}).AuditAlarm(ctx)
		return response
	}

	response := post(`{"id":"peer-1","uuid":"device-uuid","typ":1,"info":"{\"ip\":\"192.0.2.4\"}","conn_id":42}`)
	if response.Code != http.StatusOK {
		t.Fatalf("valid alarm status = %d", response.Code)
	}
	var alarm model.AuditAlarm
	if err := db.First(&alarm).Error; err != nil {
		t.Fatal(err)
	}
	if alarm.PeerId != "peer-1" || alarm.Type != 1 || alarm.ConnId != 42 || alarm.Info != `{"ip":"192.0.2.4"}` {
		t.Fatalf("stored alarm = %#v", alarm)
	}

	post(`{"id":"peer-1","uuid":"device-uuid","typ":5,"info":"{}","conn_id":43}`)
	var count int64
	if err := db.Model(&model.AuditAlarm{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("unsupported alarm type was stored; count = %d", count)
	}
}
