package flow

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
)

type Response struct {
	// Writer      flow.ResponseWriter
	contentType   string
	charset       string
	code          int
	content       string
	filePath      string
	Request       *Request
	cookies       map[string]*http.Cookie
	CookieHandler *Cookie
	Header        *http.Header
	streamFunc    func(w io.Writer) bool
	handled       bool
}

// HandledResponse creates a response indicating that output was directly handled.
func HandledResponse() *Response {
	r := NewResponse()
	r.handled = true
	return r
}

// IsHandled returns whether the response was already handled directly.
func (r *Response) IsHandled() bool {
	return r.handled
}

// FileResponse Create a response that serves a file
func FileResponse(filepath string) *Response {
	r := NewResponse()
	r.filePath = filepath
	return r
}

// SetFile set a file path to be served
func (r *Response) SetFile(filepath string) *Response {
	r.filePath = filepath
	return r
}

// SetRequest bind original request for file serving
func (r *Response) SetRequest(req *Request) *Response {
	r.Request = req
	return r
}

// GetContentType sets the Content-Type on the response.
func (r *Response) SetContentType(val string) *Response {
	r.contentType = val
	return r
}

// GetContentType sets the Charset on the response.
func (r *Response) SetCharset(val string) *Response {
	r.charset = val
	return r
}

// SetCode sets the status code on the response.
func (r *Response) SetCode(val int) *Response {
	r.code = val
	return r
}

// SetContent sets the content on the response.
func (r *Response) SetContent(val string) *Response {
	r.content = val
	return r
}

func FormatContent(v interface{}) string {
	if v == nil {
		return ""
	}
	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Bool:
		return fmt.Sprintf("%t", v)
	case reflect.String:
		return fmt.Sprintf("%s", v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%v", v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}

// GetContentType get the Content-Type on the response.
func (r *Response) GetContentType() string {
	return r.contentType
}

// GetContentType get the Charset on the response.
func (r *Response) GetCharset() string {
	return r.charset
}

// GetCode get the response status code.
func (r *Response) GetCode() int {
	return r.code
}

// GetCode get the response content.
func (r *Response) GetContent() string {
	return r.content
}

// Cookie Add a cookie to the response.
func (r *Response) Cookie(name interface{}, params ...interface{}) error {
	if r.CookieHandler == nil {
		r.CookieHandler = ParseCookieHandler()
	}
	cookie, err := r.CookieHandler.Set(name, params...)

	if err == nil && cookie != nil {
		if r.cookies == nil {
			r.cookies = make(map[string]*http.Cookie)
		}
		r.cookies[cookie.Name] = cookie
	}

	return err
}

// Send Sends HTTP headers and content.
func (r *Response) Send(w http.ResponseWriter) {
	if r.handled {
		return
	}

	for _, cookie := range r.cookies {
		http.SetCookie(w, cookie)
	}
	for key, value := range *r.Header {
		for _, val := range value {
			w.Header().Add(key, val)
		}
	}

	// If filePath is set, prioritize using http.ServeFile to serve the file
	if r.filePath != "" {
		if r.Request != nil && r.Request.Request != nil {
			http.ServeFile(w, r.Request.Request, r.filePath)
		}
		return
	}

	// If streamFunc is set, handle streaming output (stream / SSE)
	if r.streamFunc != nil {
		if r.GetContentType() != "" {
			w.Header().Set("Content-Type", r.GetContentType())
		}
		w.WriteHeader(r.GetCode())

		flusher, isFlusher := w.(http.Flusher)
		for {
			keepStreaming := r.streamFunc(w)
			if isFlusher {
				flusher.Flush()
			}
			if !keepStreaming {
				break
			}
		}
		return
	}

	w.Header().Set("Content-Type", r.GetContentType()+";"+" charset="+r.GetCharset())
	// r.Header.Write(w)
	w.WriteHeader(r.GetCode())
	w.Write([]byte(r.GetContent()))
}

// NewResponse Create a new HTTP Response
func NewResponse() *Response {
	r := &Response{
		Header: &http.Header{},
	}
	r.SetCode(http.StatusOK)
	r.SetContentType("text/html")
	r.SetCharset("utf-8")
	r.CookieHandler = ParseCookieHandler()
	return r
}

// NotFoundResponse Create a new HTTP NotFoundResponse
func NotFoundResponse() *Response {
	return NewResponse().SetCode(http.StatusNotFound).SetContent("Not Found")
}

// DownloadResponse Create a new HTTP Download Response
func DownloadResponse(filePath string, filename ...string) *Response {
	r := NewResponse()
	r.filePath = filePath

	name := filepath.Base(filePath)
	if len(filename) > 0 {
		name = filename[0]
	}

	r.Header.Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	r.Header.Set("Content-Type", "application/octet-stream")
	return r
}

// NotFoundResponse Create a new HTTP Error Response
func ErrorResponse() *Response {
	return NewResponse().SetCode(http.StatusInternalServerError).SetContent("Server Error")
}

// Redirect Create a new HTTP Redirect Response
func Redirect(to string) *Response {
	r := NewResponse().SetCode(http.StatusMovedPermanently)
	r.Header.Set("Location", to)
	return r
}

// SetStream sets a streaming callback for the response.
func (r *Response) SetStream(streamFunc func(w io.Writer) bool) *Response {
	r.streamFunc = streamFunc
	return r
}

// StreamResponse creates a new streaming HTTP Response.
func StreamResponse(streamFunc func(w io.Writer) bool) *Response {
	r := NewResponse()
	r.SetContentType("text/event-stream")
	r.Header.Set("Cache-Control", "no-cache")
	r.Header.Set("Connection", "keep-alive")
	r.streamFunc = streamFunc
	return r
}

// StreamDownload creates a new streaming download response.
func StreamDownload(streamFunc func(w io.Writer) bool, filename string) *Response {
	r := NewResponse()
	r.Header.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	r.Header.Set("Content-Type", "application/octet-stream")
	r.streamFunc = streamFunc
	return r
}

// NoContent creates a new 204 No Content Response.
func NoContent(status ...int) *Response {
	code := http.StatusNoContent
	if len(status) > 0 {
		code = status[0]
	}
	return NewResponse().SetCode(code).SetContent("")
}

// Json Create a new HTTP Response with JSON data
func Json(v interface{}) *Response {
	c, err := json.Marshal(v)
	if err != nil {
		return NewResponse().SetContent("").SetContentType("application/json")
	}
	return NewResponse().SetContent(string(c)).SetContentType("application/json")
}

// Text Create a new HTTP Response with TEXT data
func Text(s string) *Response {
	return NewResponse().SetContent(s).SetContentType("text/plain")
}

// Html Create a new HTTP Response with HTML data
func Html(s string) *Response {
	return NewResponse().SetContent(s)
}

// Download Create a new HTTP Download Response
func Download(filePath string, filename ...string) *Response {
	return DownloadResponse(filePath, filename...)
}

// MakeResponse Create a new HTTP Response by auto detecting content type
func MakeResponse(v interface{}) *Response {
	r := NewResponse()
	if v == nil {
		return r.SetContent("")
	}

	content := FormatContent(v)
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Map || t.Kind() == reflect.Slice || t.Kind() == reflect.Struct {
		r.SetContentType("application/json")
	} else {
		r.SetContentType("text/plain")
	}
	r.SetContent(content)

	return r
}
