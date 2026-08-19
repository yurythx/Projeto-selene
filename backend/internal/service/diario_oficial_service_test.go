package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// fakeDiarioOficialConfigRepository é um dublê em memória mínimo — mesmo
// espírito de fakeKeycloakConfigRepository (ver keycloak_config_service_test.go).
type fakeDiarioOficialConfigRepository struct {
	salvo *models.DiarioOficialConfig
}

func (f *fakeDiarioOficialConfigRepository) Buscar(ctx context.Context) (*models.DiarioOficialConfig, error) {
	if f.salvo == nil {
		return nil, repository.ErrDiarioOficialConfigNotFound
	}
	copia := *f.salvo
	return &copia, nil
}

func (f *fakeDiarioOficialConfigRepository) Salvar(ctx context.Context, cfg *models.DiarioOficialConfig) error {
	cfg.ID = 1
	copia := *cfg
	f.salvo = &copia
	return nil
}

func TestDiarioOficialService_Buscar(t *testing.T) {
	t.Run("sem configuração salva, devolve DTO vazio (sem erro)", func(t *testing.T) {
		repo := &fakeDiarioOficialConfigRepository{}
		svc := NewDiarioOficialService(repo)

		dto, err := svc.Buscar(context.Background())
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if dto.TemChaveConfigurada {
			t.Fatal("esperava TemChaveConfigurada=false sem configuração salva")
		}
		if dto.BaseURL != "" {
			t.Fatalf("esperava BaseURL vazia, veio %q", dto.BaseURL)
		}
	})

	t.Run("com configuração salva, devolve BaseURL e nunca a APIKey", func(t *testing.T) {
		repo := &fakeDiarioOficialConfigRepository{salvo: &models.DiarioOficialConfig{
			ID:          1,
			BaseURL:     "https://diario.example.gov.br/api",
			APIKey:      "chave-super-secreta",
			UpdatedByID: uuid.New(),
		}}
		svc := NewDiarioOficialService(repo)

		dto, err := svc.Buscar(context.Background())
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if dto.BaseURL != "https://diario.example.gov.br/api" {
			t.Fatalf("BaseURL inesperada: %q", dto.BaseURL)
		}
		if !dto.TemChaveConfigurada {
			t.Fatal("esperava TemChaveConfigurada=true")
		}
		// ConfiguracaoDiarioOficial nem declara um campo pra APIKey — a
		// garantia de "nunca volta pro cliente" é estrutural, mesma prova
		// por design já usada em ConfiguracaoKeycloak.
	})
}

func TestDiarioOficialService_Salvar(t *testing.T) {
	t.Run("base_url vazia/inválida é rejeitada", func(t *testing.T) {
		repo := &fakeDiarioOficialConfigRepository{}
		svc := NewDiarioOficialService(repo)

		err := svc.Salvar(context.Background(), uuid.New(), AtualizarConfiguracaoDiarioOficial{
			BaseURL: "não é uma url", APIKey: "x",
		})
		if !errors.Is(err, ErrDiarioOficialURLInvalida) {
			t.Fatalf("esperava ErrDiarioOficialURLInvalida, veio %v", err)
		}
	})

	t.Run("api_key vazia na primeira configuração é rejeitada", func(t *testing.T) {
		repo := &fakeDiarioOficialConfigRepository{}
		svc := NewDiarioOficialService(repo)

		err := svc.Salvar(context.Background(), uuid.New(), AtualizarConfiguracaoDiarioOficial{
			BaseURL: "https://diario.example.gov.br/api", APIKey: "",
		})
		if !errors.Is(err, ErrDiarioOficialChaveObrigatoria) {
			t.Fatalf("esperava ErrDiarioOficialChaveObrigatoria, veio %v", err)
		}
	})

	t.Run("caminho feliz: salva, e api_key em branco mantém a atual numa atualização seguinte", func(t *testing.T) {
		repo := &fakeDiarioOficialConfigRepository{}
		svc := NewDiarioOficialService(repo)
		adminID := uuid.New()

		err := svc.Salvar(context.Background(), adminID, AtualizarConfiguracaoDiarioOficial{
			BaseURL: "https://diario.example.gov.br/api/", APIKey: "chave-inicial",
		})
		if err != nil {
			t.Fatalf("erro inesperado no caminho feliz: %v", err)
		}
		if repo.salvo == nil || repo.salvo.APIKey != "chave-inicial" {
			t.Fatalf("esperava APIKey persistida, veio %+v", repo.salvo)
		}
		// Barra final removida (ver s.Salvar) — evita "//contratos" ao
		// concatenar em BuscarContratos.
		if repo.salvo.BaseURL != "https://diario.example.gov.br/api" {
			t.Fatalf("esperava barra final removida, veio %q", repo.salvo.BaseURL)
		}

		err = svc.Salvar(context.Background(), adminID, AtualizarConfiguracaoDiarioOficial{
			BaseURL: "https://diario-novo.example.gov.br/api", APIKey: "",
		})
		if err != nil {
			t.Fatalf("erro inesperado na atualização: %v", err)
		}
		if repo.salvo.BaseURL != "https://diario-novo.example.gov.br/api" {
			t.Fatalf("esperava BaseURL atualizada, veio %q", repo.salvo.BaseURL)
		}
		if repo.salvo.APIKey != "chave-inicial" {
			t.Fatalf("esperava APIKey mantida (\"chave-inicial\") quando enviada em branco, veio %q", repo.salvo.APIKey)
		}
	})
}

