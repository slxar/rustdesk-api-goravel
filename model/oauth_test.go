package model

import (
	"encoding/json"
	"testing"
)

func TestOidcUserEmailVerifiedAcceptsBooleanAndString(t *testing.T) {
	for _, tc := range []struct {
		name string
		json string
		want bool
	}{
		{name: "boolean", json: `{"email_verified":true}`, want: true},
		{name: "string", json: `{"email_verified":"TRUE"}`, want: true},
		{name: "null", json: `{"email_verified":null}`},
		{name: "absent", json: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var user OidcUser
			if err := json.Unmarshal([]byte(tc.json), &user); err != nil {
				t.Fatal(err)
			}
			if user.VerifiedEmail != tc.want {
				t.Fatalf("VerifiedEmail = %v, want %v", user.VerifiedEmail, tc.want)
			}
		})
	}
}

func TestOidcUserEmailVerifiedRejectsInvalidString(t *testing.T) {
	var user OidcUser
	if err := json.Unmarshal([]byte(`{"email_verified":"not-a-bool"}`), &user); err == nil {
		t.Fatal("expected invalid email_verified to fail")
	}
}
