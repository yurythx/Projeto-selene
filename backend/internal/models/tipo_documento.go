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

	// RestritoTipoObjeto, quando preenchido, restringe este tipo de
	// documento a contratos daquele TipoObjeto exato (ex: "Planilha de
	// Medição de Serviços" só faz sentido em contratos SERVICO). nil =
	// aplicável a qualquer tipo de contrato. Consumido por
	// service.TipoDocumentoAplicavel — mesma fonte de verdade usada tanto
	// pra bloquear o upload (DocumentoService.Upload) quanto pra filtrar o
	// select do frontend (GET /kanban/tipos-documento).
	RestritoTipoObjeto *TipoObjeto `gorm:"type:varchar(20)"`

	// RestritoTerceirizacao marca os documentos mensais do Art.9º-XXXII
	// (IN SCL 04/2021) que só se aplicam a contratos com
	// Contrato.ExigeFiscalizacaoTerceirizacao=true — ver o comentário em
	// checklist.go sobre essa flag ser mais estreita que TipoObjeto=SERVICO.
	RestritoTerceirizacao bool `gorm:"not null;default:false"`
}

func (TipoDocumento) TableName() string {
	return "tipos_documento"
}
