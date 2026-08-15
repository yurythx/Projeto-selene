package service

import (
	"net/mail"
	"time"
)

// dataLayout é o formato aceito para datas vindas da API (ISO 8601, sem
// hora): "2006-01-02".
const dataLayout = "2006-01-02"

// parseData converte uma data no formato "AAAA-MM-DD" para time.Time,
// retornando um erro explícito e legível em caso de formato inválido.
func parseData(valor string) (time.Time, error) {
	return time.Parse(dataLayout, valor)
}

// cnpjValido checa se cnpj tem o formato de um CNPJ: exatamente 14 dígitos
// depois de descartar máscara ("." "/" "-") e espaços — aceita tanto
// "12.345.678/0001-90" quanto "12345678000190". Deliberadamente NÃO
// valida os dígitos verificadores (módulo 11): a base de contratos já
// cadastrada usa CNPJs de exemplo com dígitos verificadores inválidos (ver
// fixtures de teste), e recusar checksum retroativamente exigiria migrar
// dados reais só para satisfazer uma regra que o domínio não pediu. Isso
// já elimina a falha real observada (qualquer string não vazia sendo
// aceita como CNPJ, corrompendo o agrupamento do Dossiê do Fornecedor).
func cnpjValido(cnpj string) bool {
	return len(apenasDigitos(cnpj)) == 14
}

// emailValido delega para net/mail.ParseAddress (RFC 5322) — mesma
// biblioteca padrão usada em qualquer parser de e-mail em Go, evita
// reinventar uma regex frágil. Chamado só quando o e-mail não é vazio: o
// campo é opcional (ver comentário em models.Contrato.ContratadaEmail).
func emailValido(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
