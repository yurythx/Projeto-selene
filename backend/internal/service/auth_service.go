package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"projeto-selene/internal/localauth"
	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// senhaMinimaLen é o tamanho mínimo exigido pra qualquer senha local nova
// (definida por um admin na criação da conta ou pelo próprio usuário na
// troca) — checagem simples de propósito (comprimento, não regra de
// complexidade), suficiente pra um sistema interno de uso municipal sem
// introduzir uma biblioteca de política de senha só pra isso.
const senhaMinimaLen = 8

// dummyPasswordHash é comparado no lugar do hash real quando o e-mail
// informado no login não existe (ou pertence a uma conta Keycloak, sem
// senha local) — SEM isso, pular a chamada bcrypt (deliberadamente lenta)
// nesse caso faria a resposta voltar mais rápido que quando o e-mail
// existe, vazando por TEMPO DE RESPOSTA se um e-mail está cadastrado no
// sistema, mesmo a mensagem de erro sendo idêntica nos dois casos. Não é
// uma credencial real: é bcrypt("preenchimento-de-tempo-constante"), só
// pra ter algo pra comparar.
//
//nolint:gosec // G101: falso positivo do gosec (achou que parece uma credencial).
const dummyPasswordHash = "$2a$10$CwTycUXWue0Thq9StjUM0uJ8gU9mAWvSCSdKk9wfjNPX9WeQb0nz2"

// AuthService implementa o login tradicional (usuário/senha) — a
// alternativa ao Keycloak. Contas locais são criadas só por um
// administrador (ver CriarLocal); não há autocadastro público.
type AuthService struct {
	userRepo repository.UserRepository
	keys     *localauth.KeyPair
}

// NewAuthService constrói um AuthService.
func NewAuthService(userRepo repository.UserRepository, keys *localauth.KeyPair) *AuthService {
	return &AuthService{userRepo: userRepo, keys: keys}
}

// LoginResult é devolvido por Login em caso de sucesso.
type LoginResult struct {
	AccessToken string
	User        *models.User
}

// Login autentica por e-mail/senha e, se válido, emite um token de acesso
// aceito pelo middleware de autenticação (ver internal/localauth e
// internal/middleware.NewAuthMiddleware) — do ponto de vista do resto da
// API, é indistinguível de um token do Keycloak: mesmo formato de claims,
// mesmo algoritmo, só o issuer muda.
func (s *AuthService) Login(ctx context.Context, email, senha string) (*LoginResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return nil, fmt.Errorf("service: buscar usuário para login: %w", err)
	}

	// hash começa como o de preenchimento (ver o comentário em
	// dummyPasswordHash) — só é trocado pelo hash real se a conta existir
	// E tiver senha local. bcrypt.CompareHashAndPassword roda SEMPRE,
	// mesmo quando já sabemos que vai falhar, pra manter o tempo de
	// resposta uniforme.
	hash := dummyPasswordHash
	temSenhaLocal := user != nil && user.PasswordHash != nil
	if temSenhaLocal {
		hash = *user.PasswordHash
	}
	senhaValida := bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) == nil

	if user == nil || !temSenhaLocal || !senhaValida {
		return nil, ErrCredenciaisInvalidas
	}

	token, err := s.keys.Emitir(user.ID, user.Nome, user.Email)
	if err != nil {
		return nil, fmt.Errorf("service: emitir token de login local: %w", err)
	}

	return &LoginResult{AccessToken: token, User: user}, nil
}

// CriarLocalInput agrupa os dados de uma conta local nova — sempre criada
// por um administrador, nunca por autocadastro.
type CriarLocalInput struct {
	Nome            string
	Email           string
	SenhaTemporaria string
	IsFiscal        bool
	IsAdmin         bool
}

// CriarLocal cria uma conta de login local com uma senha temporária —
// MustChangePassword nasce true, forçando a troca no primeiro login (ver
// o comentário em models.User.MustChangePassword sobre isso ser reforçado
// só no frontend, não no backend).
func (s *AuthService) CriarLocal(ctx context.Context, input CriarLocalInput) (*models.User, error) {
	if len(input.SenhaTemporaria) < senhaMinimaLen {
		return nil, ErrSenhaFraca
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.SenhaTemporaria), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("service: hash da senha temporária: %w", err)
	}
	hashStr := string(hash)

	user := &models.User{
		Nome:               input.Nome,
		Email:              input.Email,
		PasswordHash:       &hashStr,
		MustChangePassword: true,
		IsFiscal:           input.IsFiscal,
		IsAdmin:            input.IsAdmin,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("service: criar usuário local: %w", err)
	}

	return user, nil
}

// TrocarSenhaInput agrupa os dados da troca de senha — exige a senha
// atual (mesmo na troca obrigatória de primeiro login, cuja "senha atual"
// é a temporária que o admin definiu) pra confirmar que quem está
// trocando é de fato o dono da sessão, não alguém que sequestrou um
// access token de curta duração.
type TrocarSenhaInput struct {
	SenhaAtual string
	SenhaNova  string
}

// TrocarSenha troca a senha de uma conta local autenticada.
func (s *AuthService) TrocarSenha(ctx context.Context, userID uuid.UUID, input TrocarSenhaInput) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.PasswordHash == nil {
		return ErrContaSemSenhaLocal
	}
	if bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(input.SenhaAtual)) != nil {
		return ErrCredenciaisInvalidas
	}
	if len(input.SenhaNova) < senhaMinimaLen {
		return ErrSenhaFraca
	}

	novoHash, err := bcrypt.GenerateFromPassword([]byte(input.SenhaNova), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("service: hash da nova senha: %w", err)
	}
	novoHashStr := string(novoHash)

	user.PasswordHash = &novoHashStr
	user.MustChangePassword = false

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("service: salvar nova senha: %w", err)
	}

	return nil
}
