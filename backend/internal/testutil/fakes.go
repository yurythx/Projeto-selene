package testutil

// Dublês (fakes) em memória dos repositories, reutilizáveis por testes de
// service e de handler — não tocam banco, permitem montar a pilha real
// (handler -> service -> fake repository) sem precisar de Postgres nem de
// mocks gerados.

import (
	"context"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// --- UserRepository ---

type FakeUserRepository struct {
	Users map[uuid.UUID]*models.User
}

func NewFakeUserRepository(users ...*models.User) *FakeUserRepository {
	byID := make(map[uuid.UUID]*models.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	return &FakeUserRepository{Users: byID}
}

func (f *FakeUserRepository) FindByKeycloakID(ctx context.Context, keycloakID string) (*models.User, error) {
	for _, u := range f.Users {
		if u.KeycloakID == keycloakID {
			return u, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func (f *FakeUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if u, ok := f.Users[id]; ok {
		return u, nil
	}
	return nil, repository.ErrUserNotFound
}

func (f *FakeUserRepository) List(ctx context.Context) ([]models.User, error) {
	out := make([]models.User, 0, len(f.Users))
	for _, u := range f.Users {
		out = append(out, *u)
	}
	return out, nil
}

func (f *FakeUserRepository) Create(ctx context.Context, user *models.User) error {
	f.Users[user.ID] = user
	return nil
}

func (f *FakeUserRepository) Update(ctx context.Context, user *models.User) error {
	f.Users[user.ID] = user
	return nil
}

var _ repository.UserRepository = (*FakeUserRepository)(nil)

// --- ContratoRepository ---

type FakeContratoRepository struct {
	Contratos map[uuid.UUID]*models.Contrato
}

func NewFakeContratoRepository(contratos ...*models.Contrato) *FakeContratoRepository {
	byID := make(map[uuid.UUID]*models.Contrato, len(contratos))
	for _, c := range contratos {
		byID[c.ID] = c
	}
	return &FakeContratoRepository{Contratos: byID}
}

func (f *FakeContratoRepository) Create(ctx context.Context, contrato *models.Contrato) error {
	if contrato.ID == uuid.Nil {
		contrato.ID = uuid.New()
	}
	f.Contratos[contrato.ID] = contrato
	return nil
}

func (f *FakeContratoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Contrato, error) {
	if c, ok := f.Contratos[id]; ok {
		return c, nil
	}
	return nil, repository.ErrContratoNotFound
}

func (f *FakeContratoRepository) List(ctx context.Context, pagina repository.Pagina) (repository.ResultadoPaginado[models.Contrato], error) {
	pagina = pagina.Normalizada()

	out := make([]models.Contrato, 0, len(f.Contratos))
	for _, c := range f.Contratos {
		out = append(out, *c)
	}

	total := int64(len(out))
	inicio := pagina.Offset()
	if inicio > len(out) {
		inicio = len(out)
	}
	fim := inicio + pagina.Tamanho
	if fim > len(out) {
		fim = len(out)
	}

	return repository.ResultadoPaginado[models.Contrato]{
		Dados:         out[inicio:fim],
		Total:         total,
		Pagina:        pagina.Numero,
		TamanhoPagina: pagina.Tamanho,
	}, nil
}

func (f *FakeContratoRepository) Update(ctx context.Context, contrato *models.Contrato) error {
	f.Contratos[contrato.ID] = contrato
	return nil
}

func (f *FakeContratoRepository) ListAtivos(ctx context.Context) ([]models.Contrato, error) {
	var out []models.Contrato
	for _, c := range f.Contratos {
		if c.Ativo {
			out = append(out, *c)
		}
	}
	return out, nil
}

var _ repository.ContratoRepository = (*FakeContratoRepository)(nil)

// --- TipoDocumentoRepository ---

type FakeTipoDocumentoRepository struct {
	Tipos map[int]*models.TipoDocumento
}

func NewFakeTipoDocumentoRepository(tipos ...*models.TipoDocumento) *FakeTipoDocumentoRepository {
	byID := make(map[int]*models.TipoDocumento, len(tipos))
	for _, t := range tipos {
		byID[t.ID] = t
	}
	return &FakeTipoDocumentoRepository{Tipos: byID}
}

func (f *FakeTipoDocumentoRepository) List(ctx context.Context) ([]models.TipoDocumento, error) {
	out := make([]models.TipoDocumento, 0, len(f.Tipos))
	for _, t := range f.Tipos {
		out = append(out, *t)
	}
	return out, nil
}

func (f *FakeTipoDocumentoRepository) FindByID(ctx context.Context, id int) (*models.TipoDocumento, error) {
	if t, ok := f.Tipos[id]; ok {
		return t, nil
	}
	return nil, repository.ErrTipoDocumentoNotFound
}

func (f *FakeTipoDocumentoRepository) FindByNome(ctx context.Context, nome string) (*models.TipoDocumento, error) {
	for _, t := range f.Tipos {
		if t.Nome == nome {
			return t, nil
		}
	}
	return nil, repository.ErrTipoDocumentoNotFound
}

var _ repository.TipoDocumentoRepository = (*FakeTipoDocumentoRepository)(nil)

// --- KanbanEtapaRepository ---

type FakeKanbanEtapaRepository struct {
	Etapas map[int]*models.KanbanEtapa
}

func NewFakeKanbanEtapaRepository(etapas ...*models.KanbanEtapa) *FakeKanbanEtapaRepository {
	byID := make(map[int]*models.KanbanEtapa, len(etapas))
	for _, e := range etapas {
		byID[e.ID] = e
	}
	return &FakeKanbanEtapaRepository{Etapas: byID}
}

func (f *FakeKanbanEtapaRepository) List(ctx context.Context) ([]models.KanbanEtapa, error) {
	out := make([]models.KanbanEtapa, 0, len(f.Etapas))
	for _, e := range f.Etapas {
		out = append(out, *e)
	}
	return out, nil
}

func (f *FakeKanbanEtapaRepository) FindByID(ctx context.Context, id int) (*models.KanbanEtapa, error) {
	if e, ok := f.Etapas[id]; ok {
		return e, nil
	}
	return nil, repository.ErrEtapaNotFound
}

var _ repository.KanbanEtapaRepository = (*FakeKanbanEtapaRepository)(nil)

// --- ProcessoPagamentoRepository ---

type FakeProcessoPagamentoRepository struct {
	Processos map[uuid.UUID]*models.ProcessoPagamento
}

func NewFakeProcessoPagamentoRepository(processos ...*models.ProcessoPagamento) *FakeProcessoPagamentoRepository {
	byID := make(map[uuid.UUID]*models.ProcessoPagamento, len(processos))
	for _, p := range processos {
		byID[p.ID] = p
	}
	return &FakeProcessoPagamentoRepository{Processos: byID}
}

func (f *FakeProcessoPagamentoRepository) Create(ctx context.Context, processo *models.ProcessoPagamento) error {
	if processo.ID == uuid.Nil {
		processo.ID = uuid.New()
	}
	f.Processos[processo.ID] = processo
	return nil
}

func (f *FakeProcessoPagamentoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.ProcessoPagamento, error) {
	if p, ok := f.Processos[id]; ok {
		return p, nil
	}
	return nil, repository.ErrProcessoNotFound
}

func (f *FakeProcessoPagamentoRepository) ListByEtapa(ctx context.Context, etapaID int, pagina repository.Pagina) (repository.ResultadoPaginado[models.ProcessoPagamento], error) {
	pagina = pagina.Normalizada()

	var todos []models.ProcessoPagamento
	for _, p := range f.Processos {
		if p.EtapaAtualID == etapaID {
			todos = append(todos, *p)
		}
	}

	total := int64(len(todos))
	inicio := pagina.Offset()
	if inicio > len(todos) {
		inicio = len(todos)
	}
	fim := inicio + pagina.Tamanho
	if fim > len(todos) {
		fim = len(todos)
	}

	return repository.ResultadoPaginado[models.ProcessoPagamento]{
		Dados:         todos[inicio:fim],
		Total:         total,
		Pagina:        pagina.Numero,
		TamanhoPagina: pagina.Tamanho,
	}, nil
}

func (f *FakeProcessoPagamentoRepository) ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.ProcessoPagamento, error) {
	var out []models.ProcessoPagamento
	for _, p := range f.Processos {
		if p.ContratoID == contratoID {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (f *FakeProcessoPagamentoRepository) Update(ctx context.Context, processo *models.ProcessoPagamento) error {
	f.Processos[processo.ID] = processo
	return nil
}

func (f *FakeProcessoPagamentoRepository) ListAtivosComContrato(ctx context.Context) ([]models.ProcessoPagamento, error) {
	var out []models.ProcessoPagamento
	for _, p := range f.Processos {
		if p.Status == models.StatusProcessoAtivo {
			out = append(out, *p)
		}
	}
	return out, nil
}

var _ repository.ProcessoPagamentoRepository = (*FakeProcessoPagamentoRepository)(nil)

// --- DocumentoAnexoRepository ---

type FakeDocumentoAnexoRepository struct {
	Documentos []models.DocumentoAnexo
}

func (f *FakeDocumentoAnexoRepository) Create(ctx context.Context, documento *models.DocumentoAnexo) error {
	if documento.ID == uuid.Nil {
		documento.ID = uuid.New()
	}
	f.Documentos = append(f.Documentos, *documento)
	return nil
}

func (f *FakeDocumentoAnexoRepository) ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.DocumentoAnexo, error) {
	var out []models.DocumentoAnexo
	for _, d := range f.Documentos {
		if d.ProcessoPagamentoID == processoID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *FakeDocumentoAnexoRepository) FindByProcessoAndHash(ctx context.Context, processoID uuid.UUID, hash string) (*models.DocumentoAnexo, error) {
	for _, d := range f.Documentos {
		if d.ProcessoPagamentoID == processoID && d.HashArquivo == hash {
			return &d, nil
		}
	}
	return nil, repository.ErrDocumentoNotFound
}

var _ repository.DocumentoAnexoRepository = (*FakeDocumentoAnexoRepository)(nil)

// --- KanbanLogRepository ---

type FakeKanbanLogRepository struct {
	Logs []models.KanbanLog
}

func (f *FakeKanbanLogRepository) Create(ctx context.Context, log *models.KanbanLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	f.Logs = append(f.Logs, *log)
	return nil
}

func (f *FakeKanbanLogRepository) ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.KanbanLog, error) {
	var out []models.KanbanLog
	for _, l := range f.Logs {
		if l.ProcessoPagamentoID == processoID {
			out = append(out, l)
		}
	}
	return out, nil
}

var _ repository.KanbanLogRepository = (*FakeKanbanLogRepository)(nil)
