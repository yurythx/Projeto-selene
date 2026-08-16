package testutil

// Dublês (fakes) em memória dos repositories, reutilizáveis por testes de
// service e de handler — não tocam banco, permitem montar a pilha real
// (handler -> service -> fake repository) sem precisar de Postgres nem de
// mocks gerados.

import (
	"context"
	"strings"

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
		if u.KeycloakID != nil && *u.KeycloakID == keycloakID {
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

func (f *FakeUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	for _, u := range f.Users {
		if u.Email == email {
			return u, nil
		}
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

func (f *FakeContratoRepository) ListTodos(ctx context.Context) ([]models.Contrato, error) {
	out := make([]models.Contrato, 0, len(f.Contratos))
	for _, c := range f.Contratos {
		out = append(out, *c)
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

func (f *FakeKanbanLogRepository) ListByProcessos(ctx context.Context, processoIDs []uuid.UUID) ([]models.KanbanLog, error) {
	porID := make(map[uuid.UUID]bool, len(processoIDs))
	for _, id := range processoIDs {
		porID[id] = true
	}
	var out []models.KanbanLog
	for _, l := range f.Logs {
		if porID[l.ProcessoPagamentoID] {
			out = append(out, l)
		}
	}
	return out, nil
}

var _ repository.KanbanLogRepository = (*FakeKanbanLogRepository)(nil)

// --- DocumentoEmitidoRepository ---

type FakeDocumentoEmitidoRepository struct {
	Documentos []models.DocumentoEmitido
}

func (f *FakeDocumentoEmitidoRepository) Create(ctx context.Context, documento *models.DocumentoEmitido) error {
	if documento.ID == uuid.Nil {
		documento.ID = uuid.New()
	}
	f.Documentos = append(f.Documentos, *documento)
	return nil
}

func (f *FakeDocumentoEmitidoRepository) FindByCodigoVerificacao(ctx context.Context, codigo string) (*models.DocumentoEmitido, error) {
	for _, d := range f.Documentos {
		if d.CodigoVerificacao == codigo {
			return &d, nil
		}
	}
	return nil, repository.ErrDocumentoEmitidoNotFound
}

func (f *FakeDocumentoEmitidoRepository) ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.DocumentoEmitido, error) {
	var out []models.DocumentoEmitido
	for _, d := range f.Documentos {
		if d.ContratoID == contratoID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *FakeDocumentoEmitidoRepository) ListByContratoIDs(ctx context.Context, contratoIDs []uuid.UUID) ([]models.DocumentoEmitido, error) {
	porID := make(map[uuid.UUID]bool, len(contratoIDs))
	for _, id := range contratoIDs {
		porID[id] = true
	}
	var out []models.DocumentoEmitido
	for _, d := range f.Documentos {
		if porID[d.ContratoID] {
			out = append(out, d)
		}
	}
	return out, nil
}

var _ repository.DocumentoEmitidoRepository = (*FakeDocumentoEmitidoRepository)(nil)

// --- VistoriaRepository ---

type FakeVistoriaRepository struct {
	Vistorias map[uuid.UUID]*models.RegistroVistoria
}

func NewFakeVistoriaRepository(vistorias ...*models.RegistroVistoria) *FakeVistoriaRepository {
	byID := make(map[uuid.UUID]*models.RegistroVistoria, len(vistorias))
	for _, v := range vistorias {
		byID[v.ID] = v
	}
	return &FakeVistoriaRepository{Vistorias: byID}
}

func (f *FakeVistoriaRepository) Create(ctx context.Context, vistoria *models.RegistroVistoria) error {
	if vistoria.ID == uuid.Nil {
		vistoria.ID = uuid.New()
	}
	f.Vistorias[vistoria.ID] = vistoria
	return nil
}

func (f *FakeVistoriaRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.RegistroVistoria, error) {
	if v, ok := f.Vistorias[id]; ok {
		return v, nil
	}
	return nil, repository.ErrVistoriaNotFound
}

func (f *FakeVistoriaRepository) ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.RegistroVistoria, error) {
	var out []models.RegistroVistoria
	for _, v := range f.Vistorias {
		if v.ProcessoPagamentoID == processoID {
			out = append(out, *v)
		}
	}
	return out, nil
}

var _ repository.VistoriaRepository = (*FakeVistoriaRepository)(nil)

// --- FotoVistoriaRepository ---

type FakeFotoVistoriaRepository struct {
	Fotos []models.FotoVistoria
}

func (f *FakeFotoVistoriaRepository) Create(ctx context.Context, foto *models.FotoVistoria) error {
	if foto.ID == uuid.Nil {
		foto.ID = uuid.New()
	}
	f.Fotos = append(f.Fotos, *foto)

	// Reflete a foto na vistoria em memória do FakeVistoriaRepository não é
	// responsabilidade deste fake (cada repository fake é independente,
	// como os reais) — os testes que precisam de Vistoria.Fotos populado
	// montam isso diretamente no fixture, igual RadarService faz com
	// Contrato/Fiscal.
	return nil
}

func (f *FakeFotoVistoriaRepository) FindByVistoriaAndHash(ctx context.Context, vistoriaID uuid.UUID, hash string) (*models.FotoVistoria, error) {
	for _, foto := range f.Fotos {
		if foto.VistoriaID == vistoriaID && foto.HashArquivo == hash {
			return &foto, nil
		}
	}
	return nil, repository.ErrFotoVistoriaNotFound
}

var _ repository.FotoVistoriaRepository = (*FakeFotoVistoriaRepository)(nil)

// --- PortariaDesignacaoRepository ---

type FakePortariaDesignacaoRepository struct {
	Designacoes map[uuid.UUID]*models.PortariaDesignacao
}

func NewFakePortariaDesignacaoRepository(designacoes ...*models.PortariaDesignacao) *FakePortariaDesignacaoRepository {
	byID := make(map[uuid.UUID]*models.PortariaDesignacao, len(designacoes))
	for _, d := range designacoes {
		byID[d.ID] = d
	}
	return &FakePortariaDesignacaoRepository{Designacoes: byID}
}

func (f *FakePortariaDesignacaoRepository) Create(ctx context.Context, designacao *models.PortariaDesignacao) error {
	if designacao.ID == uuid.Nil {
		designacao.ID = uuid.New()
	}
	f.Designacoes[designacao.ID] = designacao
	return nil
}

func (f *FakePortariaDesignacaoRepository) Update(ctx context.Context, designacao *models.PortariaDesignacao) error {
	f.Designacoes[designacao.ID] = designacao
	return nil
}

func (f *FakePortariaDesignacaoRepository) ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.PortariaDesignacao, error) {
	var out []models.PortariaDesignacao
	for _, d := range f.Designacoes {
		if d.ContratoID == contratoID {
			out = append(out, *d)
		}
	}
	return out, nil
}

func (f *FakePortariaDesignacaoRepository) FindAtivaPorContratoEPapel(ctx context.Context, contratoID uuid.UUID, papel models.PapelDesignacao) (*models.PortariaDesignacao, error) {
	for _, d := range f.Designacoes {
		if d.ContratoID == contratoID && d.Papel == papel && d.DataRevogacao == nil {
			return d, nil
		}
	}
	return nil, repository.ErrPortariaDesignacaoNotFound
}

var _ repository.PortariaDesignacaoRepository = (*FakePortariaDesignacaoRepository)(nil)

// --- EmpenhoRepository / MovimentacaoEmpenhoRepository ---

type FakeEmpenhoRepository struct {
	Empenhos map[uuid.UUID]*models.Empenho
}

func NewFakeEmpenhoRepository(empenhos ...*models.Empenho) *FakeEmpenhoRepository {
	byID := make(map[uuid.UUID]*models.Empenho, len(empenhos))
	for _, e := range empenhos {
		byID[e.ID] = e
	}
	return &FakeEmpenhoRepository{Empenhos: byID}
}

func (f *FakeEmpenhoRepository) Create(ctx context.Context, empenho *models.Empenho) error {
	if empenho.ID == uuid.Nil {
		empenho.ID = uuid.New()
	}
	f.Empenhos[empenho.ID] = empenho
	return nil
}

func (f *FakeEmpenhoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Empenho, error) {
	if e, ok := f.Empenhos[id]; ok {
		return e, nil
	}
	return nil, repository.ErrEmpenhoNotFound
}

func (f *FakeEmpenhoRepository) ListByContrato(ctx context.Context, contratoID uuid.UUID) ([]models.Empenho, error) {
	var out []models.Empenho
	for _, e := range f.Empenhos {
		if e.ContratoID == contratoID {
			out = append(out, *e)
		}
	}
	return out, nil
}

var _ repository.EmpenhoRepository = (*FakeEmpenhoRepository)(nil)

type FakeMovimentacaoEmpenhoRepository struct {
	Movimentacoes []models.MovimentacaoEmpenho
}

func (f *FakeMovimentacaoEmpenhoRepository) Create(ctx context.Context, movimentacao *models.MovimentacaoEmpenho) error {
	if movimentacao.ID == uuid.Nil {
		movimentacao.ID = uuid.New()
	}
	f.Movimentacoes = append(f.Movimentacoes, *movimentacao)
	return nil
}

func (f *FakeMovimentacaoEmpenhoRepository) ListByEmpenho(ctx context.Context, empenhoID uuid.UUID) ([]models.MovimentacaoEmpenho, error) {
	var out []models.MovimentacaoEmpenho
	for _, m := range f.Movimentacoes {
		if m.EmpenhoID == empenhoID {
			out = append(out, m)
		}
	}
	return out, nil
}

var _ repository.MovimentacaoEmpenhoRepository = (*FakeMovimentacaoEmpenhoRepository)(nil)

// --- OcorrenciaRepository ---

type FakeOcorrenciaRepository struct {
	Ocorrencias map[uuid.UUID]*models.Ocorrencia
}

func NewFakeOcorrenciaRepository(ocorrencias ...*models.Ocorrencia) *FakeOcorrenciaRepository {
	byID := make(map[uuid.UUID]*models.Ocorrencia, len(ocorrencias))
	for _, o := range ocorrencias {
		byID[o.ID] = o
	}
	return &FakeOcorrenciaRepository{Ocorrencias: byID}
}

func (f *FakeOcorrenciaRepository) Create(ctx context.Context, ocorrencia *models.Ocorrencia) error {
	if ocorrencia.ID == uuid.Nil {
		ocorrencia.ID = uuid.New()
	}
	f.Ocorrencias[ocorrencia.ID] = ocorrencia
	return nil
}

func (f *FakeOcorrenciaRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Ocorrencia, error) {
	if o, ok := f.Ocorrencias[id]; ok {
		return o, nil
	}
	return nil, repository.ErrOcorrenciaNotFound
}

func (f *FakeOcorrenciaRepository) Update(ctx context.Context, ocorrencia *models.Ocorrencia) error {
	f.Ocorrencias[ocorrencia.ID] = ocorrencia
	return nil
}

func (f *FakeOcorrenciaRepository) ListByProcesso(ctx context.Context, processoID uuid.UUID) ([]models.Ocorrencia, error) {
	var out []models.Ocorrencia
	for _, o := range f.Ocorrencias {
		if o.ProcessoPagamentoID != nil && *o.ProcessoPagamentoID == processoID {
			out = append(out, *o)
		}
	}
	return out, nil
}

func (f *FakeOcorrenciaRepository) ListAbertasPorProcesso(ctx context.Context, processoID uuid.UUID) ([]models.Ocorrencia, error) {
	var out []models.Ocorrencia
	for _, o := range f.Ocorrencias {
		if o.ProcessoPagamentoID != nil && *o.ProcessoPagamentoID == processoID && o.Estado != models.OcorrenciaRegularizada {
			out = append(out, *o)
		}
	}
	return out, nil
}

var _ repository.OcorrenciaRepository = (*FakeOcorrenciaRepository)(nil)

// --- ModeloDocumentoRepository / ModeloDocumentoVersaoRepository ---

type FakeModeloDocumentoRepository struct {
	Modelos map[uuid.UUID]*models.ModeloDocumento
}

func NewFakeModeloDocumentoRepository(modelos ...*models.ModeloDocumento) *FakeModeloDocumentoRepository {
	byID := make(map[uuid.UUID]*models.ModeloDocumento, len(modelos))
	for _, m := range modelos {
		byID[m.ID] = m
	}
	return &FakeModeloDocumentoRepository{Modelos: byID}
}

func (f *FakeModeloDocumentoRepository) Create(ctx context.Context, modelo *models.ModeloDocumento) error {
	if modelo.ID == uuid.Nil {
		modelo.ID = uuid.New()
	}
	for _, m := range f.Modelos {
		if strings.EqualFold(m.Categoria, modelo.Categoria) {
			return repository.ErrCategoriaModeloDuplicada
		}
	}
	f.Modelos[modelo.ID] = modelo
	return nil
}

func (f *FakeModeloDocumentoRepository) Update(ctx context.Context, modelo *models.ModeloDocumento) error {
	if modelo.Gatilho != nil {
		for id, m := range f.Modelos {
			if id != modelo.ID && m.Gatilho != nil && *m.Gatilho == *modelo.Gatilho {
				return repository.ErrGatilhoModeloJaAssociado
			}
		}
	}
	f.Modelos[modelo.ID] = modelo
	return nil
}

func (f *FakeModeloDocumentoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.ModeloDocumento, error) {
	if m, ok := f.Modelos[id]; ok {
		return m, nil
	}
	return nil, repository.ErrModeloDocumentoNotFound
}

func (f *FakeModeloDocumentoRepository) FindByCategoria(ctx context.Context, categoria string) (*models.ModeloDocumento, error) {
	for _, m := range f.Modelos {
		if strings.EqualFold(m.Categoria, categoria) {
			return m, nil
		}
	}
	return nil, repository.ErrModeloDocumentoNotFound
}

func (f *FakeModeloDocumentoRepository) List(ctx context.Context) ([]models.ModeloDocumento, error) {
	out := []models.ModeloDocumento{}
	for _, m := range f.Modelos {
		out = append(out, *m)
	}
	return out, nil
}

func (f *FakeModeloDocumentoRepository) FindAtivoByGatilho(ctx context.Context, gatilho models.TipoGatilhoModelo) (*models.ModeloDocumento, error) {
	for _, m := range f.Modelos {
		if m.Gatilho != nil && *m.Gatilho == gatilho {
			return m, nil
		}
	}
	return nil, repository.ErrModeloDocumentoNotFound
}

var _ repository.ModeloDocumentoRepository = (*FakeModeloDocumentoRepository)(nil)

type FakeModeloDocumentoVersaoRepository struct {
	Versoes map[uuid.UUID]*models.ModeloDocumentoVersao
}

func NewFakeModeloDocumentoVersaoRepository(versoes ...*models.ModeloDocumentoVersao) *FakeModeloDocumentoVersaoRepository {
	byID := make(map[uuid.UUID]*models.ModeloDocumentoVersao, len(versoes))
	for _, v := range versoes {
		byID[v.ID] = v
	}
	return &FakeModeloDocumentoVersaoRepository{Versoes: byID}
}

func (f *FakeModeloDocumentoVersaoRepository) Create(ctx context.Context, versao *models.ModeloDocumentoVersao) error {
	if versao.ID == uuid.Nil {
		versao.ID = uuid.New()
	}
	f.Versoes[versao.ID] = versao
	return nil
}

func (f *FakeModeloDocumentoVersaoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.ModeloDocumentoVersao, error) {
	if v, ok := f.Versoes[id]; ok {
		return v, nil
	}
	return nil, repository.ErrModeloDocumentoVersaoNotFound
}

var _ repository.ModeloDocumentoVersaoRepository = (*FakeModeloDocumentoVersaoRepository)(nil)
