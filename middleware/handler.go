package middleware

import (
	"github.com/go-think/think/flow"
)

// Closure Anonymous function, Used in Middleware Handler
type Closure func(req *flow.Request) interface{}

// Handler Middleware Handler interface
type Handler interface {
	Process(request *flow.Request, next Closure) interface{}
}

// HandlerFunc type is an adapter to allow the use of ordinary functions as HTTP middleware.
type HandlerFunc func(request *flow.Request, next Closure) interface{}

// Process calls f(request, next).
func (f HandlerFunc) Process(request *flow.Request, next Closure) interface{} {
	return f(request, next)
}

// Terminable Terminable Middleware interface
type Terminable interface {
	Terminate(request *flow.Request, response interface{})
}
