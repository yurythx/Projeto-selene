package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/google/uuid"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
	"projeto-selene/internal/testutil"
)

// pngDeTeste gera um PNG 1x1 válido em memória — o suficiente pra
// RegisterImageOptionsReader (usado em GerarRelatorioCampo) aceitar sem
// precisar de um arquivo de imagem real no repositório de testes.
func pngDeTeste(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("falha ao gerar PNG de teste: %v", err)
	}
	return buf.Bytes()
}

func TestVistoriaService_Registrar(t *testing.T) {
	ctx := context.Background()
	fiscalID := uuid.New()
	processo := &models.ProcessoPagamento{ID: uuid.New()}

	processoRepo := testutil.NewFakeProcessoPagamentoRepository(processo)
	vistoriaRepo := testutil.NewFakeVistoriaRepository()
	svc := NewVistoriaService(vistoriaRepo, &testutil.FakeFotoVistoriaRepository{}, processoRepo, t.TempDir())

	t.Run("processo inexistente é rejeitado", func(t *testing.T) {
		_, err := svc.Registrar(ctx, RegistrarInput{ProcessoPagamentoID: uuid.New(), FiscalID: fiscalID})
		if !errors.Is(err, repository.ErrProcessoNotFound) {
			t.Fatalf("esperava ErrProcessoNotFound, veio %v", err)
		}
	})

	t.Run("caminho feliz registra a vistoria, com e sem geolocalização", func(t *testing.T) {
		lat, lng := -16.4673, -54.6372
		vistoria, err := svc.Registrar(ctx, RegistrarInput{
			ProcessoPagamentoID: processo.ID,
			FiscalID:            fiscalID,
			Latitude:            &lat,
			Longitude:           &lng,
			Observacoes:         "Tudo conforme o especificado.",
		})
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if vistoria.ID == uuid.Nil {
			t.Fatal("esperava um ID gerado")
		}
		if vistoria.Latitude == nil || *vistoria.Latitude != lat {
			t.Errorf("Latitude = %v, esperado %v", vistoria.Latitude, lat)
		}

		semGeo, err := svc.Registrar(ctx, RegistrarInput{ProcessoPagamentoID: processo.ID, FiscalID: fiscalID})
		if err != nil {
			t.Fatalf("erro inesperado (sem geolocalização): %v", err)
		}
		if semGeo.Latitude != nil || semGeo.Longitude != nil {
			t.Error("esperava Latitude/Longitude nil quando não informadas (geolocalização negada/indisponível)")
		}
	})
}

func TestVistoriaService_AnexarFoto(t *testing.T) {
	ctx := context.Background()
	vistoria := &models.RegistroVistoria{ID: uuid.New()}
	vistoriaRepo := testutil.NewFakeVistoriaRepository(vistoria)
	fotoRepo := &testutil.FakeFotoVistoriaRepository{}
	storageDir := t.TempDir()
	svc := NewVistoriaService(vistoriaRepo, fotoRepo, testutil.NewFakeProcessoPagamentoRepository(), storageDir)

	conteudo := pngDeTeste(t)

	t.Run("vistoria inexistente é rejeitada", func(t *testing.T) {
		_, err := svc.AnexarFoto(ctx, uuid.New(), "foto.png", conteudo)
		if !errors.Is(err, repository.ErrVistoriaNotFound) {
			t.Fatalf("esperava ErrVistoriaNotFound, veio %v", err)
		}
	})

	t.Run("caminho feliz grava a foto e evita duplicidade por hash", func(t *testing.T) {
		foto, err := svc.AnexarFoto(ctx, vistoria.ID, "foto.png", conteudo)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if _, err := os.Stat(foto.CaminhoStorage); err != nil {
			t.Fatalf("esperava o arquivo gravado em %q: %v", foto.CaminhoStorage, err)
		}

		// Reenviar o mesmo conteúdo pra mesma vistoria não deve duplicar.
		fotoDup, err := svc.AnexarFoto(ctx, vistoria.ID, "foto.png", conteudo)
		if err != nil {
			t.Fatalf("erro inesperado no reenvio: %v", err)
		}
		if fotoDup.ID != foto.ID {
			t.Error("esperava o mesmo registro (deduplicação por hash), veio um novo ID")
		}
		if len(fotoRepo.Fotos) != 1 {
			t.Fatalf("esperava 1 foto persistida, veio %d", len(fotoRepo.Fotos))
		}
	})

	t.Run("path traversal no nome do arquivo é neutralizado", func(t *testing.T) {
		foto, err := svc.AnexarFoto(ctx, vistoria.ID, "../../../../evil.png", conteudo)
		if err != nil {
			t.Fatalf("upload falhou inesperadamente: %v", err)
		}
		destDir := storageDir + string(os.PathSeparator) + "vistorias" + string(os.PathSeparator) + vistoria.ID.String()
		if !bytesHavePrefixPath(foto.CaminhoStorage, destDir) {
			t.Fatalf("regressão de path traversal: arquivo gravado em %q, fora de %q", foto.CaminhoStorage, destDir)
		}
	})
}

