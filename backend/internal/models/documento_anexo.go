package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DocumentoAnexo é um arquivo (PDF) anexado ao escopo de um
// ProcessoPagamento. Um documento anexado em uma etapa fica disponível e é
// validado automaticamente nas etapas seguintes, sem necessidade de
// re-upload pelo fiscal.
type DocumentoAnexo struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	ProcessoPagamentoID uuid.UUID          `gorm:"type:uuid;not null;index:idx_processo_hash,unique"`
	ProcessoPagamento   *ProcessoPagamento `gorm:"foreignKey:ProcessoPagamentoID;references:ID"`

	TipoDocumentoID int            `gorm:"not null;index"`
	TipoDocumento   *TipoDocumento `gorm:"foreignKey:TipoDocumentoID;references:ID"`

	NomeArquivo string `gorm:"type:varchar(255);not null"`

	// CaminhoStorage é o caminho local (ou chave S3) onde o arquivo está
	// gravado — detalhe de infraestrutura do servidor, não deve vazar
	// pela API (json:"-"): nenhum cliente precisa saber a estrutura de
	// diretórios do filesystem do backend.
	CaminhoStorage string `gorm:"type:text;not null" json:"-"`

	// HashArquivo é o SHA-256 (hex, 64 caracteres) do conteúdo do arquivo,
	// usado para evitar duplicidade de upload dentro do mesmo processo de
	// pagamento. O índice único é escopado por ProcessoPagamentoID (não
	// global) porque o mesmo arquivo físico pode legitimamente ser
	// reaproveitado em processos/contratos diferentes.
	HashArquivo string `gorm:"type:char(64);not null;index:idx_processo_hash,unique"`

	EnviadoPorID uuid.UUID `gorm:"type:uuid;not null;index"`
	EnviadoPor   *User     `gorm:"foreignKey:EnviadoPorID;references:ID"`

	DataUpload time.Time `gorm:"column:data_upload;autoCreateTime;not null"`

	// DataValidade só é preenchida quando TipoDocumento.ExigeValidade é
	// true (certidões) — alimenta o Radar de Alertas (Fase 1 do
	// roadmap), que avisa quando uma certidão anexada ao processo em
	// andamento está vencida ou vencendo. Nullable pros documentos que
	// não vencem (a maioria) e pros já anexados antes desta coluna
	// existir.
	DataValidade *time.Time `gorm:"type:date"`
}

func (DocumentoAnexo) TableName() string {
	return "documentos_anexos"
}

func (d *DocumentoAnexo) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
