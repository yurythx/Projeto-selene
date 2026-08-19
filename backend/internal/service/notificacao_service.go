package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// NotificacaoService gerencia as notificações in-app de prazos/
// vencimentos (Radar) — pedido explícito do usuário: "precisamos ter os
// alertas/notificacoes a respeito de prazos e vencimentos", com os dois
// canais confirmados (e-mail + dentro do app).
type NotificacaoService struct {
	repo         repository.NotificacaoRepository
	radarService *RadarService
	contratoRepo repository.ContratoRepository
	userRepo     repository.UserRepository
	notifier     Notifier
}

func NewNotificacaoService(
	repo repository.NotificacaoRepository,
	radarService *RadarService,
	contratoRepo repository.ContratoRepository,
	userRepo repository.UserRepository,
	notifier Notifier,
) *NotificacaoService {
	return &NotificacaoService{
		repo:         repo,
		radarService: radarService,
		contratoRepo: contratoRepo,
		userRepo:     userRepo,
		notifier:     notifier,
	}
}

// Listar devolve as notificações do usuário (não-lidas primeiro).
func (s *NotificacaoService) Listar(ctx context.Context, usuarioID uuid.UUID) ([]models.Notificacao, error) {
	notificacoes, err := s.repo.Listar(ctx, usuarioID)
	if err != nil {
		return nil, fmt.Errorf("service: listar notificações: %w", err)
	}
	return notificacoes, nil
}

// ContarNaoLidas devolve a contagem pro badge da TopBar.
func (s *NotificacaoService) ContarNaoLidas(ctx context.Context, usuarioID uuid.UUID) (int64, error) {
	total, err := s.repo.ContarNaoLidas(ctx, usuarioID)
	if err != nil {
		return 0, fmt.Errorf("service: contar notificações não lidas: %w", err)
	}
	return total, nil
}

// MarcarLida marca UMA notificação como lida — escopado ao usuário
// (ErrNotificacaoNotFound tanto pra ID inexistente quanto pra ID de
// outro usuário, ver o comentário no repository).
func (s *NotificacaoService) MarcarLida(ctx context.Context, usuarioID, notificacaoID uuid.UUID) error {
	if err := s.repo.MarcarLida(ctx, usuarioID, notificacaoID); err != nil {
		return fmt.Errorf("service: marcar notificação como lida: %w", err)
	}
	return nil
}

// MarcarTodasLidas marca todas as notificações não lidas do usuário.
func (s *NotificacaoService) MarcarTodasLidas(ctx context.Context, usuarioID uuid.UUID) error {
	if err := s.repo.MarcarTodasLidas(ctx, usuarioID); err != nil {
		return fmt.Errorf("service: marcar todas as notificações como lidas: %w", err)
	}
	return nil
}

