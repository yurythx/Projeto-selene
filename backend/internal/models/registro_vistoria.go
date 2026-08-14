package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RegistroVistoria é o Módulo 3 do roadmap: uma vistoria de campo
// (presencial) do fiscal a um processo de pagamento, com fotos e
// geolocalização — instrui o processo perante o TCE além do que os
// documentos anexados (DocumentoAnexo) já cobrem.
type RegistroVistoria struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	ProcessoPagamentoID uuid.UUID          `gorm:"type:uuid;not null;index"`
	ProcessoPagamento   *ProcessoPagamento `gorm:"foreignKey:ProcessoPagamentoID;references:ID"`

	FiscalID uuid.UUID `gorm:"type:uuid;not null;index"`
	Fiscal   *User     `gorm:"foreignKey:FiscalID;references:ID"`

	// DataHora é quando a vistoria efetivamente aconteceu — setada no
	// momento do registro (não editável depois), não a data de eventual
	// edição/complementação futura.
	DataHora time.Time `gorm:"not null"`

	// Latitude/Longitude são opcionais: capturados via
	// navigator.geolocation no browser, que o usuário pode negar (ou o
	// dispositivo pode não suportar) — a vistoria continua válida sem
	// coordenadas, mesmo princípio de graceful degradation de
	// Contrato.DataVigenciaFim (Fase 1).
	Latitude  *float64
	Longitude *float64

	Observacoes string `gorm:"type:text"`

	Fotos []FotoVistoria `gorm:"foreignKey:VistoriaID"`

	CreatedAt time.Time
}

func (RegistroVistoria) TableName() string {
	return "registros_vistoria"
}

func (r *RegistroVistoria) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
