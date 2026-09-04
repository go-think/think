package facades

import (
	"github.com/go-think/cache"
	"github.com/go-think/flow"
	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
)

// App is the global application instance holder for facades.
var App contract.Application

// Container returns the concrete *container.Container from the global App.
// This enables Go 1.27 method generics (Resolve[T], SingletonFunc[T], etc.)
func Container() *container.Container {
	if App == nil {
		panic("facades: App container is not initialized")
	}
	// Application embeds *container.Container, extract it via interface check
	type containerProvider interface {
		GetContainer() *container.Container
	}
	if cp, ok := App.(containerProvider); ok {
		return cp.GetContainer()
	}
	panic("facades: App does not provide concrete container access")
}

// Config returns the global Config instance.
func Config() contract.Config {
	return Container().Make[contract.Config]()
}

// Route returns the global Router instance.
func Route() flow.Router {
	return Container().Make[flow.Router]()
}

// Event returns the global EventDispatcher instance.
func Event() contract.EventDispatcher {
	return Container().Make[contract.EventDispatcher]()
}

// Log returns the global Logger instance.
func Log() contract.Logger {
	return Container().Make[contract.Logger]()
}

// Cache returns the global Cache repository instance.
func Cache() *cache.Repository {
	return Container().Make[*cache.Repository]()
}