// GerarAlertas é o gerador — chamado periodicamente por um scheduler
// (ver a goroutine em cmd/api/main.go). Roda o Radar, decide quem
// deveria ser avisado de cada item (fiscal do contrato + todos os
// admins), cria uma Notificacao por destinatário (deduplicada por
// chave_alerta, ver o repository), e manda UM e-mail de resumo por
// destinatário com só os alertas NOVOS desta execução — não reenvia
// e-mail sobre um alerta que já foi notificado antes, mesmo que ele
// continue valendo na próxima rodada.
//
// Falhas parciais (um contrato não encontrado, um e-mail que não envia)
// são logadas e a execução continua pros demais itens — um problema
// isolado não deveria impedir os outros alertas de serem gerados.
func (s *NotificacaoService) GerarAlertas(ctx context.Context) error {
	itens, err := s.radarService.Listar(ctx)
	if err != nil {
		return fmt.Errorf("service: listar itens do radar para gerar alertas: %w", err)
	}
	if len(itens) == 0 {
		return nil
	}

	admins, err := s.listarAdmins(ctx)
	if err != nil {
		return fmt.Errorf("service: listar admins para gerar alertas: %w", err)
	}

	type destinoNovo struct {
		nome  string
		email string
		itens []ItemRadar
	}
	novosPorUsuario := map[uuid.UUID]*destinoNovo{}

	contratoCache := map[uuid.UUID]*models.Contrato{}

	for _, item := range itens {
		contrato, ok := contratoCache[item.ContratoID]
		if !ok {
			c, err := s.contratoRepo.FindByID(ctx, item.ContratoID)
			if err != nil {
				slog.ErrorContext(ctx, "notificacoes: falha ao buscar contrato do alerta, pulando este item",
					"contrato_id", item.ContratoID, "erro", err)
				continue
			}
			contratoCache[item.ContratoID] = c
			contrato = c
		}

		for _, destinatario := range destinatariosDoAlerta(contrato, admins) {
			n := &models.Notificacao{
				ID:          uuid.New(),
				UsuarioID:   destinatario.ID,
				Tipo:        string(item.Tipo),
				Nivel:       string(item.Nivel),
				ContratoID:  item.ContratoID,
				ProcessoID:  item.ProcessoID,
				Mensagem:    item.Mensagem,
				ChaveAlerta: chaveAlerta(item),
			}

			criada, err := s.repo.Criar(ctx, n)
			if err != nil {
				slog.ErrorContext(ctx, "notificacoes: falha ao gravar notificação, pulando este destinatário",
					"usuario_id", destinatario.ID, "erro", err)
				continue
			}
			if !criada {
				// Já existia (mesmo usuário + mesmo alerta em execução
				// anterior) — não é um erro, é a deduplicação funcionando.
				continue
			}

			destino, ok := novosPorUsuario[destinatario.ID]
			if !ok {
				destino = &destinoNovo{nome: destinatario.Nome, email: destinatario.Email}
				novosPorUsuario[destinatario.ID] = destino
			}
			destino.itens = append(destino.itens, item)
		}
	}

	for usuarioID, destino := range novosPorUsuario {
		if err := s.notifier.EnviarResumoAlertas(ctx, destino.email, destino.nome, destino.itens); err != nil {
			slog.ErrorContext(ctx, "notificacoes: falha ao enviar resumo de alertas por e-mail",
				"usuario_id", usuarioID, "erro", err)
			// Não retorna — a notificação IN-APP já foi criada com
			// sucesso; a falha de e-mail não deveria "perder" isso.
		}
	}

	return nil
}

func (s *NotificacaoService) listarAdmins(ctx context.Context) ([]models.User, error) {
	todos, err := s.userRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	admins := make([]models.User, 0, len(todos))
	for _, u := range todos {
		if u.IsAdmin {
			admins = append(admins, u)
		}
	}
	return admins, nil
}

// destinatariosDoAlerta decide quem recebe um alerta de um contrato: o
// fiscal responsável (Contrato.Fiscal, populado via Preload em
// ContratoRepository.FindByID) + todos os admins — deduplicados por ID
// (o fiscal também pode ser admin).
func destinatariosDoAlerta(contrato *models.Contrato, admins []models.User) []models.User {
	vistos := map[uuid.UUID]bool{}
	destinatarios := make([]models.User, 0, len(admins)+1)

	if contrato.Fiscal != nil {
		vistos[contrato.Fiscal.ID] = true
		destinatarios = append(destinatarios, *contrato.Fiscal)
	}
	for _, admin := range admins {
		if vistos[admin.ID] {
			continue
		}
		vistos[admin.ID] = true
		destinatarios = append(destinatarios, admin)
	}
	return destinatarios
}

// chaveAlerta identifica a MESMA instância de alerta entre execuções
// sucessivas do gerador — ver o comentário completo na migration 000014.
// processoID vem vazio (não "<nil>") pros alertas de vigência de
// contrato, que não têm processo associado.
func chaveAlerta(item ItemRadar) string {
	processoID := ""
	if item.ProcessoID != nil {
		processoID = item.ProcessoID.String()
	}
	return fmt.Sprintf("%s:%s:%s:%s", item.Tipo, item.ContratoID, processoID, item.Nivel)
}
