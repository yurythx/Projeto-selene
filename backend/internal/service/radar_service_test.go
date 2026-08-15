package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/testutil"
)

func TestDiasAte(t *testing.T) {
	hoje := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)

	futuro := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	if got := diasAte(hoje, futuro); got != 5 {
		t.Errorf("diasAte(futuro) = %d, esperado 5", got)
	}

	passado := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	if got := diasAte(hoje, passado); got != -5 {
		t.Errorf("diasAte(passado) = %d, esperado -5", got)
	}

	if got := diasAte(hoje, hoje); got != 0 {
		t.Errorf("diasAte(hoje) = %d, esperado 0", got)
	}
}

func TestNivelVigencia(t *testing.T) {
	casos := []struct {
		dias     int
		nivel    NivelAlerta
		emAlerta bool
	}{
		{dias: 91, emAlerta: false},
		{dias: 90, nivel: NivelAlertaAtencao, emAlerta: true},
		{dias: 31, nivel: NivelAlertaAtencao, emAlerta: true},
		{dias: 30, nivel: NivelAlertaCritico, emAlerta: true},
		{dias: 0, nivel: NivelAlertaCritico, emAlerta: true},
		{dias: -10, nivel: NivelAlertaCritico, emAlerta: true},
	}

	for _, c := range casos {
		nivel, alerta := nivelVigencia(c.dias)
		if alerta != c.emAlerta {
			t.Errorf("nivelVigencia(%d): alerta = %v, esperado %v", c.dias, alerta, c.emAlerta)
			continue
		}
		if alerta && nivel != c.nivel {
			t.Errorf("nivelVigencia(%d): nivel = %v, esperado %v", c.dias, nivel, c.nivel)
		}
	}
}

func TestNivelCertidao(t *testing.T) {
	casos := []struct {
		dias     int
		nivel    NivelAlerta
		emAlerta bool
	}{
		{dias: 31, emAlerta: false},
		{dias: 30, nivel: NivelAlertaAtencao, emAlerta: true},
		{dias: 0, nivel: NivelAlertaAtencao, emAlerta: true},
		{dias: -1, nivel: NivelAlertaCritico, emAlerta: true},
	}

	for _, c := range casos {
		nivel, alerta := nivelCertidao(c.dias)
		if alerta != c.emAlerta {
			t.Errorf("nivelCertidao(%d): alerta = %v, esperado %v", c.dias, alerta, c.emAlerta)
			continue
		}
		if alerta && nivel != c.nivel {
			t.Errorf("nivelCertidao(%d): nivel = %v, esperado %v", c.dias, nivel, c.nivel)
		}
	}
}

