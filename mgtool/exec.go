package mgtool

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

// RunInDir runs the given command in the specified directory, discarding stdout.
func RunInDir(cmd, dir string, args ...string) error {
	b := io.Discard
	return run(cmd, dir, b, os.Stderr, args...)
}

// RunInDirV runs the given command in the specified directory, outputting to stdout.
func RunInDirV(cmd, dir string, args ...string) error {
	return run(cmd, dir, os.Stdout, os.Stderr, args...)
}

// OutputRunInDir run the given command in the specified directory, returning the output ad a string.
func OutputRunInDir(cmd, dir string, args ...string) (string, error) {
	b := &bytes.Buffer{}
	if err := run(cmd, dir, b, os.Stderr, args...); err != nil {
		return "", err
	}
	return b.String(), nil
}

func run(cmd, dir string, stdout, stderr io.Writer, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Env = os.Environ()
	c.Stderr = stderr
	c.Stdout = stdout
	c.Stdin = os.Stdin
	c.Dir = dir

	return c.Run()
}
