package middleware

import (
	"net/http"
	"net/http/httptest"
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

func TestRustAuthRejectsExpiredAndDisabledTokens(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserToken{}); err != nil {
		t.Fatal(err)
	}
	previousDB, previousServices, previousJwt := service.DB, service.AllService, global.Jwt
	service.DB, service.AllService, global.Jwt = db, &service.Service{UserService: &service.UserService{}}, &jwt.Jwt{}
	t.Cleanup(func() { service.DB, service.AllService, global.Jwt = previousDB, previousServices, previousJwt })
	for _, tc := range []struct {
		name, token string
		status      model.StatusCode
		expiry      int64
	}{
		{"expired", "expired", model.COMMON_STATUS_ENABLE, time.Now().Add(-time.Minute).Unix()},
		{"disabled", "disabled", model.COMMON_STATUS_DISABLED, time.Now().Add(time.Minute).Unix()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := &model.User{Username: tc.name, Status: tc.status}
			if err := db.Create(u).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&model.UserToken{UserId: u.Id, Token: tc.token, ExpiredAt: tc.expiry}).Error; err != nil {
				t.Fatal(err)
			}
			r := gin.New()
			r.Use(RustAuth())
			r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d", w.Code)
			}
		})
	}
}
