// Package service contém os casos de uso e regras de negócio do Selene,
// livres de detalhes de transporte HTTP ou de acesso a dados.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// UserService implementa os casos de uso relacionados a usuários,
// incluindo a sincronização Just-In-Time exigida pelo middleware de
// autenticação.
type UserService struct {
	userRepo repository.UserRepository
}

// NewUserService constrói um UserService a partir de um UserRepository.
func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// asserção em tempo de compilação: UserService satisfaz a interface que o
// middleware de autenticação espera, sem que o middleware precise conhecer
// este pacote (evita import cycle e mantém a Clean Architecture).
var _ middleware.UserProvisioner = (*UserService)(nil)

// FindOrCreateByKeycloakID busca o usuário local pelo KeycloakID (claim
// "sub" do token OIDC); se não existir, cria um novo registro básico
// (sincronização Just-In-Time) com IsFiscal=false por padrão — a permissão
// de fiscal precisa ser concedida explicitamente por um administrador.
func (s *UserService) FindOrCreateByKeycloakID(ctx context.Context, keycloakID, nome, email string) (*models.User, error) {
	user, err := s.userRepo.FindByKeycloakID(ctx, keycloakID)
	if err == nil {
		return user, nil
	}

	if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, fmt.Errorf("service: buscar usuário: %w", err)
	}

	newUser := &models.User{
		KeycloakID: keycloakID,
		Nome:       nome,
		Email:      email,
		IsFiscal:   false,
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, fmt.Errorf("service: provisionar usuário via JIT: %w", err)
	}

	return newUser, nil
}

// Listar retorna todos os usuários cadastrados — usado pelas rotas de
// administração (Seção 6, "Administração de Fiscais").
func (s *UserService) Listar(ctx context.Context) ([]models.User, error) {
	users, err := s.userRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: listar usuários: %w", err)
	}
	return users, nil
}

// Buscar retorna um usuário pelo ID.
func (s *UserService) Buscar(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.userRepo.FindByID(ctx, id)
}

// AtualizarPermissoesInput agrupa os campos que um administrador pode
// alterar em outro usuário. Ponteiros distinguem "campo não informado" de
// "campo definido como false/vazio" — só o que foi explicitamente
// enviado é alterado.
type AtualizarPermissoesInput struct {
	IsFiscal  *bool
	IsAdmin   *bool
	Matricula *string
}

// AtualizarPermissoes aplica as alterações de AtualizarPermissoesInput a
// um usuário existente (Seção 6: "administradores podem alterar o boolean
// IsFiscal para delegar permissões").
func (s *UserService) AtualizarPermissoes(ctx context.Context, id uuid.UUID, input AtualizarPermissoesInput) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.IsFiscal != nil {
		user.IsFiscal = *input.IsFiscal
	}
	if input.IsAdmin != nil {
		user.IsAdmin = *input.IsAdmin
	}
	if input.Matricula != nil {
		user.Matricula = *input.Matricula
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("service: atualizar permissões do usuário: %w", err)
	}

	return user, nil
}
