package router

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-think/think/flow"
	"github.com/go-think/think/contract"
	"github.com/go-think/think/facades"
)

var verbs = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

type Request interface {
	GetMethod() string
	GetPath() string
}

type Route struct {
	inited bool

	method      []string
	prefix      string
	pattern     string
	handler     interface{}
	middlewares []interface{}
	group       *Route
	name        string
	wheres      map[string]string
	patterns    map[string]string

	collects []*Route

	trees       map[string]*node
	rules       map[string]map[string]*Rule
	allRules    map[string]*Rule
	namedRoutes map[string]*Rule
	fallback    interface{}
}

// New Create a new Route instance.
func New() *Route {
	route := &Route{
		trees:    make(map[string]*node),
		rules:    make(map[string]map[string]*Rule),
		patterns: make(map[string]string),
		wheres:   make(map[string]string),
	}
	return route
}

// Dispatch executes the request and returns the response.
func (r *Route) Dispatch(request *flow.Request) interface{} {
	rule, params, err := r.MatchRequest(request)
	if err != nil {
		if r.fallback != nil {
			fallbackRule := &Rule{
				handler: r.fallback,
			}
			return RunRoute(request, fallbackRule, nil)
		}
		return flow.NotFoundResponse()
	}

	if rule.name != "" {
		request.Set("_route_name", rule.name)
	}

	for _, p := range params {
		request.SetRouteParam(p.name, p.value)
	}
	
	return RunRoute(request, rule, params)
}

// MatchRequest Dispatch the request to find a matching rule
func (r *Route) MatchRequest(request Request) (*Rule, []*parameter, error) {
	rule, treeParams, err := r.Match(request)
	if err != nil {
		return nil, nil, err
	}

	params := rule.Bind(request, request.GetPath(), treeParams)

	return rule, params, nil
}

// Match Find the first rule matching a given request using Radix Tree with fallback.
func (r *Route) Match(request Request) (*Rule, []*parameter, error) {
	method := request.GetMethod()
	path := request.GetPath()

	// Prioritize Radix Tree index
	if tree, ok := r.trees[method]; ok {
		if handle, ps, _ := tree.getValue(path); handle != nil {
			if rule, ok := handle.(*Rule); ok {
				// Sync dynamically parsed parameters from tree to local slice
				ruleParams := make([]*parameter, 0, len(ps))
				for _, p := range ps {
					ruleParams = append(ruleParams, &parameter{
						name:  p.Key,
						value: p.Value,
					})
				}
				if rule.ValidateParams(ruleParams) {
					return rule, ruleParams, nil
				}
			}
		}
	}

	// Fallback to regex rules library
	for _, rule := range r.rules[method] {
		if true == rule.Matches(method, path) {
			return rule, nil, nil
		}
	}
	return nil, nil, errors.New("Not Found")
}

// AddRule Add a Rule to the Router.Rules and Radix Tree
func (r *Route) AddRule(rule *Rule) *Rule {
	domainAndUri := rule.pattern
	for _, method := range rule.method {
		// Add to regex list as fallback
		if _, ok := r.rules[method]; !ok {
			r.rules[method] = map[string]*Rule{
				domainAndUri: rule,
			}
		} else {
			r.rules[method][domainAndUri] = rule
		}

		// Build/Get Radix Tree for Method
		rootNode, ok := r.trees[method]
		if !ok {
			rootNode = &node{}
			r.trees[method] = rootNode
		}
		rootNode.addRoute(domainAndUri, rule)
	}

	if r.allRules == nil {
		r.allRules = map[string]*Rule{}
	}
	r.allRules[strings.Join(rule.method, "|")+domainAndUri] = rule

	return rule
}

// Add Add a router
func (r *Route) Add(method []string, pattern string, handler interface{}) contract.Router {
	route := r.initRoute()
	route.method = method
	route.pattern = r.getPrefix(pattern)
	route.handler = handler
	return route
}

// Get Register a new GET rule with the router.
func (r *Route) Get(pattern string, handler interface{}) contract.Router {
	return r.Add(Method("GET", "HEAD"), pattern, handler)
}

// Head Register a new Head rule with the router.
func (r *Route) Head(pattern string, handler interface{}) contract.Router {
	return r.Add(Method("HEAD"), pattern, handler)
}

// Post Register a new POST rule with the router.
func (r *Route) Post(pattern string, handler interface{}) contract.Router {
	return r.Add(Method("POST"), pattern, handler)
}

// Put Register a new PUT rule with the router.
func (r *Route) Put(pattern string, handler interface{}) contract.Router {
	return r.Add(Method("PUT"), pattern, handler)
}

