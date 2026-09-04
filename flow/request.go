package flow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Request HTTP request
type Request struct {
	Request          *http.Request
	ctx              context.Context
	method           string
	path             string
	query            map[string]string
	queryValues      url.Values
	post             map[string]string
	postValues       url.Values
	routeParams      map[string]string
	routeParamsMu    sync.RWMutex
	routeMiddlewares []interface{}
	files            map[string]*File
	bodyContent      []byte
	session          Session
	CookieHandler    *Cookie
	writer           http.ResponseWriter
	keys             map[string]interface{}
	keysMu           sync.RWMutex
	parseOnce        sync.Once
}

// ResponseWriter returns the native http.ResponseWriter associated with this request.
func (r *Request) ResponseWriter() http.ResponseWriter {
	return r.writer
}

// SetResponseWriter sets the native http.ResponseWriter for direct writing.
func (r *Request) SetResponseWriter(w http.ResponseWriter) *Request {
	r.writer = w
	return r
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

// SetRouteParam sets a route parameter by name
func (r *Request) SetRouteParam(name, value string) {
	r.routeParamsMu.Lock()
	defer r.routeParamsMu.Unlock()
	r.routeParams[name] = value
}

// RouteMiddlewares Returns the route middlewares matched for the request.
func (r *Request) RouteMiddlewares() []interface{} {
	return r.routeMiddlewares
}

// SetRouteMiddlewares Sets the route middlewares matched for the request.
func (r *Request) SetRouteMiddlewares(middlewares []interface{}) {
	r.routeMiddlewares = append(r.routeMiddlewares, middlewares...)
}

// GetRouteParam gets a route parameter
func (r *Request) GetRouteParam(key string, defaultValue ...string) string {
	r.routeParamsMu.RLock()
	defer r.routeParamsMu.RUnlock()
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
	r.routeParamsMu.RLock()
	defer r.routeParamsMu.RUnlock()
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
			
			// Parse JSON body if applicable
			if strings.Contains(r.Request.Header.Get("Content-Type"), "application/json") {
				body, _ := r.GetContent()
				if len(body) > 0 {
					var jsonData map[string]interface{}
					if err := json.Unmarshal(body, &jsonData); err == nil {
						for k, v := range jsonData {
							r.post[k] = fmt.Sprintf("%v", v)
						}
					}
				}
			}
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

// HasFile determines if the uploaded data contains a file.
func (r *Request) HasFile(key string) bool {
	if _, ok := r.files[key]; ok {
		return true
	}
	if r.Request == nil {
		return false
	}
	_, _, err := r.Request.FormFile(key)
	return err == nil
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
	r.parseInputOnce()
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

// Header returns the value of the given header key.
func (r *Request) Header(key string) string {
	if r.Request != nil {
		return r.Request.Header.Get(key)
	}
	return ""
}

// BearerToken returns the Bearer token from the Authorization header.
func (r *Request) BearerToken() string {
	auth := r.Header("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}

// ClientIP returns the client IP.
func (r *Request) ClientIP() string {
	if r.Request == nil {
		return ""
	}
	if ip := r.Header("X-Real-Ip"); ip != "" {
		return ip
	}
	if ip := r.Header("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	
	// RemoteAddr could be IP:port
	addr := r.Request.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// WantsJson returns true if the request asks for a JSON response.
func (r *Request) WantsJson() bool {
	accept := r.Header("Accept")
	return strings.Contains(accept, "/json") || strings.Contains(accept, "+json")
}

// ExpectsJson returns true if the request expects a JSON response.
func (r *Request) ExpectsJson() bool {
	return r.IsAjax() || r.WantsJson()
}

// IsAjax returns true if the request is an AJAX request.
func (r *Request) IsAjax() bool {
	return r.Header("X-Requested-With") == "XMLHttpRequest"
}

// Boolean retrieves an input item as a boolean.
func (r *Request) Boolean(key string, defaultValue ...bool) bool {
	val, err := r.Input(key)
	if err != nil || val == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return false
	}

	lower := strings.ToLower(strings.TrimSpace(val))
	switch lower {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	}

	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return false
}

// Integer retrieves an input item as an integer.
func (r *Request) Integer(key string, defaultValue ...int) int {
	val, err := r.Input(key)
	if err != nil || val == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	intVal, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0
	}

	return intVal
}

// Float retrieves an input item as a float64.
func (r *Request) Float(key string, defaultValue ...float64) float64 {
	val, err := r.Input(key)
	if err != nil || val == "" {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0.0
	}

	floatVal, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
	if err != nil {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return 0.0
	}

	return floatVal
}

// Merge merges new input into the current request's user input.
func (r *Request) Merge(values map[string]string) *Request {
	r.parseInputOnce()
	if r.post == nil {
		r.post = make(map[string]string)
	}
	for k, v := range values {
		r.post[k] = v
		if r.query != nil {
			if _, ok := r.query[k]; ok {
				r.query[k] = v
			}
		}
	}
	return r
}

// Fingerprint gets a unique fingerprint for the request.
func (r *Request) Fingerprint() string {
	ip := r.ClientIP()
	ua := r.Header("User-Agent")
	route := r.GetPath()
	method := r.GetMethod()

	hash := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s", method, route, ip, ua)))
	return hex.EncodeToString(hash[:])
}

// UserAgent returns the client User-Agent header.
func (r *Request) UserAgent() string {
	return r.Header("User-Agent")
}

// Segments gets all segments of the request path.
func (r *Request) Segments() []string {
	trimmed := strings.Trim(r.Path(), "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

// Segment gets a 1-indexed segment of the path.
func (r *Request) Segment(index int, defaultValue ...string) string {
	segments := r.Segments()
	if index > 0 && index <= len(segments) {
		return segments[index-1]
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

// Is determines if the current request path matches given patterns.
func (r *Request) Is(patterns ...string) bool {
	path := strings.Trim(r.Path(), "/")
	for _, pattern := range patterns {
		pattern = strings.Trim(pattern, "/")
		if pattern == path {
			return true
		}
		if strings.Contains(pattern, "*") {
			pat := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*") + "$"
			if matched, _ := regexp.MatchString(pat, path); matched {
				return true
			}
		}
	}
	return false
}

// RouteIs determines if the current route name matches given patterns.
func (r *Request) RouteIs(patterns ...string) bool {
	routeName := ""
	if nameVal, ok := r.Get("_route_name"); ok {
		if nameStr, ok := nameVal.(string); ok {
			routeName = nameStr
		}
	}
	if routeName == "" {
		return false
	}
	for _, pattern := range patterns {
		if pattern == routeName {
			return true
		}
		if strings.Contains(pattern, "*") {
			pat := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*") + "$"
			if matched, _ := regexp.MatchString(pat, routeName); matched {
				return true
			}
		}
	}
	return false
}

// FullUrlWithQuery appends or replaces query parameters to current full URL.
func (r *Request) FullUrlWithQuery(query map[string]string) string {
	if r.Request == nil || r.Request.URL == nil {
		return ""
	}
	q := r.Request.URL.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	return r.Url() + "?" + q.Encode()
}

// FullUrlWithoutQuery removes specified query parameters from current full URL.
func (r *Request) FullUrlWithoutQuery(keys ...string) string {
	if r.Request == nil || r.Request.URL == nil {
		return ""
	}
	q := r.Request.URL.Query()
	for _, k := range keys {
		q.Del(k)
	}
	encoded := q.Encode()
	if encoded == "" {
		return r.Url()
	}
	return r.Url() + "?" + encoded
}
