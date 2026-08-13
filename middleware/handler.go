package middleware

import (
	"github.com/go-think/think/context"
)

// Closure Anonymous function, Used in Middleware Handler
type Closure func(req *context.Request) interface{}

// Handler Middleware Handler interface
type Handler interface {
	Process(request *context.Request, next Closure) interface{}
}
