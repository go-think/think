package router

import (
	"net/http"

	"github.com/go-think/think/context"
)

// Response an HTTP response interface
type Response interface {
	Send(w http.ResponseWriter)
}

// Closure Anonymous function, Used in Middleware Handler
type Closure func(req *context.Request) interface {
}

// MiddlewareFunc Handle an incoming request.
type Middleware func(request *context.Request, next Closure) interface{}
