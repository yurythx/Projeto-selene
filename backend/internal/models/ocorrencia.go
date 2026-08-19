package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EstadoOcorrencia é o ciclo de vida de uma Ocorrencia. Fundamentado em
// IN SCL 01/2019 Art.3º-III/Art.5º-IV,IX e IN SCL 04/2021
// Art.3º-VIII/Art.5º-VIII,XVI-XVII, que exigem registro formal de
// ocorrências e sua comunicação ao Gestor até a regularização.
//
// Esta é uma SIMPLIFICAÇÃO de Camada 2 do fluxo real da norma: a IN04
// Art.5º-XVII prevê um ramo de escalonamento (se a empresa não responde à
// notificação, o fiscal comunica de novo ao Gestor com cópia à Unidade
// Administrativa de Fiscalização, que emite Parecer em 3 dias úteis) que
// não está modelado aqui como um estado próprio — fica como lacuna
// conhecida para uma fase futura (ver plano em
// .claude/plans/projeto-selene-rippling-kite.md, seção "Lacunas
// conhecidas"), não uma omissão silenciosa.
type EstadoOcorrencia string

const (
	OcorrenciaRegistrada   EstadoOcorrencia = "REGISTRADA"
	OcorrenciaNotificada   EstadoOcorrencia = "NOTIFICADA"
	OcorrenciaEmTratamento EstadoOcorrencia = "EM_TRATAMENTO"
	OcorrenciaRegularizada EstadoOcorrencia = "REGULARIZADA"
)

// Ocorrencia é o registro formal de uma ocorrência relacionada à execução
// de um contrato — a norma fala de ocorrências "relacionadas com a
// execução do contrato" de forma geral, não amarradas a um ciclo mensal
// específico, por isso ContratoID é obrigatório e ProcessoPagamentoID é
// opcional (só preenchido quando a ocorrência está claramente associada a
// um mês de referência específico).
type Ocorrencia struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	ContratoID uuid.UUID `gorm:"type:uuid;not null;index"`
	Contrato   *Contrato `gorm:"foreignKey:ContratoID;references:ID"`

	ProcessoPagamentoID *uuid.UUID         `gorm:"type:uuid;index"`
	ProcessoPagamento   *ProcessoPagamento `gorm:"foreignKey:ProcessoPagamentoID;references:ID"`

	Descricao string `gorm:"type:text;not null"`

	Estado EstadoOcorrencia `gorm:"type:varchar(20);not null;default:'REGISTRADA';check:estado IN ('REGISTRADA','NOTIFICADA','EM_TRATAMENTO','REGULARIZADA')"`

	RegistradoPorID uuid.UUID `gorm:"type:uuid;not null"`
	RegistradoPor   *User     `gorm:"foreignKey:RegistradoPorID;references:ID"`

	// DataNotificacaoGestor/DataRegularizacao ficam nil até a ocorrência
	// avançar para o estado correspondente — não são setadas na criação.
	DataNotificacaoGestor *time.Time `gorm:"type:date"`
	DataRegularizacao     *time.Time `gorm:"type:date"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Ocorrencia) TableName() string {
	return "ocorrencias"
}

func (o *Ocorrencia) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}
