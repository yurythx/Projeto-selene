package localauth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"projeto-selene/internal/localauth"
)

func TestKeyPair_EmitirEVerificar(t *testing.T) {
	keys, err := localauth.NewKeyPair()
	if err != nil {
		t.Fatalf("falha ao gerar par de chaves: %v", err)
	}

	userID := uuid.New()
	tokenString, err := keys.Emitir(userID, "Fiscal Teste", "fiscal@teste.local")
	if err != nil {
		t.Fatalf("erro inesperado ao emitir: %v", err)
	}

	claims := &localauth.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return keys.PublicKey(), nil
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(localauth.Issuer))
	if err != nil {
		t.Fatalf("erro inesperado ao verificar token recém-emitido: %v", err)
	}
	if !token.Valid {
		t.Fatal("token deveria ser válido")
	}

	if claims.Subject != userID.String() {
		t.Errorf("Subject = %q, esperado %q", claims.Subject, userID.String())
	}
	if claims.Name != "Fiscal Teste" {
		t.Errorf("Name = %q, esperado \"Fiscal Teste\"", claims.Name)
	}
	if claims.Email != "fiscal@teste.local" {
		t.Errorf("Email = %q, esperado \"fiscal@teste.local\"", claims.Email)
	}
	if claims.Issuer != localauth.Issuer {
		t.Errorf("Issuer = %q, esperado %q", claims.Issuer, localauth.Issuer)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		t.Error("esperava um ExpiresAt no futuro")
	}
}

func TestKeyPair_TokenNaoValidaComOutraChave(t *testing.T) {
	keysA, err := localauth.NewKeyPair()
	if err != nil {
		t.Fatalf("falha ao gerar par de chaves A: %v", err)
	}
	keysB, err := localauth.NewKeyPair()
	if err != nil {
		t.Fatalf("falha ao gerar par de chaves B: %v", err)
	}

	tokenString, err := keysA.Emitir(uuid.New(), "Fiscal Teste", "fiscal@teste.local")
	if err != nil {
		t.Fatalf("erro inesperado ao emitir: %v", err)
	}

	_, err = jwt.ParseWithClaims(tokenString, &localauth.Claims{}, func(t *jwt.Token) (interface{}, error) {
		return keysB.PublicKey(), nil
	}, jwt.WithValidMethods([]string{"RS256"}))
	if err == nil {
		t.Fatal("esperava falha ao verificar um token com a chave pública errada")
	}
}
