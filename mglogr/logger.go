package mglogr

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/iancoleman/strcase"
)

// New returns a standard logger.
func New(name string) logr.Logger {
	return logr.New(&sink{
		name: strcase.ToKebab(name),
		formatter: funcr.NewFormatter(funcr.Options{
			RenderBuiltinsHook: func(kvList []interface{}) []interface{} {
				// Don't render builtins.
				return nil
			},
		}),
	})
}

type sink struct {
	formatter funcr.Formatter
	name      string
}

func (s *sink) Init(info logr.RuntimeInfo) {
}

func (s *sink) Enabled(level int) bool {
	return true
}

func (s *sink) Info(level int, msg string, keysAndValues ...interface{}) {
	_, args := s.formatter.FormatInfo(level, msg, keysAndValues)
	fmt.Printf("[%s] %s%s\n", s.name, msg, args)
}

func (s *sink) Error(err error, msg string, keysAndValues ...interface{}) {
	_, args := s.formatter.FormatError(err, msg, keysAndValues)
	fmt.Printf("[%s | ERROR] %s%s\n", s.name, msg, args)
}

func (s *sink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	panic("implement me")
}

func (s *sink) WithName(name string) logr.LogSink {
	s.name = name
	return s
}
