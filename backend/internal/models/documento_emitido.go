package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TipoDocumentoEmitido identifica qual dos 3 geradores do Módulo 2
// ("Gerador Inteligente de Documentos Legais", Fase 2 do roadmap) produziu
// o registro.
type TipoDocumentoEmitido string

const (
	TipoDocumentoEmitidoNotificacao TipoDocumentoEmitido = "NOTIFICACAO_DESCUMPRIMENTO"
	TipoDocumentoEmitidoAtesto      TipoDocumentoEmitido = "ATESTO"
	TipoDocumentoEmitidoAditivo     TipoDocumentoEmitido = "MINUTA_ADITIVO"
)

// DocumentoEmitido registra cada PDF oficial gerado pelo Selene a partir de
// um contrato/processo — não é um upload do fiscal (isso é DocumentoAnexo),
// é um documento que o PRÓPRIO sistema produziu. Serve dois propósitos:
// (a) histórico consultável (quem gerou o quê, quando, com qual
// motivo/justificativa) — é a fonte do Dossiê do Fornecedor (Fase 4); (b)
// autenticidade verificável via CodigoVerificacao, embutido como QR code no
// PDF do Atesto.
type DocumentoEmitido struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	ContratoID uuid.UUID `gorm:"type:uuid;not null;index"`
	Contrato   *Contrato `gorm:"foreignKey:ContratoID;references:ID"`

	// ProcessoPagamentoID é opcional: Notificação de Descumprimento e
	// Minuta de Aditivo são emitidas no nível do contrato (não têm um mês
	// de referência específico); só o Atesto é sempre amarrado a um
	// processo. Nullable em vez de duas tabelas separadas — os 3 tipos
	// compartilham identidade/consulta o suficiente pra não justificar
	// isso.
	ProcessoPagamentoID *uuid.UUID         `gorm:"type:uuid;index"`
	ProcessoPagamento   *ProcessoPagamento `gorm:"foreignKey:ProcessoPagamentoID;references:ID"`

	Tipo TipoDocumentoEmitido `gorm:"type:varchar(30);not null;check:tipo IN ('NOTIFICACAO_DESCUMPRIMENTO','ATESTO','MINUTA_ADITIVO')"`

	// Motivo guarda o texto principal usado na geração: o motivo
	// selecionado (Notificação) ou a justificativa técnica (Minuta de
	// Aditivo). Vazio no Atesto, que não pede motivo.
	Motivo string `gorm:"type:text"`

	// DadosExtras é um snapshot JSON dos demais campos usados na geração
	// (ex: valor/prazo solicitados na Minuta de Aditivo) — texto livre em
	// vez de colunas por tipo: os 3 geradores têm formulários bem
	// diferentes entre si, e não vale a pena normalizar isso em tabelas
	// próprias só pra um registro de auditoria/histórico.
	DadosExtras string `gorm:"type:text"`

	GeradoPorID uuid.UUID `gorm:"type:uuid;not null;index"`
	GeradoPor   *User     `gorm:"foreignKey:GeradoPorID;references:ID"`

	// CodigoVerificacao é o valor embutido no QR code do Atesto — permite
	// confirmar autenticidade via GET /api/v1/verificar/:codigo (rota
	// pública, sem autenticação: quem escaneia o QR de um papel impresso
	// não tem login no Selene) sem expor o UUID interno do registro.
	CodigoVerificacao string `gorm:"type:varchar(40);not null;uniqueIndex"`

	CreatedAt time.Time
}

func (DocumentoEmitido) TableName() string {
	return "documentos_emitidos"
}

func (d *DocumentoEmitido) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
