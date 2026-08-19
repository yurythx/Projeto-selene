package models

import (
	"time"

	"github.com/google/uuid"
)

// KeycloakConfig é a configuração do Keycloak editável em runtime pelo
// admin (Configurações → Keycloak/SSO) — linha única (singleton, ID
// fixo em 1). Ver a migration 000012 pro histórico da decisão.
type KeycloakConfig struct {
	ID int `gorm:"primaryKey"`

	ClientID string `gorm:"column:client_id;type:varchar(255);not null"`

	// ClientSecret nunca é serializado de volta pro cliente (json:"-") —
	// ver KeycloakConfigService.Buscar, que expõe só
	// TemSegredoConfigurado (bool) no DTO de leitura.
	ClientSecret string `gorm:"column:client_secret;type:text;not null" json:"-"`

	IssuerURL string  `gorm:"column:issuer_url;type:text;not null"`
	Audience  *string `gorm:"column:audience;type:varchar(255)"`

	UpdatedByID uuid.UUID `gorm:"column:updated_by_id;type:uuid;not null"`
	UpdatedBy   *User     `gorm:"foreignKey:UpdatedByID;references:ID"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;not null"`
}

func (KeycloakConfig) TableName() string {
	return "keycloak_config"
}
