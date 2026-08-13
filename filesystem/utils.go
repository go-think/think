package filesystem

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

func Exists(p ...string) (bool, error) {
	_, err := os.Stat(filepath.Join(p...))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func Get(path string) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

func Put(path string, data string) error {
	dir := filepath.Dir(path)
	if ok, _ := Exists(dir); !ok {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(data)
	return err
}

func ModTime(path string) (time.Time, error) {
	var modTime time.Time
	fileInfo, err := os.Stat(path)
	if err != nil {
		return modTime, err
	}
	return fileInfo.ModTime(), nil
}
