package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/einride/mage-tools/file"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	amd64  = "amd64"
	x86_64 = "x86_64"
)

func GrpcJava() error {
	binDir := filepath.Join(path(), "grpc-java", "1.33.0", "bin")
	binary := filepath.Join(binDir, "protoc-gen-grpc-java")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	// read the whole file at once
	b, err := os.ReadFile("pom.xml")
	if err != nil {
		panic(err)
	}
	s := string(b)

	if !strings.Contains(s, "<grpc.version>1.33.0") {
		return errors.New("pom.xml is out of sync with gRPC Java version - expecting 1.33.0")
	}

	hostOS := runtime.GOOS
	if hostOS == "darwin" {
		hostOS = "osx"
	}
	hostArch := runtime.GOARCH
	if hostArch == amd64 {
		hostArch = x86_64
	}

	binURL := fmt.Sprintf(
		"https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.33.0/protoc-gen-grpc-java-1.33.0-%s-%s.exe",
		hostOS,
		hostArch,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithRenameFile("", "protoc-gen-grpc-java"),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download grpc-java: %w", err)
	}

	return nil
}

func Protoc() error {
	version := "3.15.7"
	binDir := filepath.Join(path(), "protoc", version)
	binary := filepath.Join(binDir, "bin", "protoc")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == amd64 {
		hostArch = x86_64
	}

	binURL := fmt.Sprintf(
		"https://github.com/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-%s-%s.zip",
		version,
		version,
		hostOS,
		hostArch,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUnzip(),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download protoc: %w", err)
	}

	if err := os.RemoveAll(filepath.Join(binDir, "include")); err != nil {
		return err
	}

	return nil
}

func Terraform(version string) error {
	const defaultVersion = "1.0.0"

	if version == "" {
		version = defaultVersion
	}

	supportedVersions := []string{
		"1.0.0",
		"1.0.5",
	}
	if !contains(supportedVersions, version) {
		return fmt.Errorf(
			"the following Terraform versions are supported: %s",
			strings.Join(supportedVersions, ", "),
		)
	}

	binDir := filepath.Join(path(), "terraform", version)
	binary := filepath.Join(binDir, "terraform")
	path := fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH"))

	os.Setenv("PATH", path)

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	binURL := fmt.Sprintf(
		"https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip",
		version,
		version,
		hostOS,
		hostArch,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUnzip(),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download terraform: %w", err)
	}
	return nil
}

func Sops() error {
	version := "3.7.1"

	binDir := filepath.Join(path(), "sops", version)
	binary := filepath.Join(binDir, "sops")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS

	binURL := fmt.Sprintf(
		"https://github.com/mozilla/sops/releases/download/v%s/sops-v%s.%s",
		version,
		version,
		hostOS,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithRenameFile("", "sops"),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download sops: %w", err)
	}
	return nil
}

func Buf() error {
	version := "0.55.0"
	binDir := filepath.Join(path(), "buf", version, "bin")
	binary := filepath.Join(binDir, "buf")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == amd64 {
		hostArch = x86_64
	}

	binURL := fmt.Sprintf(
		"https://github.com/bufbuild/buf/releases/download/v%s/buf-%s-%s",
		version,
		hostOS,
		hostArch,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithRenameFile("", "buf"),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download buf: %w", err)
	}

	return nil
}

func GoogleProtoScrubber() error {
	version := "1.1.0"
	binDir := filepath.Join(path(), "google-cloud-proto-scrubber", version)
	binary := filepath.Join(binDir, "google-cloud-proto-scrubber")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == amd64 {
		hostArch = x86_64
	}

	binURL := fmt.Sprintf(
		"https://github.com/einride/google-cloud-proto-scrubber"+
			"/releases/download/v%s/google-cloud-proto-scrubber_%s_%s_%s.tar.gz",
		version,
		version,
		hostOS,
		hostArch,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download google-cloud-proto-scrubber: %w", err)
	}

	if err := os.Chmod(binary, 0o755); err != nil {
		return fmt.Errorf("unable to make google-cloud-proto-scrubber executable: %w", err)
	}

	return nil
}

func GH() error {
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	version := "2.2.0"

	dir := filepath.Join(path(), "gh")
	binDir := filepath.Join(dir, version, "bin")
	binary := filepath.Join(binDir, "gh")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	binURL := fmt.Sprintf(
		"https://github.com/cli/cli/releases/download/v%s/gh_%s_%s_%s.tar.gz",
		version,
		version,
		hostOS,
		hostArch,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
		file.WithRenameFile(fmt.Sprintf("gh_%s_%s_%s/bin/gh", version, hostOS, hostArch), "gh"),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download gh: %w", err)
	}

	return nil
}

