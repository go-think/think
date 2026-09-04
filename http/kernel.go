package http

import (
	"net/http"
	"sync"

	"github.com/go-think/flow"
	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
)

type Kernel struct {
	sync.RWMutex
	app              *container.Container
	globalMiddleware []interface{}
	routeMiddleware  map[string]interface{}
	middlewareGroups map[string][]interface{}
	bootstrapOnce    sync.Once
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
	k.bootstrapOnce.Do(func() {
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

		if r := k.app.Make[flow.Router](); r != nil {
			k.RLock()
			for name, m := range k.routeMiddleware {
				r.AliasMiddleware(name, m)
			}
			for name, g := range k.middlewareGroups {
				r.MiddlewareGroup(name, g...)
			}
			k.RUnlock()
		}
	})
}

func (k *Kernel) Handle(request interface{}) interface{} {
	// 1. Bootstrap application for HTTP request
	k.Bootstrap()

	req, ok := request.(*flow.Request)
	if !ok {
		return nil
	}

	pipe := flow.NewPipeline()

	// Append Global Middlewares (we need to cast them to flow.Handler)
	k.RLock()
	global := make([]interface{}, len(k.globalMiddleware))
	copy(global, k.globalMiddleware)
	k.RUnlock()

	for _, m := range global {
		if md, ok := m.(flow.Handler); ok {
			pipe.Pipe(md)
		}
	}

	// Dispatch to router
	if r := k.app.Make[flow.Router](); r != nil {
		pipe.Pipe(flow.NewRouteMiddleware(r))
	}

	result := pipe.Send(req).Then(nil)

	return result
}

func (k *Kernel) Terminate(req *flow.Request, response interface{}) {
	// Execute global and route terminable middlewares asynchronously
	go func() {
		defer func() {
			if err := recover(); err != nil {
				if logger := k.app.Make[contract.Logger](); logger != nil {
					logger.Error("Kernel Terminate panic: %v", err)
				}
			}
		}()

		// Global Middlewares
		k.RLock()
		global := make([]interface{}, len(k.globalMiddleware))
		copy(global, k.globalMiddleware)
		k.RUnlock()

		for _, m := range global {
			if terminable, ok := m.(flow.Terminable); ok {
				terminable.Terminate(req, response)
			}
		}

		// Route Middlewares
		for _, m := range req.RouteMiddlewares() {
			if terminable, ok := m.(flow.Terminable); ok {
				terminable.Terminate(req, response)
			}
		}

		// Dispatch kernel.terminating event
		if dispatcher := k.app.Make[contract.EventDispatcher](); dispatcher != nil {
			dispatcher.Dispatch("kernel.terminating", req)
		}
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
	k.Lock()
	defer k.Unlock()
	k.globalMiddleware = append(k.globalMiddleware, middleware...)
}

func (k *Kernel) AddRouteMiddleware(name string, middleware interface{}) {
	k.Lock()
	k.routeMiddleware[name] = middleware
	k.Unlock()

	if r := k.app.Make[flow.Router](); r != nil {
		r.AliasMiddleware(name, middleware)
	}
}

func (k *Kernel) GetRouteMiddleware(name string) interface{} {
	k.RLock()
	defer k.RUnlock()
	if m, ok := k.routeMiddleware[name]; ok {
		return m
	}
	return nil
}

func (k *Kernel) AddMiddlewareGroup(name string, middlewares []interface{}) {
	k.Lock()
	k.middlewareGroups[name] = middlewares
	k.Unlock()

	if r := k.app.Make[flow.Router](); r != nil {
		r.MiddlewareGroup(name, middlewares...)
	}
}

func (k *Kernel) GetMiddlewareGroup(name string) []interface{} {
	k.RLock()
	defer k.RUnlock()
	if m, ok := k.middlewareGroups[name]; ok {
		return m
	}
	return nil
}

func (k *Kernel) GetGlobalMiddleware() []interface{} {
	k.RLock()
	defer k.RUnlock()
	copied := make([]interface{}, len(k.globalMiddleware))
	copy(copied, k.globalMiddleware)
	return copied
}
