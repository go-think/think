package provider

import (
	"reflect"

	"github.com/go-think/flow"
	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
)

// RoutingServiceProvider registers the core router services into the container.
type RoutingServiceProvider struct {
}

// RouteServiceProvider is an alias to RoutingServiceProvider for backward compatibility.
type RouteServiceProvider = RoutingServiceProvider

// flowContainerAdapter adapts think's container.Container to flow.Container.
type flowContainerAdapter struct {
	c *container.Container
}

type flowExceptionHandlerAdapter struct {
	handler contract.ExceptionHandler
}

func (a *flowExceptionHandlerAdapter) Report(err any) {
	a.handler.Report(err)
}

func (a *flowExceptionHandlerAdapter) Render(req *flow.Request, err any) any {
	return a.handler.Render(err)
}

func (a *flowContainerAdapter) Make(key string) any {
	if key == "ExceptionHandler" {
		if h := a.c.MakeByName("ExceptionHandler"); h != nil {
			if eh, ok := h.(flow.ExceptionHandler); ok {
				return eh
			}
			if eh, ok := h.(contract.ExceptionHandler); ok {
				return &flowExceptionHandlerAdapter{handler: eh}
			}
		}
	}
	return a.c.MakeByName(key)
}

func (a *flowContainerAdapter) Bound(key string) bool {
	if key == "ExceptionHandler" {
		return a.c.Bound("ExceptionHandler")
	}
	return a.c.Bound(key)
}

func (a *flowContainerAdapter) Instance(key string, instance any) {
	a.c.InstanceNamed(key, instance)
}

// Register registers the router into the container.
func (p *RoutingServiceProvider) Register(app *container.Container) {
	r := flow.New(nil, &flowContainerAdapter{c: app})
	if setter, ok := r.(interface{ SetParameterResolver(flow.ParameterResolver) }); ok {
		setter.SetParameterResolver(p.parameterResolver(app))
	}

	generator := flow.NewUrlGenerator(r, "").SetKeyResolver(func() []string {
		cfg := app.Make[contract.Config]()
		if cfg == nil {
			return nil
		}
		return append(
			[]string{cfg.GetString("app.key")},
			cfg.GetStringSlice("app.previous_keys")...,
		)
	})

	generator.SetRequestProvider(func() *flow.Request { return r.CurrentRequest() })

	// Route lifecycle events flow through the application event dispatcher.
	r.OnRouting(func(req *flow.Request) {
		if dispatcher := app.Make[contract.EventDispatcher](); dispatcher != nil {
			dispatcher.Dispatch("flow.routing", req)
		}
	})
	r.OnRouteMatched(func(route *flow.Route, req *flow.Request) {
		if dispatcher := app.Make[contract.EventDispatcher](); dispatcher != nil {
			dispatcher.Dispatch("flow.route.matched", route)
		}
	})

	app.Instance[flow.Router](r)
	app.Instance[*flow.UrlGenerator](generator)
	app.Instance[*flow.Redirector](flow.NewRedirector(generator).SetRequestProvider(func() *flow.Request { return r.CurrentRequest() }))
	app.Alias[flow.Router]("router")
}

// parameterResolver creates a parameter resolver backed by the application container.
func (p *RoutingServiceProvider) parameterResolver(app *container.Container) flow.ParameterResolver {
	return flow.ParameterResolverFunc(func(t reflect.Type, req *flow.Request) (reflect.Value, bool) {
		if dep := app.Get(t); dep != nil {
			return reflect.ValueOf(dep), true
		}
		return reflect.Value{}, false
	})
}

// Boot compiles route rules and validates signing configuration.
// The key itself is resolved lazily by the UrlGenerator's key resolver
// (registered in Register); an unconfigured key leaves signing disabled —
// the generator fails closed.
func (p *RoutingServiceProvider) Boot(app *container.Container) {
	r := app.Make[flow.Router]()
	if r == nil {
		return
	}

	if cfg := app.Make[contract.Config](); cfg != nil && cfg.GetString("app.key") == "" {
		if logger := app.Make[contract.Logger](); logger != nil {
			logger.Error("app.key is not configured: signed URLs are disabled and signature validation always fails. Set APP_KEY / app.key before using signed URLs.")
		}
	}

	if eh := app.Make[contract.ExceptionHandler](); eh != nil {
		r.SetExceptionHandler(&flowExceptionHandlerAdapter{handler: eh})
	}

	r.Register()
}
