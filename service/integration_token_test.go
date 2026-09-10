package service

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/slxar/rustdesk-api-goravel/v3/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestIntegrationTokensAreScopedHashedAndRevocable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "integration-token-test.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	previousDB := DB
	DB = db
	t.Cleanup(func() { DB = previousDB })
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	admin := true
	issuer := &model.User{Username: "integration-fixture-admin", IsAdmin: &admin, Status: model.COMMON_STATUS_ENABLE}
	if err := db.Create(issuer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.IntegrationToken{}); err != nil {
		t.Fatal(err)
	}
	var users int64
	if err := db.Model(&model.User{}).Count(&users).Error; err != nil || users != 1 {
		t.Fatalf("additive migration changed existing users: %d %v", users, err)
	}
	token, raw, err := CreateIntegrationToken(issuer, " ERP link issuer ", "123456789, device-one\n123456789", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 49 || !strings.HasPrefix(raw, "rdapi_") || token.Name != "ERP link issuer" || token.TargetAllowlist != "123456789 device-one" || token.TokenHash == raw || len(token.TokenHash) != 64 || token.TokenPrefix != raw[:14] {
		t.Fatal("invalid token format, metadata, or plaintext storage")
	}
	if delta := time.Until(time.Unix(token.ExpiresAt, 0)); delta < 89*24*time.Hour || delta > 90*24*time.Hour {
		t.Fatalf("default expiry is not 90 days: %v", delta)
	}
	loaded, err := GetIntegrationToken(token.Id)
	if err != nil || loaded.TokenHash != integrationTokenHash(raw) {
		t.Fatal("stored hash differs")
	}
	payload, _ := json.Marshal(loaded)
	if strings.Contains(string(payload), raw) || strings.Contains(string(payload), token.TokenHash) {
		t.Fatal("secret or hash exposed by JSON metadata")
	}
	authed, user, err := AuthenticateIntegrationToken(raw)
	if err != nil || user.Id != issuer.Id || authed.LastUsedAt == 0 {
		t.Fatalf("valid token rejected: %v", err)
	}
	if !IntegrationTokenAllowsTarget(authed, "123456789") || !IntegrationTokenAllowsTarget(authed, "device-one") || IntegrationTokenAllowsTarget(authed, "987654321") || IntegrationTokenAllowsTarget(authed, "123456") || IntegrationTokenAllowsTarget(authed, "123456789@external") {
		t.Fatal("target scope escaped exact ID allowlist")
	}
	for _, invalid := range []string{"", "Bearer " + raw, raw + "x", "rdapi_" + strings.Repeat("!", 43), "rdapi_" + strings.Repeat("A", 43)} {
		if _, _, err := AuthenticateIntegrationToken(invalid); err == nil {
			t.Fatal("accepted invalid integration secret")
		}
	}
	for _, tc := range []struct {
		name, targets string
		days          int
	}{{"", "*", 90}, {"test", "", 90}, {"test", "*,123456789", 90}, {"test", "123456789@other", 90}, {"test", "*", -1}, {"test", "*", 366}} {
		if _, _, err := CreateIntegrationToken(issuer, tc.name, tc.targets, tc.days); err == nil {
			t.Fatalf("accepted invalid create arguments: %+v", tc)
		}
	}
	for _, state := range []map[string]any{{"status": model.COMMON_STATUS_DISABLED}, {"status": model.COMMON_STATUS_ENABLE, "is_admin": false}} {
		if err := db.Model(issuer).Updates(state).Error; err != nil {
			t.Fatal(err)
		}
		if _, _, err := AuthenticateIntegrationToken(raw); err == nil {
			t.Fatal("disabled/demoted issuer authenticated")
		}
		if _, _, err := CreateIntegrationToken(issuer, "forbidden", "*", 1); err == nil {
			t.Fatal("disabled/demoted issuer created token")
		}
	}
	if err := db.Model(issuer).Updates(map[string]any{"is_admin": true, "status": model.COMMON_STATUS_ENABLE}).Error; err != nil {
		t.Fatal(err)
	}
	wildcard, wildcardRaw, err := CreateIntegrationToken(issuer, "explicit wildcard", "*", 1)
	if err != nil || !IntegrationTokenAllowsTarget(wildcard, "device-two") || IntegrationTokenAllowsTarget(wildcard, "../device-two") {
		t.Fatal("explicit wildcard scope broken")
	}
	if err := db.Model(wildcard).Update("expires_at", time.Now().Add(-time.Second).Unix()).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := AuthenticateIntegrationToken(wildcardRaw); err == nil {
		t.Fatal("expired token authenticated")
	}
	if err := RevokeIntegrationToken(token.Id); err != nil {
		t.Fatal(err)
	}
	if _, _, err := AuthenticateIntegrationToken(raw); err == nil {
		t.Fatal("revoked token authenticated")
	}
	loaded, err = GetIntegrationToken(token.Id)
	if err != nil || loaded.RevokedAt == 0 || IntegrationTokenAllowsTarget(loaded, "123456789") {
		t.Fatal("revoked metadata still authorizes target")
	}
	list, err := ListIntegrationTokens()
	if err != nil || len(list) != 2 || list[0].Id != wildcard.Id {
		t.Fatal("list metadata missing or incorrectly ordered")
	}
}
