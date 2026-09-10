package web

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	frameworkpath "github.com/goravel/framework/support/path"
	"github.com/gorilla/websocket"
	"github.com/slxar/rustdesk-api-goravel/v3/global"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
	"google.golang.org/protobuf/encoding/protowire"
)

const webCookie = "rustdesk_web_access"

func WebClient(c *gin.Context) {
	w, r := c.Writer, c.Request
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	// zstddec initializes its bundled WASM using fetch(data:); this permits the
	// embedded decoder without authorizing any additional network origin.
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self'; img-src 'self' data: blob:; connect-src 'self' data:; worker-src 'self' blob:; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
	p := strings.TrimPrefix(r.URL.Path, "/webclient/")
	// Corresponding source remains available after a viewer's capability expires.
	if p == "source.tar.gz" && (r.Method == "GET" || r.Method == "HEAD") {
		http.ServeFile(w, r, frameworkpath.Resource("webclient", "source.tar.gz"))
		return
	}
	if p == "open" && r.Method == "GET" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(webOpenPage))
		return
	}
	if p == "open.js" && r.Method == "GET" {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write([]byte(webOpenScript))
		return
	}
	if p == "redeem" && r.Method == "POST" {
		if !webSameOrigin(r) || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			c.AbortWithStatus(403)
			return
		}
		var input struct {
			Token string `json:"token"`
		}
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&input) != nil {
			c.AbortWithStatus(400)
			return
		}
		cookie, a, err := service.WebAccesses.Redeem(input.Token)
		if err != nil || !webIssuerEnabled(a) {
			c.JSON(403, gin.H{"error": "This link is invalid, expired, or already used."})
			return
		}
		http.SetCookie(w, &http.Cookie{Name: webCookie, Value: cookie, Path: "/webclient", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: 3600})
		c.JSON(200, gin.H{"url": "/webclient/"})
		return
	}
	cookie, err := r.Cookie(webCookie)
	var a *service.WebAccess
	if err == nil {
		a = service.WebAccesses.Session(cookie.Value)
	}
	if a == nil || !webIssuerEnabled(a) {
		if p == "" && r.Method == "GET" {
			c.Redirect(302, "/_admin/webclient")
		} else {
			c.AbortWithStatus(401)
		}
		return
	}
	if p == "logout" && r.Method == "POST" {
		if !webSameOrigin(r) {
			c.AbortWithStatus(403)
			return
		}
		service.WebAccesses.Revoke(cookie.Value)
		http.SetCookie(w, &http.Cookie{Name: webCookie, Path: "/webclient", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
		c.Status(204)
		return
	}
	if r.Method != "GET" {
		c.AbortWithStatus(405)
		return
	}
	if p == "session-config" {
		c.JSON(200, gin.H{"id": a.PeerID, "key": strings.TrimSpace(global.Config.Rustdesk.Key), "id_ws": "/webclient/ws/id", "relay_ws": "/webclient/ws/relay", "expires_at": a.Expires.UTC().Format(time.RFC3339)})
		return
	}
	if p == "ws/id" || p == "ws/relay" {
		if !webSameOrigin(r) || !a.Acquire() {
			c.AbortWithStatus(403)
			return
		}
		defer a.Release()
		webProxy(w, r, a, cookie.Value, p == "ws/relay")
		return
	}
	if p == "" {
		p = "index.html"
	}
	// No directory listing, SPA fallback, or files outside the protected asset root.
	if strings.Contains(p, "\\") || filepath.IsAbs(p) || filepath.Clean(p) != p || strings.HasPrefix(p, ".") {
		c.AbortWithStatus(404)
		return
	}
	file := frameworkpath.Resource("webclient", p)
	info, err := os.Stat(file)
	if err != nil || !info.Mode().IsRegular() {
		c.AbortWithStatus(404)
		return
	}
	http.ServeFile(w, r, file)
}

func webIssuerEnabled(a *service.WebAccess) bool {
	return service.WebAccessValid(a)
}

func webSameOrigin(r *http.Request) bool {
	origin := service.WebAccessOrigin()
	return origin != "" && r.Header.Get("Origin") == origin
}

func webProxy(w http.ResponseWriter, r *http.Request, a *service.WebAccess, cookie string, relay bool) {
	endpoint := os.Getenv("RUSTDESK_WEBCLIENT_ID_WS")
	if endpoint == "" {
		endpoint = "ws://hbbs:21118"
	}
	if relay {
		endpoint = os.Getenv("RUSTDESK_WEBCLIENT_RELAY_WS")
		if endpoint == "" {
			endpoint = "ws://hbbr:21119"
		}
	}
	upgrader := websocket.Upgrader{CheckOrigin: webSameOrigin, HandshakeTimeout: 10 * time.Second}
	client, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer client.Close()
	client.SetReadLimit(64 * 1024)
	_ = client.SetReadDeadline(time.Now().Add(10 * time.Second))
	kind, first, err := client.ReadMessage()
	if err != nil || kind != websocket.BinaryMessage || !webFirstMessage(first, a, relay) {
		return
	}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	upstream, response, err := dialer.DialContext(r.Context(), endpoint, http.Header{"Origin": {service.WebAccessOrigin()}})
	if err != nil {
		if response != nil {
			_ = response.Body.Close()
		}
		return
	}
	defer upstream.Close()
	upstream.SetReadLimit(32 * 1024 * 1024)
	client.SetReadLimit(32 * 1024 * 1024)
	_ = client.SetReadDeadline(a.Expires)
	_ = client.SetWriteDeadline(a.Expires)
	_ = upstream.SetReadDeadline(a.Expires)
	_ = upstream.SetWriteDeadline(a.Expires)
	if upstream.WriteMessage(websocket.BinaryMessage, first) != nil {
		return
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if service.WebAccesses.Session(cookie) != a || !webIssuerEnabled(a) {
					_ = client.Close()
					_ = upstream.Close()
					return
				}
			}
		}
	}()
	go func() {
		defer client.Close()
		defer upstream.Close()
		for {
			// ReadMessage reassembles WebSocket continuation frames. Forwarding
			// individual frames splits encrypted RustDesk messages and breaks nonces.
			kind, data, err := client.ReadMessage()
			if err != nil || kind != websocket.BinaryMessage {
				return
			}
			// The ID socket performs exactly one target lookup. Never forward
			// a second request, registration, key exchange, or alternate target.
			if !relay && len(data) != 0 {
				return
			}
			if upstream.WriteMessage(websocket.BinaryMessage, data) != nil {
				return
			}
		}
	}()
	for {
		kind, data, err := upstream.ReadMessage()
		if err != nil || kind != websocket.BinaryMessage {
			return
		}
		if !relay && len(data) > 0 && !webRendezvousResponse(data, a) {
			return
		}
		if client.WriteMessage(websocket.BinaryMessage, data) != nil {
			return
		}
	}
}

