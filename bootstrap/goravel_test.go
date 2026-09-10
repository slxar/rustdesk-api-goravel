package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/slxar/rustdesk-api-goravel/v3/global"
	"github.com/slxar/rustdesk-api-goravel/v3/utils"
)

func TestBootProvidesGinRouteFacade(t *testing.T) {
	t.Setenv("SESSION_FILES", t.TempDir())
	previousLogger, previousLimiter := global.Logger, global.LoginLimiter
	global.Logger = logrus.New()
	global.LoginLimiter = utils.NewLoginLimiter(utils.SecurityPolicy{CaptchaThreshold: -1})
	t.Cleanup(func() {
		global.Logger, global.LoginLimiter = previousLogger, previousLimiter
	})
	router := Boot("127.0.0.1:0").MakeRoute()
	if router == nil {
		t.Fatal("Goravel Gin route facade is not configured")
	}

	request := httptest.NewRequest(http.MethodGet, "/_admin/login", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "RustDesk Admin") {
		t.Fatalf("admin login response = %d %q", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/_admin/", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusFound || response.Header().Get("Location") != "/_admin/login" {
		t.Fatalf("unauthenticated admin response = %d %q", response.Code, response.Header().Get("Location"))
	}

	request = httptest.NewRequest(http.MethodGet, "/_admin/login", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	cookie := response.Header().Get("Set-Cookie")
	for _, required := range []string{"rustdesk_admin_session=", "Path=/_admin", "HttpOnly", "Secure", "SameSite=Lax"} {
		if !strings.Contains(cookie, required) {
			t.Errorf("session cookie %q does not contain %q", cookie, required)
		}
	}
	for name, value := range map[string]string{
		"Content-Security-Policy": "default-src 'self'",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
	} {
		if !strings.Contains(response.Header().Get(name), value) {
			t.Errorf("%s = %q, want %q", name, response.Header().Get(name), value)
		}
	}
	csrfToken := response.Header().Get("X-CSRF-TOKEN")
	if csrfToken == "" {
		t.Fatal("admin login response did not issue a CSRF token")
	}
	form := url.Values{"_token": {csrfToken}}
	request = httptest.NewRequest(http.MethodPost, "https://example.test/_admin/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range response.Result().Cookies() {
		request.AddCookie(cookie)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Invalid sign-in details") {
		t.Fatalf("CSRF-authenticated form response = %d %q", response.Code, response.Body.String())
	}

	global.LoginLimiter = utils.NewLoginLimiter(utils.SecurityPolicy{CaptchaThreshold: 0})
	global.LoginLimiter.RegisterProvider(utils.B64StringCaptchaProvider{})
	request = httptest.NewRequest(http.MethodGet, "https://example.test/_admin/login", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), `name="captcha"`) || !strings.Contains(response.Body.String(), `data:image/png;base64,`) {
		t.Fatalf("admin login did not render the required captcha: %q", response.Body.String())
	}
	form = url.Values{
		"_token":   {response.Header().Get("X-CSRF-TOKEN")},
		"username": {"admin"},
		"password": {"wrong"},
	}
	request = httptest.NewRequest(http.MethodPost, "https://example.test/_admin/login", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range response.Result().Cookies() {
		request.AddCookie(cookie)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Invalid sign-in details") {
		t.Fatalf("captcha-protected login response = %d %q", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/public/vendor/htmx/htmx.min.js", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "htmx") {
		t.Fatalf("HTMX asset response = %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	for _, path := range []string{"/webclient/open", "/webclient/open.js"} {
		response = httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("web-client fallback %s returned %d: %.200s", path, response.Code, response.Body.String())
		}
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || response.Body.String() != "404 not found" {
		t.Fatalf("legacy fallback response = %d %q", response.Code, response.Body.String())
	}
	checkWebClientSocketOutlivesHTTPTimeout(t, router)
}
