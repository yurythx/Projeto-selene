// Package models contém os structs de domínio do Projeto Selene, com tags
// GORM para leitura/escrita via ORM (nomes de coluna, tipos, associações).
//
// As tags GORM aqui NÃO são a fonte de verdade do schema do banco — essa
// fonte é internal/database/migrations/ (SQL versionado, aplicado via
// golang-migrate). Uma mudança de campo neste pacote precisa vir
// acompanhada de uma migration nova; alterar só o struct não altera o
// banco.
package models
