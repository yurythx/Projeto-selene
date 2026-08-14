package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User representa um usuário autenticado via Keycloak (OIDC). O registro é
// criado automaticamente (sincronização Just-In-Time) pelo middleware de
// autenticação na primeira requisição válida de um novo `sub`.
type User struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// KeycloakID é o claim "sub" do token OIDC — identificador único e
	// imutável do usuário no Keycloak. Tratado como string opaca (não como
	// uuid.UUID) porque o OIDC não garante esse formato para todo IdP.
	// json:"-": é um identificador interno de correlação com o Keycloak,
	// sem uso na UI hoje — não tem por que ir pra fora em nenhuma resposta
	// (inclusive quando User aparece aninhado como Contrato.Fiscal em
	// /contratos e /processos, visível a QUALQUER usuário autenticado, não
	// só administradores).
	KeycloakID string `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"`

	Nome  string `gorm:"type:varchar(255);not null"`
	Email string `gorm:"type:varchar(255);uniqueIndex;not null"`

	// IsFiscal delega permissão de escrita/movimentação no Kanban.
	// Usuários recém-provisionados via JIT começam como false
	// (princípio do menor privilégio) até um administrador liberar.
	IsFiscal bool `gorm:"not null;default:false"`

	// IsAdmin delega permissão para administrar contas de outros usuários
	// (rotas /admin/users), incluindo alterar IsFiscal e o próprio IsAdmin.
	// Assim como IsFiscal, começa false na criação JIT — o primeiro admin
	// do sistema precisa ser promovido manualmente (ex: UPDATE direto no
	// Postgres), já que não há como uma rota autenticada conceder o
	// primeiro admin a si mesma.
	IsAdmin bool `gorm:"not null;default:false"`

	// Matricula é opcional na criação JIT — pode ser completada depois
	// pelo próprio usuário ou por um administrador.
	Matricula string `gorm:"type:varchar(50)"`

	CriadoEm     time.Time `gorm:"column:criado_em;autoCreateTime;not null"`
	AtualizadoEm time.Time `gorm:"column:atualizado_em;autoUpdateTime;not null"`
}

// TableName fixa o nome da tabela em português, conforme o domínio Selene.
func (User) TableName() string {
	return "users"
}

// BeforeCreate gera explicitamente o UUID da PK em código Go, em vez de
// depender de uma extensão do Postgres (uuid-ossp/pgcrypto) para isso —
// mantém a geração de identificadores visível e sob controle da aplicação.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
