package sg

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"go.einride.tech/sage/internal/strcase"
)

type loggerContextKey struct{}

// NewLogger returns a standard logger.
func NewLogger(name string) *log.Logger {
	return log.New(os.Stderr, fmt.Sprintf("[%s] ", strcase.ToKebab(strings.TrimPrefix(name, "main."))), 0)
}

// WithLogger attaches a log.Logger to the provided context.
func WithLogger(ctx context.Context, logger *log.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// Logger returns the log.Logger attached to ctx, or a default logger.
func Logger(ctx context.Context) *log.Logger {
	if logger := ctx.Value(loggerContextKey{}).(*log.Logger); logger != nil {
		return logger
	}
	return NewLogger("sage")
}
