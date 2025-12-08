package session

import (
	"path"
	"time"

	"github.com/go-think/think/filesystem"
)

type FileHandler struct {
	Path     string
	Lifetime time.Duration
}

func (c *FileHandler) Read(id string) string {

	savePath := path.Join(c.Path, id)

	if ok, _ := filesystem.Exists(savePath); ok {
		modTime, _ := filesystem.ModTime(savePath)
		if modTime.After(time.Now().Add(-c.Lifetime)) {
			data, err := filesystem.Get(savePath)
			if err != nil {
				panic(err)
			}
			return string(data)
		}
	}
	return ""
}

func (c *FileHandler) Write(id string, data string) {
	savePath := path.Join(c.Path, id)

	err := filesystem.Put(savePath, data)

	if err != nil {
		panic(err)
	}
}
