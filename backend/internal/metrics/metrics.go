// Package metrics define as métricas Prometheus da aplicação como
// variáveis de pacote, registradas automaticamente no registry padrão via
// promauto — é o padrão idiomático da client_golang: os coletores vivem
// aqui, e qualquer pacote que precise incrementar/observar uma métrica só
// importa este pacote, sem precisar receber nada por injeção de
// dependência.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal conta requisições HTTP por método, rota (o
	// PADRÃO da rota, ex: "/api/v1/contratos/:id" — nunca a URL crua com
	// o ID de verdade, o que explodiria a cardinalidade da métrica) e
	// status.
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "selene_http_requests_total",
		Help: "Total de requisições HTTP, por método, rota e status.",
	}, []string{"method", "route", "status"})

	// HTTPRequestDuration observa a duração das requisições HTTP.
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "selene_http_request_duration_seconds",
		Help:    "Duração das requisições HTTP em segundos.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})

	// KanbanTransicoesTotal conta transições de etapa do funil de
	// compliance, por etapa de origem e destino — permite ver no
	// Grafana/Prometheus onde os processos estão se acumulando ou
	// travando (ex: muita gente parada na Etapa 5).
	KanbanTransicoesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "selene_kanban_transicoes_total",
		Help: "Total de transições de etapa do Kanban, por etapa de origem e destino.",
	}, []string{"etapa_origem", "etapa_destino"})
)
