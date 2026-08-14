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

// respondBindError responde a uma falha de c.ShouldBindJSON/ShouldBind sem
// repassar err.Error() ao cliente — a mensagem do validador
// (go-playground/validator) inclui o nome da struct e do campo Go
// internos (ex: "Key: 'criarContratoRequest.NumeroContrato' Error:Field
// validation for 'NumeroContrato' failed on the 'required' tag"), o que
// não é segredo, mas também não é da conta do cliente. O detalhe vai só
// pro log do servidor.
func respondBindError(c *gin.Context, err error) {
	slog.WarnContext(c.Request.Context(), "corpo da requisição inválido", "erro", err)
	c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido ou campos obrigatórios ausentes"})
}

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
