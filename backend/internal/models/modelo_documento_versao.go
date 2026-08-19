package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ModeloDocumentoVersao é uma versão publicada do arquivo-modelo de uma
// categoria — mesma infra de storage local em disco de DocumentoAnexo/
// FotoVistoria (hash SHA-256, caminho nunca exposto pela API), mas SEM
// dedup por hash: aqui cada upload é semanticamente "publicar uma versão
// nova" (o admin decidiu substituir), mesmo que o conteúdo seja idêntico
// ao anterior — diferente de DocumentoAnexo.Upload, que rejeita reenvio
// do mesmo arquivo como duplicidade.
type ModeloDocumentoVersao struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	ModeloDocumentoID uuid.UUID `gorm:"type:uuid;not null;index"`

	NomeArquivo string `gorm:"type:varchar(255);not null"`

	// CaminhoStorage nunca é serializado (json:"-") — detalhe de
	// implementação do servidor, não da conta do client. Mesmo
	// tratamento de DocumentoAnexo.CaminhoStorage/FotoVistoria.CaminhoStorage.
	CaminhoStorage string `gorm:"type:text;not null" json:"-"`

	HashArquivo  string `gorm:"type:char(64);not null"`
	TamanhoBytes int64  `gorm:"not null"`

	EnviadoPorID uuid.UUID `gorm:"type:uuid;not null;index"`
	EnviadoPor   *User     `gorm:"foreignKey:EnviadoPorID;references:ID"`

	CreatedAt time.Time
}

func (ModeloDocumentoVersao) TableName() string {
	return "modelo_documento_versoes"
}

func (v *ModeloDocumentoVersao) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
