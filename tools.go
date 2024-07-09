package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module. Any variable of this type will have access to all the methods with the receiver *Tools.
type Tools struct {
	MaxFileSize      int
	AllowedFileTypes []string
}

// RandomString generates a random string of length n, using the characters in randomStringSource.
func (t *Tools) RandomString(n int) string {
	s, r := make([]rune, n), []rune(randomStringSource)

	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))

		s[i] = r[x%y]
	}
	return string(s)
}

// UploadedFile is the type used to store information about a file that has been uploaded.
type UploadedFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int64
}

// UploadOneFile processes a single file upload request. It accepts an HTTP request, a directory to upload the file to, and an optional boolean
// parameter to indicate whether the file should be renamed. It returns a pointer to the uploaded file information or an error if the upload fails.
func (t *Tools) UploadOneFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
	renameFile := true

	if len(rename) > 0 {
		renameFile = rename[0]
	}

	files, err := t.UploadFiles(r, uploadDir, renameFile)

	if err != nil {
		return nil, err
	}

	return files[0], nil
}

// UploadFiles handles multiple file uploads from an HTTP request. It takes the request, the directory where files should be uploaded, and an optional
// boolean parameter to indicate whether files should be renamed upon upload. It returns a slice of pointers to the uploaded file information or an error if the upload process fails.
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFile := true

	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024
	}

	err := r.ParseMultipartForm(int64(t.MaxFileSize))

	if err != nil {
		return nil, errors.New("the uploaded file is too big")
	}

	for _, fHeaders := range r.MultipartForm.File {
		for _, hdr := range fHeaders {
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
				var uploadedFile UploadedFile

				infoFile, err := hdr.Open()

				if err != nil {
					return nil, err
				}

				defer infoFile.Close()

				buff := make([]byte, 512)

				_, err = infoFile.Read(buff)

				if err != nil {
					return nil, err
				}

				allowed := false
				fileType := http.DetectContentType(buff)

				if len(t.AllowedFileTypes) > 0 {
					for _, x := range t.AllowedFileTypes {
						if strings.EqualFold(fileType, x) {
							allowed = true
						}
					}
				} else {
					allowed = true
				}

				if !allowed {
					return nil, errors.New("file type not allowed")
				}

				_, err = infoFile.Seek(0, 0)

				if err != nil {
					return nil, err
				}

				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(hdr.Filename))
				} else {
					uploadedFile.NewFileName = hdr.Filename
				}

				uploadedFile.OriginalFileName = hdr.Filename

				var outFile *os.File

				defer outFile.Close()

				if outFile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outFile, infoFile)

					if err != nil {
						return nil, err
					}

					uploadedFile.FileSize = fileSize
				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil
			}(uploadedFiles)

			if err != nil {
				return uploadedFiles, err
			}
		}
	}

	return uploadedFiles, nil
}

// CreateDirIfNotExist checks if a directory exists and creates it if it does not. It takes a string parameter representing the path to the directory and returns an error if the directory cannot be created.
func (t *Tools) CreateDirIfNotExist(path string) error {
	const mode = 0755
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, mode)

		if err != nil {
			return err
		}
	}
	return nil
}
