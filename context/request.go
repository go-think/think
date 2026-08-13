package context

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// Request HTTP request
type Request struct {
	Request       *http.Request
	ctx           context.Context
	method        string
	path          string
	query         map[string]string
	queryValues   url.Values
	post          map[string]string
	postValues    url.Values
	routeParams   map[string]string
	files         map[string]*File
	bodyContent   []byte
	session       Session
	CookieHandler *Cookie
	keys          map[string]interface{}
	keysMu        sync.RWMutex
	parseOnce     sync.Once
}

// NewRequest create a new HTTP request from *http.Request
func NewRequest(req *http.Request) *Request {
	ctx := context.Background()
	method := ""
	path := ""
	if req != nil {
		if req.Context() != nil {
			ctx = req.Context()
		}
		method = req.Method
		if req.URL != nil {
			path = req.URL.Path
		}
	}
	return &Request{
		Request:     req,
		ctx:         ctx,
		method:      method,
		path:        path,
		routeParams: make(map[string]string),
		files:       make(map[string]*File),
		keys:        make(map[string]interface{}),
	}
}

// Context returns the request's context.Context
func (r *Request) Context() context.Context {
	if r.ctx == nil {
		return context.Background()
	}
	return r.ctx
}

// WithContext sets the request's context.Context
func (r *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		return r
	}
	r.ctx = ctx
	if r.Request != nil {
		r.Request = r.Request.WithContext(ctx)
	}
	return r
}

// Set store a new key/value pair in this context
func (r *Request) Set(key string, value interface{}) {
	r.keysMu.Lock()
	defer r.keysMu.Unlock()
	if r.keys == nil {
		r.keys = make(map[string]interface{})
	}
	r.keys[key] = value
}

// Get returns the value for the given key
func (r *Request) Get(key string) (value interface{}, exists bool) {
	r.keysMu.RLock()
	defer r.keysMu.RUnlock()
	value, exists = r.keys[key]
	return
}

// GetMethod get the request method.
func (r *Request) GetMethod() string {
	return r.method
}

// GetPath get the request path.
func (r *Request) GetPath() string {
	return r.path
}

// GetHttpRequest get Current *http.Request
func (r *Request) GetHttpRequest() *http.Request {
	return r.Request
}

//IsMethod checks if the request method is of specified type.
func (r *Request) IsMethod(m string) bool {
	return strings.ToUpper(m) == r.GetMethod()
}

// SetRouteParam sets a route parameter
func (r *Request) SetRouteParam(key, value string) {
	r.keysMu.Lock()
	defer r.keysMu.Unlock()
	if r.routeParams == nil {
		r.routeParams = make(map[string]string)
	}
	r.routeParams[key] = value
}

