package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// fakeUserRepository e fakeContratoRepository são dublês de teste em
// memória — permitem testar as regras de ContratoService sem banco real.
type fakeUserRepository struct {
	usersByID map[uuid.UUID]*models.User
}

func newFakeUserRepository(users ...*models.User) *fakeUserRepository {
	byID := make(map[uuid.UUID]*models.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	return &fakeUserRepository{usersByID: byID}
}

func (f *fakeUserRepository) FindByKeycloakID(ctx context.Context, keycloakID string) (*models.User, error) {
	for _, u := range f.usersByID {
		if u.KeycloakID != nil && *u.KeycloakID == keycloakID {
			return u, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func (f *fakeUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if u, ok := f.usersByID[id]; ok {
		return u, nil
	}
	return nil, repository.ErrUserNotFound
}

func (f *fakeUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	for _, u := range f.usersByID {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func (f *fakeUserRepository) List(ctx context.Context) ([]models.User, error) {
	var out []models.User
	for _, u := range f.usersByID {
		out = append(out, *u)
	}
	return out, nil
}

func (f *fakeUserRepository) Create(ctx context.Context, user *models.User) error {
	f.usersByID[user.ID] = user
	return nil
}

func (f *fakeUserRepository) Update(ctx context.Context, user *models.User) error {
	f.usersByID[user.ID] = user
	return nil
}

var _ repository.UserRepository = (*fakeUserRepository)(nil)

type fakeContratoRepository struct {
	criados []*models.Contrato
}

func (f *fakeContratoRepository) Create(ctx context.Context, contrato *models.Contrato) error {
	// Simula o índice único idx_contratos_numero_contrato (migration
	// 000001) — mesmo comportamento do Postgres real, pra testar o
	// mapeamento em repository.ErrNumeroContratoDuplicado sem banco.
	for _, c := range f.criados {
		if c.NumeroContrato == contrato.NumeroContrato {
			return repository.ErrNumeroContratoDuplicado
		}
	}
	contrato.ID = uuid.New()
	f.criados = append(f.criados, contrato)
	return nil
}

func (f *fakeContratoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Contrato, error) {
	for _, c := range f.criados {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, repository.ErrContratoNotFound
}

func (f *fakeContratoRepository) List(ctx context.Context, pagina repository.Pagina, filtro repository.FiltroContrato) (repository.ResultadoPaginado[models.Contrato], error) {
	var out []models.Contrato
	for _, c := range f.criados {
		out = append(out, *c)
	}
	return repository.ResultadoPaginado[models.Contrato]{
		Dados: out,
		Total: int64(len(out)),
	}, nil
}

func (f *fakeContratoRepository) Update(ctx context.Context, contrato *models.Contrato) error {
	return nil
}

func (f *fakeContratoRepository) ListAtivos(ctx context.Context) ([]models.Contrato, error) {
	return nil, nil
}

func (f *fakeContratoRepository) ListTodos(ctx context.Context) ([]models.Contrato, error) {
	return nil, nil
}

var _ repository.ContratoRepository = (*fakeContratoRepository)(nil)

func TestContratoServiceCriar(t *testing.T) {
	ctx := context.Background()
	fiscal := &models.User{ID: uuid.New(), Nome: "Maria Fiscal", IsFiscal: true}
	naoFiscal := &models.User{ID: uuid.New(), Nome: "Carlos Sem Permissão", IsFiscal: false}

	baseInput := func() NovoContratoInput {
		return NovoContratoInput{
			NumeroContrato:  "125/2026",
			DataAssinatura:  "2026-01-15",
			ContratadaNome:  "Empresa Teste",
			ContratadaCNPJ:  "12.345.678/0001-90",
			ContratadaEmail: "contato@empresateste.com.br",
			FiscalID:        fiscal.ID,
			TipoObjeto:      models.TipoObjetoServico,
		}
	}

	t.Run("caminho feliz cria o contrato", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		contrato, err := svc.Criar(ctx, baseInput())
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if contrato.ID == uuid.Nil {
			t.Fatal("esperava um ID gerado para o contrato")
		}
		if len(contratoRepo.criados) != 1 {
			t.Fatalf("esperava 1 contrato persistido, veio %d", len(contratoRepo.criados))
		}
	})

	t.Run("tipo de objeto inválido é rejeitado", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		input := baseInput()
		input.TipoObjeto = "OUTRO_QUALQUER"

		_, err := svc.Criar(ctx, input)
		if !errors.Is(err, ErrTipoObjetoInvalido) {
			t.Fatalf("esperava ErrTipoObjetoInvalido, veio %v", err)
		}
	})

	t.Run("fiscal inexistente é rejeitado", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		input := baseInput()
		input.FiscalID = uuid.New() // não cadastrado

		_, err := svc.Criar(ctx, input)
		if !errors.Is(err, ErrFiscalInvalido) {
			t.Fatalf("esperava ErrFiscalInvalido, veio %v", err)
		}
	})

	t.Run("usuário sem IsFiscal é rejeitado como fiscal do contrato", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal, naoFiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		input := baseInput()
		input.FiscalID = naoFiscal.ID

		_, err := svc.Criar(ctx, input)
		if !errors.Is(err, ErrFiscalInvalido) {
			t.Fatalf("esperava ErrFiscalInvalido, veio %v", err)
		}
	})

	t.Run("data de assinatura em formato inválido é rejeitada", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		input := baseInput()
		input.DataAssinatura = "15/01/2026" // formato errado, esperado AAAA-MM-DD

		_, err := svc.Criar(ctx, input)
		if err == nil {
			t.Fatal("esperava erro de data inválida, veio nil")
		}
	})

	t.Run("ExigeFiscalizacaoTerceirizacao não informado nasce false", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		contrato, err := svc.Criar(ctx, baseInput())
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if contrato.ExigeFiscalizacaoTerceirizacao {
			t.Fatal("esperava ExigeFiscalizacaoTerceirizacao=false por default")
		}
	})

	t.Run("ExigeFiscalizacaoTerceirizacao=true é persistido", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		input := baseInput()
		input.ExigeFiscalizacaoTerceirizacao = true

		contrato, err := svc.Criar(ctx, input)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !contrato.ExigeFiscalizacaoTerceirizacao {
			t.Fatal("esperava ExigeFiscalizacaoTerceirizacao=true")
		}
	})

	t.Run("CNPJ com formato inválido é rejeitado", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		input := baseInput()
		input.ContratadaCNPJ = "123"

		_, err := svc.Criar(ctx, input)
		if !errors.Is(err, ErrCNPJInvalido) {
			t.Fatalf("esperava ErrCNPJInvalido, veio %v", err)
		}
	})

	t.Run("e-mail da contratada inválido é rejeitado", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		input := baseInput()
		input.ContratadaEmail = "não é um e-mail"

		_, err := svc.Criar(ctx, input)
		if !errors.Is(err, ErrEmailInvalido) {
			t.Fatalf("esperava ErrEmailInvalido, veio %v", err)
		}
	})

	t.Run("e-mail vazio (opcional) é aceito", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		input := baseInput()
		input.ContratadaEmail = ""

		_, err := svc.Criar(ctx, input)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
	})

	t.Run("vigência final antes da assinatura é rejeitada", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		input := baseInput()
		input.DataAssinatura = "2026-01-15"
		input.DataVigenciaFim = "2026-01-10" // antes da assinatura

		_, err := svc.Criar(ctx, input)
		if !errors.Is(err, ErrVigenciaAntesDaAssinatura) {
			t.Fatalf("esperava ErrVigenciaAntesDaAssinatura, veio %v", err)
		}
	})

	t.Run("vigência final igual à assinatura é rejeitada", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		input := baseInput()
		input.DataAssinatura = "2026-01-15"
		input.DataVigenciaFim = "2026-01-15" // mesmo dia — não é um prazo de vigência válido

		_, err := svc.Criar(ctx, input)
		if !errors.Is(err, ErrVigenciaAntesDaAssinatura) {
			t.Fatalf("esperava ErrVigenciaAntesDaAssinatura, veio %v", err)
		}
	})

	t.Run("número de contrato duplicado é rejeitado", func(t *testing.T) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		if _, err := svc.Criar(ctx, baseInput()); err != nil {
			t.Fatalf("falha ao criar o primeiro contrato: %v", err)
		}

		_, err := svc.Criar(ctx, baseInput()) // mesmo NumeroContrato
		if !errors.Is(err, repository.ErrNumeroContratoDuplicado) {
			t.Fatalf("esperava ErrNumeroContratoDuplicado, veio %v", err)
		}
	})
}

