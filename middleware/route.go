package middleware

import (
	"github.com/go-think/think/flow"
	"github.com/go-think/think/router"
)

type RouteHandler struct {
	Route *router.Route
}

// NewRouteHandler The default RouteHandler
func NewRouteHandler(r *router.Route) Handler {
	return &RouteHandler{
		Route: r,
	}
}

// Process Process the request to a router and return the response.
func (h *RouteHandler) Process(request *flow.Request, next Closure) interface{} {
	return h.Route.Dispatch(request)
}
