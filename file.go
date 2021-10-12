package tools

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func ExtractZip(src string, dest string) ([]string, error) {
	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
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

func DownloadBinary(binDir string, binURL string, binary string) (bool, error) {
	// Check if binary already exist
	if _, err := os.Stat(binary); err == nil {
		return false, nil
	}

	// Get the data
	resp, err := http.Get(binURL)
	if err != nil {
		return false, fmt.Errorf("unable to get url: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("unable to download %s - %d", binURL, resp.StatusCode)
	}

	// Create the file
	err = os.MkdirAll(binDir, 00755)
	if err != nil {
		return false, fmt.Errorf("unable to create %s: %w", binDir, err)
	}

	out, err := os.OpenFile(binary, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return false, fmt.Errorf("unable to open %s: %w", binary, err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return false, fmt.Errorf("unable to copy %s from http stream: %w", binary, err)
	}

	return true, nil
}

func ExtractTarGz(src string, dest string) error {
	srcStream, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcStream.Close()
	uncompressedStream, err := gzip.NewReader(srcStream)
	if err != nil {
		return fmt.Errorf("ExtractTarGz: NewReader failed: %w", err)
	}

	tarReader := tar.NewReader(uncompressedStream)

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("ExtractTarGz: Next() failed: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(header.Name, 0755); err != nil {
				return fmt.Errorf("ExtractTarGz: Mkdir() failed: %w", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(header.Name)
			if err != nil {
				return fmt.Errorf("ExtractTarGz: Create() failed: %w", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("ExtractTarGz: Copy() failed: %s", err)
			}
			outFile.Close()

		default:
			return fmt.Errorf(
				"ExtractTarGz: uknown type: %s in %s",
				header.Typeflag,
				header.Name)
		}
	}
	return nil
}
