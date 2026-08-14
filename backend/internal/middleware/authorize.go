package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireFiscal exige que o usuário autenticado (já resolvido por
// NewAuthMiddleware, que deve rodar antes deste na cadeia) tenha
// IsFiscal=true. Usado nas rotas de escrita/movimentação do Kanban.
func RequireFiscal() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := UserFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
			return
		}

		if !user.IsFiscal {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "ação restrita a fiscais habilitados"})
			return
		}

		c.Next()
	}
}

// RequireAdmin exige que o usuário autenticado tenha IsAdmin=true. Usado
// nas rotas de administração de contas (Seção 6).
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := UserFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "usuário não autenticado"})
			return
		}

		if !user.IsAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "ação restrita a administradores"})
			return
		}

		c.Next()
	}
}
