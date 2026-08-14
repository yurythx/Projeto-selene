package models

// KanbanEtapa representa uma das 6 colunas fixas do funil de compliance
// documental (ex: "1. Elaborar OF", "2. Tramitar Planejamento / Contabilidade").
// É uma tabela de referência semeada uma única vez na implantação — não faz
// parte do fluxo de auditoria e por isso não carrega timestamps próprios.
type KanbanEtapa struct {
	ID int `gorm:"primaryKey"`

	Nome string `gorm:"type:varchar(100);not null"`

	// Posicao define a ordenação das colunas exibidas pelo frontend Next.js.
	Posicao int `gorm:"not null;uniqueIndex"`
}

func (KanbanEtapa) TableName() string {
	return "kanban_etapas"
}
