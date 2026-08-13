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
