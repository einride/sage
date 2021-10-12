package tools

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func GrpcJava() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	binDir := filepath.Join(cwd, "tools", "grpc-java", "1.33.0", "bin")
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

	var binURL string
	if runtime.GOOS == "linux" {
		binURL = "https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.33.0/protoc-gen-grpc-java-1.33.0-linux-x86_64.exe"
	} else if runtime.GOOS == "darwin" {
		binURL = "https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.33.0/protoc-gen-grpc-java-1.33.0-osx-x86_64.exe"
	}

	DownloadBinary(binDir, binURL, binary)
}

func Protoc() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	version := "3.15.7"
	binDir := filepath.Join(cwd, "tools", "protoc", version)
	binary := filepath.Join(binDir, "protoc")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == "amd64" {
		hostArch = "x86_64"
	}

	binURL := fmt.Sprintf("https://github.com/protocolbuffers/protobuf/releases/download/v%s/protoc-%s-%s-%s.zip", version, version, hostOS, hostArch)

	zip := binary + ".zip"
	DownloadBinary(binDir, binURL, zip)

	_, err = ExtractZip(zip, binDir)
	if err != nil {
		panic(err)
	}

	if err := os.RemoveAll(filepath.Join(binDir, "include")); err != nil {
		panic(err)
	}
}

func Buf() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	version := "0.55.0"
	binDir := filepath.Join(cwd, "tools", "buf", version, "bin")
	binary := filepath.Join(binDir, "buf")

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == "amd64" {
		hostArch = "x86_64"
	}

	binURL := fmt.Sprintf("https://github.com/bufbuild/buf/releases/download/v%s/buf-%s-%s", version, hostOS, hostArch)

	DownloadBinary(binDir, binURL, binary)
}

func GoogleProtoScrubber() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	version := "1.1.0"
	binDir := filepath.Join(cwd, "tools", "google-cloud-proto-scrubber", version)
	binary := filepath.Join(binDir, "google-cloud-proto-scrubber")
	archive := binary + ".tar.gz"

	os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(binary), os.Getenv("PATH")))

	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
	if hostArch == "amd64" {
		hostArch = "x86_64"
	}

	binURL := fmt.Sprintf("https://github.com/einride/google-cloud-proto-scrubber/releases/download/v%s/google-cloud-proto-scrubber_%s_%s_%s.tar.gz", version, version, hostOS, hostArch)

	ok, err := DownloadBinary(binDir, binURL, archive)
	if err != nil {
		panic(err)
	}
	if !ok {
		f, err := os.Open(archive)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		uncompressedStream, err := gzip.NewReader(f)
		if err != nil {
			panic(err)
		}
		tarReader := tar.NewReader(uncompressedStream)

	LOOP:
		for true {
			header, err := tarReader.Next()
			if err == io.EOF {
				panic("Found no files in the tar.gz archive")
			}
			if err != nil {
				panic(err)
			}

			switch header.Typeflag {
			case tar.TypeReg:
				if header.Name != "google-cloud-proto-scrubber" {
					continue
				}
				outFile, err := os.OpenFile(filepath.Join(binDir, header.Name), os.O_RDWR|os.O_CREATE, 0o755)
				if err != nil {
					panic(err)
				}
				defer outFile.Close()
				if _, err := io.Copy(outFile, tarReader); err != nil {
					panic(err)
				}
				break LOOP
			}
		}
	}
}
