package sgtool

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.einride.tech/sage/sg"
)

type archiveType int

const (
	None archiveType = iota
	Zip
	Tar
	TarGz
)

const (
	Darwin = "darwin"
)

const (
	AMD64 = "amd64"
	X8664 = "x86_64"
)

type Opt func(f *fileState)

type fileState struct {
	archiveType  archiveType
	dstPath      string
	archiveFiles map[string]string
	skipFile     string
	symlink      string
	httpHeader   http.Header
}

func newFileState() *fileState {
	return &fileState{
		archiveFiles: make(map[string]string),
		httpHeader:   make(http.Header),
	}
}

func FromRemote(ctx context.Context, addr string, opts ...Opt) error {
	s := newFileState()
	for _, o := range opts {
		o(s)
	}
	if s.skipFile != "" {
		// Check if binary already exist
		if _, err := os.Stat(s.skipFile); err == nil {
			if s.symlink != "" {
				if _, err := CreateSymlink(s.symlink); err != nil {
					return err
				}
			}
			return nil
		}
	}
	sg.Logger(ctx).Printf("fetching %s ...", addr)
	rStream, cleanup, err := s.downloadBinary(ctx, addr)
	if err != nil {
		return fmt.Errorf("unable to download file: %w", err)
	}
	defer cleanup()
	return s.handleFileStream(rStream, path.Base(addr))
}

func (s *fileState) handleFileStream(inFile io.Reader, filename string) error {
	if s.dstPath == "" {
		return fmt.Errorf("destination directory is missing")
	}

	f, err := os.Open(s.dstPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("unable to read destination path")
		}
		if err := os.MkdirAll(s.dstPath, 0o755); err != nil {
			return fmt.Errorf("unable to create destination path")
		}
	}
	f.Close()

	switch s.archiveType {
	case None:
		// There should be only 1 entry in the map
		if len(s.archiveFiles) > 1 {
			return fmt.Errorf("only 1 destination file should be specified on direct downloads")
		}
		for _, v := range s.archiveFiles {
			filename = v
			break
		}
		out, err := os.OpenFile(filepath.Join(s.dstPath, filename), os.O_RDWR|os.O_CREATE, 0o755)
		if err != nil {
			return fmt.Errorf("unable to open %s: %w", filename, err)
		}
		defer out.Close()
		// write the body to file
		_, err = io.Copy(out, inFile)
		if err != nil {
			return fmt.Errorf("unable to download remote file: %w", err)
		}
	case Tar:
		if err := s.extractTar(inFile); err != nil {
			return fmt.Errorf("unable to untar the file: %w", err)
		}
	case TarGz:
		gzipStream, err := gzip.NewReader(inFile)
		if err != nil {
			return fmt.Errorf("unable to setup gzip stream: %w", err)
		}
		defer gzipStream.Close()
		if err := s.extractTar(gzipStream); err != nil {
			return fmt.Errorf("unable to untarGz the file: %w", err)
		}
	case Zip:
		// Zip archives require random access for reading, so we need to figure out the
		// entire file size first by reading it completely
		buff := bytes.NewBuffer([]byte{})
		size, err := io.Copy(buff, inFile)
		if err != nil {
			return fmt.Errorf("unable to read remote file: %w", err)
		}
		reader := bytes.NewReader(buff.Bytes())

		zipStream, err := zip.NewReader(reader, size)
		if err != nil {
			return fmt.Errorf("unable to unzip file: %w", err)
		}
		if _, err := s.extractZip(zipStream); err != nil {
			return fmt.Errorf("unable to extract zip file: %w", err)
		}
	}
	if s.symlink != "" {
		if _, err := CreateSymlink(s.symlink); err != nil {
			return err
		}
	}
	return nil
}

func WithUnzip() Opt {
	return func(f *fileState) {
		f.archiveType = Zip
	}
}

