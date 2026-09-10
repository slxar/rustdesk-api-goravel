package adminui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/goravel/gin"
)

func TestManagementMutationValidation(t *testing.T) {
	if err := validateManagementInput("user-delete", "0", "delete"); err == nil {
		t.Fatal("zero IDs must be rejected")
	}
	if err := validateManagementInput("user-delete", "1", ""); err == nil {
		t.Fatal("destructive actions must require confirmation")
	}
	if err := validateManagementInput("token-revoke", "1", "delete"); err != nil {
		t.Fatalf("confirmed token revocation should accept a positive ID: %v", err)
	}
	if err := validateManagementInput("token-revoke", "1", ""); err == nil {
		t.Fatal("token revocation must require confirmation")
	}
	if err := validateManagementInput("unknown", "1", ""); err == nil {
		t.Fatal("unknown actions must be rejected")
	}
}

func TestManagementRedirectResource(t *testing.T) {
	if got := managementResource("user-delete"); got != "users" {
		t.Fatalf("user redirect resource = %q", got)
	}
	if got := managementResource("token-revoke"); got != "tokens" {
		t.Fatalf("token redirect resource = %q", got)
	}
}

func TestManagementTemplateRendersProtectedForms(t *testing.T) {
	renderer, err := gin.DefaultTemplate()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		resource       string
		formCount      int
		destructiveURL string
	}{
		{"users", 4, "/_admin/manage/user-delete"},
		{"groups", 3, "/_admin/manage/group-delete"},
		{"tokens", 1, "/_admin/manage/token-revoke"},
		{"oauth", 3, "/_admin/manage/oauth-delete"},
	} {
		t.Run(tc.resource, func(t *testing.T) {
			var output bytes.Buffer
			if err := renderer.Template.ExecuteTemplate(&output, "management.tmpl", map[string]any{
				"Title": tc.resource, "Resource": tc.resource, "CsrfToken": "test", "Notice": "Saved.",
				"Columns": []string{"ID"}, "Items": [][]string{{"1"}},
			}); err != nil {
				t.Fatal(err)
			}
			body := output.String()
			if got := strings.Count(body, `name="_token" value="test"`); got != tc.formCount+1 {
				t.Fatalf("CSRF fields = %d, want %d operations plus sign-out: %s", got, tc.formCount+1, body)
			}
			if !strings.Contains(body, `action="`+tc.destructiveURL+`"`) || !strings.Contains(body, `name="confirm" type="checkbox" value="delete" required`) {
				t.Fatalf("destructive form is not explicitly confirmed: %s", body)
			}
			if !strings.Contains(body, "Saved.") || !strings.Contains(body, `<th scope="col">ID</th>`) {
				t.Fatalf("management notice or table heading is missing: %s", body)
			}
			if got := strings.Count(body, `<section class="operation`); got != tc.formCount {
				t.Fatalf("labelled management sections = %d, want %d: %s", got, tc.formCount, body)
			}
			if !strings.Contains(body, `<form method="post" action="/_admin/logout">`) || !strings.Contains(body, `RustDesk Admin`) {
				t.Fatalf("management page must retain the standard admin header and sign-out action: %s", body)
			}
		})
	}

	var users bytes.Buffer
	if err := renderer.Template.ExecuteTemplate(&users, "management.tmpl", map[string]any{"Title": "Users", "Resource": "users", "CsrfToken": "test"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(users.String(), `<option value="2">Disabled</option>`) {
		t.Fatal("disabled user option must use the model status value 2")
	}
	if got := strings.Count(users.String(), `name="password"`); got != 2 {
		t.Fatalf("user management renders %d password fields, want create and reset only", got)
	}
}
