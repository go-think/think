package router

import (
	"github.com/go-think/think/context"
)

// RunRoute Return the response for the given rule.
func RunRoute(request *context.Request, rule *Rule, params ...[]*parameter) interface{} {
	return PrepareResponse(
		request,
		rule,
		runMiddlewares(request, rule, params...),
	)
}

// PrepareResponse Create a response instance from the given value.
func PrepareResponse(request *context.Request, rule *Rule, result interface{}) interface{} {
	return result
}

// runMiddlewares Run the given route within Middlewares instance.
func runMiddlewares(request *context.Request, rule *Rule, params ...[]*parameter) interface{} {
	pipeline := NewPipeline()

	for _, m := range rule.GatherRouteMiddleware() {
		pipeline.Pipe(m)
	}
	return pipeline.Passable(request).Run(func(request *context.Request, next Closure) interface{} {
		return PrepareResponse(
			request,
			rule,
			rule.Run(request, params...),
		)
	})
}
