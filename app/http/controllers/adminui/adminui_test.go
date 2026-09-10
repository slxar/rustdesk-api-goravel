package adminui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goravel/gin"
	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"github.com/slxar/rustdesk-api-goravel/v3/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAdminTemplatesUseCSRFAndNoBrowserTokenStorage(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	for _, name := range []string{"login.tmpl", "page.tmpl", filepath.Join("partials", "table.tmpl")} {
		contents, err := os.ReadFile(filepath.Join(root, "resources", "views", "admin", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		body := string(contents)
		if strings.Contains(body, "localStorage") || strings.Contains(body, "sessionStorage") {
			t.Fatalf("%s must not store credentials in browser storage", name)
		}
	}
	login, _ := os.ReadFile(filepath.Join(root, "resources", "views", "admin", "login.tmpl"))
	if !strings.Contains(string(login), `name="_token"`) {
		t.Fatal("login form must carry the CSRF token")
	}
	if _, err := os.Stat(filepath.Join(root, "public", "vendor", "htmx", "htmx.min.js")); err != nil {
		t.Fatalf("self-hosted HTMX is required: %v", err)
	}
}

func TestAdminTemplatesRender(t *testing.T) {
	renderer, err := gin.DefaultTemplate()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := renderer.Template.ExecuteTemplate(&output, "login.tmpl", map[string]any{"CsrfToken": "test"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "RustDesk Admin") {
		t.Fatal("login template did not render")
	}
	output.Reset()
	if err := renderer.Template.ExecuteTemplate(&output, "page.tmpl", pageData("Users", "/_admin/users", "<script>", []string{"Name"}, [][]string{{"<b>unsafe</b>"}}, 1, 25, 1)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "<b>unsafe</b>") || !strings.Contains(output.String(), "&lt;b&gt;unsafe&lt;/b&gt;") {
		t.Fatal("page template must escape list values")
	}
	if !strings.Contains(output.String(), `class="admin-nav"`) || !strings.Contains(output.String(), `href="/_admin/users" aria-current="page"`) || !strings.Contains(output.String(), `aria-label="Users table" tabindex="0"`) {
		t.Fatal("page template must expose current navigation and a keyboard-scrollable table")
	}
	for name, path := range map[string]string{"integration_tokens.tmpl": "/_admin/integration-tokens", "webclient.tmpl": "/_admin/webclient"} {
		output.Reset()
		if err := renderer.Template.ExecuteTemplate(&output, name, map[string]any{"Path": path, "CsrfToken": "csrf-test"}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(output.String(), `class="admin-nav"`) || !strings.Contains(output.String(), `href="`+path+`" aria-current="page"`) || !strings.Contains(output.String(), `name="_token" value="csrf-test"`) {
			t.Fatalf("%s must render the shared navigation, active page and protected forms", name)
		}
	}
}

func TestAdminTemplatesProvideAccessiblePaginationAndSafeOAuthStatus(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	page, err := os.ReadFile(filepath.Join(root, "resources", "views", "admin", "page.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	partial, err := os.ReadFile(filepath.Join(root, "resources", "views", "admin", "partials", "table.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	login, err := os.ReadFile(filepath.Join(root, "resources", "views", "admin", "login.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(page, []byte("page_size")) || !bytes.Contains(page, []byte("hx-indicator")) || !bytes.Contains(partial, []byte("Page {{.Page}}")) || !bytes.Contains(partial, []byte("hx-indicator")) {
		t.Fatal("list templates must provide bounded-page controls and an HTMX loading indicator")
	}
	if !bytes.Contains(login, []byte("OAuth providers are configured, but browser sign-in is unavailable.")) {
		t.Fatal("login must make OAuth's intentionally unavailable browser entry point explicit")
	}
}

func TestPageBoundsPaginationAndPreservesFilter(t *testing.T) {
	page, size := pagination("-9", "999")
	if page != 1 || size != 100 {
		t.Fatalf("pagination bounds = %d, %d", page, size)
	}
	data := pageData("Users", "/_admin/users", "x & y", []string{"ID"}, nil, 2, 25, 60)
	if data["PreviousURL"] != "/_admin/users?page=1&page_size=25&q=x+%26+y" || data["NextURL"] != "/_admin/users?page=3&page_size=25&q=x+%26+y" {
		t.Fatalf("pagination URLs = %#v", data)
	}
}

func TestOAuthProvidersAreSafeBeforeServicesInitialize(t *testing.T) {
	previous := service.AllService
	service.AllService = nil
	t.Cleanup(func() { service.AllService = previous })
	if providers := oauthProviders(); providers != nil {
		t.Fatalf("providers before service initialization = %#v", providers)
	}
}

func TestAdminAlarmListFiltersByPeer(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AuditAlarm{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AuditAlarm{PeerId: "peer-1", Uuid: "uuid", Type: 1, Info: "{}"}).Error; err != nil {
		t.Fatal(err)
	}
	previousDB, previousServices := service.DB, service.AllService
	service.DB, service.AllService = db, &service.Service{AuditService: &service.AuditService{}}
	t.Cleanup(func() { service.DB, service.AllService = previousDB, previousServices })

	title, _, rows, total, err := listPage("alarms", "peer-1", 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Alarm logs" || total != 1 || len(rows) != 1 || rows[0][1] != "peer-1" {
		t.Fatalf("alarm list = %q %d %#v", title, total, rows)
	}
}
