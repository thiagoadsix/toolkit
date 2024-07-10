# Toolkit Library

The Toolkit Library is a Go package that provides a collection of utilities for handling various tasks such as file uploads, JSON processing, random string generation, and more. This library aims to simplify common operations in Go web applications.

## Features

- Generate random strings.
- Upload and process single or multiple files.
- Create directories if they do not exist.
- Convert strings to URL-friendly slugs.
- Serve static files for download.
- Read and write JSON with custom error handling.
- Push JSON data to remote servers.

## Installation

To use the Toolkit Library in your project, you can import it as a module.

```sh
go get -u github.com/thiagoadsix/toolkit
```

## Usage

### Initializing the Toolkit

Create an instance of the Tools struct to access the utility methods.
```go
import "github.com/your-username/toolkit"

var tools toolkit.Tools
```

#### Generate a Random String
Generate a random string of a specified length.
```go
randomString := tools.RandomString(10)
```

#### Uploading Files
Upload a single file.
```go
uploadedFile, err := tools.UploadOneFile(r *http.Request, uploadDir string, rename ...bool)
if err != nil {
    // Handle error
}
```

Upload multiple files.
```go
uploadedFile, err := tools.UploadOneFile(r *http.Request, uploadDir string, rename ...bool)
if err != nil {
    // Handle error
}
```
#### Create Directory if Not Exists
Create a directory if it does not exist.
```go
err := tools.CreateDirIfNotExist(path string)
if err != nil {
    // Handle error
}
```

#### Slugify a String
Convert a string to a URL-friendly slug.
```go
slug, err := tools.Slugify("Your String Here")
if err != nil {
    // Handle error
}
```

#### Download Static Files
Serve a static file for download.
```go
tools.DownloadStaticFile(w http.ResponseWriter, r *http.Request, path string, file string, displayName string)
```

#### Read JSON from Request
Read JSON data from an HTTP request.
```go
var data YourStruct
err := tools.ReadJSON(w http.ResponseWriter, r *http.Request, &data)
if err != nil {
    // Handle error
}
```

#### Write JSON Response
Write a JSON response to the client.
```go
err := tools.WriteJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header)
if err != nil {
    // Handle error
}
```

#### Send JSON Error Response
Send a JSON-formatted error response.
```go
err := tools.ErrorJSON(w http.ResponseWriter, err error, status ...int)
if err != nil {
    // Handle error
}
```

#### Push JSON to Remote Server
Send JSON data to a remote server using HTTP POST.
```go
response, statusCode, err := tools.PushJSONToRemote(uri string, data interface{}, client ...*http.Client)
if err != nil {
    // Handle error
}
```