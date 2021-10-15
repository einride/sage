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

func GrpcJava() {
	binDir := filepath.Join(toolsPath(), "grpc-java", "1.33.0", "bin")
	binary := filepath.Join(binDir, "protoc-gen-grpc-java")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	// read the whole file at once
	b, err := os.ReadFile("pom.xml")
	if err != nil {
		panic(err)
	}
	s := string(b)

	if !strings.Contains(s, "<grpc.version>1.33.0") {
		panic("pom.xml is out of sync with gRPC Java version - expecting 1.33.0")
	}

	hostOS := runtime.GOOS
	if hostOS == "darwin" {
		hostOS = "osx"
	}
	hostArch := runtime.GOARCH
	if hostArch == "amd64" {
		hostArch = "x86_64"
	}

	binURL := fmt.Sprintf("https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.33.0/protoc-gen-grpc-java-1.33.0-%s-%s.exe", hostOS, hostArch)

	if err := file.FromRemote(
		binURL,
		file.WithDestinationDir(binDir),
		file.WithRenameFile("", "protoc-gen-grpc-java"),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		panic(fmt.Sprintf("Unable to download grpc-java: %v", err))
	}
}

func Protoc() {
	version := "3.15.7"
	binDir := filepath.Join(toolsPath(), "protoc", version)
	binary := filepath.Join(binDir, "bin", "protoc")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == "amd64" {
		hostArch = "x86_64"
	}

	binURL := fmt.Sprintf("https://github.com/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-%s-%s.zip", version, version, hostOS, hostArch)

	if err := file.FromRemote(
		binURL,
		file.WithDestinationDir(binDir),
		file.WithUnzip(),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		panic(fmt.Sprintf("Unable to download protoc: %v", err))
	}

	if err := os.RemoveAll(filepath.Join(binDir, "include")); err != nil {
		panic(err)
	}
}

func Terraform() {
	version := "1.0.0"
	binDir := filepath.Join(toolsPath(), "terraform", version)
	binary := filepath.Join(binDir, "terraform")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	binURL := fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip", version, version, hostOS, hostArch)

	if err := file.FromRemote(
		binURL,
		file.WithDestinationDir(binDir),
		file.WithUnzip(),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		panic(fmt.Sprintf("Unable to download terraform: %v", err))
	}
}

func Buf() {
	version := "0.55.0"
	binDir := filepath.Join(toolsPath(), "buf", version, "bin")
	binary := filepath.Join(binDir, "buf")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == "amd64" {
		hostArch = "x86_64"
	}

	binURL := fmt.Sprintf("https://github.com/bufbuild/buf/releases/download/v%s/buf-%s-%s", version, hostOS, hostArch)

	if err := file.FromRemote(
		binURL,
		file.WithDestinationDir(binDir),
		file.WithRenameFile("", "buf"),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		panic(fmt.Sprintf("Unable to download buf: %v", err))
	}
}

func GoogleProtoScrubber() {
	version := "1.1.0"
	binDir := filepath.Join(toolsPath(), "google-cloud-proto-scrubber", version)
	binary := filepath.Join(binDir, "google-cloud-proto-scrubber")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == "amd64" {
		hostArch = "x86_64"
	}

	binURL := fmt.Sprintf("https://github.com/einride/google-cloud-proto-scrubber/releases/download/v%s/google-cloud-proto-scrubber_%s_%s_%s.tar.gz", version, version, hostOS, hostArch)

	if err := file.FromRemote(
		binURL,
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		panic(fmt.Sprintf("Unable to download google-cloud-proto-scrubber: %v", err))
	}

	if err := os.Chmod(binary, 0o755); err != nil {
		panic(fmt.Sprintf("Unable to make google-cloud-proto-scrubber executable: %v", err))
	}
}

func GH() {
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	version := "1.4.0"

	// ghFolder := fmt.Sprintf("gh_%s_%s_%s", version, hostOS, hostArch)
	dir := filepath.Join(toolsPath(), "gh")
	binDir := filepath.Join(dir, version, "bin")
	binary := filepath.Join(binDir, "gh")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	binURL := fmt.Sprintf("https://github.com/cli/cli/releases/download/v%s/gh_%s_%s_%s.tar.gz", version, version, hostOS, hostArch)

	if err := file.FromRemote(
		binURL,
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
		file.WithRenameFile(fmt.Sprintf("gh_%s_%s_%s/bin/gh", version, hostOS, hostArch), "gh"),
		file.WithSkipIfFileExists(binary),
	); err != nil {
		panic(fmt.Sprintf("Unable to download gh: %v", err))
	}
}

func GHComment() {
	mg.Deps(GH)

	version := "0.2.1"
	binDir := filepath.Join(toolsPath(), "ghcomment", version, "bin")
	binary := filepath.Join(binDir, "ghcomment")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	// Check if binary already exist
	if _, err := os.Stat(binary); err == nil {
		return
	}

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	ghVersion := "v" + version
	pattern := fmt.Sprintf("*%s_%s.tar.gz", hostOS, hostArch)
	archive := fmt.Sprintf("%s/ghcomment_%s_%s_%s.tar.gz", binDir, version, hostOS, hostArch)

	if err := sh.Run("gh", "release", "download", "--repo", "einride/ghcomment", ghVersion, "--pattern", pattern, "--dir", binDir); err != nil {
		panic(fmt.Sprintf("unable to download ghcomment: %v", err))
	}

	if err := file.FromLocal(
		archive,
		file.WithDestinationDir(binDir),
		file.WithUntarGz(),
	); err != nil {
		panic(fmt.Sprintf("Unable to download gh: %v", err))
	}
}
