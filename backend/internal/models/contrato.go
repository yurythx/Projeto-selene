package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TipoObjeto classifica a natureza do contrato, o que determina qual
// checklist condicional (Coluna 5 do Kanban) se aplica ao processo de
// pagamento — ver internal/service (etapa futura) para as regras de
// validação por tipo.
type TipoObjeto string

const (
	TipoObjetoConsumo    TipoObjeto = "CONSUMO"
	TipoObjetoPermanente TipoObjeto = "PERMANENTE"
	TipoObjetoServico    TipoObjeto = "SERVICO"
)

// Contrato representa um contrato administrativo sob fiscalização de um
// Fiscal de Contratos. Não armazena saldo orçamentário — isso é
// responsabilidade exclusiva dos sistemas corporativos da prefeitura.
type Contrato struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	NumeroContrato   string    `gorm:"type:varchar(50);not null;uniqueIndex"` // ex: "125/2026"
	PortariaNomeacao string    `gorm:"type:varchar(255)"`
	DataAssinatura   time.Time `gorm:"type:date;not null"`

	// DataVigenciaFim alimenta o Radar de Alertas (Fase 1 do roadmap) —
	// "faltam N dias pro fim do contrato". Nullable de propósito:
	// contratos cadastrados antes desta coluna existir não têm esse dado,
	// e não faz sentido travar retroativamente algo que não existia. Sem
	// essa data, o contrato simplesmente não aparece no radar de vigência
	// (não é tratado como vencido nem como OK).
	DataVigenciaFim *time.Time `gorm:"type:date"`

	ContratadaNome string `gorm:"type:varchar(255);not null"`
	ContratadaCNPJ string `gorm:"type:varchar(18);not null;index"`

	// ContratadaEmail não está na lista original de campos da Seção 4.2 da
	// documentação de domínio, mas é indispensável para a Ação Assíncrona
	// da Etapa 3 ("envio de pacote digital... por e-mail para a empresa
	// contratada") — sem um endereço de destino, essa notificação não
	// pode ser implementada de verdade. Adicionado como campo opcional
	// para não quebrar contratos já cadastrados sem esse dado.
	ContratadaEmail string `gorm:"type:varchar(255)"`

	FiscalID uuid.UUID `gorm:"type:uuid;not null;index"`
	Fiscal   *User     `gorm:"foreignKey:FiscalID;references:ID"`

	TipoObjeto TipoObjeto `gorm:"type:varchar(20);not null;check:tipo_objeto IN ('CONSUMO','PERMANENTE','SERVICO')"`

	// Ativo permite "encerrar" um contrato sem apagar o registro nem seu
	// histórico de processos/documentos — contratos encerrados não podem
	// receber novos processos de pagamento, mas continuam consultáveis.
	//
	// ARMADILHA: por causa da tag `default:true`, o GORM OMITE esta
	// coluna do INSERT sempre que o valor Go for o zero-value (false),
	// deixando o Postgres aplicar o DEFAULT — ou seja, `Create(&Contrato{
	// ..., Ativo: false})` silenciosamente persiste Ativo=TRUE. Para
	// criar já encerrado (não deveria acontecer no fluxo normal, mas se
	// precisar), crie com Ativo=true e faça um Update logo em seguida —
	// Update não tem essa omissão. Ver ContratoService.Criar, que
	// documenta e evita isso setando Ativo:true explicitamente.
	Ativo bool `gorm:"not null;default:true"`

	// ExigeFiscalizacaoTerceirizacao é Camada 2 (regra do SGF, não da
	// norma): marca contratos de mão de obra terceirizada, sujeitos à
	// IN SCL Nº 04/2021 (que é mais estreita que TipoObjeto == SERVICO —
	// nem todo contrato de serviço é terceirização de mão de obra). Quando
	// true, o checklist da Etapa 5 passa a exigir também os documentos
	// mensais do Art.9º-XXXII (alíneas a/b.1/b.2/b.3) — ver
	// internal/service/checklist.go. Default false para não afetar
	// contratos já cadastrados.
	ExigeFiscalizacaoTerceirizacao bool `gorm:"not null;default:false"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Contrato) TableName() string {
	return "contratos"
}

func (c *Contrato) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
