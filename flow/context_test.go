package flow

import (
	"bytes"
	"context"
	"io"
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

	// Test standard context.Context wrapping
	assert.NotNil(t, r.Context())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	r.WithContext(ctx)
	assert.Equal(t, ctx, r.Context())

	// Test request key-value pass-through
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
	// CookieHandler being nil should not trigger panic
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

	// Ensure path is cleaned by filepath.Base and exists in uploads directory, preventing path traversal
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

func TestRequestTypeConversionsAndFingerprint(t *testing.T) {
	req, _ := http.NewRequest("GET", "/test?page=3&rate=3.14&active=true&fallback_check=", nil)
	req.Header.Set("User-Agent", "Go-Test-Agent")
	r := NewRequest(req)

	// Test Integer
	assert.Equal(t, 3, r.Integer("page"))
	assert.Equal(t, 10, r.Integer("non_exist", 10))

	// Test Float
	assert.Equal(t, 3.14, r.Float("rate"))
	assert.Equal(t, 0.5, r.Float("non_exist", 0.5))

	// Test Boolean
	assert.True(t, r.Boolean("active"))
	assert.False(t, r.Boolean("non_exist"))
	assert.True(t, r.Boolean("non_exist", true))

	// Test Merge
	r.Merge(map[string]string{"merged_key": "merged_val"})
	val, err := r.Input("merged_key")
	assert.NoError(t, err)
	assert.Equal(t, "merged_val", val)

	// Test Fingerprint
	fp1 := r.Fingerprint()
	assert.NotEmpty(t, fp1)
	assert.Equal(t, fp1, r.Fingerprint())
}

func TestResponseStreamingAndNoContent(t *testing.T) {
	// Test NoContent
	noContent := NoContent()
	rec := httptest.NewRecorder()
	noContent.Send(rec)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())

	// Test StreamResponse
	chunks := []string{"chunk1", "chunk2", "chunk3"}
	idx := 0
	streamResp := StreamResponse(func(w io.Writer) bool {
		if idx >= len(chunks) {
			return false
		}
		_, _ = w.Write([]byte(chunks[idx]))
		idx++
		return idx < len(chunks)
	})

	recStream := httptest.NewRecorder()
	streamResp.Send(recStream)
	assert.Equal(t, http.StatusOK, recStream.Code)
	assert.Equal(t, "chunk1chunk2chunk3", recStream.Body.String())
	assert.Equal(t, "text/event-stream", recStream.Header().Get("Content-Type"))
}

func TestRequestPathAndUrlMethods(t *testing.T) {
	httpReq, _ := http.NewRequest("GET", "/api/v1/users/42?sort=asc&page=1", nil)
	httpReq.Header.Set("User-Agent", "ThinkGo-Agent/1.0")
	r := NewRequest(httpReq)
	r.Set("_route_name", "users.show")

	// 1. UserAgent
	assert.Equal(t, "ThinkGo-Agent/1.0", r.UserAgent())

	// 2. Segments & Segment
	segments := r.Segments()
	assert.Equal(t, []string{"api", "v1", "users", "42"}, segments)
	assert.Equal(t, "api", r.Segment(1))
	assert.Equal(t, "42", r.Segment(4))
	assert.Equal(t, "default", r.Segment(5, "default"))

	// 3. Is & RouteIs
	assert.True(t, r.Is("api/*"))
	assert.True(t, r.Is("api/v1/users/42"))
	assert.False(t, r.Is("web/*"))

	assert.True(t, r.RouteIs("users.show"))
	assert.True(t, r.RouteIs("users.*"))
	assert.False(t, r.RouteIs("orders.*"))

	// 4. FullUrlWithQuery & FullUrlWithoutQuery
	withQuery := r.FullUrlWithQuery(map[string]string{"page": "2", "limit": "20"})
	assert.Contains(t, withQuery, "page=2")
	assert.Contains(t, withQuery, "limit=20")
	assert.Contains(t, withQuery, "sort=asc")

	withoutQuery := r.FullUrlWithoutQuery("sort")
	assert.NotContains(t, withoutQuery, "sort=")
	assert.Contains(t, withoutQuery, "page=1")
}
