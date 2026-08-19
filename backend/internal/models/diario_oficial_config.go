package models

import (
	"time"

	"github.com/google/uuid"
)

// DiarioOficialConfig é a configuração da integração com a API do Diário
// Oficial da cidade, editável em runtime pelo admin (Configurações →
// Diário Oficial) — linha única (singleton, ID fixo em 1), mesmo padrão
// de KeycloakConfig. Ver a migration 000013 pro histórico da decisão.
type DiarioOficialConfig struct {
	ID int `gorm:"primaryKey"`

	BaseURL string `gorm:"column:base_url;type:text;not null"`

	// APIKey nunca é serializada de volta pro cliente (json:"-") — ver
	// DiarioOficialService.Buscar, que expõe só TemChaveConfigurada
	// (bool) no DTO de leitura.
	APIKey string `gorm:"column:api_key;type:text;not null" json:"-"`

	UpdatedByID uuid.UUID `gorm:"column:updated_by_id;type:uuid;not null"`
	UpdatedBy   *User     `gorm:"foreignKey:UpdatedByID;references:ID"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;not null"`
}

func (DiarioOficialConfig) TableName() string {
	return "diario_oficial_config"
}
