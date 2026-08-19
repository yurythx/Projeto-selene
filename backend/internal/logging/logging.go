// Package logging monta o logger estruturado (log/slog) usado em toda a
// aplicação, e o registra como logger padrão do processo (slog.SetDefault)
// — código em qualquer pacote pode simplesmente chamar slog.Info/Error/...
// sem precisar receber o logger por injeção de dependência.
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// contextKey é um tipo próprio (não string) para a chave de contexto do
// request ID — evita colisão com outras chaves de contexto do pacote
// padrão ou de outras bibliotecas.
type contextKey string

const requestIDContextKey contextKey = "request_id"

// New constrói o slog.Logger de acordo com level/format e o registra como
// logger padrão do processo. Chame uma única vez, na inicialização
// (main.go) — antes disso, chamadas a slog.Info/Error/etc. usam o handler
// texto padrão da stdlib.
func New(level, format string) (*slog.Logger, error) {
	slogLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	handlerOptions := &slog.HandlerOptions{Level: slogLevel}

	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, handlerOptions)
	case "text":
		handler = slog.NewTextHandler(os.Stdout, handlerOptions)
	default:
		return nil, fmt.Errorf("logging: formato de log %q inválido (use \"json\" ou \"text\")", format)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
}

// parseLevel converte o valor textual de LOG_LEVEL ("debug"/"info"/
// "warn"/"error") no slog.Level correspondente.
func parseLevel(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("logging: nível de log %q inválido (use debug/info/warn/error)", level)
	}
}

// WithRequestID devolve um novo context.Context carregando o request ID —
// usado pelo middleware de request ID para propagar o valor até onde o
// código de negócio/log precisar dele.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// RequestIDFromContext recupera o request ID gravado por WithRequestID,
// ou "" se não houver nenhum (ex: chamada fora do ciclo de uma requisição
// HTTP, como um job de background).
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

// FromContext retorna um *slog.Logger com o request_id da requisição já
// anexado como atributo (quando presente), pronto para logar dentro de um
// handler ou service chamado durante o ciclo de vida de uma requisição.
func FromContext(ctx context.Context) *slog.Logger {
	logger := slog.Default()
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		logger = logger.With("request_id", requestID)
	}
	return logger
}
