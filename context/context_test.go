package context

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRequestContextAndKV(t *testing.T) {
	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	r := NewRequest(req)

	// 测试标准 context.Context 包装
	assert.NotNil(t, r.Context())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	r.WithContext(ctx)
	assert.Equal(t, ctx, r.Context())

	// 测试请求键值透传
	r.Set("trace_id", "abc-123")
	val, ok := r.Get("trace_id")
	assert.True(t, ok)
	assert.Equal(t, "abc-123", val)

	_, ok = r.Get("non_exist")
	assert.False(t, ok)
}

func TestFileResponse(t *testing.T) {
	tmpDir := os.TempDir()
	filePath := filepath.Join(tmpDir, "test_file.txt")
	err := os.WriteFile(filePath, []byte("hello file response"), 0644)
	assert.NoError(t, err)
	defer os.Remove(filePath)

	req, err := http.NewRequest("GET", "/file", nil)
	assert.NoError(t, err)

	r := NewRequest(req)
	resp := FileResponse(filePath).SetRequest(r)

	rec := httptest.NewRecorder()
	resp.Send(rec)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello file response", rec.Body.String())
}

func TestRouteParamsIsolation(t *testing.T) {
	req, err := http.NewRequest("GET", "/user/42", nil)
	assert.NoError(t, err)
	r := NewRequest(req)

	r.SetRouteParam("id", "42")
	val, err := r.RouteParam("id")
	assert.NoError(t, err)
	assert.Equal(t, "42", val)
	assert.Equal(t, "42", r.GetRouteParam("id"))

	assert.Equal(t, "default", r.GetRouteParam("non_exist", "default"))
}

func TestCookieNilHandlerSafety(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	r := NewRequest(req)
	// CookieHandler 为 nil 时不应触发 panic
	val, err := r.Cookie("session_id", "default_val")
	assert.NoError(t, err)
	assert.Equal(t, "default_val", val)
}

func TestFileMovePathTraversalProtection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "file_test_*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "../../malicious.txt")
	assert.NoError(t, err)
	_, _ = part.Write([]byte("malicious content"))
	_ = writer.Close()

	req, err := http.NewRequest("POST", "/upload", body)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	r := NewRequest(req)
	file, err := r.File("file")
	assert.NoError(t, err)

	targetDir := filepath.Join(tmpDir, "uploads")
	ok, err := file.Move(targetDir)
	assert.NoError(t, err)
	assert.True(t, ok)

	// 确认路径被 filepath.Base 洗盘，存在于 uploads 目录下，而不是逃逸出去
	assert.FileExists(t, filepath.Join(targetDir, "malicious.txt"))
}

func TestConcurrentSetAndGet(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	r := NewRequest(req)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r.Set("key", idx)
			_, _ = r.Get("key")
			r.SetRouteParam("p", "v")
			_ = r.GetRouteParam("p")
		}(i)
	}
	wg.Wait()
}
