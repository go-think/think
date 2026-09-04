package router

import (
	"net/http"

	"github.com/go-think/think/flow"
)

// Response an HTTP response interface
type Response interface {
	Send(w http.ResponseWriter)
}

// Closure Anonymous function, Used in Middleware Handler
type Closure func(req *flow.Request) interface {
}

// Middleware Handle an incoming request.
type Middleware func(request *flow.Request, next Closure) interface{}

// ParameterizedMiddleware Handle an incoming request with parameters.
type ParameterizedMiddleware func(params ...string) Middleware
