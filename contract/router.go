package contract

import "github.com/go-think/think/flow"

// Router defines the interface for the routing system.
type Router interface {
	// Add registers a new route.
	Add(method []string, pattern string, handler interface{}) Router
	// Get registers a GET route.
	Get(pattern string, handler interface{}) Router
	// Post registers a POST route.
	Post(pattern string, handler interface{}) Router
	// Put registers a PUT route.
	Put(pattern string, handler interface{}) Router
	// Patch registers a PATCH route.
	Patch(pattern string, handler interface{}) Router
	// Delete registers a DELETE route.
	Delete(pattern string, handler interface{}) Router
	// Options registers an OPTIONS route.
	Options(pattern string, handler interface{}) Router
	// Any registers a route responding to all standard verbs.
	Any(pattern string, handler interface{}) Router
	// Group creates a route group.
	Group(callback func(group Router))
	// Prefix adds a prefix to the current route group.
	Prefix(prefix string) Router
	// Middleware adds middleware to the current route or group.
	Middleware(middlewares ...interface{}) Router
	// Dispatch resolves the request to a handler and executes it.
	Dispatch(request *flow.Request) interface{}
	// Name names the route.
	Name(name string) Router
	// Url generates a URL for a named route.
	Url(name string, params map[string]string) string
	// Fallback registers a fallback route.
	Fallback(handler interface{})
	// Where adds a regex constraint to a route parameter.
	Where(name string, expression string) Router
	// WhereNumber adds a numeric regex constraint to parameters.
	WhereNumber(names ...string) Router
	// WhereAlpha adds an alphabetic regex constraint to parameters.
	WhereAlpha(names ...string) Router
	// WhereIn adds an allowed values constraint to a parameter.
	WhereIn(name string, allowed []string) Router
	// Has determines if the route collection contains a given named route.
	Has(name string) bool
	// CurrentRouteName returns the current route name for the request.
	CurrentRouteName(req *flow.Request) string
	// Is determines if the current route's name matches given patterns.
	Is(req *flow.Request, patterns ...string) bool
}
