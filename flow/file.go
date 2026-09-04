package flow

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type File struct {
	FileHeader *multipart.FileHeader
}

func (f *File) Move(directory string, name ...string) (bool, error) {
	if f.FileHeader == nil {
		return false, errors.New("nil file header")
	}

	src, err := f.FileHeader.Open()
	if err != nil {
		return false, err
	}
	defer src.Close()

	fname := filepath.Base(f.FileHeader.Filename)
	if len(name) > 0 && name[0] != "" {
		fname = filepath.Base(name[0])
	}

	if err := os.MkdirAll(directory, 0755); err != nil {
		return false, err
	}

	dst := filepath.Join(directory, fname)

	out, err := os.Create(dst)
	if err != nil {
		return false, err
	}
	defer out.Close()

	if _, err = io.Copy(out, src); err != nil {
		return false, err
	}

	return true, nil
}
