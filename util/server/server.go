package server

import (
	"io/fs"
	"net/http"
	"path"
	"path/filepath"

	"github.com/go-openapi/runtime/middleware"
)

const (
	redocBundleURL = "https://cdn.jsdelivr.net/npm/redoc/bundles/redoc.standalone.js"
	specURL        = "/swagger.json"
)

var defaultHeaders = map[string]string{
	"X-Frame-Options":  "1",
	"X-XSS-Protection": "1",
}

type staticFS struct {
	fsys http.FileSystem
}

func (sfs *staticFS) Open(path string) (http.File, error) {
	f, err := sfs.fsys.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if s.IsDir() {
		index := filepath.Join(path, "index.html")
		if _, err := sfs.fsys.Open(index); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func NewStaticAssetsHandler(fsys fs.FS) http.HandlerFunc {
	fileserver := http.FileServer(&staticFS{http.FS(fsys)})

	return func(w http.ResponseWriter, r *http.Request) {
		for k, v := range defaultHeaders {
			w.Header().Set(k, v)
		}

		fileserver.ServeHTTP(w, r)
	}
}

func NewOpenAPIHandler(fsys fs.FS, handlePath string) http.HandlerFunc {
	mux := http.NewServeMux()

	mux.Handle(handlePath, middleware.Redoc(middleware.RedocOpts{
		SpecURL:  specURL,
		Path:     path.Base(handlePath),
		RedocURL: redocBundleURL,
	}, http.NotFoundHandler()))

	return mux.ServeHTTP
}

type handlerSwitch struct {
	base                http.Handler
	contentTypeHandlers map[string]http.Handler
}

func NewGRPCWebHandler(base http.Handler, grpcWebHandler http.Handler) http.Handler {
	return &handlerSwitch{
		base: base,
		contentTypeHandlers: map[string]http.Handler{
			"application/grpc-web+proto": grpcWebHandler,
		},
	}
}

func (h *handlerSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler := h.contentTypeHandlers[r.Header.Get("Content-Type")]; handler != nil {
		handler.ServeHTTP(w, r)
	} else {
		h.base.ServeHTTP(w, r)
	}
}
