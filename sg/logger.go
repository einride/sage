package sg

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/iancoleman/strcase"
)

// NewLogger returns a standard logger.
func NewLogger(name string) logr.Logger {
	return logr.New(&logSink{
		name: strcase.ToKebab(strings.TrimPrefix(name, "main.")),
		formatter: funcr.NewFormatter(funcr.Options{
			RenderBuiltinsHook: func(kvList []interface{}) []interface{} {
				// Don't render builtins.
				return nil
			},
		}),
	})
}

type logSink struct {
	formatter funcr.Formatter
	name      string
}

var _ logr.LogSink = &logSink{}

func (s *logSink) Init(info logr.RuntimeInfo) {
}

func (s *logSink) Enabled(level int) bool {
	return true
}

func (s *logSink) Info(level int, msg string, keysAndValues ...interface{}) {
	_, args := s.formatter.FormatInfo(level, msg, keysAndValues)
	fmt.Printf("[%s] %s%s\n", s.name, msg, args)
}

func (s *logSink) Error(err error, msg string, keysAndValues ...interface{}) {
	_, args := s.formatter.FormatError(err, msg, keysAndValues)
	fmt.Printf("[%s | ERROR] %s%s\n", s.name, msg, args)
}

func (s *logSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	panic("implement me")
}

func (s *logSink) WithName(name string) logr.LogSink {
	s.name = name
	return s
}
