package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Empenho é um registro PARALELO e INFORMATIVO do empenho orçamentário de
// um contrato — existe para dar suporte à obrigação do FISCAL (não do
// sistema financeiro) de "controlar o saldo do empenho em função do valor
// da fatura, de modo a possibilitar reforço de novos valores ou anulações
// parciais" (IN SCL 01/2019 Art.5º-VIII; IN SCL 04/2021 Art.5º-XXII).
//
// IMPORTANTE — isto NÃO é fonte de verdade orçamentária. Contrato já
// documenta explicitamente que "não armazena saldo orçamentário — isso é
// responsabilidade exclusiva dos sistemas corporativos da prefeitura", e
// essa decisão de arquitetura continua valendo: o valor/saldo aqui é
// alimentado manualmente pelo fiscal (a partir da Nota de Empenho e de
// reforços/anulações que ele recebe do sistema financeiro oficial), como
// apoio de acompanhamento dentro do Kanban — nunca como o número
// oficial/legal do saldo. Em caso de divergência, o sistema financeiro
// corporativo da prefeitura prevalece.
type Empenho struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	ContratoID uuid.UUID `gorm:"type:uuid;not null;index"`
	Contrato   *Contrato `gorm:"foreignKey:ContratoID;references:ID"`

	NumeroEmpenho string    `gorm:"type:varchar(50);not null"`
	DataEmissao   time.Time `gorm:"type:date;not null"`

	// ValorInicial em centavos (int64), não float64 — evita erro de
	// arredondamento em soma/subtração de valores monetários. Convenção
	// nova neste domínio: nenhum outro model do Selene armazena valor
	// monetário hoje (Contrato deliberadamente não guarda valor de
	// contrato/saldo, ver comentário acima).
	ValorInicial int64 `gorm:"not null"`

	CreatedAt time.Time
}

func (Empenho) TableName() string {
	return "empenhos"
}

func (e *Empenho) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// TipoMovimentacaoEmpenho classifica cada lançamento no histórico de um
// Empenho. INICIAL/REFORCO/ANULACAO correspondem diretamente à letra da
// norma (IN01 Art.5º-VIII, IN04 Art.5º-XXII: "reforço de novos valores ou
// anulações parciais"); FATURA_APROPRIADA é a mecânica de débito que
// operacionaliza "controlar o saldo em função do valor da fatura" — não é
// um termo usado literalmente pela norma, é uma decisão de modelagem do
// SGF (Camada 2) para tornar o saldo reconstruível a partir do histórico.
type TipoMovimentacaoEmpenho string

const (
	MovimentacaoInicial          TipoMovimentacaoEmpenho = "INICIAL"
	MovimentacaoReforco          TipoMovimentacaoEmpenho = "REFORCO"
	MovimentacaoAnulacao         TipoMovimentacaoEmpenho = "ANULACAO"
	MovimentacaoFaturaApropriada TipoMovimentacaoEmpenho = "FATURA_APROPRIADA"
)

// MovimentacaoEmpenho é um lançamento imutável no histórico de um Empenho
// — o saldo nunca é armazenado denormalizado num campo próprio; é sempre
// reconstruído somando/subtraindo os valores desta tabela (INICIAL e
// REFORCO somam, ANULACAO e FATURA_APROPRIADA subtraem), evitando que o
// saldo divirja do histórico que o sustenta.
type MovimentacaoEmpenho struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	EmpenhoID uuid.UUID `gorm:"type:uuid;not null;index"`
	Empenho   *Empenho  `gorm:"foreignKey:EmpenhoID;references:ID"`

	Tipo TipoMovimentacaoEmpenho `gorm:"type:varchar(20);not null;check:tipo IN ('INICIAL','REFORCO','ANULACAO','FATURA_APROPRIADA')"`

	// Valor é sempre positivo em centavos — o sinal (soma ou subtração do
	// saldo) é implícito pelo Tipo, nunca codificado aqui.
	Valor int64 `gorm:"not null"`

	// ProcessoPagamentoID só é preenchido quando Tipo == FATURA_APROPRIADA
	// — liga o débito ao ciclo mensal (ProcessoPagamento) que o originou.
	ProcessoPagamentoID *uuid.UUID         `gorm:"type:uuid;index"`
	ProcessoPagamento   *ProcessoPagamento `gorm:"foreignKey:ProcessoPagamentoID;references:ID"`

	Observacao string `gorm:"type:text"`

	RegistradoPorID uuid.UUID `gorm:"type:uuid;not null"`

	CreatedAt time.Time
}

func (MovimentacaoEmpenho) TableName() string {
	return "movimentacoes_empenho"
}

func (m *MovimentacaoEmpenho) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
