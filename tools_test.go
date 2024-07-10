package toolkit

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)

	if len(s) != 10 {
		t.Errorf("Expected string length to be 10, got %d", len(s))

	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{
		name:         "allowed to not rename file",
		allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false,
	},
	{
		name:         "allowed to rename file",
		allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false,
	},
	{
		name:         "not allow",
		allowedTypes: []string{"image/jpeg"}, renameFile: true, errorExpected: true,
	},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}

		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			// create the form data field 'file'
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			file, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer file.Close()

			img, _, err := image.Decode(file)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads", e.renameFile)
		if e.errorExpected && err == nil {
			t.Error(err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}

			//  clean up
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but none received", e.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		// create the form data field 'file'
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		file, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			t.Error("error decoding image", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	// read from the pipe which receives data
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFiles, err := testTools.UploadOneFile(request, "./testdata/uploads", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	//  clean up
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles.NewFileName))
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTools Tools

	err := testTools.CreateDirIfNotExist("./testdata/dir")

	if err != nil {
		t.Error(err)
	}

	err = testTools.CreateDirIfNotExist("./testdata/dir")

	if err != nil {
		t.Error(err)
	}

	_ = os.Remove("./testdata/dir")
}

var slugifyTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{name: "valid string", s: "Hello, World!", expected: "hello-world", errorExpected: false},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "complex string", s: "L3TS, make #& - A  +- G00D test HERE!", expected: "l3ts-make-a-g00d-test-here", errorExpected: false},
	{name: "Japanese string", s: "こんにちは世界", expected: "", errorExpected: true},
	{name: "Japanese and Roman string", s: "こんにちは世界 HELLO-worlD", expected: "hello-world", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTools Tools

	for _, e := range slugifyTests {
		slug, err := testTools.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error received when none expected: %s", e.name, err.Error())
		}

		if !e.errorExpected && slug != e.expected {
			t.Errorf("%s: wrong slug returned; expected %s, got %s", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "", nil)

	var testTools Tools

	testTools.DownloadStaticFile(rr, req, "./testdata", "pic.jpg", "puppy.png")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "98827" {
		t.Errorf("Expected Content-Length to be 98827, got %s", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"puppy.png\"" {
		t.Errorf("Expected Content-Disposition to be attachment; filename=\"puppy.png\", got %s", res.Header["Content-Disposition"][0])
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

var readJsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{name: "valid json", json: `{"foo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "invalid json", json: `{"foo": "bar"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "json too big", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 1, allowUnknown: false},
	{name: "json with unknown fields", json: `{"foo": "bar", "baz": "qux"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "json with unknown fields allowed", json: `{"foo": "bar", "baz": "qux"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "json with unknown fields disallowed", json: `{"foo": "bar", "baz": "qux"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty json", json: `{}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "empty json with unknown fields", json: `{}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "empty json with unknown fields disallowed", json: `{}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "empty json with unknown fields allowed", json: `{}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "json with incorrect type", json: `{"foo": 1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "json duplicated", json: `{"foo": "bar"}{"foo": "bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "json missing field name", json: `{"": "bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "json is not json", json: `foo`, errorExpected: true, maxSize: 1024, allowUnknown: false},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTools Tools

	for _, e := range readJsonTests {
		// set the max file size
		testTools.MaxJSONSize = e.maxSize

		// allow/disallow unknown fields
		testTools.AllowUnknownFields = e.allowUnknown

		// declare a variable to read the decoded JSON into
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		// create a new request with the JSON data
		req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error:", err)
		}

		// create a new response recorder
		rr := httptest.NewRecorder()

		err = testTools.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: expected error but none received", e.name)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected, but one received: %s", e.name, err.Error())
		}

		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write json: %v", err)
	}
}