func GHComment() error {
	mg.Deps(GH)

	version := "0.2.1"
	binDir := filepath.Join(path(), "ghcomment", version, "bin")
	binary := filepath.Join(binDir, "ghcomment")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	// Check if binary already exist
	if file.Exists(binary) == nil {
		return nil
	}

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	ghVersion := "v" + version
	pattern := fmt.Sprintf("*%s_%s.tar.gz", hostOS, hostArch)
	archive := fmt.Sprintf("%s/ghcomment_%s_%s_%s.tar.gz", binDir, version, hostOS, hostArch)

	if err := sh.Run(
		"gh",
		"release",
		"download",
		"--repo",
		"einride/ghcomment",
		ghVersion,
		"--pattern",
		pattern,
		"--dir",
		binDir,
	); err != nil {
		return fmt.Errorf("unable to download ghcomment: %w", err)
	}

	if err := file.FromLocal(
		archive,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
	); err != nil {
		return fmt.Errorf("unable to download ghcomment: %w", err)
	}

	return nil
}

func GolangciLint() error {
	version := "1.42.1"
	toolDir := filepath.Join(path(), "golangci-lint")
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, "golangci-lint")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	golangciLint := fmt.Sprintf("golangci-lint-%s-%s-%s", version, hostOS, hostArch)

	binURL := fmt.Sprintf(
		"https://github.com/golangci/golangci-lint/releases/download/v%s/%s.tar.gz",
		version,
		golangciLint,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
		file.WithRenameFile(fmt.Sprintf("%s/golangci-lint", golangciLint), "golangci-lint"),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download golangci-lint: %w", err)
	}
	return nil
}

func Goreview() error {
	mg.Deps(GH)

	version := "0.18.0"
	binDir := filepath.Join(path(), "goreview", version, "bin")
	binary := filepath.Join(binDir, "goreview")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	// Check if binary already exist
	if file.Exists(binary) == nil {
		return nil
	}

	hostOS := strings.Title(runtime.GOOS)
	hostArch := runtime.GOARCH
	if hostArch == amd64 {
		hostArch = x86_64
	}
	goreviewVersion := "v" + version
	pattern := fmt.Sprintf("*%s_%s.tar.gz", hostOS, hostArch)
	archive := fmt.Sprintf("%s/goreview_%s_%s_%s.tar.gz", binDir, version, hostOS, hostArch)

	if err := sh.Run(
		"gh",
		"release",
		"download",
		"--repo",
		"einride/goreview",
		goreviewVersion,
		"--pattern",
		pattern,
		"--dir",
		binDir,
	); err != nil {
		return fmt.Errorf("unable to download goreview: %w", err)
	}

	if err := file.FromLocal(
		archive,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
	); err != nil {
		return fmt.Errorf("unable to download goreview: %w", err)
	}

	return nil
}

func SemanticRelease(branch string) error {
	// Check if npm is installed
	if err := sh.Run("npm", "version"); err != nil {
		return err
	}

	toolDir := filepath.Join(path(), "semantic-release")
	binary := filepath.Join(toolDir, "node_modules", ".bin", "semantic-release")
	releasercJSON := filepath.Join(toolDir, ".releaserc.json")
	packageJSON := filepath.Join(toolDir, "package.json")

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}

	packageFileContent := `{
    "devDependencies": {
        "semantic-release": "^17.3.7",
        "@semantic-release/github": "^7.2.0",
        "@semantic-release/release-notes-generator": "^9.0.1",
        "conventional-changelog-conventionalcommits": "^4.5.0"
    }
}`
	releasercFileContent := fmt.Sprintf(`{
  "plugins": [
    [
      "@semantic-release/commit-analyzer",
      {
        "preset": "conventionalcommits",
        "releaseRules": [
          {
            "type": "chore",
            "release": "patch"
          },
          {
            "breaking": true,
            "release": "minor"
          }
        ]
      }
    ],
    "@semantic-release/release-notes-generator",
    "@semantic-release/github"
  ],
  "branches": [
    "%s"
  ],
  "success": false,
  "fail": false
}`, branch)

	fp, err := os.Create(packageJSON)
	if err != nil {
		return err
	}
	defer fp.Close()

	if _, err = fp.WriteString(packageFileContent); err != nil {
		return err
	}

	fr, err := os.Create(releasercJSON)
	if err != nil {
		return err
	}
	defer fr.Close()

	if _, err = fr.WriteString(releasercFileContent); err != nil {
		return err
	}

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))
	fmt.Println("[semantic-release] installing packages...")
	err = sh.Run(
		"npm",
		"--silent",
		"install",
		"--prefix",
		toolDir,
		"--no-save",
		"--no-audit",
		"--ignore-script",
	)
	if err != nil {
		return err
	}
	return nil
}

func Ko() error {
	hostOS := runtime.GOOS

	version := "0.9.3"

	binDir := filepath.Join(path(), "ko", version, "bin")
	binary := filepath.Join(binDir, "ko")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	binURL := fmt.Sprintf(
		"https://github.com/google/ko/releases/download/v%s/ko_%s_%s_x86_64.tar.gz",
		version,
		version,
		hostOS,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download ko: %w", err)
	}

	return nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
