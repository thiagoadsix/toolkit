package toolkit

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_+"

// Tools is the type used to instantiate this module. Any variable of this type will have access to all the methods with the receiver *Tools.
type Tools struct {
	MaxFileSize        int
	AllowedFileTypes   []string
	MaxJSONSize        int
	AllowUnknownFields bool
}

// RandomString generates a random string of a specified length using a predefined set of characters.
// Parameters:
// - n: The length of the random string to be generated.
// Returns a string consisting of randomly selected characters from the predefined set.
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

// UploadOneFile processes a single file upload from an HTTP request, saving it to a specified directory.
// Optionally, the file can be renamed during the upload process.
// Parameters:
// - r: The *http.Request containing the file to be uploaded.
// - uploadDir: The directory path where the file will be uploaded.
// - rename: An optional boolean slice indicating whether the file should be renamed (true by default if not specified).
// Returns a pointer to UploadedFile containing information about the uploaded file, or an error if the upload fails.
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

// UploadFiles handles the upload of multiple files from an HTTP request, saving them to a specified directory.
// Optionally, files can be renamed during the upload process.
// Parameters:
// - r: The *http.Request containing the files to be uploaded.
// - uploadDir: The directory path where the files will be uploaded.
// - rename: An optional boolean slice indicating whether the files should be renamed (true by default if not specified).
// Returns a slice of pointers to UploadedFile containing information about the uploaded files, or an error if the upload fails.
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFile := true

	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024
	}

	err := t.CreateDirIfNotExist(uploadDir)
	if err != nil {
		return nil, err
	}

	err = r.ParseMultipartForm(int64(t.MaxFileSize))

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

// CreateDirIfNotExist checks for the existence of a directory and creates it if it does not exist.
// Parameters:
// - path: The path of the directory to check or create.
// Returns an error if the directory cannot be created.
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

// Slugify converts a string into a slug format suitable for URLs, filenames, etc., by removing or replacing characters.
// Parameters:
// - s: The string to be slugified.
// Returns the slugified string and an error if the input string is empty or results in an empty string after processing.
func (t *Tools) Slugify(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty string")
	}

	var regex = regexp.MustCompile(`[^a-z\d]+`)
	slug := strings.Trim(regex.ReplaceAllString(strings.ToLower(s), "-"), "-")

	if len(slug) == 0 {
		return "", errors.New("after removing characters, the string is empty")
	}

	return slug, nil
}

// DownloadStaticFile sends a static file located at the specified path to the client as a downloadable file.
// It sets the HTTP response header to indicate that the content is an attachment, which prompts the browser to download the file.
// Parameters:
// - w: The http.ResponseWriter that is used to write the HTTP response.
// - r: The *http.Request that represents the client's request.
// - path: The base directory path where the static file is located.
// - file: The name of the file to be downloaded.
// - displayName: The name that will be used for the downloaded file on the client's side.
// This function constructs the full file path by joining the base path and the file name, sets the Content-Disposition header
// to make the browser treat the response as a file to be downloaded, and then serves the file using http.ServeFile.
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, path, file, displayName string) {
	filePath := filepath.Join(path, file)

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))

	http.ServeFile(w, r, filePath)
}

// JSONResponse represents the structure of a JSON response.
// Fields:
// - Error: A boolean indicating if the response signifies an error.
// - Message: A string containing a message, typically used for providing feedback to the client.
// - Data: An interface{} that can hold any data type, used for sending the actual response data. It's omitted if empty.
type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ReadJSON reads and decodes JSON from an HTTP request body into a specified data structure.
// It enforces a maximum size for the request body and optionally disallows unknown fields in the JSON payload.
// Parameters:
// - w: The http.ResponseWriter to write responses to.
// - r: The *http.Request containing the JSON to be read.
// - data: The data structure where the decoded JSON will be stored.
// Returns an error if the request body exceeds the maximum size, is empty, contains badly-formed JSON, or other decoding issues occur.
func (t *Tools) ReadJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	maxBytes := 1024 * 1024
	if t.MaxJSONSize != 0 {
		maxBytes = t.MaxJSONSize
	}

	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)

	if !t.AllowUnknownFields {
		dec.DisallowUnknownFields()
	}

	err := dec.Decode(data)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("request body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("request body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("request body contains an invalid value for the %q field", unmarshalTypeError.Field)
			}

			return fmt.Errorf("request body contains an invalid value (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("request body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("request body contains unknown field %s", fieldName)

		case err.Error() == "http: request body too large":
			return fmt.Errorf("request body must not be larger than %d bytes", maxBytes)

		case errors.As(err, &invalidUnmarshalError):
			return fmt.Errorf("error unmarshalling JSON: %s", err.Error())

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON object")
	}

	return nil
}

// WriteJSON sends a JSON response with custom HTTP headers to the client.
// This method marshals the provided data into JSON, sets any provided custom headers, and writes the response to the client.
// Parameters:
// - w: The http.ResponseWriter to write the JSON response to.
// - status: The HTTP status code for the response.
// - data: The data to be marshaled into JSON and sent in the response body.
// - headers: An optional slice of http.Header, allowing for custom headers to be set. Only the first header in the slice is considered if provided.
// Returns an error if marshaling the data into JSON fails or if writing the response fails.
func (t *Tools) WriteJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

// ErrorJSON sends a JSON-formatted error response to the client with an optional HTTP status code.
// This function constructs a JSONResponse struct with the error flag set to true and the error message from the provided error.
// If an HTTP status code is provided in the variadic 'status' parameter, it uses that status code for the response; otherwise, it defaults to http.StatusBadRequest (400).
// Parameters:
// - w: The http.ResponseWriter to write the error response to.
// - err: The error object whose message will be included in the JSON response.
// - status: An optional variadic parameter that allows specifying the HTTP status code for the response. Only the first value is used if multiple are provided.
// Returns an error if writing the JSON response fails.
func (t *Tools) ErrorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload JSONResponse
	payload.Error = true
	payload.Message = err.Error()

	return t.WriteJSON(w, statusCode, payload)
}

// PushJSONToRemote sends a JSON payload to a specified URI using an HTTP POST request.
// This function allows for an optional http.Client to be specified for the request; if none is provided, a default client is used.
// Parameters:
// - uri: The URI where the JSON data will be sent.
// - data: The data to be marshaled into JSON and sent in the request body.
// - client: An optional variadic parameter that allows specifying a custom http.Client for the request. Only the first client is used if multiple are provided.
// Returns the HTTP response, the response status code, and an error if the request fails at any point.
func (t *Tools) PushJSONToRemote(uri string, data interface{}, client ...*http.Client) (*http.Response, int, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, 0, err
	}

	httpClient := &http.Client{}
	if len(client) > 0 {
		httpClient = client[0]
	}

	request, err := http.NewRequest(http.MethodPost, uri, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()

	return response, response.StatusCode, nil
}