// GetRouteParam gets a route parameter
func (r *Request) GetRouteParam(key string, defaultValue ...string) string {
	r.keysMu.RLock()
	defer r.keysMu.RUnlock()
	if v, ok := r.routeParams[key]; ok {
		return v
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// RouteParam returns a route parameter with error if not present
func (r *Request) RouteParam(key string, defaultValue ...string) (string, error) {
	r.keysMu.RLock()
	defer r.keysMu.RUnlock()
	if v, ok := r.routeParams[key]; ok {
		return v, nil
	}
	if len(defaultValue) > 0 {
		return defaultValue[0], nil
	}
	return "", errors.New("route parameter not present")
}

func (r *Request) parseInputOnce() {
	r.parseOnce.Do(func() {
		if r.Request != nil && r.Request.URL != nil {
			r.queryValues = r.Request.URL.Query()
			r.query = parseQuery(r.queryValues)
		}
		if r.Request != nil {
			r.postValues, r.post = parsePost(r.Request)
		}
	})
}

// Query returns a query string item from the request.
func (r *Request) Query(key string, value ...string) (string, error) {
	r.parseInputOnce()
	if v, ok := r.query[key]; ok {
		return v, nil
	}
	if len(value) > 0 {
		return value[0], nil
	}
	return "", errors.New("named query not present")
}

// Input returns a input item from the request.
func (r *Request) Input(key string, value ...string) (string, error) {
	r.parseInputOnce()
	if v, ok := r.post[key]; ok {
		return v, nil
	}

	if v, ok := r.query[key]; ok {
		return v, nil
	}

	if len(value) > 0 {
		return value[0], nil
	}
	return "", errors.New("named input not present")
}

// Post returns a post item from the request.
func (r *Request) Post(key string, value ...string) (string, error) {
	r.parseInputOnce()
	if v, ok := r.post[key]; ok {
		return v, nil
	}

	if len(value) > 0 {
		return value[0], nil
	}
	return "", errors.New("named post not present")
}

//Cookie Retrieve a cookie from the request.
func (r *Request) Cookie(key string, value ...string) (string, error) {
	var err error
	if r.Request == nil {
		if len(value) > 0 {
			return value[0], nil
		}
		return "", errors.New("nil http request")
	}
	if r.CookieHandler == nil {
		r.CookieHandler = ParseCookieHandler()
	}
	prefix := ""
	if r.CookieHandler != nil && r.CookieHandler.Config != nil {
		prefix = r.CookieHandler.Config.Prefix
	}
	key = prefix + key
	cookie, err := r.Request.Cookie(key)
	if err == nil {
		c, _ := url.QueryUnescape(cookie.Value)
		return c, err
	}
	if len(value) > 0 {
		return value[0], nil
	}
	return "", err
}

// File returns a file from the request.
func (r *Request) File(key string) (*File, error) {
	if f, ok := r.files[key]; ok {
		return f, nil
	}
	if r.Request == nil {
		return nil, errors.New("nil http request")
	}
	_, fh, err := r.Request.FormFile(key)
	if err != nil {
		return nil, err
	}
	r.files[key] = &File{fh}

	return r.files[key], nil
}

// AllFiles returns all files from the request.
func (r *Request) AllFiles() (map[string]*File, error) {
	if r.Request == nil {
		return nil, errors.New("nil http request")
	}
	err := r.Request.ParseMultipartForm(32 << 20)
	if err != nil && err != http.ErrNotMultipart {
		return nil, err
	}
	if r.Request.MultipartForm != nil && r.Request.MultipartForm.File != nil {
		for key, fh := range r.Request.MultipartForm.File {
			if len(fh) > 0 {
				r.files[key] = &File{fh[0]}
			}
		}
	}
	return r.files, nil
}

// All get all of the input and query for the request.
func (r *Request) All(keys ...string) map[string]string {
	all := mergeForm(r.query, r.post)

	if len(keys) == 0 {
		return all
	}

	result := make(map[string]string)

	for _, key := range keys {
		if v, ok := all[key]; ok {
			result[key] = v
		} else {
			result[key] = ""
		}
	}

	return result
}

//Only get a subset of the items from the input data.
func (r *Request) Only(keys ...string) map[string]string {
	all := r.All()

	result := make(map[string]string)

	for _, key := range keys {
		if v, ok := all[key]; ok {
			result[key] = v
		}
	}

	return result
}

// Except Get all of the input except for a specified array of items.
func (r *Request) Except(keys ...string) map[string]string {
	all := r.All()

	for _, key := range keys {
		delete(all, key)
	}

	return all
}

// Has Determine if the request contains a given input item key.
func (r *Request) Exists(keys ...string) bool {
	all := r.All()

	for _, key := range keys {
		if _, ok := all[key]; !ok {
			return false
		}
	}

	return true
}

// Filled Determine if the request contains a non-empty value for an input item.
func (r *Request) Has(keys ...string) bool {
	all := r.All()

	for _, key := range keys {
		if _, ok := all[key]; !ok {
			return false
		}
		if len(all[key]) == 0 {
			return false
		}
	}

	return true
}

//Url get the URL (no query string) for the request.
func (r *Request) Url() string {
	return r.Request.URL.Path
}

// FullUrl get the full URL for the request.
func (r *Request) FullUrl() string {
	return r.Url() + "?" + r.Request.URL.RawQuery
}

// Path get the current path info for the request.
func (r *Request) Path() string {
	return r.path
}

// Method get the current method for the request.
func (r *Request) Method() string {
	return r.method
}

// GetContent Returns the request body content.
func (r *Request) GetContent() ([]byte, error) {
	if r.bodyContent != nil {
		return r.bodyContent, nil
	}

	if r.Request == nil || r.Request.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(r.Request.Body)
	if err != nil {
		return nil, err
	}

	r.bodyContent = body
	r.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	return body, nil
}

// Session get the session associated with the request.
func (r *Request) Session() Session {
	return r.session
}

// Session set the session associated with the request.
func (r *Request) SetSession(s Session) {
	r.session = s
}

func parseQuery(q url.Values) map[string]string {
	query := make(map[string]string)
	for k, v := range q {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}
	return query
}

func parsePost(r *http.Request) (url.Values, map[string]string) {
	postMap := make(map[string]string)
	if r == nil {
		return nil, postMap
	}

	_ = r.ParseForm()
	values := url.Values{}

	for k, v := range r.PostForm {
		values[k] = append(values[k], v...)
		if len(v) > 0 {
			postMap[k] = v[0]
		}
	}

	_ = r.ParseMultipartForm(32 << 20)
	if r.MultipartForm != nil {
		for k, v := range r.MultipartForm.Value {
			values[k] = append(values[k], v...)
			if len(v) > 0 {
				postMap[k] = v[0]
			}
		}
	}

	return values, postMap
}

func mergeForm(slices ...map[string]string) map[string]string {
	r := make(map[string]string)

	for _, slice := range slices {
		for k, v := range slice {
			r[k] = v
		}
	}

	return r
}
