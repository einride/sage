package file

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type archiveType int

const (
	None archiveType = iota
	Zip
	TarGz
)

type opt func(f *FileState)

type FileState struct {
	name         string
	archiveType  archiveType
	dstPath      string
	archiveFiles map[string]string
	skipFile     string
}

func FromRemote(addr string, opts ...opt) error {
	s := &FileState{
		archiveFiles: make(map[string]string),
	}

	for _, o := range opts {
		o(s)
	}

	if s.skipFile != "" {
		// Check if binary already exist
		if _, err := os.Stat(s.skipFile); err == nil {
			return nil
		}
	}

	if s.name == "" {
		return fmt.Errorf("fileState.name is empty")
	}
	fmt.Printf("[%s] Fetching %s\n", s.name, addr)

	rStream, cleanup, err := downloadBinary(addr)
	if err != nil {
		return fmt.Errorf("unable to download file: %w", err)
	}
	defer cleanup()

	return s.handleFileStream(rStream, path.Base(addr))
}

func FromLocal(filepath string, opts ...opt) error {
	s := &FileState{
		archiveFiles: make(map[string]string),
	}
	for _, o := range opts {
		o(s)
	}

	if s.skipFile != "" {
		// Check if binary already exist
		if _, err := os.Stat(s.skipFile); err == nil {
			return nil
		}
	}

	f, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("unable to open local file: %w", err)
	}
	defer f.Close()

	return s.handleFileStream(f, path.Base(f.Name()))
}

func (s *FileState) handleFileStream(inFile io.Reader, filename string) error {
	// If no destination path is set we create one with a random uuid
	if s.dstPath == "" {
		// Set a default destination on a temporary path and output filename has
		path, err := os.MkdirTemp("", uuid.NewString())
		if err != nil {
			return fmt.Errorf("unable to creatre temporary directory: %w", err)
		}
		defer os.RemoveAll(path)
		s.dstPath = path
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
		out, err := os.OpenFile(filepath.Join(s.dstPath, filename), os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			return fmt.Errorf("unable to open %s: %w", filename, err)
		}
		defer out.Close()

		// Write the body to file
		_, err = io.Copy(out, inFile)
		if err != nil {
			return fmt.Errorf("unable to download remote file: %w", err)
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
	return nil
}

func WithUnzip() opt {
	return func(f *FileState) {
		f.archiveType = Zip
	}
}

func WithUntarGz() opt {
	return func(f *FileState) {
		f.archiveType = TarGz
	}
}

func WithName(name string) opt {
	return func(f *FileState) {
		f.name = name
	}
}

func WithDestinationDir(path string) opt {
	return func(f *FileState) {
		f.dstPath = path
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
func WithRenameFile(src string, dst string) opt {
	return func(f *FileState) {
		f.archiveFiles[src] = dst
	}
}

func WithSkipIfFileExists(filepath string) opt {
	return func(f *FileState) {
		f.skipFile = filepath
	}
}

func downloadBinary(url string) (io.ReadCloser, func(), error) {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return nil, func() {}, fmt.Errorf("unable to get url: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, func() {}, fmt.Errorf("unable to download %s - %d", url, resp.StatusCode)
	}

	return resp.Body, func() { resp.Body.Close() }, nil
}

// extractZip will decompress a zip archive from the given gzip.Reader into
// the destination path.
// The path must exist already.
func (s *FileState) extractZip(reader *zip.Reader) ([]string, error) {
	var filenames []string
	for _, f := range reader.File {
		dstName := f.Name
		if name, ok := s.archiveFiles[f.Name]; ok {
			dstName = name
		}

		// Store filename/path for returning and using later on
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

func (s *FileState) extractTar(reader io.Reader) error {
	if reader == nil {
		return errors.New("unable to untar nil file")
	}
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("extractTar: Next() failed: %w", err)
		}

		dstName := header.Name
		if name, ok := s.archiveFiles[dstName]; ok {
			dstName = name
		}

		path := filepath.Join(s.dstPath, dstName)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(path, 0o755); err != nil {
				return fmt.Errorf("extractTar: Mkdir() failed: %w", err)
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
			if err := os.Chmod(path, 0775); err != nil {
				return fmt.Errorf("extractTar: Chmod() failed: %s", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("extractTar: Copy() failed: %s", err)
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

func Exists(file string) error {
	if _, err := os.Stat(file); err != nil {
		return err
	}
	return nil
}
