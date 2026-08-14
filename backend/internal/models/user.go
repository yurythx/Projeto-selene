package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User representa um usuário do Selene, autenticado OU via Keycloak
// (OIDC — registro criado automaticamente por sincronização Just-In-Time
// na primeira requisição válida de um novo `sub`) OU por login local
// tradicional (usuário/senha — conta criada só por um administrador, sem
// autocadastro público; ver AuthService). Um usuário tem exatamente uma
// dessas origens: KeycloakID preenchido e PasswordHash nulo, ou vice-versa
// — nunca as duas (nem nenhuma).
type User struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey"`

	// KeycloakID é o claim "sub" do token OIDC — identificador único e
	// imutável do usuário no Keycloak. Tratado como string opaca (não como
	// uuid.UUID) porque o OIDC não garante esse formato para todo IdP.
	// Nulo para usuários locais (nunca logaram via Keycloak). json:"-": é
	// um identificador interno de correlação com o Keycloak, sem uso na
	// UI hoje — não tem por que ir pra fora em nenhuma resposta (inclusive
	// quando User aparece aninhado como Contrato.Fiscal em /contratos e
	// /processos, visível a QUALQUER usuário autenticado, não só
	// administradores).
	KeycloakID *string `gorm:"type:varchar(255);uniqueIndex" json:"-"`

	// PasswordHash (bcrypt) só é preenchido pra contas de login local —
	// nulo pra usuários provisionados via Keycloak. json:"-": um hash de
	// senha nunca deveria ir pra fora em resposta nenhuma, mesmo sendo
	// bcrypt (defesa em profundidade, não depende de "é seguro o
	// suficiente pra vazar").
	PasswordHash *string `gorm:"type:text" json:"-"`

	// MustChangePassword força a troca de senha no próximo login local —
	// true quando o admin definiu uma senha temporária na criação da
	// conta. Não tem efeito pra contas Keycloak. Aplicado como um "soft
	// gate" no FRONTEND (redireciona pra tela de troca antes de liberar
	// navegação) — o backend não bloqueia as demais rotas por este flag;
	// ver o comentário em AuthService sobre essa limitação conhecida.
	MustChangePassword bool `gorm:"not null;default:false"`

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
