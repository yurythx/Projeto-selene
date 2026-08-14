package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// KanbanLog registra toda transição bem-sucedida de etapa de um
// ProcessoPagamento, mapeando quem moveu, de onde, para onde e quando —
// exigência de auditoria do domínio Selene. É um registro imutável
// (insert-only), por isso carrega apenas MovidoEm, sem UpdatedAt.
type KanbanLog struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	ProcessoPagamentoID uuid.UUID          `gorm:"type:uuid;not null;index"`
	ProcessoPagamento   *ProcessoPagamento `gorm:"foreignKey:ProcessoPagamentoID;references:ID"`

	UsuarioID uuid.UUID `gorm:"type:uuid;not null;index"`
	Usuario   *User     `gorm:"foreignKey:UsuarioID;references:ID"`

	// EtapaOrigemID é nulo para o primeiro log de um processo (entrada
	// inicial no Kanban não tem etapa de origem).
	EtapaOrigemID  *int         `gorm:"index"`
	EtapaOrigem    *KanbanEtapa `gorm:"foreignKey:EtapaOrigemID;references:ID"`
	EtapaDestinoID int          `gorm:"not null;index"`
	EtapaDestino   *KanbanEtapa `gorm:"foreignKey:EtapaDestinoID;references:ID"`

	MovidoEm time.Time `gorm:"column:movido_em;autoCreateTime;not null"`
}

func (KanbanLog) TableName() string {
	return "kanban_logs"
}

func (k *KanbanLog) BeforeCreate(tx *gorm.DB) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}
	return nil
}