// Patch Register a new PATCH rule with the router.
func (r *Route) Patch(pattern string, handler interface{}) contract.Router {
	return r.Add(Method("PATCH"), pattern, handler)
}

// Delete Register a new DELETE rule with the router.
func (r *Route) Delete(pattern string, handler interface{}) contract.Router {
	return r.Add(Method("DELETE"), pattern, handler)
}

// Options Register a new OPTIONS rule with the router.
func (r *Route) Options(pattern string, handler interface{}) contract.Router {
	return r.Add(Method("OPTIONS"), pattern, handler)
}

// Any Register a new rule responding to all verbs.
func (r *Route) Any(pattern string, handler interface{}) contract.Router {
	return r.Add(verbs, pattern, handler)
}

// Static Register a new Static rule.
func (r *Route) Static(path, root string) {
	cleanPrefix := "/" + strings.Trim(path, "/")
	wildcardPath := cleanPrefix + "/*"

	h := NewStaticHandle(cleanPrefix, root)

	r.Get(wildcardPath, h)
	r.Head(wildcardPath, h)
}

// Statics Bulk register Static rule.
func (r *Route) Statics(statics map[string]string) {
	for path, root := range statics {
		r.Static(path, root)
	}
}

// Prefix Add a prefix to the route URI.
func (r *Route) Prefix(prefix string) contract.Router {
	route := r.initRoute()
	route.prefix = route.getPrefix(prefix)
	return route
}

// Group Create a route group
func (r *Route) Group(callback func(group contract.Router)) {
	route := r.initRoute()
	group := route.cloneRoute()
	
	var md []interface{}
	for _, m := range route.middlewares {
		md = append(md, m)
	}
	group.Middleware(md...)
	
	callback(group)
}

// Middleware Set the middleware attached to the route.
func (r *Route) Middleware(middlewares ...interface{}) contract.Router {
	route := r.initRoute()
	route.middlewares = append(route.middlewares, middlewares...)
	return route
}

// Name Set the name attached to the route.
func (r *Route) Name(name string) contract.Router {
	route := r.initRoute()
	route.name = r.name + name
	return route
}

// Url generates a URL for a named route.
func (r *Route) Url(name string, params map[string]string) string {
	if r.namedRoutes == nil {
		return ""
	}
	rule, ok := r.namedRoutes[name]
	if !ok {
		return ""
	}
	urlPath := rule.pattern
	for k, v := range params {
		urlPath = strings.Replace(urlPath, "{"+k+"}", v, -1)
	}
	// Clean up any remaining optional parameters e.g., {param?} or {param}
	// A regex could be used here, but string replacement is sufficient for exact matches.
	return urlPath
}

// Fallback registers a fallback route.
func (r *Route) Fallback(handler interface{}) {
	r.fallback = handler
}

// Where adds a regex constraint to a route parameter.
func (r *Route) Where(name string, expression string) contract.Router {
	route := r.initRoute()
	if route.wheres == nil {
		route.wheres = make(map[string]string)
	}
	route.wheres[name] = expression
	return route
}

// WhereNumber adds a numeric regex constraint to parameters.
func (r *Route) WhereNumber(names ...string) contract.Router {
	for _, name := range names {
		r.Where(name, "^[0-9]+$")
	}
	return r
}

// WhereAlpha adds an alphabetic regex constraint to parameters.
func (r *Route) WhereAlpha(names ...string) contract.Router {
	for _, name := range names {
		r.Where(name, "^[a-zA-Z]+$")
	}
	return r
}

// WhereIn adds an allowed values constraint to a parameter.
func (r *Route) WhereIn(name string, allowed []string) contract.Router {
	escaped := make([]string, len(allowed))
	for i, val := range allowed {
		escaped[i] = regexpQuote(val)
	}
	return r.Where(name, "^("+strings.Join(escaped, "|")+")$")
}

// Pattern sets a global regex pattern for a parameter.
func (r *Route) Pattern(name string, expression string) {
	if r.patterns == nil {
		r.patterns = make(map[string]string)
	}
	r.patterns[name] = expression
}

