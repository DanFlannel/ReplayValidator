package docs

import (
	"net/http"
	"os"
	"path"

	"github.com/mvrilo/go-redoc"
)

// specFile is where `make gen-api` writes each service's spec, relative to the
// service's working directory.
const specFile = "./static/openapi.yaml"

// GetHandler returns a handler that serves Redoc API documentation at docsPath,
// with the raw spec available at docsPath + "/openapi.yaml".
//
// redoc's own Handler panics when the spec file cannot be read, which would take
// a service down at startup over its documentation. The file is checked here
// first so a missing spec degrades to a 503 that says what is wrong.
func GetHandler(docsPath string) http.HandlerFunc {
	_, err := os.ReadFile(specFile)
	if err != nil {
		return func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "API documentation unavailable: "+specFile+" could not be read",
				http.StatusServiceUnavailable)
		}
	}

	doc := redoc.Redoc{
		Title:       "API documentation",
		Description: "OpenAPI specification",
		SpecFile:    specFile,
		SpecPath:    path.Join(docsPath, "openapi.yaml"),
		DocsPath:    docsPath,
	}

	return doc.Handler()
}
