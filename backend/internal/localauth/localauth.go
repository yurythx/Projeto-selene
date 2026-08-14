// Package localauth implementa a emissão/verificação dos tokens de acesso
// do login tradicional (usuário/senha) — a alternativa ao Keycloak.
//
// É um pacote-folha de propósito: tanto internal/service (AuthService, que
// EMITE tokens no login) quanto internal/middleware (que VERIFICA tokens
// em toda requisição) precisam do mesmo par de chaves RSA; colocar o tipo
// aqui, sem depender de nenhum dos dois, evita um ciclo de import entre
// eles.
package localauth

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Issuer é o valor do claim "iss" emitido pra tokens de login local —
// distinto do issuer do Keycloak (uma URL do realm da prefeitura). O
// middleware usa isso pra saber, depois de validar a assinatura, que tipo
// de conta resolver: usuário local por ID direto (FindByID — a conta já
// existe, criada por um admin), não a sincronização JIT usada pra contas
// Keycloak.
const Issuer = "selene-login-local"

// tokenTTL: um turno de trabalho. Sem refresh token pra contas locais —
// diferente do fluxo Keycloak (que tem refresh_token de verdade), aqui é
// simplesmente fazer login de novo quando expirar. Login local é a via
// secundária (contas criadas por admin, uso ocasional/fallback), não vale
// a complexidade de replicar o ciclo de refresh do OAuth pra ela.
const tokenTTL = 8 * time.Hour

// Claims usa a mesma forma que middleware.Claims espera (Name/Email +
// RegisteredClaims) — os dois tipos são estruturalmente idênticos de
// propósito, pra middleware.Claims conseguir decodificar tanto tokens do
// Keycloak quanto tokens locais sem precisar de dois parses diferentes.
type Claims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// KeyPair gera e guarda em memória o par de chaves RSA usado pra
// assinar/verificar os tokens de login local — nunca persistido em disco
// nem em variável de ambiente.
//
// LIMITAÇÃO CONHECIDA: a chave é gerada uma vez por processo (main.go, na
// inicialização) e nunca persiste — todo reinício/redeploy do backend
// invalida silenciosamente as sessões locais ativas (o usuário só precisa
// logar de novo; contas Keycloak não são afetadas, usam a infraestrutura
// de chaves do Keycloak, inteiramente separada). Aceitável pro volume
// esperado de contas locais (poucas, via de acesso secundária) —
// documentado aqui em vez de adicionar gestão de chave persistente
// (arquivo/secret externo) sem necessidade comprovada.
type KeyPair struct {
	private *rsa.PrivateKey
}

// NewKeyPair gera um novo par de chaves RSA-2048.
func NewKeyPair() (*KeyPair, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("localauth: gerar par de chaves RSA: %w", err)
	}
	return &KeyPair{private: key}, nil
}

// PublicKey expõe a chave pública — usada pelo middleware pra verificar a
// assinatura dos tokens emitidos por Emitir.
func (k *KeyPair) PublicKey() *rsa.PublicKey {
	return &k.private.PublicKey
}

// Emitir assina um novo token de acesso pro usuário local informado.
func (k *KeyPair) Emitir(userID uuid.UUID, nome, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		Name:  nome,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(k.private)
	if err != nil {
		return "", fmt.Errorf("localauth: assinar token: %w", err)
	}

	return signed, nil
}
