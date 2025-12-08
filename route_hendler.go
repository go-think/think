package thinkgo

import (
	"github.com/go-think/think/context"
	"github.com/go-think/think/router"
)

type RouteHandler struct {
	Route *router.Route
}

// NewRouteHandler The default RouteHandler
func NewRouteHandler(app *Application) Handler {
	return &RouteHandler{
		Route: app.GetRoute(),
	}
}

// Process Process the request to a router and return the response.
func (h *RouteHandler) Process(request *context.Request, next Closure) interface{} {
	rule, err := h.Route.Dispatch(request)

	if err != nil {
		return context.NotFoundResponse()
	}

	return router.RunRoute(request, rule)
}
