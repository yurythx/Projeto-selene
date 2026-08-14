package repository

// Defaults e limites de paginação, aplicados por Pagina.Normalizada —
// nenhuma rota de listagem aceita "tamanho ilimitado" por acidente.
const (
	PaginaPadrao        = 1
	TamanhoPaginaPadrao = 20
	TamanhoPaginaMaximo = 100
)

// Pagina representa uma página de resultados solicitada (1-based, como a
// API expõe — não 0-based como o OFFSET do SQL).
type Pagina struct {
	Numero  int
	Tamanho int
}

// Normalizada aplica os defaults/limites: Numero nunca fica abaixo de 1,
// Tamanho nunca fica fora de [1, TamanhoPaginaMaximo].
func (p Pagina) Normalizada() Pagina {
	n := p
	if n.Numero < 1 {
		n.Numero = PaginaPadrao
	}
	if n.Tamanho < 1 {
		n.Tamanho = TamanhoPaginaPadrao
	}
	if n.Tamanho > TamanhoPaginaMaximo {
		n.Tamanho = TamanhoPaginaMaximo
	}
	return n
}

// Offset converte a página normalizada no OFFSET usado pela consulta SQL.
func (p Pagina) Offset() int {
	return (p.Numero - 1) * p.Tamanho
}

// ResultadoPaginado agrupa uma página de itens com os metadados
// necessários para o cliente montar paginação (total de itens, página
// atual, tamanho da página).
type ResultadoPaginado[T any] struct {
	Dados         []T   `json:"dados"`
	Total         int64 `json:"total"`
	Pagina        int   `json:"pagina"`
	TamanhoPagina int   `json:"tamanho_pagina"`
}
