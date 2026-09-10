package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/slxar/rustdesk-api-goravel/v3/model"
)

var ErrIntegrationToken = errors.New("integration token is invalid, expired, revoked, or its administrator is disabled")

func integrationTokenHash(raw string) string {
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func integrationTokenIssuer(id uint) (*model.User, error) {
	var issuer model.User
	if id == 0 || DB.Where("id = ?", id).First(&issuer).Error != nil || issuer.IsAdmin == nil || !*issuer.IsAdmin || issuer.Status != model.COMMON_STATUS_ENABLE {
		return nil, ErrIntegrationToken
	}
	return &issuer, nil
}

func integrationTargets(value string) (string, error) {
	if len(value) > 8192 {
		return "", errors.New("target allowlist is too long")
	}
	targets := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
	if len(targets) == 1 && targets[0] == "*" {
		return "*", nil
	}
	if len(targets) == 0 {
		return "", errors.New("specify target IDs or explicitly allow all targets with *")
	}
	seen := make(map[string]bool)
	unique := make([]string, 0, len(targets))
	for _, target := range targets {
		if !webPeerID.MatchString(target) {
			return "", errors.New("target allowlist contains an invalid RustDesk ID")
		}
		if !seen[target] {
			unique = append(unique, target)
			seen[target] = true
		}
	}
	return strings.Join(unique, " "), nil
}

// CreateIntegrationToken returns the raw secret once; only its hash and a short
// display prefix are stored. days=0 selects the default 90-day expiry.
func CreateIntegrationToken(issuer *model.User, name, targets string, days int) (*model.IntegrationToken, string, error) {
	if issuer == nil {
		return nil, "", ErrIntegrationToken
	}
	if _, err := integrationTokenIssuer(issuer.Id); err != nil {
		return nil, "", err
	}
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return nil, "", errors.New("token name must contain 1 to 100 characters")
	}
	if days == 0 {
		days = 90
	}
	if days < 1 || days > 365 {
		return nil, "", errors.New("token expiry must be between 1 and 365 days")
	}
	allowlist, err := integrationTargets(targets)
	if err != nil {
		return nil, "", err
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, "", err
	}
	raw := "rdapi_" + base64.RawURLEncoding.EncodeToString(secret)
	token := &model.IntegrationToken{
		IssuerID: issuer.Id, Name: name, TokenHash: integrationTokenHash(raw), TokenPrefix: raw[:14],
		TargetAllowlist: allowlist, ExpiresAt: time.Now().Add(time.Duration(days) * 24 * time.Hour).Unix(),
	}
	if err := DB.Create(token).Error; err != nil {
		return nil, "", err
	}
	return token, raw, nil
}

// AuthenticateIntegrationToken is for the web-link issuance endpoint only.
// Call IntegrationTokenAllowsTarget for the requested target before issuing.
func AuthenticateIntegrationToken(raw string) (*model.IntegrationToken, *model.User, error) {
	if len(raw) != 49 || !strings.HasPrefix(raw, "rdapi_") {
		return nil, nil, ErrIntegrationToken
	}
	secret, err := base64.RawURLEncoding.DecodeString(raw[6:])
	if err != nil || len(secret) != 32 || base64.RawURLEncoding.EncodeToString(secret) != raw[6:] {
		return nil, nil, ErrIntegrationToken
	}
	now := time.Now().Unix()
	var token model.IntegrationToken
	if DB.Where("token_hash = ? AND revoked_at = 0 AND expires_at > ?", integrationTokenHash(raw), now).First(&token).Error != nil {
		return nil, nil, ErrIntegrationToken
	}
	issuer, err := integrationTokenIssuer(token.IssuerID)
	if err != nil {
		return nil, nil, err
	}
	// Recheck revocation/expiry at the write so a concurrent revocation cannot
	// authenticate after its update has completed.
	result := DB.Model(&model.IntegrationToken{}).Where("id = ? AND revoked_at = 0 AND expires_at > ?", token.Id, now).Update("last_used_at", now)
	if result.Error != nil {
		return nil, nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, nil, ErrIntegrationToken
	}
	token.LastUsedAt = now
	return &token, issuer, nil
}

func IntegrationTokenAllowsTarget(token *model.IntegrationToken, target string) bool {
	if token == nil || !webPeerID.MatchString(target) || token.RevokedAt != 0 || token.ExpiresAt <= time.Now().Unix() {
		return false
	}
	if token.TargetAllowlist == "*" {
		return true
	}
	for _, allowed := range strings.Fields(token.TargetAllowlist) {
		if allowed == target {
			return true
		}
	}
	return false
}

func ListIntegrationTokens() ([]model.IntegrationToken, error) {
	tokens := make([]model.IntegrationToken, 0)
	err := DB.Order("id DESC").Find(&tokens).Error
	return tokens, err
}

func RevokeIntegrationToken(id uint) error {
	if id == 0 {
		return ErrIntegrationToken
	}
	result := DB.Model(&model.IntegrationToken{}).Where("id = ?", id).Update("revoked_at", time.Now().Unix())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrIntegrationToken
	}
	return nil
}

// GetIntegrationToken returns metadata for capability revocation checks.
func GetIntegrationToken(id uint) (*model.IntegrationToken, error) {
	var token model.IntegrationToken
	if id == 0 {
		return nil, ErrIntegrationToken
	}
	if err := DB.First(&token, id).Error; err != nil {
		return nil, err
	}
	return &token, nil
}
