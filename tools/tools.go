package tools

import (
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

var (
	ghVersion               string
	GrpcJavaPath            string
	ProtocPath              string
	TerraformPath           string
	SopsPath                string
	BufPath                 string
	GoogleProtoScrubberPath string
	GHPath                  string
	GHCommentPath           string
	GolangciLintPath        string
	GoReviewPath            string
	SemanticReleasePath     string
	CommitlintPath          string
	KoPath                  string
)

func SetGhVersion(v string) (string, error) {
	ghVersion = v
	return ghVersion, nil
}

func GrpcJava(version string) error {
	const binaryName = "protoc-gen-grpc-java"
	const defaultVersion = "1.33.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"1.33.0"}
		if err := IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(path(), "grpc-java", version, "bin")
	binary := filepath.Join(binDir, binaryName)
	GrpcJavaPath = binary

	// read the whole file at once
	b, err := os.ReadFile("pom.xml")
	if err != nil {
		panic(err)
	}
	s := string(b)

	if !strings.Contains(s, fmt.Sprintf("<grpc.version>%s", version)) {
		return fmt.Errorf("pom.xml is out of sync with gRPC Java version - expecting %s", version)
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
		"https://repo1.maven.org/maven2/io/grpc/%s/%s/%s-%s-%s-%s.exe",
		binaryName,
		version,
		binaryName,
		version,
		hostOS,
		hostArch,
	)

	if err := file.FromRemote(
		binURL,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithRenameFile("", binaryName),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download grpc-java: %w", err)
	}

	return nil
}

func Protoc(version string) error {
	const binaryName = "protoc"
	const defaultVersion = "3.15.7"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"3.15.7"}
		if err := IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(path(), binaryName, version)
	binary := filepath.Join(binDir, "bin", binaryName)
	ProtocPath = binary

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
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	if err := os.RemoveAll(filepath.Join(binDir, "include")); err != nil {
		return err
	}

	return nil
}

func Terraform(version string) error {
	const binaryName = "terraform"
	const defaultVersion = "1.0.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{
			"1.0.0",
			"1.0.5",
		}
		if err := IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(path(), binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	TerraformPath = binary

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
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}

func Sops(version string) error {
	const binaryName = "sops"
	const defaultVersion = "3.7.1"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"3.7.1"}
		if err := IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(path(), binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	SopsPath = binary

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
		file.WithRenameFile("", binaryName),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}

func Buf(version string) error {
	const binaryName = "buf"
	const defaultVersion = "0.55.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"0.55.0"}
		if err := IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(path(), binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	BufPath = binary

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
		file.WithRenameFile("", binaryName),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	return nil
}

func GoogleProtoScrubber(version string) error {
	const binaryName = "google-cloud-proto-scrubber"
	const defaultVersion = "1.1.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"1.1.0"}
		if err := IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}
	binDir := filepath.Join(path(), binaryName, version)
	binary := filepath.Join(binDir, binaryName)
	GoogleProtoScrubberPath = binary

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
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	if err := os.Chmod(binary, 0o755); err != nil {
		return fmt.Errorf("unable to make %s executable: %w", binaryName, err)
	}

	return nil
}

func GH(version string) error {
	const binaryName = "gh"
	const defaultVersion = "2.2.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"2.2.0"}
		if err := IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	dir := filepath.Join(path(), binaryName)
	binDir := filepath.Join(dir, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	GHPath = binary

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
		file.WithRenameFile(fmt.Sprintf("gh_%s_%s_%s/bin/gh", version, hostOS, hostArch), binaryName),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	return nil
}

func GHComment(version string) error {
	mg.Deps(mg.F(GH, version))
	const binaryName = "ghcomment"
	const defaultVersion = "0.2.1"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"0.2.1"}
		if err := IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(path(), binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	GHCommentPath = binary

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
		GHPath,
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
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	if err := file.FromLocal(
		archive,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	return nil
}

func GolangciLint(version string) error {
	const binaryName = "golangci-lint"
	const defaultVersion = "1.42.1"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"1.42.1"}
		if err := IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}
	toolDir := filepath.Join(path(), binaryName)
	binDir := filepath.Join(toolDir, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	GolangciLintPath = binary

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
		file.WithRenameFile(fmt.Sprintf("%s/golangci-lint", golangciLint), binaryName),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}
	return nil
}

func Goreview(version string) error {
	mg.Deps(mg.F(GH, version))
	const binaryName = "goreview"
	const defaultVersion = "0.18.0"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"0.18.0"}
		if err := IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	binDir := filepath.Join(path(), binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	GoReviewPath = binary

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
		GHPath,
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
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	if err := file.FromLocal(
		archive,
		file.WithName(filepath.Base(binary)),
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
	); err != nil {
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
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

	SemanticReleasePath = binary

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

func Commitlint() error {
	// Check if npm is installed
	if err := sh.Run("npm", "version"); err != nil {
		return err
	}

	toolDir := filepath.Join(path(), "commitlint")
	binary := filepath.Join(toolDir, "node_modules", ".bin", "commitlint")
	packageJSON := filepath.Join(toolDir, "package.json")

	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		return err
	}

	packageFileContent := `{
  "devDependencies": {
    "@commitlint/cli": "^11.0.0",
    "@commitlint/config-conventional": "^11.0.0"
  }
}`

	fp, err := os.Create(packageJSON)
	if err != nil {
		return err
	}
	defer fp.Close()

	if _, err = fp.WriteString(packageFileContent); err != nil {
		return err
	}

	CommitlintPath = binary

	fmt.Println("[commitlint] installing packages...")
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

func Ko(version string) error {
	const binaryName = "ko"
	const defaultVersion = "0.9.3"

	if version == "" {
		version = defaultVersion
	} else {
		supportedVersions := []string{"0.9.3"}
		if err := IsSupportedVersion(supportedVersions, version, binaryName); err != nil {
			return err
		}
	}

	hostOS := runtime.GOOS

	binDir := filepath.Join(path(), binaryName, version, "bin")
	binary := filepath.Join(binDir, binaryName)
	KoPath = binary

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
		return fmt.Errorf("unable to download %s: %w", binaryName, err)
	}

	return nil
}
