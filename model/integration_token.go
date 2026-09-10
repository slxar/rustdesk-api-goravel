package model

// IntegrationToken may mint target-scoped browser links only. It is not a
// RustDesk account token and must never authorize general API/admin routes.
type IntegrationToken struct {
	IdModel
	IssuerID        uint   `json:"issuer_id" gorm:"not null;index"`
	Name            string `json:"name" gorm:"size:100;not null"`
	TokenHash       string `json:"-" gorm:"size:64;not null;uniqueIndex"`
	TokenPrefix     string `json:"token_prefix" gorm:"size:14;not null"`
	TargetAllowlist string `json:"target_allowlist" gorm:"type:text;not null"`
	ExpiresAt       int64  `json:"expires_at" gorm:"not null"`
	RevokedAt       int64  `json:"revoked_at" gorm:"not null;default:0"`
	LastUsedAt      int64  `json:"last_used_at" gorm:"not null;default:0"`
	TimeModel
}
