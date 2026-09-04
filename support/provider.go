package support

import (
	"github.com/go-think/think/container"
)

// ServiceProvider defines the interface for all framework service providers.
type ServiceProvider interface {
	// Register is used to bind things into the container.
	Register(app *container.Container)

	// Boot is called after all providers have been registered,
	// meaning you have access to all other services.
	Boot(app *container.Container)
}
