package mglog

import (
	"fmt"

	"github.com/go-logr/logr"
)

// Logger returns a standard logger.
func Logger(name string) logr.Logger {
	return logr.New(&sink{}).WithName(name)
}

type sink struct {
	name string
}

func (s *sink) Init(info logr.RuntimeInfo) {
}

func (s *sink) Enabled(level int) bool {
	return true
}

func (s *sink) Info(level int, msg string, keysAndValues ...interface{}) {
	if len(keysAndValues) > 0 {
		fmt.Printf("[%s] %s (%v)\n", s.name, msg, keysAndValues)
	} else {
		fmt.Printf("[%s] %s\n", s.name, msg)
	}
}

func (s *sink) Error(err error, msg string, keysAndValues ...interface{}) {
	if len(keysAndValues) > 0 {
		fmt.Printf("[%s - ERROR] %s (%v)\n", s.name, msg, keysAndValues)
	} else {
		fmt.Printf("[%s - ERROR] %s\n", s.name, msg)
	}
}

func (s *sink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	panic("implement me")
}

func (s *sink) WithName(name string) logr.LogSink {
	s.name = name
	return s
}
