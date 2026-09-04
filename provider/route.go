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

// Register registers the router into the container.
func (p *RoutingServiceProvider) Register(app *container.Container) {
	opts := []flow.Option{
		flow.WithParameterResolver(p.parameterResolver(app)),
	}
	if cfg := app.Make[contract.Config](); cfg != nil {
		if key := cfg.GetString("app.key"); key != "" {
			opts = append(opts, flow.WithSignatureKey(key))
		}
	}

	r := flow.New(opts...)

	app.Instance[flow.Router](r)
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

// Boot boots the provider and registers all route collections.
func (p *RoutingServiceProvider) Boot(app *container.Container) {
	r := app.Make[flow.Router]()
	if r != nil {
		r.Register()
	}
}
