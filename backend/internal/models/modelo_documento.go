package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TipoGatilhoModelo identifica qual dos fluxos de geração já existentes
// (Módulo 2 — Gerador Inteligente de Documentos Legais, e Relatório de
// Pagamento) uma categoria de ModeloDocumento está associada a. Uma
// categoria sem gatilho (nil) é só biblioteca de referência — upload,
// consulta, substituição — sem nenhuma geração automática ligada a ela;
// "categoria livre" (o admin digita o nome) não dá ao sistema nenhum
// dado pra preencher um template sem um gatilho conhecido, então só os 4
// valores abaixo habilitam geração real via merge fields (ver
// internal/service/modelo_documento_render.go).
type TipoGatilhoModelo string

const (
	GatilhoNotificacaoDescumprimento TipoGatilhoModelo = "NOTIFICACAO_DESCUMPRIMENTO"
	GatilhoMinutaAditivo             TipoGatilhoModelo = "MINUTA_ADITIVO"
	GatilhoAtesto                    TipoGatilhoModelo = "ATESTO"
	GatilhoRelatorioPagamento        TipoGatilhoModelo = "RELATORIO_PAGAMENTO"
)

// ModeloDocumento é uma categoria de arquivo-modelo (.docx) cadastrada em
// Configurações — ex: "Ofício", "Relatório Quadrimestral", "OF/Pré-Empenho".
// O arquivo em si (e seu histórico de versões) fica em
// ModeloDocumentoVersao; este struct guarda só a identidade da categoria
// e qual versão está ativa agora.
type ModeloDocumento struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	Categoria string `gorm:"type:varchar(150);not null"`

	// Gatilho é nullable — ver o comentário em TipoGatilhoModelo. O
	// índice único parcial (só quando não-nulo, ver a migration) garante
	// que no máximo uma categoria reivindica cada gatilho.
	Gatilho *TipoGatilhoModelo `gorm:"type:varchar(40);check:gatilho IS NULL OR gatilho IN ('NOTIFICACAO_DESCUMPRIMENTO','MINUTA_ADITIVO','ATESTO','RELATORIO_PAGAMENTO')"`

	VersaoAtivaID *uuid.UUID             `gorm:"type:uuid"`
	VersaoAtiva   *ModeloDocumentoVersao `gorm:"foreignKey:VersaoAtivaID;references:ID"`

	// Versoes é o histórico completo (mais recente primeiro, ver
	// ModeloDocumentoRepository.FindByID) — nunca apagado quando uma
	// versão nova é publicada, mesmo espírito auditável de
	// PortariaDesignacao/MovimentacaoEmpenho.
	Versoes []ModeloDocumentoVersao `gorm:"foreignKey:ModeloDocumentoID;references:ID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ModeloDocumento) TableName() string {
	return "modelos_documento"
}

func (m *ModeloDocumento) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