func WithUntar() Opt {
	return func(f *fileState) {
		f.archiveType = Tar
	}
}

func WithUntarGz() Opt {
	return func(f *fileState) {
		f.archiveType = TarGz
	}
}

func WithDestinationDir(path string) Opt {
	return func(f *fileState) {
		f.dstPath = path
	}
}

func WithSymlink(path string) Opt {
	return func(f *fileState) {
		f.symlink = path
	}
}

// WithRenameFile renames a source file to the given
// destination file when writing it.
// For archives the source file should be the path relative
// to the root of the archive. If the archive does not contain a file
// with a matching src path, it is ignored.
// For direct downloads (no archive) the src does not matter and the
// output file is stored as per dst.
// The output file is stored relative to the destination dir given by
// WithDestinationDir.
func WithRenameFile(src, dst string) Opt {
	return func(f *fileState) {
		f.archiveFiles[src] = dst
	}
}

func WithSkipIfFileExists(filepath string) Opt {
	return func(f *fileState) {
		f.skipFile = filepath
	}
}

func WithHTTPHeader(key, value string) Opt {
	return func(f *fileState) {
		f.httpHeader.Add(key, value)
	}
}

func (s *fileState) downloadBinary(ctx context.Context, url string) (io.ReadCloser, func(), error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, func() {}, fmt.Errorf("download binary %s: %w", url, err)
	}

	req.Header = s.httpHeader
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, func() {}, fmt.Errorf("download binary %s: %w", url, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, func() {}, fmt.Errorf("download binary %s: status code %d", url, resp.StatusCode)
	}
	return resp.Body, func() { resp.Body.Close() }, nil
}

// extractZip will decompress a zip archive from the given gzip.Reader into
// the destination path.
// The path must exist already.
func (s *fileState) extractZip(reader *zip.Reader) ([]string, error) {
	filenames := make([]string, 0)
	for _, f := range reader.File {
		dstName := f.Name
		if name, ok := s.archiveFiles[f.Name]; ok {
			dstName = name
		}

		// Store filename/path for returning and using later on
		// nolint: gosec // allow file traversal when extracting archive
		fpath := filepath.Join(s.dstPath, dstName)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(s.dstPath)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return nil, err
			}
			continue
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		// nolint: gosec // allow potential decompression bomb
		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

func (s *fileState) extractTar(reader io.Reader) error {
	if reader == nil {
		return errors.New("unable to untar nil file")
	}
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("extractTar: Next() failed: %w", err)
		}

		dstName := header.Name
		if name, ok := s.archiveFiles[dstName]; ok {
			dstName = name
		}

		// nolint: gosec // allow traversal into archive
		path := filepath.Join(s.dstPath, dstName)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return fmt.Errorf("extractTar: MkdirAll() failed: %w", err)
			}
		case tar.TypeReg:
			// Not all directories in the tar file are TypeDir so we have to make
			// sure to create any paths that might only show up as TypeReg
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("extractTar: MkdirAll() failed: %w", err)
			}
			outFile, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("extractTar: Create() failed: %w", err)
			}
			if err := os.Chmod(path, 0o775); err != nil {
				return fmt.Errorf("extractTar: Chmod() failed: %w", err)
			}
			// nolint: gosec // allow potential decompression bomb
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("extractTar: Copy() failed: %w", err)
			}
			outFile.Close()

		default:
			return fmt.Errorf(
				"extractTar: uknown type: %v in %s",
				header.Typeflag,
				header.Name,
			)
		}
	}
	return nil
}

func CreateSymlink(src string) (string, error) {
	symlink := filepath.Join(sg.FromBinDir(), filepath.Base(src))
	if err := os.MkdirAll(sg.FromBinDir(), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Lstat(symlink); err == nil {
		if err := os.Remove(symlink); err != nil {
			return "", err
		}
	}
	if err := os.Symlink(src, symlink); err != nil {
		return "", err
	}
	return symlink, nil
}
