package service

import "time"

// dataLayout é o formato aceito para datas vindas da API (ISO 8601, sem
// hora): "2006-01-02".
const dataLayout = "2006-01-02"

// parseData converte uma data no formato "AAAA-MM-DD" para time.Time,
// retornando um erro explícito e legível em caso de formato inválido.
func parseData(valor string) (time.Time, error) {
	return time.Parse(dataLayout, valor)
}
