package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PapelDesignacao classifica o papel exercido por um servidor designado
// para um contrato, conforme os agentes definidos nas duas Instruções
// Normativas de fiscalização de contrato:
//   - IN SCL 01/2019 (Versão II), Art.3º: Gestor do Contrato, Fiscal de
//     Contrato.
//   - IN SCL 04/2021, Art.3º-V: acrescenta o Fiscal Setorial, específico de
//     contratos de mão de obra terceirizada.
type PapelDesignacao string

const (
	PapelFiscal         PapelDesignacao = "FISCAL"
	PapelFiscalSuplente PapelDesignacao = "FISCAL_SUPLENTE"
	PapelGestor         PapelDesignacao = "GESTOR"
	// PapelFiscalSetorial — IN SCL 04/2021 Art.3º-V, Art.6º: agente
	// designado para atuar no local onde o trabalho é realizado,
	// auxiliando o Fiscal e o Gestor, atestando a prestação efetiva do
	// objeto. Só se aplica a contratos de mão de obra terceirizada.
	PapelFiscalSetorial PapelDesignacao = "FISCAL_SETORIAL"
)

// PortariaDesignacao é o histórico auditável e imutável de designação de
// servidores (fiscal, suplente, gestor, fiscal setorial) por contrato —
// fundamentado em IN01 Art.4º-I/Art.6º/Art.7º e IN04 Art.4º-I/Art.10/Art.11,
// que exigem que toda designação de fiscal seja formalizada por Portaria
// publicada no Diário Oficial, com dados do servidor e do contrato.
//
// Contrato.FiscalID NÃO é substituído por esta tabela — continua existindo
// e sendo a leitura rápida (denormalizada) do último registro papel=FISCAL
// ativo de um contrato, usada por todo o código que já depende dele hoje.
// PortariaDesignacao é quem passa a ser a fonte de verdade auditável: um
// contrato pode ter vários registros ao longo do tempo (troca de fiscal,
// suplente, gestor), nenhum é sobrescrito — uma substituição fecha o
// registro anterior via DataRevogacao e insere um novo, nunca dá UPDATE
// nos campos de identidade de um registro existente (mesmo espírito de
// KanbanLog: log é fato histórico, não estado mutável).
type PortariaDesignacao struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	ContratoID uuid.UUID `gorm:"type:uuid;not null;index"`
	Contrato   *Contrato `gorm:"foreignKey:ContratoID;references:ID"`

	ServidorID uuid.UUID `gorm:"type:uuid;not null;index"`
	Servidor   *User     `gorm:"foreignKey:ServidorID;references:ID"`

	Papel PapelDesignacao `gorm:"type:varchar(20);not null;check:papel IN ('FISCAL','FISCAL_SUPLENTE','GESTOR','FISCAL_SETORIAL')"`

	// NumeroPortaria e PublicadoDiorondon documentam a formalização exigida
	// pela norma (Portaria + publicação no Diário Oficial/Diorondon) — os
	// dois são opcionais porque designações retroativas (ver backfill da
	// migration 000007) nem sempre têm esse dado capturado historicamente.
	NumeroPortaria     string `gorm:"type:varchar(50)"`
	PublicadoDiorondon string `gorm:"type:varchar(50)"`

	DataDesignacao time.Time `gorm:"type:date;not null"`

	// DataRevogacao é preenchida quando este registro é substituído por
	// uma nova designação — o registro em si nunca é apagado nem tem seus
	// demais campos alterados; é o único campo desta struct que uma
	// operação de Update pode legitimamente tocar.
	DataRevogacao *time.Time `gorm:"type:date"`

	CriadoPorID uuid.UUID `gorm:"type:uuid;not null"`

	CreatedAt time.Time
}

func (PortariaDesignacao) TableName() string {
	return "portarias_designacao"
}

func (p *PortariaDesignacao) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
