// Package handler contém os controllers HTTP (Gin): tratam JSON/multipart,
// chamam a camada de service e traduzem o resultado (ou erro) em respostas
// HTTP. Não contêm regra de negócio.
package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/repository"
	"projeto-selene/internal/service"
)

// respondError centraliza a tradução de erros vindos de repository/service
// em respostas HTTP, para que cada handler não precise reimplementar essa
// lógica de mapeamento.
func respondError(c *gin.Context, err error) {
	var checklistErr *service.ErrChecklistIncompleto
	if errors.As(err, &checklistErr) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":                "checklist incompleto",
			"documentos_pendentes": checklistErr.Pendentes,
		})
		return
	}

	switch {
	case errors.Is(err, repository.ErrUserNotFound),
		errors.Is(err, repository.ErrContratoNotFound),
		errors.Is(err, repository.ErrEtapaNotFound),
		errors.Is(err, repository.ErrTipoDocumentoNotFound),
		errors.Is(err, repository.ErrProcessoNotFound),
		errors.Is(err, repository.ErrDocumentoNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "recurso não encontrado"})

	case errors.Is(err, service.ErrEtapaFinal),
		errors.Is(err, service.ErrProcessoNaoElegivelParaConclusao),
		errors.Is(err, service.ErrFiscalInvalido),
		errors.Is(err, service.ErrTipoObjetoInvalido),
		errors.Is(err, service.ErrContratoEncerrado):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	default:
		// Erro não mapeado = inesperado (bug, falha de infraestrutura) —
		// vale registrar com detalhe no log do servidor. O detalhe NÃO
		// vai para o cliente, só a mensagem genérica.
		slog.ErrorContext(c.Request.Context(), "erro interno não mapeado", "erro", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro interno"})
	}
}
