package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConcurrentPutAndGet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fs_test_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	var wg sync.WaitGroup
	count := 50

	for i := 0; i < count; i++ {
		wg.Add(1)
		fileIdx := i
		go func(idx int) {
			defer wg.Done()
			fpath := filepath.Join(tmpDir, fmt.Sprintf("file_%d.txt", idx))
			content := fmt.Sprintf("data_%d", idx)

			err := Put(fpath, content)
			assert.NoError(t, err)

			exists, err := Exists(fpath)
			assert.NoError(t, err)
			assert.True(t, exists)

			data, err := Get(fpath)
			assert.NoError(t, err)
			assert.Equal(t, content, string(data))
		}(fileIdx)
	}

	wg.Wait()
}

func TestDirectoryOperationsAndSize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fs_dir_test_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	dstDir := filepath.Join(tmpDir, "dst")

	// 1. Create nested structure
	f1 := filepath.Join(srcDir, "sub", "hello.txt")
	assert.NoError(t, Put(f1, "hello filesystem"))

	// 2. Test Size
	sz, err := Size(f1)
	assert.NoError(t, err)
	assert.Equal(t, int64(len("hello filesystem")), sz)

	// 3. Test CopyDirectory
	assert.NoError(t, CopyDirectory(srcDir, dstDir))
	copiedFile := filepath.Join(dstDir, "sub", "hello.txt")
	exists, err := Exists(copiedFile)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 4. Test CleanDirectory
	assert.NoError(t, CleanDirectory(dstDir))
	existsAfterClean, _ := Exists(copiedFile)
	assert.False(t, existsAfterClean)
	// Directory itself should still exist
	dstExists, _ := Exists(dstDir)
	assert.True(t, dstExists)

	// 5. Test Delete
	assert.NoError(t, Delete(srcDir, dstDir))
	srcExists, _ := Exists(srcDir)
	assert.False(t, srcExists)
}