func TestContratoServiceAtualizarEEncerrar(t *testing.T) {
	ctx := context.Background()
	fiscal := &models.User{ID: uuid.New(), Nome: "Maria Fiscal", IsFiscal: true}

	novoServico := func() (*ContratoService, *models.Contrato) {
		userRepo := newFakeUserRepository(fiscal)
		contratoRepo := &fakeContratoRepository{}
		svc := NewContratoService(contratoRepo, userRepo)

		contrato, err := svc.Criar(ctx, NovoContratoInput{
			NumeroContrato:  "125/2026",
			DataAssinatura:  "2026-01-15",
			ContratadaNome:  "Empresa Original",
			ContratadaCNPJ:  "11.111.111/0001-11",
			ContratadaEmail: "original@empresa.com.br",
			FiscalID:        fiscal.ID,
			TipoObjeto:      models.TipoObjetoConsumo,
		})
		if err != nil {
			t.Fatalf("falha ao preparar contrato para o teste: %v", err)
		}
		return svc, contrato
	}

	t.Run("contrato criado nasce ativo", func(t *testing.T) {
		_, contrato := novoServico()
		if !contrato.Ativo {
			t.Fatal("contrato recém-criado deveria nascer Ativo=true")
		}
	})

	t.Run("atualizar só altera os campos informados", func(t *testing.T) {
		svc, contrato := novoServico()

		novoNome := "Empresa Renomeada"
		atualizado, err := svc.Atualizar(ctx, contrato.ID, AtualizarContratoInput{ContratadaNome: &novoNome})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if atualizado.ContratadaNome != novoNome {
			t.Fatalf("esperava ContratadaNome=%q, veio %q", novoNome, atualizado.ContratadaNome)
		}
		if atualizado.ContratadaEmail != "original@empresa.com.br" {
			t.Fatalf("campo não informado não deveria mudar, veio %q", atualizado.ContratadaEmail)
		}
	})

	t.Run("encerrar marca Ativo=false", func(t *testing.T) {
		svc, contrato := novoServico()

		encerrado, err := svc.Encerrar(ctx, contrato.ID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if encerrado.Ativo {
			t.Fatal("esperava Ativo=false após Encerrar")
		}
	})

	t.Run("atualizar ExigeFiscalizacaoTerceirizacao", func(t *testing.T) {
		svc, contrato := novoServico()

		valor := true
		atualizado, err := svc.Atualizar(ctx, contrato.ID, AtualizarContratoInput{ExigeFiscalizacaoTerceirizacao: &valor})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !atualizado.ExigeFiscalizacaoTerceirizacao {
			t.Fatal("esperava ExigeFiscalizacaoTerceirizacao=true após atualizar")
		}
	})

	t.Run("atualizar com CNPJ inválido é rejeitado", func(t *testing.T) {
		svc, contrato := novoServico()

		cnpjInvalido := "123"
		_, err := svc.Atualizar(ctx, contrato.ID, AtualizarContratoInput{ContratadaCNPJ: &cnpjInvalido})
		if !errors.Is(err, ErrCNPJInvalido) {
			t.Fatalf("esperava ErrCNPJInvalido, veio %v", err)
		}
	})

	t.Run("atualizar com e-mail inválido é rejeitado", func(t *testing.T) {
		svc, contrato := novoServico()

		emailInvalido := "não é um e-mail"
		_, err := svc.Atualizar(ctx, contrato.ID, AtualizarContratoInput{ContratadaEmail: &emailInvalido})
		if !errors.Is(err, ErrEmailInvalido) {
			t.Fatalf("esperava ErrEmailInvalido, veio %v", err)
		}
	})

	t.Run("atualizar vigência para antes da assinatura é rejeitado", func(t *testing.T) {
		svc, contrato := novoServico() // DataAssinatura = 2026-01-15

		vigenciaInvalida := "2026-01-01"
		_, err := svc.Atualizar(ctx, contrato.ID, AtualizarContratoInput{DataVigenciaFim: &vigenciaInvalida})
		if !errors.Is(err, ErrVigenciaAntesDaAssinatura) {
			t.Fatalf("esperava ErrVigenciaAntesDaAssinatura, veio %v", err)
		}
	})

	t.Run("atualizar vigência para string vazia limpa o campo", func(t *testing.T) {
		svc, contrato := novoServico()

		vazio := ""
		atualizado, err := svc.Atualizar(ctx, contrato.ID, AtualizarContratoInput{DataVigenciaFim: &vazio})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if atualizado.DataVigenciaFim != nil {
			t.Fatal("esperava DataVigenciaFim=nil após limpar o campo")
		}
	})
}
