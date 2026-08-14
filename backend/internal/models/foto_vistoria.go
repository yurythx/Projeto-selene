package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FotoVistoria é uma foto anexada a um RegistroVistoria (Módulo 3 do
// roadmap) — reaproveita a mesma infra de storage local em disco usada
// por DocumentoAnexo (ver internal/service/documento_service.go e
// internal/service/vistoria_service.go), generalizada pra um segundo tipo
// de anexo.
type FotoVistoria struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	VistoriaID uuid.UUID `gorm:"type:uuid;not null;index"`

	NomeArquivo string `gorm:"type:varchar(255);not null"`

	// CaminhoStorage nunca é serializado (json:"-") — é um detalhe de
	// implementação do servidor (path local em disco), não da conta do
	// client. Mesmo tratamento de DocumentoAnexo.CaminhoStorage.
	CaminhoStorage string `gorm:"type:text;not null" json:"-"`

	HashArquivo string `gorm:"type:char(64);not null"`

	CreatedAt time.Time
}

func (FotoVistoria) TableName() string {
	return "fotos_vistoria"
}

func (f *FotoVistoria) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
