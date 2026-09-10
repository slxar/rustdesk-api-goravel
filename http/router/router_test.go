package router

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/slxar/rustdesk-api-goravel/v3/global"
)

func TestWebInitDoesNotExposeLegacyAdminStaticTree(t *testing.T) {
	router := gin.New()
	WebInit(router)
	for _, route := range router.Routes() {
		if strings.HasPrefix(route.Path, "/_admin") {
			t.Fatalf("legacy administrator route is still public: %s %s", route.Method, route.Path)
		}
	}
}

func TestLegacyWebClientRoutesAreNeverRegistered(t *testing.T) {
	t.Chdir("../..")

	previous := global.Config.App.WebClient
	global.Config.App.WebClient = 1
	t.Cleanup(func() { global.Config.App.WebClient = previous })

	router := gin.New()
	WebInit(router)
	ApiInit(router)

	for _, route := range router.Routes() {
		if route.Path == "/api/shared-peer" ||
			route.Path == "/api/server-config" ||
			route.Path == "/api/server-config-v2" {
			t.Fatalf("legacy web-client route is still registered: %s %s", route.Method, route.Path)
		}
	}
	if !hasRoute(router, "POST", "/api/webclient/access-tokens") || !hasRoute(router, "GET", "/webclient/*path") {
		t.Fatal("authenticated web-client routes are missing")
	}

	if !hasRoute(router, "GET", "/api/version") {
		t.Fatal("native RustDesk API route was removed")
	}
}

func TestRustDesk149SupportedClientRoutesAreRegistered(t *testing.T) {
	t.Chdir("../..")

	router := gin.New()
	ApiInit(router)

	// RustDesk 1.4.9 client sources:
	// https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/models/user_model.dart
	// https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/models/group_model.dart
	// https://github.com/rustdesk/rustdesk/blob/1.4.9/flutter/lib/models/ab_model.dart
	// https://github.com/rustdesk/rustdesk/blob/1.4.9/src/hbbs_http/sync.rs
	// https://github.com/rustdesk/rustdesk/blob/1.4.9/src/hbbs_http/account.rs
	// https://github.com/rustdesk/rustdesk/blob/1.4.9/src/server/connection.rs
	required := []struct {
		method string
		path   string
	}{
		{"POST", "/api/heartbeat"},
		{"GET", "/api/login-options"},
		{"POST", "/api/login"},
		{"POST", "/api/oidc/auth"},
		{"GET", "/api/oidc/auth-query"},
		{"GET", "/api/oidc/callback"},
		{"POST", "/api/sysinfo"},
		{"POST", "/api/sysinfo_ver"},
		{"POST", "/api/audit/conn"},
		{"POST", "/api/audit/file"},
		{"POST", "/api/audit/alarm"},
		{"POST", "/api/currentUser"},
		{"POST", "/api/logout"},
		{"GET", "/api/users"},
		{"GET", "/api/peers"},
		{"GET", "/api/device-group/accessible"},
		{"GET", "/api/ab"},
		{"POST", "/api/ab"},
		{"POST", "/api/ab/personal"},
		{"POST", "/api/ab/settings"},
		{"POST", "/api/ab/shared/profiles"},
		{"POST", "/api/ab/peers"},
		{"POST", "/api/ab/tags/:guid"},
		{"POST", "/api/ab/peer/add/:guid"},
		{"DELETE", "/api/ab/peer/:guid"},
		{"PUT", "/api/ab/peer/update/:guid"},
		{"POST", "/api/ab/tag/add/:guid"},
		{"PUT", "/api/ab/tag/rename/:guid"},
		{"PUT", "/api/ab/tag/update/:guid"},
		{"DELETE", "/api/ab/tag/:guid"},
	}

	for _, route := range required {
		if !hasRoute(router, route.method, route.path) {
			t.Errorf("RustDesk 1.4.9 client route is missing: %s %s", route.method, route.path)
		}
	}
}

func hasRoute(router *gin.Engine, method, path string) bool {
	for _, route := range router.Routes() {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}
