// Package repository desacopla o acesso ao banco de dados (GORM/Postgres)
// das regras de negócio, que vivem em internal/service.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"projeto-selene/internal/models"
)

// ErrUserNotFound é retornado por FindByKeycloakID/FindByID quando nenhum
// usuário local corresponde ao identificador informado. A camada de
// service usa este erro sentinela (via errors.Is) tanto para decidir
// quando provisionar um novo usuário (JIT) quanto para responder 404 nas
// rotas de administração — não é tratado como falha de infraestrutura.
var ErrUserNotFound = errors.New("repository: usuário não encontrado")

// UserRepository abstrai o acesso à tabela `users`.
type UserRepository interface {
	FindByKeycloakID(ctx context.Context, keycloakID string) (*models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	List(ctx context.Context) ([]models.User, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
}

type gormUserRepository struct {
	db *gorm.DB
}

// NewUserRepository constrói um UserRepository apoiado em GORM/Postgres.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

func (r *gormUserRepository) FindByKeycloakID(ctx context.Context, keycloakID string) (*models.User, error) {
	var user models.User

	err := r.db.WithContext(ctx).Where("keycloak_id = ?", keycloakID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("repository: buscar usuário por keycloak_id: %w", err)
	}

	return &user, nil
}

func (r *gormUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User

	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("repository: buscar usuário por id: %w", err)
	}

	return &user, nil
}

func (r *gormUserRepository) List(ctx context.Context) ([]models.User, error) {
	var users []models.User

	if err := r.db.WithContext(ctx).Order("nome").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("repository: listar usuários: %w", err)
	}

	return users, nil
}

func (r *gormUserRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("repository: criar usuário: %w", err)
	}

	return nil
}

func (r *gormUserRepository) Update(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("repository: atualizar usuário: %w", err)
	}

	return nil
}
