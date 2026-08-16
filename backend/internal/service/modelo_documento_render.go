package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	docx "github.com/lukasjarosch/go-docx"

	"projeto-selene/internal/models"
	"projeto-selene/internal/repository"
)

// CamposMerge são os valores de merge field disponíveis pra um gatilho de
// geração — a chave é o nome do placeholder SEM delimitador. O delimitador
// usado no arquivo .docx é chave simples (ex: "{numero_contrato}"), não
// duplo ("{{numero_contrato}}") — convenção de
// github.com/lukasjarosch/go-docx, documentar isso na tela de
// Configurações pro admin não errar ao montar o template.
type CamposMerge map[string]string

// renderizarComModelo tenta preencher o modelo .docx ativo associado ao
// gatilho informado. usado=false sem erro é o caminho NORMAL quando o
// admin não cadastrou modelo pra este gatilho (ou a categoria existe mas
// está num estado inconsistente sem versão ativa) — o chamador cai no
// fallback fpdf existente, comportamento inalterado.
//
// Função package-level (não método de ModeloDocumentoService) porque o
// pacote service não tem precedente de um service injetar outro, só
// repositórios — GeradorDocumentosService/RelatorioService continuam só
// dependendo de repository, igual todo o resto do pacote.
func renderizarComModelo(
	ctx context.Context,
	modeloRepo repository.ModeloDocumentoRepository,
	gatilho models.TipoGatilhoModelo,
	campos CamposMerge,
) (conteudo []byte, usado bool, err error) {
	modelo, err := modeloRepo.FindAtivoByGatilho(ctx, gatilho)
	if err != nil {
		if errors.Is(err, repository.ErrModeloDocumentoNotFound) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("service: buscar modelo de documento pro gatilho %s: %w", gatilho, err)
	}
	if modelo.VersaoAtiva == nil {
		// Estado inconsistente (categoria sem versão ativa) — não deveria
		// acontecer via ModeloDocumentoService (toda categoria nasce com
		// uma versão), mas trata como "sem modelo" em vez de propagar um
		// erro: um problema no módulo de templates nunca deve quebrar a
		// geração de documento, que tem o fallback fpdf como rede de
		// segurança.
		return nil, false, nil
	}

	templateBytes, err := os.ReadFile(modelo.VersaoAtiva.CaminhoStorage)
	if err != nil {
		return nil, false, fmt.Errorf("service: ler arquivo do modelo de documento: %w", err)
	}

	doc, err := docx.OpenBytes(templateBytes)
	if err != nil {
		return nil, false, fmt.Errorf("service: abrir modelo de documento como .docx: %w", err)
	}
	defer doc.Close()

	placeholders := make(docx.PlaceholderMap, len(campos))
	for chave, valor := range campos {
		placeholders[chave] = valor
	}
	if err := doc.ReplaceAll(placeholders); err != nil {
		return nil, false, fmt.Errorf("service: preencher modelo de documento: %w", err)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		return nil, false, fmt.Errorf("service: serializar modelo de documento preenchido: %w", err)
	}

	return buf.Bytes(), true, nil
}
