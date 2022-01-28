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

// Command should be used when returning exec.Cmd from tools to set opinionated standard fields.
func Command(ctx context.Context, path string, args ...string) *exec.Cmd {
	// TODO: use exec.CommandContext when we have determined there are no side-effects.
	cmd := exec.Command(path)
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = FromGitRoot(".")
	cmd.Env = prependPath(os.Environ(), FromBinDir())
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
		}
		if l.hasFileReferences {
			// Don't prefix output from processes that print file references (e.g. lint errors).
			// Instead prefix on the line above, these lines can be multiline hence the conditional.
			// This enables GitHub to autodetect the file references and print them in the PR review.
			if hasFileReferences(line) {
				_, _ = fmt.Fprintln(l.out, l.logger.Prefix(), "\n", line)
			} else {
				_, _ = fmt.Fprintln(l.out, line)
			}
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
