package handler

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"projeto-selene/internal/middleware"
	"projeto-selene/internal/service"
)

// DocumentoHandler expõe as rotas HTTP de upload/consulta de documentos
// anexos (PDFs) de um processo de pagamento.
type DocumentoHandler struct {
	documentoService *service.DocumentoService
}

// NewDocumentoHandler constrói um DocumentoHandler.
func NewDocumentoHandler(documentoService *service.DocumentoService) *DocumentoHandler {
	return &DocumentoHandler{documentoService: documentoService}
}

// maxUploadBytes limita o tamanho de um único arquivo anexado (20 MiB) —
// evita que um upload malformado ou abusivo esgote memória/disco.
const maxUploadBytes = 20 << 20

// Upload trata POST /api/v1/processos/:id/documentos (multipart/form-data
// com os campos "tipo_documento_id" e "arquivo").
func (h *DocumentoHandler) Upload(c *gin.Context) {
	usuario, ok := middleware.UserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
		return
	}

	processoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	// Envolve o corpo da requisição ANTES de qualquer parse de
	// multipart/form-data (c.PostForm/c.FormFile disparam esse parse na
	// primeira chamada). Sem isso, o Gin bufferiza/grava em disco o corpo
	// inteiro antes da checagem de tamanho abaixo rodar — um upload de
	// vários GB já teria consumido memória/disco antes de ser rejeitado.
	// Com MaxBytesReader, o próprio parser aborta assim que o limite é
	// excedido, com uma margem pequena pro overhead do multipart
	// (boundary, headers de cada parte, o campo tipo_documento_id).
	const margemMultipart = 64 << 10 // 64KiB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+margemMultipart)

	tipoDocumentoID, err := strconv.Atoi(c.PostForm("tipo_documento_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'tipo_documento_id' é obrigatório e precisa ser um número, ou o arquivo excede o limite de 20MB"})
		return
	}

	arquivoHeader, err := c.FormFile("arquivo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'arquivo' é obrigatório, ou o arquivo excede o limite de 20MB"})
		return
	}
	if arquivoHeader.Size > maxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "arquivo excede o limite de 20MB"})
		return
	}

	arquivo, err := arquivoHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao abrir arquivo enviado"})
		return
	}
	defer func() {
		if err := arquivo.Close(); err != nil {
			slog.WarnContext(c.Request.Context(), "falha ao fechar arquivo de upload", "arquivo", arquivoHeader.Filename, "erro", err)
		}
	}()

	conteudo, err := io.ReadAll(arquivo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao ler arquivo enviado"})
		return
	}

	// "data_validade" é opcional — só faz sentido pra certidões (ver
	// TipoDocumento.ExigeValidade). Formato "AAAA-MM-DD", igual
	// Contrato.DataAssinatura no resto da API. Se vier ausente ou
	// malformado, o documento é anexado do mesmo jeito, só sem entrar no
	// radar de certidões (ver o comentário em DocumentoService.Upload).
	var dataValidade *time.Time
	if bruto := c.PostForm("data_validade"); bruto != "" {
		if parsed, err := time.Parse("2006-01-02", bruto); err == nil {
			dataValidade = &parsed
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'data_validade' precisa estar no formato AAAA-MM-DD"})
			return
		}
	}

	documento, err := h.documentoService.Upload(c.Request.Context(), processoID, tipoDocumentoID, arquivoHeader.Filename, conteudo, usuario.ID, dataValidade)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, documento)
}

// Listar trata GET /api/v1/processos/:id/documentos.
func (h *DocumentoHandler) Listar(c *gin.Context) {
	processoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	documentos, err := h.documentoService.Listar(c.Request.Context(), processoID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, documentos)
}

// documentoIDsFromParams lê e valida os dois UUIDs da rota
// /processos/:id/documentos/:docId — repetido em Baixar e Excluir.
func documentoIDsFromParams(c *gin.Context) (processoID, documentoID uuid.UUID, ok bool) {
	processoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de processo inválido"})
		return uuid.Nil, uuid.Nil, false
	}
	documentoID, err = uuid.Parse(c.Param("docId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de documento inválido"})
		return uuid.Nil, uuid.Nil, false
	}
	return processoID, documentoID, true
}

