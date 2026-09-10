package api

import (
	"testing"

	"github.com/slxar/rustdesk-api-goravel/v3/model"
)

func TestUserPayloadFromUserIncludesClientProfileFields(t *testing.T) {
	admin := true
	user := &model.User{
		Username: "alice",
		Nickname: "Alice Example",
		Avatar:   "/upload/alice.png",
		Remark:   "Support account",
		IsAdmin:  &admin,
	}

	payload := (&UserPayload{}).FromUser(user)
	if payload.Name != "alice" || payload.DisplayName != "Alice Example" ||
		payload.Avatar != "/upload/alice.png" || payload.Note != "Support account" {
		t.Fatalf("profile fields not mapped: %#v", payload)
	}
}
