package web

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	messagews "github.com/gorilla/websocket"
	"github.com/slxar/rustdesk-api-goravel/v3/global"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/encoding/protowire"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func pb(n protowire.Number, b []byte) []byte {
	return protowire.AppendBytes(protowire.AppendTag(nil, n, protowire.BytesType), b)
}
func vi(n protowire.Number, v uint64) []byte {
	return protowire.AppendVarint(protowire.AppendTag(nil, n, protowire.VarintType), v)
}
func join(parts ...[]byte) []byte { return bytes.Join(parts, nil) }

func TestWebGatewayScopeReplayAndWireProxy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	isAdmin := true
	u := &model.User{Username: "isolated-web-test", IsAdmin: &isAdmin, Status: model.COMMON_STATUS_ENABLE}
	if err := db.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	oldDB, oldServices, oldConfig, oldStore := service.DB, service.AllService, global.Config, service.WebAccesses
	service.DB, service.AllService, service.WebAccesses = db, &service.Service{}, &service.WebAccessStore{}
	t.Cleanup(func() {
		service.DB, service.AllService, global.Config, service.WebAccesses = oldDB, oldServices, oldConfig, oldStore
	})
	global.Config.Rustdesk.ApiServer = "https://example.test"
	pk, sk, _ := ed25519.GenerateKey(nil)
	global.Config.Rustdesk.Key = base64.StdEncoding.EncodeToString(pk)
	token, _ := service.WebAccesses.Issue("123456789", u.Id, time.Minute)
	r := gin.New()
	r.Any("/webclient/*path", WebClient)
	request := func(path, body, origin, cookie string) *httptest.ResponseRecorder {
		method := "GET"
		if body != "" {
			method = "POST"
		}
		req := httptest.NewRequest(method, "https://example.test"+path, strings.NewReader(body))
		req.Header.Set("Origin", origin)
		req.Header.Set("Content-Type", "application/json")
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: webCookie, Value: cookie})
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	body, _ := json.Marshal(map[string]string{"token": token})
	if w := request("/webclient/redeem", string(body), "https://evil.test", ""); w.Code != 403 {
		t.Fatal("cross-origin redemption accepted")
	}
	w := request("/webclient/redeem", string(body), "https://example.test", "")
	if !strings.Contains(w.Header().Get("Content-Security-Policy"), "connect-src 'self' data:;") {
		t.Fatal("bundled zstd WASM must initialize without permitting external network origins")
	}
	if w.Code != 200 {
		t.Fatalf("redemption: %d %s", w.Code, w.Body)
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.Path != "/webclient" || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("unsafe cookie")
	}
	if w := request("/webclient/redeem", string(body), "https://example.test", ""); w.Code != 403 {
		t.Fatal("replay accepted")
	}
	if w := request("/webclient/session-config", "", "", ""); w.Code != 401 {
		t.Fatal("config is public")
	}
	if w := request("/webclient/session-config", "", "", cookie.Value); w.Code != 200 || !strings.Contains(w.Body.String(), "123456789") {
		t.Fatal("scoped config failed")
	}
	a := service.WebAccesses.Session(cookie.Value)
	punch := func(id string) []byte {
		return pb(8, join(pb(1, []byte(id)), vi(2, 2), pb(3, []byte(global.Config.Rustdesk.Key))))
	}
	if webFirstMessage(punch("987654321"), a, false) {
		t.Fatal("different target accepted")
	}
	for _, connType := range []uint64{0, 1, 2, 3, 4} {
		data := pb(8, join(pb(1, []byte(a.PeerID)), vi(2, 2), pb(3, []byte(global.Config.Rustdesk.Key)), vi(4, connType)))
		if webFirstMessage(data, a, false) != (connType <= 1) {
			t.Fatalf("connection type %d must permit only desktop and file transfer", connType)
		}
		other := pb(8, join(pb(1, []byte("987654321")), vi(2, 2), pb(3, []byte(global.Config.Rustdesk.Key)), vi(4, connType)))
		if webFirstMessage(other, a, false) {
			t.Fatal("file transfer must remain bound to the capability host")
		}
	}
	if webFirstMessage(join(punch(a.PeerID), punch("987654321")), a, false) {
		t.Fatal("multiple oneofs accepted")
	}
	if webFirstMessage(pb(8, join(pb(1, []byte(a.PeerID)), pb(1, []byte("987654321")))), a, false) {
		t.Fatal("duplicate target accepted")
	}
	if webFirstMessage(pb(15, pb(1, []byte(a.PeerID))), a, false) {
		t.Fatal("registration accepted")
	}
	if webFirstMessage(pb(8, join(pb(1, []byte(a.PeerID)), pb(2, []byte{2}), pb(3, []byte(global.Config.Rustdesk.Key)))), a, false) {
		t.Fatal("incorrect protobuf wire type accepted")
	}
	relayID := "isolated-relay-uuid-123456"
	identity := join(pb(1, []byte(a.PeerID)), pb(2, pk))
	signed := append(ed25519.Sign(sk, identity), identity...)
	reply := pb(19, join(pb(2, []byte(relayID)), pb(5, signed)))
	bad := bytes.Clone(reply)
	bad[len(bad)-1] ^= 1
	if webRendezvousResponse(bad, a) {
		t.Fatal("invalid signature accepted")
	}
	idServer := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		var data []byte
		if websocket.Message.Receive(ws, &data) != nil || !bytes.Equal(data, punch(a.PeerID)) {
			return
		}
		_ = websocket.Message.Send(ws, reply)
		_ = websocket.Message.Receive(ws, &data)
	}))
	defer idServer.Close()
	t.Setenv("RUSTDESK_WEBCLIENT_ID_WS", "ws"+strings.TrimPrefix(idServer.URL, "http"))
	relayServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := messagews.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		if _, _, err := ws.ReadMessage(); err != nil {
			return
		}
		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if len(data) > 0 && data[0] == 0xAC {
				// Also fragment the host response so both proxy directions must
				// preserve message boundaries, including interleaved ping frames.
				writer, err := ws.NextWriter(messagews.BinaryMessage)
				if err != nil {
					return
				}
				for len(data) > 0 {
					n := min(len(data), 4096)
					if _, err := writer.Write(data[:n]); err != nil {
						return
					}
					data = data[n:]
					if err := ws.WriteControl(messagews.PingMessage, nil, time.Now().Add(time.Second)); err != nil {
						return
					}
				}
				if writer.Close() != nil {
					return
				}
			} else if ws.WriteMessage(messagews.BinaryMessage, data) != nil {
				return
			}
		}
	}))
	defer relayServer.Close()
	t.Setenv("RUSTDESK_WEBCLIENT_RELAY_WS", "ws"+strings.TrimPrefix(relayServer.URL, "http"))
	gateway := httptest.NewServer(r)
	defer gateway.Close()
	dial := func(path, origin string) *websocket.Conn {
		cfg, _ := websocket.NewConfig("ws"+strings.TrimPrefix(gateway.URL, "http")+path, origin)
		cfg.Header.Set("Cookie", webCookie+"="+cookie.Value)
		ws, err := websocket.DialConfig(cfg)
		if err != nil {
			t.Fatal(err)
		}
		_ = ws.SetDeadline(time.Now().Add(3 * time.Second))
		return ws
	}
	ws := dial("/webclient/ws/id", "https://example.test")
	if websocket.Message.Send(ws, punch(a.PeerID)) != nil {
		t.Fatal("send failed")
	}
	var received []byte
	if err := websocket.Message.Receive(ws, &received); err != nil || !bytes.Equal(received, reply) {
		t.Fatalf("rendezvous proxy: %v", err)
	}
	_ = ws.Close()
	relayRequest := func(id string) []byte {
		return pb(18, join(pb(1, []byte(a.PeerID)), pb(2, []byte(id)), pb(6, []byte(global.Config.Rustdesk.Key))))
	}
	if webFirstMessage(relayRequest("unrelated-relay-uuid"), a, true) {
		t.Fatal("unrelated relay accepted")
	}
	ws = dial("/webclient/ws/relay", "https://example.test")
	_ = websocket.Message.Send(ws, relayRequest(relayID))
	_ = websocket.Message.Send(ws, []byte("encrypted desktop frame"))
	if err := websocket.Message.Receive(ws, &received); err != nil || string(received) != "encrypted desktop frame" {
		t.Fatalf("relay proxy: %v", err)
	}
	// Uploads send receive/digest pairs back to back, then larger blocks. The
	// gateway must preserve each encrypted frame, its contents and its order.
	frames := make([][]byte, 100)
	for i := range frames {
		size := 37 + i%2*136
		if i%10 == 0 {
			size = 131110
		}
		frames[i] = bytes.Repeat([]byte{byte(i)}, size)
	}
	sendErr := make(chan error, 1)
	go func() {
		for _, frame := range frames {
			if err := websocket.Message.Send(ws, frame); err != nil {
				sendErr <- err
				return
			}
		}
		sendErr <- nil
	}()
	for i, frame := range frames {
		if err := websocket.Message.Receive(ws, &received); err != nil || !bytes.Equal(received, frame) {
			t.Fatalf("relay burst frame %d: %v", i, err)
		}
	}
	if err := <-sendErr; err != nil {
		t.Fatal(err)
	}
	_ = ws.Close()
	if webFirstMessage(relayRequest(relayID), a, true) {
		t.Fatal("relay replay accepted")
	}
	// Browsers split large encrypted messages into continuation frames. Each
	// complete message must reach RustDesk intact or its nonce/decryption fails.
	relayID = "isolated-fragmented-relay-uuid"
	if !webRendezvousResponse(pb(19, join(pb(2, []byte(relayID)), pb(5, signed))), a) {
		t.Fatal("fragmented relay authorization failed")
	}
	fragmentDialer := messagews.Dialer{WriteBufferSize: 64 * 1024}
	fragmented, _, err := fragmentDialer.Dial("ws"+strings.TrimPrefix(gateway.URL, "http")+"/webclient/ws/relay", http.Header{
		"Origin": {"https://example.test"}, "Cookie": {webCookie + "=" + cookie.Value},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fragmented.Close()
	_ = fragmented.SetReadDeadline(time.Now().Add(3 * time.Second))
	if err := fragmented.WriteMessage(messagews.BinaryMessage, relayRequest(relayID)); err != nil {
		t.Fatal(err)
	}
	large := bytes.Repeat([]byte{0xAC}, 128*1024+31)
	if err := fragmented.WriteMessage(messagews.BinaryMessage, large); err != nil {
		t.Fatal(err)
	}
	_, received, err = fragmented.ReadMessage()
	if err != nil || !bytes.Equal(received, large) {
		t.Fatalf("fragmented encrypted message split or corrupted: got %d bytes, want %d: %v", len(received), len(large), err)
	}
	_ = fragmented.Close()
	if w := request("/webclient/ws/id", "", "https://evil.test", cookie.Value); w.Code != 403 {
		t.Fatal("cross-origin websocket accepted")
	}
	db.Model(u).Update("status", model.COMMON_STATUS_DISABLED)
	if w := request("/webclient/session-config", "", "", cookie.Value); w.Code != 401 {
		t.Fatal("disabled issuer retained access")
	}
}
