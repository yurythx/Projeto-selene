// Package tracing configura o tracer distribuído (OpenTelemetry) usado em
// toda a aplicação, seguindo o mesmo princípio de graceful degradation já
// usado em internal/logging e internal/service.NewNotifier: se nenhum
// endpoint OTLP for configurado, o TracerProvider registrado ainda
// funciona (otel.Tracer(...) nunca é nil em lugar nenhum do código), só
// que com um sampler que nunca grava nada — sem overhead, sem exportar.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// New constrói e registra o TracerProvider global do processo. Chame uma
// única vez, na inicialização (main.go), e faça defer da função de
// shutdown retornada para drenar spans pendentes no encerramento
// gracioso.
//
// otlpEndpoint é "host:porta" (sem esquema — ex: "jaeger:4318"), apontando
// pro endpoint OTLP/HTTP de um coletor (Jaeger, Tempo, etc.). Vazio
// desabilita a exportação real.
func New(ctx context.Context, serviceName, otlpEndpoint string) (shutdown func(context.Context) error, err error) {
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	if err != nil {
		return nil, fmt.Errorf("tracing: criar resource: %w", err)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	if otlpEndpoint == "" {
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.NeverSample()),
		)
		otel.SetTracerProvider(tp)
		return tp.Shutdown, nil
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otlpEndpoint),
		otlptracehttp.WithInsecure(), // OTLP local/interno ao docker-compose, sem TLS
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: criar exporter OTLP a partir de %q: %w", otlpEndpoint, err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
