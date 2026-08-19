package repository_test

import (
	"context"
	"errors"
	"testing"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/testutil"
)

// TestKeycloakConfigRepository_BuscarSalvar é o teste de integração
// (Postgres real) do padrão singleton (linha única, id=1 fixo) e do
// upsert de KeycloakConfigRepository.Salvar — a checagem `id = 1` mais o
// clause.OnConflict são o tipo de detalhe SQL que um dublê em memória não
// exercitaria de verdade.
func TestKeycloakConfigRepository_BuscarSalvar(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(db)
	keycloakConfigRepo := repository.NewKeycloakConfigRepository(db)

	keycloakID := "admin-keycloak-config-teste"
	admin := &models.User{KeycloakID: &keycloakID, Nome: "Admin Teste", Email: "admin.kc.config@teste.local", IsAdmin: true}
	if err := userRepo.Create(ctx, admin); err != nil {
		t.Fatalf("falha ao criar admin de teste: %v", err)
	}

	t.Run("Buscar sem nenhuma configuração salva devolve ErrKeycloakConfigNotFound", func(t *testing.T) {
		// keycloak_config está entre as tabelas truncadas a cada teste
		// (ver testutil.OpenTestDB) — garantido vazio neste ponto.
		_, err := keycloakConfigRepo.Buscar(ctx)
		if !errors.Is(err, repository.ErrKeycloakConfigNotFound) {
			t.Fatalf("esperava ErrKeycloakConfigNotFound, veio %v", err)
		}
	})

	t.Run("Salvar cria na primeira vez, Buscar devolve com UpdatedBy preloadado", func(t *testing.T) {
		audience := "selene-api"
		cfg := &models.KeycloakConfig{
			ClientID:     "selene-client",
			ClientSecret: "segredo-inicial",
			IssuerURL:    "https://keycloak.example.org/realms/selene",
			Audience:     &audience,
			UpdatedByID:  admin.ID,
		}
		if err := keycloakConfigRepo.Salvar(ctx, cfg); err != nil {
			t.Fatalf("falha ao salvar configuração: %v", err)
		}

		encontrado, err := keycloakConfigRepo.Buscar(ctx)
		if err != nil {
			t.Fatalf("erro inesperado ao buscar: %v", err)
		}
		if encontrado.ID != 1 {
			t.Fatalf("esperava ID singleton = 1, veio %d", encontrado.ID)
		}
		if encontrado.ClientID != "selene-client" {
			t.Fatalf("ClientID inesperado: %q", encontrado.ClientID)
		}
		if encontrado.ClientSecret != "segredo-inicial" {
			t.Fatalf("ClientSecret inesperado: %q", encontrado.ClientSecret)
		}
		if encontrado.UpdatedBy == nil || encontrado.UpdatedBy.Nome != "Admin Teste" {
			t.Fatalf("esperava UpdatedBy preloadado com o admin de teste, veio %+v", encontrado.UpdatedBy)
		}
	})

	t.Run("Salvar de novo faz upsert (substitui a linha única, não duplica)", func(t *testing.T) {
		cfg := &models.KeycloakConfig{
			ClientID:     "selene-client-v2",
			ClientSecret: "segredo-novo",
			IssuerURL:    "https://keycloak.example.org/realms/selene",
			UpdatedByID:  admin.ID,
		}
		if err := keycloakConfigRepo.Salvar(ctx, cfg); err != nil {
			t.Fatalf("falha ao salvar segunda versão: %v", err)
		}

		encontrado, err := keycloakConfigRepo.Buscar(ctx)
		if err != nil {
			t.Fatalf("erro inesperado ao buscar: %v", err)
		}
		if encontrado.ClientID != "selene-client-v2" {
			t.Fatalf("esperava upsert refletindo a versão mais recente, ClientID veio %q", encontrado.ClientID)
		}
		if encontrado.Audience != nil {
			t.Fatalf("esperava Audience nil (não foi reenviado na segunda versão), veio %q", *encontrado.Audience)
		}
	})

	t.Run("Buscar de um ID diferente de 1 nunca é alcançável (singleton reforçado pelo CHECK)", func(t *testing.T) {
		// Regressão de esquema: a constraint CHECK (id = 1) da migration
		// 000012 é o que realmente impede múltiplas linhas — Salvar já
		// força ID=1 no código Go, então este teste confirma que o
		// registro alcançável por Buscar é sempre o mesmo depois de
		// múltiplos Salvar (já coberto acima), não duplica.
		encontrado, err := keycloakConfigRepo.Buscar(ctx)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if encontrado.ID != 1 {
			t.Fatalf("esperava sempre ID=1, veio %d", encontrado.ID)
		}
	})
}