func TestDiarioOficialService_TestarConexao(t *testing.T) {
	t.Run("sem configuração salva", func(t *testing.T) {
		repo := &fakeDiarioOficialConfigRepository{}
		svc := NewDiarioOficialService(repo)

		_, err := svc.TestarConexao(context.Background())
		if !errors.Is(err, ErrDiarioOficialNaoConfigurado) {
			t.Fatalf("esperava ErrDiarioOficialNaoConfigurado, veio %v", err)
		}
	})

	t.Run("servidor responde (mesmo com 404) conta como sucesso — não validamos o schema", func(t *testing.T) {
		var recebeuAuth string
		servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recebeuAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusNotFound)
		}))
		defer servidor.Close()

		repo := &fakeDiarioOficialConfigRepository{salvo: &models.DiarioOficialConfig{
			ID: 1, BaseURL: servidor.URL, APIKey: "chave-teste", UpdatedByID: uuid.New(),
		}}
		svc := NewDiarioOficialService(repo)

		resultado, err := svc.TestarConexao(context.Background())
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !resultado.Sucesso {
			t.Fatalf("esperava Sucesso=true (servidor respondeu), veio %+v", resultado)
		}
		if resultado.StatusHTTP != http.StatusNotFound {
			t.Fatalf("esperava StatusHTTP 404, veio %d", resultado.StatusHTTP)
		}
		if recebeuAuth != "Bearer chave-teste" {
			t.Fatalf("esperava header Authorization Bearer, veio %q", recebeuAuth)
		}
	})

	t.Run("servidor inalcançável conta como falha", func(t *testing.T) {
		repo := &fakeDiarioOficialConfigRepository{salvo: &models.DiarioOficialConfig{
			ID: 1, BaseURL: "http://endereco-que-nao-existe.invalid", APIKey: "x", UpdatedByID: uuid.New(),
		}}
		svc := NewDiarioOficialService(repo)

		resultado, err := svc.TestarConexao(context.Background())
		if err != nil {
			t.Fatalf("erro inesperado (a falha vai no campo Sucesso, não em err): %v", err)
		}
		if resultado.Sucesso {
			t.Fatal("esperava Sucesso=false pra um host inalcançável")
		}
		if resultado.Erro == "" {
			t.Fatal("esperava Erro preenchido")
		}
	})
}

func TestDiarioOficialService_BuscarContratos(t *testing.T) {
	t.Run("sem configuração salva", func(t *testing.T) {
		repo := &fakeDiarioOficialConfigRepository{}
		svc := NewDiarioOficialService(repo)

		_, err := svc.BuscarContratos(context.Background(), FiltroBuscaContratos{Nome: "Fulano"})
		if !errors.Is(err, ErrDiarioOficialNaoConfigurado) {
			t.Fatalf("esperava ErrDiarioOficialNaoConfigurado, veio %v", err)
		}
	})

	t.Run("repassa os 3 filtros na query string e devolve o JSON decodificado", func(t *testing.T) {
		var queryRecebida string
		servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			queryRecebida = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"resultados": []map[string]string{{"contratada_nome": "Fulano de Tal"}},
			})
		}))
		defer servidor.Close()

		repo := &fakeDiarioOficialConfigRepository{salvo: &models.DiarioOficialConfig{
			ID: 1, BaseURL: servidor.URL, APIKey: "chave-teste", UpdatedByID: uuid.New(),
		}}
		svc := NewDiarioOficialService(repo)

		resultado, err := svc.BuscarContratos(context.Background(), FiltroBuscaContratos{
			Nome: "Fulano de Tal", CPF: "111.111.111-11", Data: "2026-08-18",
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}

		mapa, ok := resultado.(map[string]any)
		if !ok {
			t.Fatalf("esperava um map decodificado, veio %T", resultado)
		}
		if _, ok := mapa["resultados"]; !ok {
			t.Fatalf("esperava a chave \"resultados\" no JSON decodificado, veio %+v", mapa)
		}

		if queryRecebida == "" {
			t.Fatal("esperava query string com os filtros")
		}
	})

	t.Run("API externa fora do ar vira ErrDiarioOficialFalhaNaBusca", func(t *testing.T) {
		repo := &fakeDiarioOficialConfigRepository{salvo: &models.DiarioOficialConfig{
			ID: 1, BaseURL: "http://endereco-que-nao-existe.invalid", APIKey: "x", UpdatedByID: uuid.New(),
		}}
		svc := NewDiarioOficialService(repo)

		_, err := svc.BuscarContratos(context.Background(), FiltroBuscaContratos{})
		if !errors.Is(err, ErrDiarioOficialFalhaNaBusca) {
			t.Fatalf("esperava ErrDiarioOficialFalhaNaBusca, veio %v", err)
		}
	})

	t.Run("resposta que não é JSON válido vira ErrDiarioOficialFalhaNaBusca", func(t *testing.T) {
		servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("isto não é json"))
		}))
		defer servidor.Close()

		repo := &fakeDiarioOficialConfigRepository{salvo: &models.DiarioOficialConfig{
			ID: 1, BaseURL: servidor.URL, APIKey: "x", UpdatedByID: uuid.New(),
		}}
		svc := NewDiarioOficialService(repo)

		_, err := svc.BuscarContratos(context.Background(), FiltroBuscaContratos{})
		if !errors.Is(err, ErrDiarioOficialFalhaNaBusca) {
			t.Fatalf("esperava ErrDiarioOficialFalhaNaBusca, veio %v", err)
		}
	})
}
