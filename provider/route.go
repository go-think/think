package provider

import (
	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
	"github.com/go-think/think/router"
)

// RoutingServiceProvider registers the core router services into the container.
type RoutingServiceProvider struct {
}

// RouteServiceProvider is an alias to RoutingServiceProvider for backward compatibility.
type RouteServiceProvider = RoutingServiceProvider

// Register registers the router into the container.
func (p *RoutingServiceProvider) Register(app *container.Container) {
	r := router.New()

	app.Instance[*router.Route](r)
	app.Instance[contract.Router](r)
	app.Alias("router", "Route")
}

// Boot boots the provider and registers all route collections.
func (p *RoutingServiceProvider) Boot(app *container.Container) {
	r := app.Make[*router.Route]()
	r.Register()
}