// Strict protobuf parsing prevents duplicate fields/oneof ambiguity at the ACL
// boundary. The upstream RustDesk decoder and our decision must see one message.
func webFields(data []byte, varints ...protowire.Number) (map[protowire.Number][]byte, error) {
	out := map[protowire.Number][]byte{}
	for len(data) > 0 {
		n, typ, k := protowire.ConsumeTag(data)
		if k < 0 || n < 1 {
			return nil, service.ErrWebAccess
		}
		data = data[k:]
		want := protowire.BytesType
		for _, field := range varints {
			if n == field {
				want = protowire.VarintType
			}
		}
		if typ != want {
			return nil, service.ErrWebAccess
		}
		if _, exists := out[n]; exists {
			return nil, service.ErrWebAccess
		}
		var value []byte
		switch typ {
		case protowire.BytesType:
			value, k = protowire.ConsumeBytes(data)
		case protowire.VarintType:
			var number uint64
			number, k = protowire.ConsumeVarint(data)
			value = protowire.AppendVarint(nil, number)
		default:
			return nil, service.ErrWebAccess
		}
		if k < 0 {
			return nil, service.ErrWebAccess
		}
		out[n] = value
		data = data[k:]
	}
	return out, nil
}

func webEnvelope(data []byte, expected protowire.Number) (map[protowire.Number][]byte, error) {
	f, err := webFields(data)
	if err != nil || len(f) != 1 || f[expected] == nil {
		return nil, service.ErrWebAccess
	}
	switch expected {
	case 8:
		return webFields(f[expected], 2, 4)
	case 11:
		return webFields(f[expected], 3, 5, 6)
	default:
		return webFields(f[expected])
	}
}

func webFirstMessage(data []byte, a *service.WebAccess, relay bool) bool {
	n := protowire.Number(8)
	if relay {
		n = 18
	}
	f, err := webEnvelope(data, n)
	if err != nil || string(f[1]) != a.PeerID {
		return false
	}
	if relay {
		for k := range f {
			if k != 1 && k != 2 && k != 6 {
				return false
			}
		}
		return string(f[6]) == strings.TrimSpace(global.Config.Rustdesk.Key) && a.ClaimRelay(string(f[2]))
	}
	for k := range f {
		if k < 1 || k > 4 {
			return false
		}
	}
	return string(f[3]) == strings.TrimSpace(global.Config.Rustdesk.Key) && (len(f[4]) == 0 || string(f[4]) == "\x00" || string(f[4]) == "\x01") && string(f[2]) == "\x02"
}

func webRendezvousResponse(data []byte, a *service.WebAccess) bool {
	f, err := webEnvelope(data, 19)
	if err != nil {
		_, err = webEnvelope(data, 11) // Offline/error response; no relay permission.
		return err == nil
	}
	if len(f[6]) > 0 {
		return true
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(global.Config.Rustdesk.Key))
	signed := f[5]
	if err != nil || len(key) != ed25519.PublicKeySize || len(signed) < ed25519.SignatureSize || !ed25519.Verify(key, signed[64:], signed[:64]) {
		return false
	}
	peer, err := webFields(signed[64:])
	return err == nil && string(peer[1]) == a.PeerID && len(peer[2]) == 32 && a.AllowRelay(string(f[2]))
}

const webOpenPage = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Open RustDesk access</title><link rel="stylesheet" href="/public/css/admin.css"><script src="/webclient/open.js" defer></script></head><body><main id="content"><h1>Open remote desktop</h1><p>This single-use link opens one RustDesk host. Its password or approval is still required.</p><button id="open">Use access link</button><p id="status" role="status"></p></main></body></html>`

const webOpenScript = `(() => { const token=location.hash.slice(1); history.replaceState(null,'','/webclient/open'); const button=document.getElementById('open'),status=document.getElementById('status'); if(!/^[A-Za-z0-9_-]{43}$/.test(token)){button.disabled=true;status.textContent='An access link is required.';return;} button.onclick=async()=>{button.disabled=true;status.textContent='Opening…';try{if('serviceWorker' in navigator){for(const registration of await navigator.serviceWorker.getRegistrations()){const scope=new URL(registration.scope);if(scope.origin===location.origin&&scope.pathname.startsWith('/webclient/'))await registration.unregister();}}const response=await fetch('/webclient/redeem',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({token})});if(!response.ok)throw new Error('This link is invalid, expired, or already used.');location.replace('/webclient/?session='+Date.now());}catch(error){status.textContent=error.message;}};})();`
