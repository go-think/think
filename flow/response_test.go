package flow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJson(t *testing.T) {
	data := map[string]string{"foo": "bar"}
	res := Json(data)

	if res.GetContentType() != "application/json" {
		t.Fatalf("expected content type application/json, got %s", res.GetContentType())
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(res.GetContent()), &parsed); err != nil {
		t.Fatalf("failed to parse json content: %v", err)
	}

	if parsed["foo"] != "bar" {
		t.Fatalf("expected foo=bar, got %s", parsed["foo"])
	}
}

func TestText(t *testing.T) {
	text := "hello thinkgo"
	res := Text(text)

	if res.GetContentType() != "text/plain" {
		t.Fatalf("expected content type text/plain, got %s", res.GetContentType())
	}

	if res.GetContent() != text {
		t.Fatalf("expected %s, got %s", text, res.GetContent())
	}
}

func TestHtml(t *testing.T) {
	html := "<h1>hello</h1>"
	res := Html(html)

	if res.GetContentType() != "text/html" {
		t.Fatalf("expected content type text/html, got %s", res.GetContentType())
	}

	if res.GetContent() != html {
		t.Fatalf("expected %s, got %s", html, res.GetContent())
	}
}

func TestDownload(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmpFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	res := Download(tmpFile, "custom.txt")
	if res.filePath != tmpFile {
		t.Fatalf("expected filePath %s, got %s", tmpFile, res.filePath)
	}

	disposition := res.Header.Get("Content-Disposition")
	expected := "attachment; filename=\"custom.txt\""
	if disposition != expected {
		t.Fatalf("expected disposition %s, got %s", expected, disposition)
	}
}

func TestMakeResponse(t *testing.T) {
	// 1. nil
	nilRes := MakeResponse(nil)
	if nilRes.GetContent() != "" {
		t.Fatalf("expected empty content for nil, got %s", nilRes.GetContent())
	}

	// 2. string
	strRes := MakeResponse("plain string")
	if strRes.GetContentType() != "text/plain" || strRes.GetContent() != "plain string" {
		t.Fatalf("unexpected string response: %v, %s", strRes.GetContentType(), strRes.GetContent())
	}

	// 3. map -> json
	mapData := map[string]int{"num": 42}
	mapRes := MakeResponse(mapData)
	if mapRes.GetContentType() != "application/json" {
		t.Fatalf("expected application/json for map, got %s", mapRes.GetContentType())
	}

	// 4. slice -> json
	sliceData := []string{"a", "b"}
	sliceRes := MakeResponse(sliceData)
	if sliceRes.GetContentType() != "application/json" {
		t.Fatalf("expected application/json for slice, got %s", sliceRes.GetContentType())
	}

	// 5. struct -> json
	type Sample struct {
		Name string `json:"name"`
	}
	structRes := MakeResponse(Sample{Name: "think"})
	if structRes.GetContentType() != "application/json" {
		t.Fatalf("expected application/json for struct, got %s", structRes.GetContentType())
	}
}
