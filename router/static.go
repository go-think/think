package router

import (
	"net/http"

	"github.com/go-think/think/filesystem"
)

type staticHandle struct {
	fileServer http.Handler
	fs         http.FileSystem
}

// NewStaticHandle A Handler responds to a Static HTTP request.
func NewStaticHandle(root string) http.Handler {
	fs := filesystem.NewFileFileSystem(root, false)
	return &staticHandle{
		fileServer: http.FileServer(fs),
		fs:         fs,
	}
}

// ServeHTTP  responds to an Static HTTP request.
func (s *staticHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.fileServer.ServeHTTP(w, r)
}
