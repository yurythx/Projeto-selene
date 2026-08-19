package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StatusProcesso indica se o card de medição mensal ainda está em
// tramitação pelo Kanban ou já foi finalizado (arquivado como "Pago").
type StatusProcesso string

const (
	StatusProcessoAtivo     StatusProcesso = "Ativo"
	StatusProcessoConcluido StatusProcesso = "Concluido"
)

// ProcessoPagamento é o "Card" do Kanban: o processo de medição/pagamento
// de um contrato referente a um mês específico, percorrendo as 6 etapas
// fixas definidas em KanbanEtapa.
type ProcessoPagamento struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	ContratoID uuid.UUID `gorm:"type:uuid;not null;index:idx_contrato_mes,unique"`
	Contrato   *Contrato `gorm:"foreignKey:ContratoID;references:ID"`

	// MesReferencia segue o formato "MM/AAAA" (ex: "07/2026"), conforme
	// exemplo dado na especificação do domínio.
	MesReferencia string `gorm:"type:varchar(7);not null;index:idx_contrato_mes,unique"`

	EtapaAtualID int          `gorm:"not null;index"`
	EtapaAtual   *KanbanEtapa `gorm:"foreignKey:EtapaAtualID;references:ID"`

	Status StatusProcesso `gorm:"type:varchar(20);not null;default:'Ativo';check:status IN ('Ativo','Concluido')"`

	// EmpenhoID liga este processo ao registro paralelo/informativo de
	// Empenho (ver empenho.go) quando a fatura do mês foi apropriada
	// contra ele — nullable de propósito: processos já existentes
	// continuam com NULL, e nem todo contrato precisa adotar o
	// acompanhamento de empenho no Selene (ver comentário em empenho.go
	// sobre este não ser a fonte de verdade orçamentária).
	EmpenhoID *uuid.UUID `gorm:"type:uuid"`
	Empenho   *Empenho   `gorm:"foreignKey:EmpenhoID;references:ID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ProcessoPagamento) TableName() string {
	return "processos_pagamento"
}

func (p *ProcessoPagamento) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
