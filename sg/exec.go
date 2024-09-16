package sg

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

type cmdEnvCtxKey string

//nolint:gochecknoglobals
var cmdEnvKey cmdEnvCtxKey = "cmdEnv"

// ContextWithEnv returns a context with environment variables which are appended to Command.
func ContextWithEnv(ctx context.Context, env ...string) context.Context {
	return context.WithValue(ctx, cmdEnvKey, env)
}

// Command should be used when returning exec.Cmd from tools to set opinionated standard fields.
func Command(ctx context.Context, path string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, path)
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = FromGitRoot(".")
	cmd.Env = os.Environ()
	if v := ctx.Value(cmdEnvKey); v != nil {
		env, ok := v.([]string)
		if ok {
			cmd.Env = append(cmd.Env, env...)
		}
	}
	cmd.Env = prependPath(cmd.Env, FromBinDir())
	cmd.Stderr = newLogWriter(ctx, os.Stderr)
	cmd.Stdout = newLogWriter(ctx, os.Stdout)
	return cmd
}

func newLogWriter(ctx context.Context, out io.Writer) *logWriter {
	logger := log.New(out, Logger(ctx).Prefix(), 0)
	return &logWriter{logger: logger, out: out}
}

type logWriter struct {
	logger            *log.Logger
	out               io.Writer
	hasFileReferences bool
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	in := bufio.NewScanner(bytes.NewReader(p))
	for in.Scan() {
		line := in.Text()
		if !l.hasFileReferences {
			l.hasFileReferences = hasFileReferences(line)
			if l.hasFileReferences {
				// If line has file reference (e.g. lint errors), print empty line with logger prefix.
				// This enables GitHub to autodetect the file references and print them in the PR review.
				l.logger.Println()
			}
		}
		if l.hasFileReferences {
			// Prints line without logger prefix.
			// Trim space to ensure that file references start at the beginning of the line.
			line = strings.TrimSpace(line)
			_, _ = fmt.Fprintln(l.out, line)
		} else {
			l.logger.Print(line)
		}
	}
	if err := in.Err(); err != nil {
		l.logger.Fatal(err)
	}
	return len(p), nil
}

func hasFileReferences(line string) bool {
	line = strings.TrimSpace(line)
	if i := strings.IndexByte(line, ':'); i > 0 {
		if _, err := os.Lstat(line[:i]); err == nil {
			return true
		}
	}
	return false
}

// Output runs the given command, and returns all output from stdout in a neatly, trimmed manner,
// panicking if an error occurs.
func Output(cmd *exec.Cmd) string {
	cmd.Stdout = nil
	output, err := cmd.Output()
	if err != nil {
		panic(fmt.Sprintf("%s failed: %v", cmd.Path, err))
	}
	return strings.TrimSpace(string(output))
}

func prependPath(environ []string, paths ...string) []string {
	for i, kv := range environ {
		if !strings.HasPrefix(kv, "PATH=") {
			continue
		}
		environ[i] = fmt.Sprintf("PATH=%s:%s", strings.Join(paths, ":"), strings.TrimPrefix(kv, "PATH="))
		return environ
	}
	return append(environ, fmt.Sprintf("PATH=%s", strings.Join(paths, ":")))
}
