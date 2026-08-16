package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// EmpenhoService implementa o acompanhamento PARALELO/informativo do
// saldo de empenho pedido por IN01 Art.5º-VIII / IN04 Art.5º-XXII — ver o
// comentário em models.Empenho sobre este NÃO ser a fonte de verdade
// orçamentária.
type EmpenhoService struct {
	empenhoRepo      repository.EmpenhoRepository
	movimentacaoRepo repository.MovimentacaoEmpenhoRepository
	contratoRepo     repository.ContratoRepository
}

// NewEmpenhoService constrói um EmpenhoService.
func NewEmpenhoService(
	empenhoRepo repository.EmpenhoRepository,
	movimentacaoRepo repository.MovimentacaoEmpenhoRepository,
	contratoRepo repository.ContratoRepository,
) *EmpenhoService {
	return &EmpenhoService{empenhoRepo: empenhoRepo, movimentacaoRepo: movimentacaoRepo, contratoRepo: contratoRepo}
}

// CriarEmpenhoInput agrupa os dados de abertura de um novo Empenho.
// DataEmissao é string "AAAA-MM-DD" (mesma convenção de
// ContratoService.CriarContratoInput.DataAssinatura) — o parsing acontece
// aqui no service via parseData/dataLayout (internal/service/util.go),
// não no handler; camada HTTP nunca deveria dar significado a um formato
// de domínio.
type CriarEmpenhoInput struct {
	ContratoID      uuid.UUID
	NumeroEmpenho   string
	DataEmissao     string
	ValorInicial    int64 // centavos
	RegistradoPorID uuid.UUID
}

// CriarEmpenho abre um novo Empenho e já registra sua movimentação
// INICIAL — o saldo de um empenho recém-criado é sempre igual ao
// ValorInicial, expresso como o primeiro lançamento do histórico (nunca
// como um campo denormalizado à parte).
func (s *EmpenhoService) CriarEmpenho(ctx context.Context, input CriarEmpenhoInput) (*models.Empenho, error) {
	if input.ValorInicial <= 0 {
		return nil, ErrValorInvalido
	}
	if _, err := s.contratoRepo.FindByID(ctx, input.ContratoID); err != nil {
		return nil, err
	}

	dataEmissao, err := parseData(input.DataEmissao)
	if err != nil {
		return nil, fmt.Errorf("service: data de emissão inválida: %w: %w", ErrDataInvalida, err)
	}

	empenho := &models.Empenho{
		ContratoID:    input.ContratoID,
		NumeroEmpenho: input.NumeroEmpenho,
		DataEmissao:   dataEmissao,
		ValorInicial:  input.ValorInicial,
	}
	if err := s.empenhoRepo.Create(ctx, empenho); err != nil {
		return nil, fmt.Errorf("service: criar empenho: %w", err)
	}

	inicial := &models.MovimentacaoEmpenho{
		EmpenhoID:       empenho.ID,
		Tipo:            models.MovimentacaoInicial,
		Valor:           input.ValorInicial,
		RegistradoPorID: input.RegistradoPorID,
	}
	if err := s.movimentacaoRepo.Create(ctx, inicial); err != nil {
		return nil, fmt.Errorf("service: registrar movimentação inicial do empenho: %w", err)
	}

	return empenho, nil
}

// RegistrarMovimentacaoInput agrupa os dados de um lançamento novo no
// histórico de um Empenho já existente.
type RegistrarMovimentacaoInput struct {
	EmpenhoID           uuid.UUID
	Tipo                models.TipoMovimentacaoEmpenho
	Valor               int64 // centavos, sempre positivo
	ProcessoPagamentoID *uuid.UUID
	Observacao          string
	RegistradoPorID     uuid.UUID
}

// RegistrarMovimentacao lança um REFORCO, ANULACAO ou FATURA_APROPRIADA
// no histórico do empenho informado.
func (s *EmpenhoService) RegistrarMovimentacao(ctx context.Context, input RegistrarMovimentacaoInput) (*models.MovimentacaoEmpenho, error) {
	if input.Valor <= 0 {
		return nil, ErrValorInvalido
	}
	if input.Tipo == models.MovimentacaoInicial {
		return nil, ErrTipoMovimentacaoInvalido
	}
	if _, err := s.empenhoRepo.FindByID(ctx, input.EmpenhoID); err != nil {
		return nil, err
	}

	// ANULACAO e FATURA_APROPRIADA debitam o saldo (ver CalcularSaldo) —
	// não podem exceder o que está disponível. REFORCO só credita, não
	// precisa dessa checagem.
	if input.Tipo == models.MovimentacaoAnulacao || input.Tipo == models.MovimentacaoFaturaApropriada {
		saldoAtual, err := s.CalcularSaldo(ctx, input.EmpenhoID)
		if err != nil {
			return nil, fmt.Errorf("service: conferir saldo antes de registrar movimentação: %w", err)
		}
		if input.Valor > saldoAtual {
			return nil, ErrSaldoInsuficiente
		}
	}

	movimentacao := &models.MovimentacaoEmpenho{
		EmpenhoID:           input.EmpenhoID,
		Tipo:                input.Tipo,
		Valor:               input.Valor,
		ProcessoPagamentoID: input.ProcessoPagamentoID,
		Observacao:          input.Observacao,
		RegistradoPorID:     input.RegistradoPorID,
	}
	if err := s.movimentacaoRepo.Create(ctx, movimentacao); err != nil {
		return nil, fmt.Errorf("service: registrar movimentação de empenho: %w", err)
	}

	return movimentacao, nil
}

// CalcularSaldo reconstrói o saldo de um empenho somando/subtraindo todo o
// histórico de movimentações — nunca lido de um campo denormalizado (ver
// o comentário em models.MovimentacaoEmpenho), evitando que o saldo
// divirja do que o sustenta.
func (s *EmpenhoService) CalcularSaldo(ctx context.Context, empenhoID uuid.UUID) (int64, error) {
	movimentacoes, err := s.movimentacaoRepo.ListByEmpenho(ctx, empenhoID)
	if err != nil {
		return 0, fmt.Errorf("service: carregar movimentações para calcular saldo: %w", err)
	}

	var saldo int64
	for _, m := range movimentacoes {
		switch m.Tipo {
		case models.MovimentacaoInicial, models.MovimentacaoReforco:
			saldo += m.Valor
		case models.MovimentacaoAnulacao, models.MovimentacaoFaturaApropriada:
			saldo -= m.Valor
		}
	}

	return saldo, nil
}

// ListarPorContrato retorna todos os empenhos de um contrato.
func (s *EmpenhoService) ListarPorContrato(ctx context.Context, contratoID uuid.UUID) ([]models.Empenho, error) {
	empenhos, err := s.empenhoRepo.ListByContrato(ctx, contratoID)
	if err != nil {
		return nil, fmt.Errorf("service: listar empenhos do contrato: %w", err)
	}
	return empenhos, nil
}

// BuscarEmpenho retorna um empenho pelo ID.
func (s *EmpenhoService) BuscarEmpenho(ctx context.Context, id uuid.UUID) (*models.Empenho, error) {
	return s.empenhoRepo.FindByID(ctx, id)
}
