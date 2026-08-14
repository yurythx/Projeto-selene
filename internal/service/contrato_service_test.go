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
		if u.KeycloakID == keycloakID {
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

func (f *fakeContratoRepository) List(ctx context.Context) ([]models.Contrato, error) {
	var out []models.Contrato
	for _, c := range f.criados {
		out = append(out, *c)
	}
	return out, nil
}

func (f *fakeContratoRepository) Update(ctx context.Context, contrato *models.Contrato) error {
	return nil
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
}
