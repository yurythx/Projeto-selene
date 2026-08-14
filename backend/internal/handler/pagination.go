package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"projeto-selene/internal/repository"
)

// paginaFromQuery lê os parâmetros de query "pagina" e "tamanho" (ambos
// opcionais). Valores ausentes ou inválidos caem nos defaults de
// repository.Pagina.Normalizada — uma rota de listagem nunca falha por
// causa de paginação malformada, só ignora o que não faz sentido.
func paginaFromQuery(c *gin.Context) repository.Pagina {
	numero, _ := strconv.Atoi(c.Query("pagina"))
	tamanho, _ := strconv.Atoi(c.Query("tamanho"))

	return repository.Pagina{Numero: numero, Tamanho: tamanho}.Normalizada()
}