func TestRadarService_Listar(t *testing.T) {
	ctx := context.Background()
	hoje := time.Now().UTC()

	fiscal := &models.User{ID: uuid.New(), Nome: "Fiscal Teste", IsFiscal: true}

	// Contrato 1: vigência vencendo em 10 dias (CRITICO).
	vigenciaCritica := hoje.AddDate(0, 0, 10)
	contratoCritico := &models.Contrato{
		ID: uuid.New(), NumeroContrato: "1/2026", FiscalID: fiscal.ID, Fiscal: fiscal,
		Ativo: true, DataVigenciaFim: &vigenciaCritica,
	}

	// Contrato 2: vigência tranquila (200 dias) — não deve gerar alerta.
	vigenciaOK := hoje.AddDate(0, 0, 200)
	contratoOK := &models.Contrato{
		ID: uuid.New(), NumeroContrato: "2/2026", FiscalID: fiscal.ID, Fiscal: fiscal,
		Ativo: true, DataVigenciaFim: &vigenciaOK,
	}

	// Contrato 3: encerrado (Ativo=false), com vigência já vencida — não
	// deve gerar alerta (contrato encerrado sai do radar).
	vigenciaVencidaEncerrado := hoje.AddDate(0, 0, -100)
	contratoEncerrado := &models.Contrato{
		ID: uuid.New(), NumeroContrato: "3/2026", FiscalID: fiscal.ID, Fiscal: fiscal,
		Ativo: false, DataVigenciaFim: &vigenciaVencidaEncerrado,
	}

	etapa1 := &models.KanbanEtapa{ID: 1, Nome: "Elaborar OF / Pré-Empenho", Posicao: 1}
	tipoCND := &models.TipoDocumento{ID: 1, Nome: "CND Federal", ExigeValidade: true}

	// Processo do contrato OK: tem uma CND já vencida anexada.
	validadeVencida := hoje.AddDate(0, 0, -5)
	processoComCertidaoVencida := &models.ProcessoPagamento{
		ID: uuid.New(), ContratoID: contratoOK.ID, Contrato: contratoOK,
		MesReferencia: "01/2026", EtapaAtualID: 1, EtapaAtual: etapa1,
		Status: models.StatusProcessoAtivo,
	}

	contratoRepo := testutil.NewFakeContratoRepository(contratoCritico, contratoOK, contratoEncerrado)
	processoRepo := testutil.NewFakeProcessoPagamentoRepository(processoComCertidaoVencida)
	docRepo := &testutil.FakeDocumentoAnexoRepository{
		Documentos: []models.DocumentoAnexo{
			{
				ID: uuid.New(), ProcessoPagamentoID: processoComCertidaoVencida.ID,
				TipoDocumentoID: tipoCND.ID, TipoDocumento: tipoCND,
				NomeArquivo: "cnd.pdf", DataValidade: &validadeVencida,
			},
		},
	}
	logRepo := &testutil.FakeKanbanLogRepository{
		Logs: []models.KanbanLog{
			{
				ID: uuid.New(), ProcessoPagamentoID: processoComCertidaoVencida.ID,
				UsuarioID: fiscal.ID, EtapaDestinoID: 1,
				// Bem recente — não deve disparar o alerta de "parado".
				MovidoEm: hoje.AddDate(0, 0, -1),
			},
		},
	}

	radarService := NewRadarService(contratoRepo, processoRepo, docRepo, logRepo)

	itens, err := radarService.Listar(ctx)
	if err != nil {
		t.Fatalf("Listar() erro inesperado: %v", err)
	}

	var temVigenciaCritica, temCertidaoVencida, temContratoEncerrado, temProcessoParado bool
	for _, item := range itens {
		switch {
		case item.Tipo == TipoAlertaVigenciaContrato && item.ContratoID == contratoCritico.ID:
			temVigenciaCritica = true
			if item.Nivel != NivelAlertaCritico {
				t.Errorf("alerta de vigência do contrato crítico veio com nível %v, esperado CRITICO", item.Nivel)
			}
		case item.Tipo == TipoAlertaCertidao && item.ContratoID == contratoOK.ID:
			temCertidaoVencida = true
			if item.Nivel != NivelAlertaCritico {
				t.Errorf("alerta de certidão vencida veio com nível %v, esperado CRITICO", item.Nivel)
			}
		case item.ContratoID == contratoEncerrado.ID:
			temContratoEncerrado = true
		case item.Tipo == TipoAlertaProcessoParado:
			temProcessoParado = true
		}
	}

	if !temVigenciaCritica {
		t.Error("esperava um alerta de vigência CRITICO pro contrato 1/2026, não encontrado")
	}
	if !temCertidaoVencida {
		t.Error("esperava um alerta de certidão vencida pro processo do contrato 2/2026, não encontrado")
	}
	if temContratoEncerrado {
		t.Error("contrato encerrado (Ativo=false) não deveria gerar nenhum item no radar")
	}
	if temProcessoParado {
		t.Error("processo movido há 1 dia não deveria disparar o alerta de 'parado há muito tempo'")
	}
}

// TestRadarService_Listar_SemAlertasNaoRetornaNil reproduz o bug
// encontrado rodando docker-compose.prod.yml de verdade: sem nenhum
// alerta em aberto (o caso "tudo em dia", nada excepcional), Listar
// devolvia um slice nil — RadarHandler.Listar serializa isso pra JSON
// como `null`, não `[]` (encoding/json), e o frontend quebrava
// (RadarPage.ordenar faz [...itens], "TypeError: ... is not iterable").
// `len(itens) == 0` sozinho não pega essa regressão (nil também tem
// len 0) — por isso a checagem explícita `itens == nil`.
func TestRadarService_Listar_SemAlertasNaoRetornaNil(t *testing.T) {
	ctx := context.Background()

	contratoRepo := testutil.NewFakeContratoRepository()
	processoRepo := testutil.NewFakeProcessoPagamentoRepository()
	docRepo := &testutil.FakeDocumentoAnexoRepository{}
	logRepo := &testutil.FakeKanbanLogRepository{}

	radarService := NewRadarService(contratoRepo, processoRepo, docRepo, logRepo)

	itens, err := radarService.Listar(ctx)
	if err != nil {
		t.Fatalf("Listar() erro inesperado: %v", err)
	}
	if itens == nil {
		t.Fatal("Listar() sem nenhum alerta devolveu nil — deveria ser []ItemRadar{} (serializa como [] em JSON, não null)")
	}
	if len(itens) != 0 {
		t.Fatalf("esperava 0 itens, veio %d", len(itens))
	}
}