func bytesHavePrefixPath(caminho, prefixo string) bool {
	return len(caminho) >= len(prefixo) && caminho[:len(prefixo)] == prefixo
}

func TestVistoriaService_GerarRelatorioCampo(t *testing.T) {
	ctx := context.Background()
	fiscal := &models.User{ID: uuid.New(), Nome: "Fiscal Teste"}
	contrato := &models.Contrato{ID: uuid.New(), NumeroContrato: "10/2026", ContratadaNome: "Fornecedora Teste"}
	processo := &models.ProcessoPagamento{ID: uuid.New(), ContratoID: contrato.ID, Contrato: contrato}

	lat, lng := -16.4673, -54.6372
	vistoria := &models.RegistroVistoria{
		ID: uuid.New(), ProcessoPagamentoID: processo.ID, ProcessoPagamento: processo,
		FiscalID: fiscal.ID, Fiscal: fiscal, Latitude: &lat, Longitude: &lng,
		Observacoes: "Execução dentro do esperado.",
	}

	vistoriaRepo := testutil.NewFakeVistoriaRepository(vistoria)
	fotoRepo := &testutil.FakeFotoVistoriaRepository{}
	storageDir := t.TempDir()
	svc := NewVistoriaService(vistoriaRepo, fotoRepo, testutil.NewFakeProcessoPagamentoRepository(processo), storageDir)

	t.Run("sem fotos", func(t *testing.T) {
		pdf, err := svc.GerarRelatorioCampo(ctx, vistoria.ID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !bytes.HasPrefix(pdf, []byte("%PDF")) {
			t.Fatal("saída não parece ser um PDF válido")
		}
	})

	t.Run("com foto embutida", func(t *testing.T) {
		foto, err := svc.AnexarFoto(ctx, vistoria.ID, "foto.png", pngDeTeste(t))
		if err != nil {
			t.Fatalf("falha ao preparar foto para o teste: %v", err)
		}
		// FakeFotoVistoriaRepository não atualiza vistoria.Fotos
		// automaticamente (fakes são independentes, ver comentário em
		// testutil/fakes.go) — simula o Preload("Fotos") do repository real.
		vistoria.Fotos = []models.FotoVistoria{*foto}

		pdf, err := svc.GerarRelatorioCampo(ctx, vistoria.ID)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !bytes.HasPrefix(pdf, []byte("%PDF")) {
			t.Fatal("saída não parece ser um PDF válido")
		}
	})

	t.Run("vistoria inexistente é rejeitada", func(t *testing.T) {
		_, err := svc.GerarRelatorioCampo(ctx, uuid.New())
		if !errors.Is(err, repository.ErrVistoriaNotFound) {
			t.Fatalf("esperava ErrVistoriaNotFound, veio %v", err)
		}
	})
}
