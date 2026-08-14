package database

import (
	"fmt"

	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// etapasSeed define as 6 colunas fixas e lineares do Kanban de compliance,
// conforme a Seção 5 da documentação de domínio. Os IDs são explícitos
// (não deixados para o auto-increment) porque a engine de checklist em
// internal/service/checklist.go referencia essas etapas por ID.
var etapasSeed = []models.KanbanEtapa{
	{ID: 1, Nome: "Elaborar OF / Pré-Empenho", Posicao: 1},
	{ID: 2, Nome: "Tramitar Planejamento / Contabilidade", Posicao: 2},
	{ID: 3, Nome: "Emitir OS / Envio à Empresa", Posicao: 3},
	{ID: 4, Nome: "Execução e Recepção", Posicao: 4},
	{ID: 5, Nome: "Relatório de Pagamento", Posicao: 5},
	{ID: 6, Nome: "Contabilidade (Liquidação e Pagamento)", Posicao: 6},
}

// tiposDocumentoSeed é a união de todos os documentos citados na Seção 5 da
// documentação de domínio, obrigatórios em algum checklist de transição.
// Os nomes aqui precisam bater exatamente com os usados em
// internal/service/checklist.go, que resolve os requisitos por nome.
var tiposDocumentoSeed = []string{
	"Ordem de Fornecimento (OF)",
	"Pré-Empenho",
	"Ofício de Solicitação",
	"Nota de Empenho",
	"Nota Fiscal / Fatura",
	"Ordem de Recepção",
	"Extrato do Empenho",
	"Declaração do Simples Nacional",
	"CND Trabalhista",
	"CND FGTS",
	"CND Municipal",
	"CND Estadual",
	"CND Federal",
	"CND INSS",
	"Relatório de Pagamento Assinado",
	"Planilha de Medição de Serviços",
	"Boleto DAM",
}

// Seed popula as tabelas de referência (kanban_etapas, tipos_documento) de
// forma idempotente — seguro de rodar em todo boot da aplicação, já que
// usa FirstOrCreate: registros já existentes não são duplicados nem
// sobrescritos.
func Seed(db *gorm.DB) error {
	for _, etapa := range etapasSeed {
		if err := db.Where(models.KanbanEtapa{ID: etapa.ID}).FirstOrCreate(&etapa).Error; err != nil {
			return fmt.Errorf("database: falha ao semear etapa %q: %w", etapa.Nome, err)
		}
	}

	for _, nome := range tiposDocumentoSeed {
		tipo := models.TipoDocumento{Nome: nome}
		if err := db.Where(models.TipoDocumento{Nome: nome}).FirstOrCreate(&tipo).Error; err != nil {
			return fmt.Errorf("database: falha ao semear tipo de documento %q: %w", nome, err)
		}
	}

	return nil
}
