package models

import (
	"time"

	"github.com/google/uuid"
)

// Notificacao é um alerta de prazo/vencimento (Radar) entregue a um
// usuário específico pelo canal in-app — ver a migration 000014 pro
// raciocínio completo de deduplicação (chave_alerta). Tipo/Nivel usam os
// mesmos valores de string de service.TipoAlerta/NivelAlerta (não
// reimportados aqui pra evitar acoplar models a service — este pacote é
// a camada mais baixa da aplicação).
type Notificacao struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	UsuarioID uuid.UUID `gorm:"column:usuario_id;type:uuid;not null;index"`
	Usuario   *User     `gorm:"foreignKey:UsuarioID;references:ID"`

	Tipo  string `gorm:"column:tipo;type:varchar(30);not null"`
	Nivel string `gorm:"column:nivel;type:varchar(20);not null"`

	ContratoID uuid.UUID `gorm:"column:contrato_id;type:uuid;not null"`
	Contrato   *Contrato `gorm:"foreignKey:ContratoID;references:ID"`

	// ProcessoID é nulo pra alertas de tipo "vigencia_contrato" (não têm
	// um processo associado, só o contrato).
	ProcessoID *uuid.UUID         `gorm:"column:processo_id;type:uuid"`
	Processo   *ProcessoPagamento `gorm:"foreignKey:ProcessoID;references:ID"`

	Mensagem string `gorm:"column:mensagem;type:text;not null"`

	// ChaveAlerta nunca é exposta na API — é só o identificador interno
	// de deduplicação (ver a migration).
	ChaveAlerta string `gorm:"column:chave_alerta;type:text;not null" json:"-"`

	Lida bool `gorm:"column:lida;not null;default:false"`

	CriadaEm time.Time  `gorm:"column:criada_em;not null;default:now()"`
	LidaEm   *time.Time `gorm:"column:lida_em"`
}

func (Notificacao) TableName() string {
	return "notificacoes"
}
