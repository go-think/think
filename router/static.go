package router

import (
	"net/http"
	"strings"

	"github.com/go-think/think/filesystem"
)

type staticHandle struct {
	fileServer http.Handler
	fs         http.FileSystem
}

// NewStaticHandle A Handler responds to a Static HTTP request.
func NewStaticHandle(prefixAndRoot ...string) http.Handler {
	prefix := ""
	root := "."
	if len(prefixAndRoot) == 1 {
		root = prefixAndRoot[0]
	} else if len(prefixAndRoot) >= 2 {
		prefix = "/" + strings.Trim(prefixAndRoot[0], "/")
		root = prefixAndRoot[1]
	}

	fs := filesystem.NewFileFileSystem(root, false)
	var srv http.Handler = http.FileServer(fs)
	if prefix != "" && prefix != "/" {
		srv = http.StripPrefix(prefix, srv)
	}

	return &staticHandle{
		fileServer: srv,
		fs:         fs,
	}
}

// ServeHTTP responds to an Static HTTP request.
func (s *staticHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.fileServer.ServeHTTP(w, r)
}
