package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"projeto-selene/internal/localauth"
	"projeto-selene/internal/models"
	"projeto-selene/internal/testutil"
)

func novoUsuarioLocalDeTeste(t *testing.T, email, senha string) *models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("falha ao gerar hash de senha para o fixture: %v", err)
	}
	hashStr := string(hash)
	return &models.User{
		ID:           uuid.New(),
		Nome:         "Fiscal Local",
		Email:        email,
		PasswordHash: &hashStr,
	}
}

func novoAuthServiceDeTeste(t *testing.T, usuarios ...*models.User) (*AuthService, *testutil.FakeUserRepository) {
	t.Helper()
	keys, err := localauth.NewKeyPair()
	if err != nil {
		t.Fatalf("falha ao gerar par de chaves: %v", err)
	}
	userRepo := testutil.NewFakeUserRepository(usuarios...)
	return NewAuthService(userRepo, keys), userRepo
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	usuarioLocal := novoUsuarioLocalDeTeste(t, "fiscal@teste.local", "senha-correta-123")
	keycloakID := "sub-keycloak-1"
	usuarioKeycloak := &models.User{ID: uuid.New(), Nome: "Fiscal Keycloak", Email: "keycloak@teste.local", KeycloakID: &keycloakID}

	svc, _ := novoAuthServiceDeTeste(t, usuarioLocal, usuarioKeycloak)

	t.Run("caminho feliz emite um token", func(t *testing.T) {
		resultado, err := svc.Login(ctx, "fiscal@teste.local", "senha-correta-123")
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if resultado.AccessToken == "" {
			t.Fatal("esperava um AccessToken não vazio")
		}
		if resultado.User.ID != usuarioLocal.ID {
			t.Errorf("User.ID = %v, esperado %v", resultado.User.ID, usuarioLocal.ID)
		}
	})

	t.Run("senha errada é rejeitada", func(t *testing.T) {
		_, err := svc.Login(ctx, "fiscal@teste.local", "senha-errada")
		if !errors.Is(err, ErrCredenciaisInvalidas) {
			t.Fatalf("esperava ErrCredenciaisInvalidas, veio %v", err)
		}
	})

	t.Run("e-mail inexistente é rejeitado com a mesma mensagem", func(t *testing.T) {
		_, err := svc.Login(ctx, "ninguem@teste.local", "qualquer-coisa")
		if !errors.Is(err, ErrCredenciaisInvalidas) {
			t.Fatalf("esperava ErrCredenciaisInvalidas, veio %v", err)
		}
	})

	t.Run("conta provisionada via Keycloak (sem senha local) é rejeitada", func(t *testing.T) {
		_, err := svc.Login(ctx, "keycloak@teste.local", "qualquer-coisa")
		if !errors.Is(err, ErrCredenciaisInvalidas) {
			t.Fatalf("esperava ErrCredenciaisInvalidas, veio %v", err)
		}
	})
}

func TestAuthService_CriarLocal(t *testing.T) {
	ctx := context.Background()
	svc, userRepo := novoAuthServiceDeTeste(t)

	t.Run("senha temporária fraca é rejeitada", func(t *testing.T) {
		_, err := svc.CriarLocal(ctx, CriarLocalInput{Nome: "X", Email: "x@teste.local", SenhaTemporaria: "curta"})
		if !errors.Is(err, ErrSenhaFraca) {
			t.Fatalf("esperava ErrSenhaFraca, veio %v", err)
		}
	})

	t.Run("caminho feliz cria a conta com MustChangePassword=true", func(t *testing.T) {
		user, err := svc.CriarLocal(ctx, CriarLocalInput{
			Nome:            "Novo Fiscal",
			Email:           "novo.fiscal@teste.local",
			SenhaTemporaria: "temporaria123",
			IsFiscal:        true,
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !user.MustChangePassword {
			t.Error("esperava MustChangePassword=true numa conta recém-criada por um admin")
		}
		if user.PasswordHash == nil {
			t.Fatal("esperava PasswordHash preenchido")
		}
		if bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte("temporaria123")) != nil {
			t.Error("o hash salvo não corresponde à senha temporária informada")
		}
		if !user.IsFiscal {
			t.Error("esperava IsFiscal=true (repassado do input)")
		}
		if user.KeycloakID != nil {
			t.Error("uma conta local não deveria ter KeycloakID")
		}
		if _, ok := userRepo.Users[user.ID]; !ok {
			t.Error("esperava o usuário persistido no repositório")
		}
	})
}

func TestAuthService_TrocarSenha(t *testing.T) {
	ctx := context.Background()

	t.Run("caminho feliz troca a senha e desliga MustChangePassword", func(t *testing.T) {
		usuario := novoUsuarioLocalDeTeste(t, "fiscal@teste.local", "senha-temporaria")
		usuario.MustChangePassword = true
		svc, _ := novoAuthServiceDeTeste(t, usuario)

		err := svc.TrocarSenha(ctx, usuario.ID, TrocarSenhaInput{SenhaAtual: "senha-temporaria", SenhaNova: "senha-nova-123"})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if usuario.MustChangePassword {
			t.Error("esperava MustChangePassword=false após a troca")
		}
		if bcrypt.CompareHashAndPassword([]byte(*usuario.PasswordHash), []byte("senha-nova-123")) != nil {
			t.Error("o hash não foi atualizado pra nova senha")
		}
	})

	t.Run("senha atual errada é rejeitada", func(t *testing.T) {
		usuario := novoUsuarioLocalDeTeste(t, "fiscal@teste.local", "senha-correta")
		svc, _ := novoAuthServiceDeTeste(t, usuario)

		err := svc.TrocarSenha(ctx, usuario.ID, TrocarSenhaInput{SenhaAtual: "senha-errada", SenhaNova: "senha-nova-123"})
		if !errors.Is(err, ErrCredenciaisInvalidas) {
			t.Fatalf("esperava ErrCredenciaisInvalidas, veio %v", err)
		}
	})

	t.Run("nova senha fraca é rejeitada", func(t *testing.T) {
		usuario := novoUsuarioLocalDeTeste(t, "fiscal@teste.local", "senha-correta")
		svc, _ := novoAuthServiceDeTeste(t, usuario)

		err := svc.TrocarSenha(ctx, usuario.ID, TrocarSenhaInput{SenhaAtual: "senha-correta", SenhaNova: "curta"})
		if !errors.Is(err, ErrSenhaFraca) {
			t.Fatalf("esperava ErrSenhaFraca, veio %v", err)
		}
	})

	t.Run("conta Keycloak (sem senha local) é rejeitada", func(t *testing.T) {
		keycloakID := "sub-1"
		usuario := &models.User{ID: uuid.New(), Nome: "Fiscal Keycloak", Email: "k@teste.local", KeycloakID: &keycloakID}
		svc, _ := novoAuthServiceDeTeste(t, usuario)

		err := svc.TrocarSenha(ctx, usuario.ID, TrocarSenhaInput{SenhaAtual: "qualquer", SenhaNova: "senha-nova-123"})
		if !errors.Is(err, ErrContaSemSenhaLocal) {
			t.Fatalf("esperava ErrContaSemSenhaLocal, veio %v", err)
		}
	})
}
