package bootstrap

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	contractsroute "github.com/goravel/framework/contracts/route"
	"github.com/sirupsen/logrus"
	"github.com/slxar/rustdesk-api-goravel/v3/global"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
	"github.com/slxar/rustdesk-api-goravel/v3/utils"
	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/encoding/protowire"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func checkWebClientSocketOutlivesHTTPTimeout(t *testing.T, router contractsroute.Route) {
	t.Setenv("SESSION_FILES", t.TempDir())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	yes := true
	user := &model.User{Username: "isolated-web-socket", IsAdmin: &yes, Status: model.COMMON_STATUS_ENABLE}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	oldDB, oldServices, oldStore, oldConfig, oldLogger, oldLimiter := service.DB, service.AllService, service.WebAccesses, global.Config, global.Logger, global.LoginLimiter
	service.DB, service.AllService, service.WebAccesses = db, &service.Service{}, &service.WebAccessStore{}
	global.Logger = logrus.New()
	global.LoginLimiter = utils.NewLoginLimiter(utils.SecurityPolicy{CaptchaThreshold: -1})
	global.Config.Rustdesk.ApiServer = "https://example.test"
	global.Config.Rustdesk.Key = "test-public-key"
	t.Cleanup(func() {
		service.DB, service.AllService, service.WebAccesses, global.Config, global.Logger, global.LoginLimiter = oldDB, oldServices, oldStore, oldConfig, oldLogger, oldLimiter
	})
	token, _ := service.WebAccesses.Issue("123456789", user.Id, time.Minute)
	cookie, _, err := service.WebAccesses.Redeem(token)
	if err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		var request []byte
		if websocket.Message.Receive(ws, &request) != nil {
			return
		}
		time.Sleep(4 * time.Second) // Longer than Goravel's default three-second timeout.
		_ = websocket.Message.Send(ws, []byte{0x5a, 2, 0x18, 2})
		_ = websocket.Message.Receive(ws, &request)
	}))
	defer upstream.Close()
	t.Setenv("RUSTDESK_WEBCLIENT_ID_WS", "ws"+strings.TrimPrefix(upstream.URL, "http"))
	server := httptest.NewServer(router)
	defer server.Close()
	cfg, _ := websocket.NewConfig("ws"+strings.TrimPrefix(server.URL, "http")+"/webclient/ws/id", "https://example.test")
	cfg.Header.Set("Cookie", "rustdesk_web_access="+cookie)
	ws, err := websocket.DialConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	_ = ws.SetDeadline(time.Now().Add(8 * time.Second))
	field := func(n protowire.Number, value string) []byte {
		return protowire.AppendString(protowire.AppendTag(nil, n, protowire.BytesType), value)
	}
	request := append(field(1, "123456789"), 0x10, 2)
	request = append(request, field(3, "test-public-key")...)
	if err := websocket.Message.Send(ws, field(8, string(request))); err != nil {
		t.Fatal(err)
	}
	var reply []byte
	if err := websocket.Message.Receive(ws, &reply); err != nil || string(reply) != string([]byte{0x5a, 2, 0x18, 2}) {
		t.Fatalf("WebSocket failed across HTTP timeout: %v %x", err, reply)
	}
}
