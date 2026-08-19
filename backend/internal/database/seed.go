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
//
// ExigeValidade marca os tipos que vencem (certidões) — usado pelo Radar
// de Alertas (Fase 1 do roadmap) pra saber quais documentos pedir data de
// validade no upload e avisar quando estiverem vencidos/vencendo. Numa
// base nova, Migrate() roda antes de Seed() (ver cmd/api/main.go) — o
// UPDATE da migration 000003 roda contra uma tabela ainda vazia e não
// tem efeito nenhum, então É este slice aqui, não a migration, que
// garante o flag certo em bases novas. A migration continua necessária
// pra fazer o mesmo backfill em bases que já existiam antes dela.
var tiposDocumentoSeed = []struct {
	Nome                  string
	ExigeValidade         bool
	RestritoTipoObjeto    *models.TipoObjeto
	RestritoTerceirizacao bool
}{
	{Nome: "Ordem de Fornecimento (OF)"},
	{Nome: "Pré-Empenho"},
	{Nome: "Ofício de Solicitação"},
	{Nome: "Nota de Empenho"},
	{Nome: "Nota Fiscal / Fatura"},
	{Nome: "Ordem de Recepção"},
	{Nome: "Extrato do Empenho"},
	{Nome: "Declaração do Simples Nacional"},
	{Nome: "CND Trabalhista", ExigeValidade: true},
	{Nome: "CND FGTS", ExigeValidade: true},
	{Nome: "CND Municipal", ExigeValidade: true},
	{Nome: "CND Estadual", ExigeValidade: true},
	{Nome: "CND Federal", ExigeValidade: true},
	{Nome: "CND INSS", ExigeValidade: true},
	{Nome: "Relatório de Pagamento Assinado"},
	// RestritoTipoObjeto=SERVICO: só aparecem no select de upload e só
	// podem ser anexados a processos de contratos SERVICO (ver
	// service.TipoDocumentoAplicavel) — mesma regra que já valia pro
	// checklist de avanço de etapa (checklistCondicionalServico), agora
	// também aplicada ao upload em si.
	{Nome: "Planilha de Medição de Serviços", RestritoTipoObjeto: &tipoObjetoServicoSeed},
	{Nome: "Boleto DAM", RestritoTipoObjeto: &tipoObjetoServicoSeed},
	// SGF-Rondonópolis, Fase 6: específicos de contratos de mão de obra
	// terceirizada (IN SCL 04/2021 Art.9º-XXXII, alíneas a/b.1/b.2/b.3) —
	// só entram no checklist e só podem ser anexados quando
	// Contrato.ExigeFiscalizacaoTerceirizacao é true, ver
	// internal/service/checklist.go.
	{Nome: "Comprovante de Pagamento de Salário", RestritoTerceirizacao: true},
	{Nome: "Protocolo GFIP", RestritoTerceirizacao: true},
	{Nome: "Guia GRF/GPS", RestritoTerceirizacao: true},
	{Nome: "Relação de Trabalhadores (SEFIP)", RestritoTerceirizacao: true},
}

// tipoObjetoServicoSeed existe só pra dar um endereço estável a
// models.TipoObjetoServico — os literais do slice acima precisam de um
// ponteiro, e não dá pra tirar o endereço de uma constante diretamente.
var tipoObjetoServicoSeed = models.TipoObjetoServico

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

	for _, t := range tiposDocumentoSeed {
		tipo := models.TipoDocumento{
			Nome:                  t.Nome,
			ExigeValidade:         t.ExigeValidade,
			RestritoTipoObjeto:    t.RestritoTipoObjeto,
			RestritoTerceirizacao: t.RestritoTerceirizacao,
		}
		if err := db.Where(models.TipoDocumento{Nome: t.Nome}).FirstOrCreate(&tipo).Error; err != nil {
			return fmt.Errorf("database: falha ao semear tipo de documento %q: %w", t.Nome, err)
		}
	}

	return nil
}
