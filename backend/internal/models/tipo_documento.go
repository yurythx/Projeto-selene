package models

// TipoDocumento classifica um DocumentoAnexo (ex: "Nota Fiscal", "Boleto
// DAM", "CND Federal"). Tabela de referência semeada na implantação.
type TipoDocumento struct {
	ID int `gorm:"primaryKey"`

	Nome string `gorm:"type:varchar(100);not null;uniqueIndex"`
}

func (TipoDocumento) TableName() string {
	return "tipos_documento"
}
