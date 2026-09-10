package admin

import (
	"reflect"
	"strings"
	"testing"

	api "github.com/slxar/rustdesk-api-goravel/v3/http/request/api"
)

func TestUsernameValidationAllows64Characters(t *testing.T) {
	for _, field := range []reflect.StructField{reflect.TypeOf(UserForm{}).Field(1), reflect.TypeOf(RegisterForm{}).Field(0), reflect.TypeOf(api.LoginForm{}).Field(5)} {
		if !strings.Contains(field.Tag.Get("validate"), "lte=64") {
			t.Fatalf("%s validation is %q, want lte=64", field.Name, field.Tag.Get("validate"))
		}
	}
}