func regexpQuote(s string) string {
	var b strings.Builder
	for _, ch := range s {
		if strings.ContainsRune(`\.+*?()|[]{}^$`, ch) {
			b.WriteRune('\\')
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// Register Register route from the collect.
func (r *Route) Register() {
	r.register(r)
}

func (r *Route) Dump() []byte {
	var b bytes.Buffer
	for _, rule := range r.allRules {
		fmt.Fprintf(&b, "%s %s %T \r\n", strings.Join(rule.method, "|"), rule.pattern, rule.handler)
	}

	return b.Bytes()
}

func (r *Route) register(root *Route) {
	for _, route := range r.collects {
		route.prefix = r.getPrefix(route.prefix)

		var middlewares []interface{}
		for _, m := range r.middlewares {
			middlewares = append(middlewares, m)
		}
		for _, m := range route.middlewares {
			middlewares = append(middlewares, m)
		}
		route.middlewares = middlewares

		route.register(root)

		if route.handler == nil {
			continue
		}
		
		routePattern := route.getPrefix(route.pattern)
		
		combinedWheres := make(map[string]string)
		if root.patterns != nil {
			for k, v := range root.patterns {
				combinedWheres[k] = v
			}
		}
		if route.wheres != nil {
			for k, v := range route.wheres {
				combinedWheres[k] = v
			}
		}

		rule := &Rule{
			name:        route.name,
			method:      route.method,
			pattern:     routePattern,
			handler:     route.handler,
			middlewares: route.middlewares,
			wheres:      combinedWheres,
		}

		root.AddRule(rule)
		if route.name != "" {
			if root.namedRoutes == nil {
				root.namedRoutes = make(map[string]*Rule)
			}
			root.namedRoutes[route.name] = rule
		}
	}
	r.collects = r.collects[0:0]
}

// initRoute Initialize a new Route if not initialized
func (r *Route) initRoute() *Route {
	route := r
	if !r.inited {
		route = &Route{
			inited: true,
			rules:  make(map[string]map[string]*Rule),
			name:   r.name,
			// prefix:      r.prefix,
			// middlewares: r.middlewares,
		}
		r.collects = append(r.collects, route)
	}
	return route
}

func (r *Route) cloneRoute() *Route {
	route := &Route{
		inited: false,
		rules:  make(map[string]map[string]*Rule),
		name:   r.name,
		// prefix:      r.prefix,
		// middlewares: r.middlewares,
	}
	r.collects = append(r.collects, route)

	return route
}

func (r *Route) getPrefix(pattern string) string {
	return path.Join("/", r.prefix, pattern)
}

// SignedUrl creates a signed URL for a named route.
func (r *Route) SignedUrl(name string, expiration time.Duration, params map[string]string) string {
	urlPath := r.Url(name, params)
	if urlPath == "" {
		return ""
	}

	key := getSignatureKey()
	expires := time.Now().Add(expiration).Unix()

	queryVals := url.Values{}
	queryVals.Set("expires", strconv.FormatInt(expires, 10))

	// Build query string sorted
	queryString := queryVals.Encode()
	fullUrl := urlPath + "?" + queryString

	// Calculate HMAC signature
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(fullUrl))
	sig := hex.EncodeToString(mac.Sum(nil))

	return fullUrl + "&signature=" + sig
}

// HasValidSignature checks if the given request has a valid signature.
func (r *Route) HasValidSignature(req *flow.Request) bool {
	if req == nil || req.Request == nil || req.Request.URL == nil {
		return false
	}

	sig, err := req.Query("signature")
	if err != nil || sig == "" {
		return false
	}

	expiresStr, err := req.Query("expires")
	if err != nil || expiresStr == "" {
		return false
	}

	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil || time.Now().Unix() > expires {
		return false
	}

	// Recreate query string without signature
	u := req.Request.URL
	rawQuery := u.Query()
	rawQuery.Del("signature")

	keys := make([]string, 0, len(rawQuery))
	for k := range rawQuery {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		for _, v := range rawQuery[k] {
			pairs = append(pairs, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}

	target := u.Path + "?" + strings.Join(pairs, "&")
	key := getSignatureKey()

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(target))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

// Has determines if the route collection contains a given named route.
func (r *Route) Has(name string) bool {
	if r.namedRoutes == nil {
		return false
	}
	_, ok := r.namedRoutes[name]
	return ok
}

// CurrentRouteName returns the current route name for the request.
func (r *Route) CurrentRouteName(req *flow.Request) string {
	if req == nil {
		return ""
	}
	if nameVal, ok := req.Get("_route_name"); ok {
		if name, ok := nameVal.(string); ok {
			return name
		}
	}
	return ""
}

// Is determines if the current route's name matches given patterns.
func (r *Route) Is(req *flow.Request, patterns ...string) bool {
	currentName := r.CurrentRouteName(req)
	if currentName == "" {
		return false
	}
	for _, pattern := range patterns {
		if pattern == currentName {
			return true
		}
		if strings.Contains(pattern, "*") {
			pat := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*") + "$"
			if matched, _ := regexp.MatchString(pat, currentName); matched {
				return true
			}
		}
	}
	return false
}

func getSignatureKey() string {
	if facades.App != nil {
		if cfg := facades.Config(); cfg != nil {
			if k := cfg.GetString("app.key"); k != "" {
				return k
			}
		}
	}
	return "thinkgo-default-secret-signature-key"
}