// Baixar trata GET /api/v1/processos/:id/documentos/:docId/download —
// serve o arquivo com Content-Disposition "inline" (não "attachment"),
// de propósito: o frontend abre esta URL numa aba nova (target="_blank",
// ver processo-page.tsx) pra pré-visualização — "inline" faz o
// navegador renderizar com seu próprio visualizador nativo em vez de
// forçar um download direto (o usuário ainda pode salvar via "Salvar
// como" se quiser).
//
// Otimização pedida pelo usuário ("a visualização de documento é tão
// lenta pra aparecer"): usa http.ServeContent sobre o *os.File aberto
// por DocumentoService.Baixar (streaming direto do disco pro socket, sem
// bufferizar o arquivo inteiro em memória antes de começar a responder)
// e cache HTTP agressivo. Um documento anexado é imutável pelo próprio
// ID — pra trocar o conteúdo, a Regra 1 (ver ErrTipoDocumentoJaAnexado)
// obriga excluir e reenviar, o que sempre gera um documento com ID novo
// — então o hash SHA-256 já calculado no upload serve de ETag perfeito
// (mesmo conteúdo ⇒ mesmo ETag, sempre) e "immutable" evita até a
// revalidação condicional em recarregamentos dentro da mesma sessão do
// navegador: antes, reabrir a pré-visualização do MESMO documento
// relia o arquivo inteiro do disco e retransmitia tudo de novo, sem
// nenhum reaproveitamento.
func (h *DocumentoHandler) Baixar(c *gin.Context) {
	processoID, documentoID, ok := documentoIDsFromParams(c)
	if !ok {
		return
	}

	arquivo, documento, err := h.documentoService.Baixar(c.Request.Context(), processoID, documentoID)
	if err != nil {
		respondError(c, err)
		return
	}
	defer func() {
		if err := arquivo.Close(); err != nil {
			slog.WarnContext(c.Request.Context(), "falha ao fechar arquivo do documento anexo", "caminho", documento.CaminhoStorage, "erro", err)
		}
	}()

	c.Header("Content-Disposition", `inline; filename="`+documento.NomeArquivo+`"`)
	c.Header("ETag", `"`+documento.HashArquivo+`"`)
	c.Header("Cache-Control", "private, max-age=31536000, immutable")

	// Sniff dos primeiros bytes pra um Content-Type confiável — os
	// uploads deste app só aceitam PDF/imagem no cliente (accept=
	// "application/pdf,image/*" em processo-page.tsx), mas nada valida
	// isso no servidor hoje (diferente do .docx de Modelos de Documento,
	// que checa a assinatura ZIP), então sniffar de verdade em vez de
	// confiar só na extensão do nome do arquivo é mais robusto. Sem isso,
	// http.ServeContent tentaria adivinhar pela extensão de NomeArquivo
	// antes de sniffar sozinho.
	var buf [512]byte
	n, err := arquivo.Read(buf[:])
	if err != nil && err != io.EOF {
		respondError(c, fmt.Errorf("handler: ler cabeçalho do arquivo pra detectar content-type: %w", err))
		return
	}
	c.Header("Content-Type", http.DetectContentType(buf[:n]))
	if _, err := arquivo.Seek(0, io.SeekStart); err != nil {
		respondError(c, fmt.Errorf("handler: rebobinar arquivo do documento anexo: %w", err))
		return
	}

	// http.ServeContent cuida do resto: escreve o Content-Length correto
	// (via Seek), responde 304 sozinho quando o If-None-Match do cliente
	// bate com o ETag acima (nem chega a transmitir o corpo), e suporta
	// Range requests.
	http.ServeContent(c.Writer, c.Request, documento.NomeArquivo, documento.DataUpload, arquivo)
}

// Excluir trata DELETE /api/v1/processos/:id/documentos/:docId —
// restrito a fiscais (ver o grupo de rotas em cmd/api/main.go, mesma
// exigência do upload).
func (h *DocumentoHandler) Excluir(c *gin.Context) {
	processoID, documentoID, ok := documentoIDsFromParams(c)
	if !ok {
		return
	}

	if err := h.documentoService.Excluir(c.Request.Context(), processoID, documentoID); err != nil {
		respondError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
