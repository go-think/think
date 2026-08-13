package context

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

type Response struct {
	// Writer      context.ResponseWriter
	contentType   string
	charset       string
	code          int
	content       string
	filePath      string
	Request       *Request
	cookies       map[string]*http.Cookie
	CookieHandler *Cookie
	Header        *http.Header
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
	for _, cookie := range r.cookies {
		http.SetCookie(w, cookie)
	}
	for key, value := range *r.Header {
		for _, val := range value {
			w.Header().Add(key, val)
		}
	}

	// 如果设置了文件路径，优先使用 http.ServeFile 传输文件
	if r.filePath != "" {
		if r.Request != nil && r.Request.Request != nil {
			http.ServeFile(w, r.Request.Request, r.filePath)
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
