package codegen

import (
	"bufio"
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
)

// File is a code generation file.
type File struct {
	cfg     FileConfig
	imports *imports
	buf     bytes.Buffer
	err     error
}

// FileConfig configures a code generation file.
type FileConfig struct {
	// Filename of the generated file.
	Filename string
	// Package of the generated file.
	Package string
	// GeneratedBy line to print to the generated file.
	GeneratedBy string
	// BuildTag is an optional build tag to include in the file header.
	BuildTag string
}

// NewFile creates a new code generation file.
func NewFile(cfg FileConfig) *File {
	f := &File{
		cfg:     cfg,
		imports: NewImports(),
	}
	f.buf.Grow(102400) // 100kiB
	if cfg.BuildTag != "" {
		f.P("// +build ", cfg.BuildTag)
		f.P()
	}
	f.P("package ", cfg.Package)
	if cfg.GeneratedBy != "" {
		f.P()
		f.P("// Code generated by ", cfg.GeneratedBy, ". DO NOT EDIT.")
	}
	f.P()
	f.P("import ()")
	return f
}

// P prints args to the generated file.
func (f *File) P(args ...interface{}) {
	for _, arg := range args {
		_, _ = fmt.Fprint(f, arg)
	}
	_, _ = fmt.Fprintln(f)
}

// Write implements io.Writer.
func (f *File) Write(p []byte) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	n, err := f.buf.Write(p)
	if err != nil {
		f.err = fmt.Errorf("write: %w", err)
	}
	return n, err // nolint: wrapcheck // false positive
}

// Content returns the formatted Go source of the file.
func (f *File) Content() (_ []byte, err error) {
	if f.err != nil {
		return nil, fmt.Errorf("content of %s: %w", f.cfg.Filename, f.err)
	}
	content := bytes.Replace(f.buf.Bytes(), []byte("import ()\n"), f.imports.Bytes(), 1)
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, f.cfg.Filename, content, parser.ParseComments)
	if err != nil {
		var src bytes.Buffer
		s := bufio.NewScanner(bytes.NewReader(f.buf.Bytes()))
		for line := 1; s.Scan(); line++ {
			if _, err := fmt.Fprintf(&src, "%5d\t%s\n", line, s.Bytes()); err != nil {
				return nil, fmt.Errorf("content of %s: %w", f.cfg.Filename, err)
			}
		}
		return nil, fmt.Errorf("content of %s:\n%v: %w", f.cfg.Filename, src.String(), err)
	}
	var out bytes.Buffer
	if err := (&printer.Config{
		Mode:     printer.TabIndent | printer.UseSpaces,
		Tabwidth: 8,
	}).Fprint(&out, fileSet, file); err != nil {
		return nil, fmt.Errorf("content of %s: print source: %w", f.cfg.Filename, err)
	}
	return out.Bytes(), nil
}

// Import includes the provided import path in the file's imports and returns a package identifier.
func (f *File) Import(path string) string {
	return f.imports.Import(path)
}
