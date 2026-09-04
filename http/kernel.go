package http

import (
	"net/http"

	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
	"github.com/go-think/think/flow"
	"github.com/go-think/think/middleware"
	"github.com/go-think/think/pipeline"
	"github.com/go-think/think/router"
)

type Kernel struct {
	app              *container.Container
	globalMiddleware []interface{}
	routeMiddleware  map[string]interface{}
	middlewareGroups map[string][]interface{}
}

func NewKernel(app *container.Container) contract.HttpKernel {
	return &Kernel{
		app:              app,
		globalMiddleware: make([]interface{}, 0),
		routeMiddleware:  make(map[string]interface{}),
		middlewareGroups: make(map[string][]interface{}),
	}
}

// Bootstrap bootstraps the application for HTTP requests.
func (k *Kernel) Bootstrap() {
	if app := k.app.Make[contract.Application](); app != nil {
		if b, ok := app.(interface {
			HasBeenBootstrapped() bool
			Bootstrap()
			Boot()
		}); ok {
			if !b.HasBeenBootstrapped() {
				b.Bootstrap()
			}
			b.Boot()
		}
	}
}

func (k *Kernel) Handle(request interface{}) interface{} {
	// 1. Bootstrap application for HTTP request
	k.Bootstrap()

	req, ok := request.(*flow.Request)
	if !ok {
		return nil
	}

	pipe := pipeline.NewPipeline()

	// Append Global Middlewares (we need to cast them to middleware.Handler)
	for _, m := range k.globalMiddleware {
		if md, ok := m.(middleware.Handler); ok {
			pipe.Pipe(md)
		}
	}

	// Dispatch to router
	if r := k.app.Make[*router.Route](); r != nil {
		pipe.Pipe(middleware.NewRouteHandler(r))
	}

	result := pipe.Run(req)

	return result
}

func (k *Kernel) Terminate(req *flow.Request, response interface{}) {
	// Execute global and route terminable middlewares asynchronously
	go func() {
		// Global Middlewares
		for _, m := range k.globalMiddleware {
			if terminable, ok := m.(middleware.Terminable); ok {
				terminable.Terminate(req, response)
			}
		}

		// Route Middlewares
		for _, m := range req.RouteMiddlewares() {
			if terminable, ok := m.(middleware.Terminable); ok {
				terminable.Terminate(req, response)
			}
		}

		// Dispatch kernel.terminating event
		if dispatcher := k.app.Make[contract.EventDispatcher](); dispatcher != nil {
			dispatcher.Dispatch("kernel.terminating", req)
		}

		// Flush request-scoped container instances
		k.app.FlushScoped()
	}()
}

func (k *Kernel) ServeHTTP(w interface{}, r interface{}) {
	writer, ok1 := w.(http.ResponseWriter)
	request, ok2 := r.(*http.Request)
	if !ok1 || !ok2 {
		return
	}

	ctxReq := flow.NewRequest(request)
	ctxReq.SetResponseWriter(writer)
	ctxReq.CookieHandler = flow.ParseCookieHandler()

	result := k.Handle(ctxReq)

	if result != nil {
		if res, ok := result.(*flow.Response); ok {
			res.Send(writer)
		} else {
			flow.NewResponse().SetContent(flow.FormatContent(result)).Send(writer)
		}
	}
	
	// Dispatch kernel.handled event
	if dispatcher := k.app.Make[contract.EventDispatcher](); dispatcher != nil {
		dispatcher.Dispatch("kernel.handled", ctxReq)
	}
	
	k.Terminate(ctxReq, result)
}

func (k *Kernel) AddGlobalMiddleware(middleware ...interface{}) {
	k.globalMiddleware = append(k.globalMiddleware, middleware...)
}

func (k *Kernel) AddRouteMiddleware(name string, middleware interface{}) {
	k.routeMiddleware[name] = middleware
}

func (k *Kernel) GetRouteMiddleware(name string) interface{} {
	if m, ok := k.routeMiddleware[name]; ok {
		return m
	}
	return nil
}

func (k *Kernel) AddMiddlewareGroup(name string, middlewares []interface{}) {
	k.middlewareGroups[name] = middlewares
}

func (k *Kernel) GetMiddlewareGroup(name string) []interface{} {
	if m, ok := k.middlewareGroups[name]; ok {
		return m
	}
	return nil
}

func (k *Kernel) GetGlobalMiddleware() []interface{} {
	return k.globalMiddleware
}
