package sgapilinter

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.einride.tech/sage/sg"
	"go.einride.tech/sage/sgtool"
	"go.einride.tech/sage/tools/sgbuf"
)

const (
	version = "1.36.2"
	name    = "api-linter"
)

//go:embed api-linter.yaml
var defaultConfig []byte

// Command returns an *exec.Cmd for the API Linter.
func Command(ctx context.Context, args ...string) *exec.Cmd {
	sg.Deps(ctx, PrepareCommand)
	return sg.Command(ctx, sg.FromBinDir(name), args...)
}

// Run the API Linter on all the Buf modules in the repo.
func Run(ctx context.Context, args ...string) error {
	// Write default config.
	defaultConfigPath := sg.FromToolsDir("api-linter", "api-linter.yaml")
	if err := os.MkdirAll(filepath.Dir(defaultConfigPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(defaultConfigPath, defaultConfig, 0o600); err != nil {
		return err
	}
	var hasProblem bool
	if err := filepath.WalkDir(sg.FromGitRoot(), func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() || d.Name() != "buf.yaml":
			return nil
		}
		moduleDir := filepath.Dir(path)
		configPath := filepath.Join(moduleDir, "api-linter.yaml")
		if _, err := os.Lstat(configPath); errors.Is(err, os.ErrNotExist) {
			configPath = defaultConfigPath
		}
		var protoFiles []string
		if err := filepath.WalkDir(moduleDir, func(path string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return err
			case !d.IsDir() && filepath.Ext(path) == ".proto":
				protoFiles = append(protoFiles, path)
			}
			return nil
		}); err != nil {
			return err
		}
		if len(protoFiles) == 0 {
			return nil
		}
		relativeModuleDir, err := filepath.Rel(sg.FromGitRoot(), moduleDir)
		if err != nil {
			return err
		}
		descriptorFile := sg.FromBuildDir("api-linter", relativeModuleDir, "descriptor.pb")
		if err := os.MkdirAll(filepath.Dir(descriptorFile), 0o755); err != nil {
			return err
		}
		// TODO: Investigate why this call sometimes fails with context timeout.
		if err := retryMaxTimes(3, func() error {
			bufBuildCmd := sgbuf.Command(ctx, "build", "-o", descriptorFile)
			bufBuildCmd.Dir = moduleDir
			return bufBuildCmd.Run()
		}); err != nil {
			return err
		}
		cmd := Command(
			ctx,
			append(
				append(
					[]string{
						"--output-format", "json",
						"--descriptor-set-in", descriptorFile,
						"--config", configPath,
					},
					args...,
				),
				protoFiles...,
			)...,
		)
		cmd.Dir = moduleDir
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			return err
		}
		var results []result
		if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
			return err
		}
		for _, r := range results {
			for _, p := range r.Problems {
				hasProblem = true
				relativeFilePath, err := filepath.Rel(sg.FromGitRoot(), filepath.Join(moduleDir, r.FilePath))
				if err != nil {
					return err
				}
				// Put the file reference on a new line to help GitHub Actions detect the lint error.
				sg.Logger(ctx).Printf(
					"\n%s:%d:%d: %s (%s)",
					relativeFilePath,
					p.Location.Start.Line,
					p.Location.Start.Column,
					strings.TrimSuffix(p.Message, "."),
					p.RuleID,
				)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if hasProblem {
		return fmt.Errorf("found lint errors")
	}
	return nil
}

func PrepareCommand(ctx context.Context) error {
	const binaryName = "api-linter"
	hostOS := runtime.GOOS
	binDir := sg.FromToolsDir(binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	binURL := fmt.Sprintf(
		"https://github.com/googleapis/api-linter/releases/download/v%s/api-linter-%s-%s-amd64.tar.gz",
		version,
		version,
		hostOS,
	)
	if err := sgtool.FromRemote(
		ctx,
		binURL,
		sgtool.WithDestinationDir(binDir),
		sgtool.WithUntarGz(),
		sgtool.WithSkipIfFileExists(binary),
		sgtool.WithSymlink(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}

type result struct {
	FilePath string `json:"file_path"`
	Problems []struct {
		Message    string `json:"message"`
		Suggestion string `json:"suggestion,omitempty"`
		Location   struct {
			Start struct {
				Line   int `json:"line_number"`
				Column int `json:"column_number"`
			} `json:"start_position"`
		} `json:"location"`
		RuleID     string `json:"rule_id"`
		RuleDocURI string `json:"rule_doc_uri"`
	} `json:"problems"`
}

func retryMaxTimes(n int, fn func() error) error {
	var err error
	for i := 0; i < n; i++ {
		err = fn()
		if err == nil {
			break
		}
	}
	return err
}
