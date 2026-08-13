package session

import (
	"crypto/sha256"
	"encoding/hex"
	"path"
	"time"

	"github.com/go-think/think/filesystem"
)

type FileHandler struct {
	Path     string
	Lifetime time.Duration
}

func (c *FileHandler) getSavePath(id string) string {
	if id == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(id))
	filename := hex.EncodeToString(hash[:])
	return path.Join(c.Path, filename)
}

func (c *FileHandler) Read(id string) string {
	savePath := c.getSavePath(id)
	if savePath == "" {
		return ""
	}

	if ok, _ := filesystem.Exists(savePath); ok {
		modTime, _ := filesystem.ModTime(savePath)
		if modTime.After(time.Now().Add(-c.Lifetime)) {
			data, err := filesystem.Get(savePath)
			if err != nil {
				return ""
			}
			return string(data)
		}
	}
	return ""
}

func (c *FileHandler) Write(id string, data string) {
	savePath := c.getSavePath(id)
	if savePath == "" {
		return
	}

	_ = filesystem.Put(savePath, data)
}

