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
	prefix := name
	prefix = strings.TrimPrefix(prefix, "main.")
	prefix = strings.TrimPrefix(prefix, "go.einride.tech/sage/tools/")

	// Separate namespace and target with colon, expecting the string to be
	// of the format `namespace.target`.
	if len(strings.Split(prefix, ".")) > 1 {
		prefix = strings.Join(strings.Split(prefix, "."), ":")
	}
	prefix = strcase.ToKebab(prefix)
	prefix = fmt.Sprintf("[%s] ", prefix)
	return log.New(os.Stderr, prefix, 0)
}

// WithLogger attaches a log.Logger to the provided context.
func WithLogger(ctx context.Context, logger *log.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// AppendLoggerPrefix appends a prefix to the current logger.
func AppendLoggerPrefix(ctx context.Context, prefix string) context.Context {
	logger := Logger(ctx)
	return WithLogger(ctx, log.New(logger.Writer(), logger.Prefix()+prefix, logger.Flags()))
}

// Logger returns the log.Logger attached to ctx, or a default logger.
func Logger(ctx context.Context) *log.Logger {
	if value := ctx.Value(loggerContextKey{}); value != nil {
		return value.(*log.Logger)
	}
	return NewLogger("sage")
}
