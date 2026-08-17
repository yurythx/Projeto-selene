package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"projeto-selene/internal/models"
)

// ErrKeycloakConfigNotFound é devolvido quando nenhum admin salvou ainda
// uma configuração customizada de Keycloak — a aplicação continua
// funcionando normalmente nesse caso (ver config.Config.KeycloakX),
// simplesmente não há linha em keycloak_config ainda.
var ErrKeycloakConfigNotFound = errors.New("repository: configuração de keycloak não encontrada")

// KeycloakConfigRepository abstrai a persistência da configuração de
// Keycloak editável em runtime (linha única, ver a migration 000012).
type KeycloakConfigRepository interface {
	Buscar(ctx context.Context) (*models.KeycloakConfig, error)
	Salvar(ctx context.Context, cfg *models.KeycloakConfig) error
}

type gormKeycloakConfigRepository struct {
	db *gorm.DB
}

func NewKeycloakConfigRepository(db *gorm.DB) KeycloakConfigRepository {
	return &gormKeycloakConfigRepository{db: db}
}

func (r *gormKeycloakConfigRepository) Buscar(ctx context.Context) (*models.KeycloakConfig, error) {
	var cfg models.KeycloakConfig
	err := r.db.WithContext(ctx).Preload("UpdatedBy").First(&cfg, "id = 1").Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrKeycloakConfigNotFound
		}
		return nil, fmt.Errorf("repository: buscar configuração de keycloak: %w", err)
	}
	return &cfg, nil
}

// Salvar faz upsert da linha única (id=1) — cria na primeira vez que um
// admin salva, atualiza nas seguintes. clause.OnConflict evita uma
// leitura prévia só pra decidir entre Create/Update (a linha inteira é
// sempre substituída, nunca um PATCH parcial — KeycloakConfigService é
// quem decide se mantém o ClientSecret atual quando o admin deixa o
// campo em branco no formulário).
func (r *gormKeycloakConfigRepository) Salvar(ctx context.Context, cfg *models.KeycloakConfig) error {
	cfg.ID = 1
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"client_id", "client_secret", "issuer_url", "audience", "updated_by_id", "updated_at"}),
	}).Create(cfg).Error
	if err != nil {
		return fmt.Errorf("repository: salvar configuração de keycloak: %w", err)
	}
	return nil
}
