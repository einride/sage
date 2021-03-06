package sg

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"

	"go.einride.tech/sage/internal/strcase"
)

type loggerContextKey struct{}

// NewLogger returns a standard logger.
func NewLogger(name string) *log.Logger {
	prefix := name
	prefix = strings.TrimPrefix(prefix, "main.")
	prefix = strings.TrimPrefix(prefix, "go.einride.tech/sage/tools/")
	// Strip namespace from name.
	if len(strings.Split(prefix, ".")) > 1 {
		if unicode.IsUpper([]rune(strings.Split(prefix, ".")[0])[0]) {
			prefix = strings.Join(strings.Split(prefix, ".")[1:], "")
		}
	}
	prefix = strcase.ToKebab(prefix)
	prefix = fmt.Sprintf("[%s] ", prefix)
	return log.New(os.Stderr, prefix, 0)
}

// WithLogger attaches a log.Logger to the provided context.
func WithLogger(ctx context.Context, logger *log.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// Logger returns the log.Logger attached to ctx, or a default logger.
func Logger(ctx context.Context) *log.Logger {
	if value := ctx.Value(loggerContextKey{}); value != nil {
		return value.(*log.Logger)
	}
	return NewLogger("sage")
}
