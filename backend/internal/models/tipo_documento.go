package models

// TipoDocumento classifica um DocumentoAnexo (ex: "Nota Fiscal", "Boleto
// DAM", "CND Federal"). Tabela de referência semeada na implantação.
type TipoDocumento struct {
	ID int `gorm:"primaryKey"`

	Nome string `gorm:"type:varchar(100);not null;uniqueIndex"`

	// ExigeValidade marca os tipos que vencem (certidões — CND, FGTS
	// etc.), usado pelo Radar de Alertas (Fase 1 do roadmap) pra saber
	// quais documentos pedem data de validade no upload. Documentos que
	// não vencem (ex: Nota Fiscal) ficam false — sem data de validade.
	ExigeValidade bool `gorm:"not null;default:false"`
}

func (TipoDocumento) TableName() string {
	return "tipos_documento"
}
